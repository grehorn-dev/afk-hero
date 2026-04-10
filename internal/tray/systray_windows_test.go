//go:build windows

package tray

import "testing"

func TestManagerSignalStopIsIdempotent(t *testing.T) {
	manager := NewManager(Callbacks{})

	manager.signalStop()
	manager.signalStop()

	select {
	case <-manager.stopCh:
	default:
		t.Fatal("expected stop channel to be closed after signalStop")
	}
}
