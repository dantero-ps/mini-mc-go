package game

import (
	"mini-mc/internal/config"
	"time"
)

// FPSLimiter provides high-precision frame rate limiting
type FPSLimiter struct {
	next time.Time
}

// NewFPSLimiter creates a new FPS limiter
func NewFPSLimiter() *FPSLimiter {
	return &FPSLimiter{}
}

// Wait blocks until the next frame should be rendered based on the FPS limit.
// Uses a hybrid sleep/spin approach for better precision on high FPS caps.
func (f *FPSLimiter) Wait(paused bool) {
	effectiveLimit := config.GetFPSLimit()
	if paused {
		effectiveLimit = 120
	}

	if effectiveLimit <= 0 {
		f.next = time.Time{}
		return
	}

	target := time.Second / time.Duration(effectiveLimit)

	if f.next.IsZero() {
		f.next = time.Now().Add(target)
	} else {
		f.next = f.next.Add(target)
	}

	for {
		remaining := time.Until(f.next)
		if remaining <= 0 {
			break
		}
		if remaining > 200*time.Microsecond {
			time.Sleep(remaining - 200*time.Microsecond)
		}
		// busy-wait for the final few microseconds
		// yields substantially better precision on high FPS caps
		if time.Until(f.next) <= 0 {
			break
		}
	}

	// If we're significantly late (e.g., hitch), resync to avoid drift
	if late := -time.Until(f.next); late > target {
		f.next = time.Now().Add(target)
	}
}
