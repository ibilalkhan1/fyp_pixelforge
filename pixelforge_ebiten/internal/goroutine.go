package internal

import (
	"bytes"
	"fmt"
	"runtime"
)

func CurrentGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	// Example header: "goroutine 1 [running]:"
	var id uint64
	_, _ = fmt.Sscanf(string(bytes.Split(b, []byte(" "))[1]), "%d", &id)
	return id
}
