package cachefs

import (
	"hash/adler32"
	"time"
)

// quantize returns an integer to use as part of a cache key so that
// time is "quantized" for a certain interval. Also, the checksum of
// a string as a second offset. This prevents all the items from
// expiring at once, but does create variability in how long they
// can be cached. This function allows us to use a cache like
// golang/groupcache, which has no expiry mechanism of its own.
func quantize(t time.Time, d time.Duration, s string) int64 {
	if d == 0 {
		return 0
	}
	sum := adler32.Checksum([]byte(s))
	offset := time.Duration(float64(sum) * float64(d) / float64(2<<31) / 4.0)
	return t.UnixNano() / int64(d.Nanoseconds()+offset.Nanoseconds())
}

// quantizeOffset returns the maximum cache duration for a given sum.
func quantizeOffset(sum int32, d time.Duration) time.Duration {
	return time.Duration(float64(sum) * float64(d) / float64(2<<31) / 4.0)
}
