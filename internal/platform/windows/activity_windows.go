//go:build windows

package windows

import (
	"fmt"
	"time"
	"unsafe"
)

// ActivityMonitor implements platform.UserActivityMonitor using GetLastInputInfo.
type ActivityMonitor struct{}

// NewActivityMonitor creates a new Windows activity monitor.
func NewActivityMonitor() *ActivityMonitor {
	return &ActivityMonitor{}
}

// IdleTime returns the duration since the last user input event.
func (m *ActivityMonitor) IdleTime() (time.Duration, error) {
	var info lastInputInfo
	info.CbSize = uint32(unsafe.Sizeof(info))

	r1, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return 0, fmt.Errorf("GetLastInputInfo: %w", err)
	}

	tickCount, _, err := procGetTickCount64.Call()
	if tickCount == 0 {
		return 0, fmt.Errorf("GetTickCount64: %w", err)
	}

	// GetLastInputInfo returns a 32-bit tick count, handle wraparound
	lastInput := uint64(info.DwTime)
	current := uint64(tickCount)

	// Handle 32-bit wraparound for DwTime
	if current > 0xFFFFFFFF {
		// The system has been up for more than 49 days
		// DwTime wraps around at 2^32 ms
		epoch := current & 0xFFFFFFFF00000000
		lastInput64 := epoch | lastInput
		if lastInput64 > current {
			lastInput64 -= 0x100000000
		}
		lastInput = lastInput64
	}

	idle := current - lastInput
	return time.Duration(idle) * time.Millisecond, nil
}
