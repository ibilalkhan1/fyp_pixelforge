//nolint:testpackage // Verifies modal editor tick audio servicing.
package editor

import "testing"

type recordingAudioDriver struct{ calls []string }

func (d *recordingAudioDriver) OnBeforeUpdate() { d.calls = append(d.calls, "before") }
func (d *recordingAudioDriver) OnAfterUpdate()  { d.calls = append(d.calls, "after") }

func TestAudioDriverFlushesOnModalTick(t *testing.T) {
	e := New()
	driver := &recordingAudioDriver{}
	e.SetAudioDriver(driver)
	e.ConfirmDialog().Show("test", "test", func() {})
	if err := e.Update(); err != nil {
		t.Fatal(err)
	}
	if len(driver.calls) != 2 || driver.calls[0] != "before" || driver.calls[1] != "after" {
		t.Fatalf("audio commands were not flushed around modal tick: %v", driver.calls)
	}
}
