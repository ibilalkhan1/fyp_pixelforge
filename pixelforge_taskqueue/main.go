// Package pixelforge_taskqueue implements the Work-Stealing Priority
// Job System that schedules player input, physics, and enemy AI
// across a pool of worker goroutines. Player input is treated as a
// high-priority real-time constraint, while AI pathfinding is a
// lower-priority task that can be time-sliced across multiple
// frames. When the main thread blocks, idle workers steal jobs
// through Go channels, keeping the frame rate stable.
package pixelforge_taskqueue

import (
	"container/heap"
	"runtime"
	"sync"
)

// Priority defines the three execution tiers used by the engine.
type Priority int

const (
	// PriorityTop is reserved for Input Polling and Audio/Video
	// buffer flips. These jobs must complete before the next V-Blank
	// or the player will perceive lag.
	PriorityTop Priority = iota

	// PriorityMiddle covers Game State transitions and Physics
	// simulation. Missing a middle-priority deadline produces a
	// visible stutter but does not break responsiveness.
	PriorityMiddle

	// PriorityBottom is for Enemy AI tracking, pathfinding, and
	// ambient world simulation. These jobs can be deferred or
	// split across several frames without the player noticing.
	PriorityBottom
)

// Job is the atomic unit of work. It carries a priority, an
// identifier for profiling, and the closure to execute.
type Job struct {
	ID       string
	Priority Priority
	Fn       func()

	// wg is injected by Fork so that the join phase knows when
	// every sub-task has finished.
	wg *sync.WaitGroup
}

// jobHeap implements heap.Interface so that the dispatcher always
// pulls the highest-priority task first.
type jobHeap []*Job

func (h jobHeap) Len() int           { return len(h) }
func (h jobHeap) Less(i, j int) bool { return h[i].Priority < h[j].Priority }
func (h jobHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *jobHeap) Push(x interface{}) { *h = append(*h, x.(*Job)) }
func (h *jobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// TaskQueue is the central scheduler. It owns the multilevel
// priority heap, one local queue per worker, and the global steal
// channel that workers use to rescue jobs when the main thread
// blocks.
type TaskQueue struct {
	mu            sync.Mutex
	priorityQueue *jobHeap

	// mainQueue is the primary execution channel for the main
	// (render) thread.
	mainQueue chan *Job

	// workerQueues holds one buffered channel per worker goroutine.
	workerQueues []chan *Job

	// steal is the overflow channel. When the main thread is
	// saturated, new jobs are pushed here so that idle workers
	// can pick them up without waiting.
	steal chan *Job

	workerCount int
	quit        chan struct{}
	wg          sync.WaitGroup
}

// NewTaskQueue creates a scheduler sized to the host CPU count.
func NewTaskQueue() *TaskQueue {
	n := runtime.NumCPU()
	pq := &jobHeap{}
	heap.Init(pq)
	tq := &TaskQueue{
		priorityQueue: pq,
		mainQueue:     make(chan *Job, 128),
		workerQueues:  make([]chan *Job, n),
		steal:         make(chan *Job, n*4),
		workerCount:   n,
		quit:          make(chan struct{}),
	}
	for i := 0; i < n; i++ {
		tq.workerQueues[i] = make(chan *Job, 64)
	}
	return tq
}

// Start spawns the worker goroutine pool. Each worker owns one
// local queue and participates in the work-stealing protocol.
func (tq *TaskQueue) Start() {
	for i := 0; i < tq.workerCount; i++ {
		tq.wg.Add(1)
		go tq.runWorker(i)
	}
}

// runWorker is the lifetime of a single goroutine in the pool.
// It services its local queue, the global steal channel, and
// finally attempts to steal from the main thread or sibling
// workers when idle.
func (tq *TaskQueue) runWorker(id int) {
	defer tq.wg.Done()
	local := tq.workerQueues[id]
	for {
		select {
		case job := <-local:
			tq.execute(job)
		case job := <-tq.steal:
			tq.execute(job)
		case <-tq.quit:
			return
		default:
			if stolen := tq.stealWork(id); stolen != nil {
				tq.execute(stolen)
			}
		}
	}
}

// execute runs the job closure and signals completion to the
// Fork-Join WaitGroup if one is attached.
func (tq *TaskQueue) execute(job *Job) {
	job.Fn()
	if job.wg != nil {
		job.wg.Done()
	}
}

// stealWork attempts to take a job from the main thread or from
// any sibling worker without blocking. It returns nil when no
// work is available.
func (tq *TaskQueue) stealWork(selfID int) *Job {
	// First try to steal from the main thread.
	select {
	case job := <-tq.mainQueue:
		return job
	default:
	}
	// Then scan sibling workers in round-robin order.
	for i := 0; i < tq.workerCount; i++ {
		if i == selfID {
			continue
		}
		select {
		case job := <-tq.workerQueues[i]:
			return job
		default:
		}
	}
	return nil
}

// Submit enqueues a job. The priority heap orders it; if the main
// thread is busy the job overflows onto the steal channel where an
// idle worker will claim it.
func (tq *TaskQueue) Submit(j *Job) {
	tq.mu.Lock()
	heap.Push(tq.priorityQueue, j)
	tq.mu.Unlock()

	select {
	case tq.mainQueue <- j:
	default:
		select {
		case tq.steal <- j:
		default:
		}
	}
}

// Fork dispatches a batch of jobs and blocks until every job has
// completed. This is the Fork-Join primitive used by the engine
// for parallel physics island solving and batch AI updates.
func (tq *TaskQueue) Fork(jobs []*Job) {
	var wg sync.WaitGroup
	wg.Add(len(jobs))
	for _, j := range jobs {
		j.wg = &wg
		tq.Submit(j)
	}
	wg.Wait()
}

// Stop signals all workers to exit and waits for them to finish.
func (tq *TaskQueue) Stop() {
	close(tq.quit)
	tq.wg.Wait()
}
