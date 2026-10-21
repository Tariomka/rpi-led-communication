package common

import "time"

var start = time.Now()

// Millis is the millisecond clock used by the cube protocol; it wraps after ~49 days.
func Millis() uint32 {
	return uint32(time.Since(start).Milliseconds())
}
