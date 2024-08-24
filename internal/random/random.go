package random

import (
	"math/rand/v2"
	"time"
)

// String returns a randomly generated string with size of n.
func String(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = letters[rand.IntN(len(letters))]
	}

	return string(b)
}

// Time returns a randomly generated time.
func Time() time.Time {
	return time.Date(
		rand.IntN(10000),            // random year [0,9999]
		time.Month(rand.IntN(12)+1), // random month [1,12]
		rand.IntN(31)+1,             // random day [1, 31]
		rand.IntN(24),               // random hour [0, 24]
		rand.IntN(60),               // random minute [0, 59]
		rand.IntN(60),               // random second [0, 59]
		rand.IntN(1000000)*1000,     // random microseconds [0, 999999μs]
		time.Local,
	)
}
