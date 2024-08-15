package albumcatalog

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

// randomString returns a randomly generated string with size of n.
func randomString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = letters[rand.IntN(len(letters))]
	}

	return string(b)
}

// randomTime returns a randomly generated time.
func randomTime() time.Time {
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

// randomAlbum returns a randomply generated Album.
func randomAlbum() Album {
	return Album{
		ID:        uuid.New(),
		Title:     randomString(20 + rand.IntN(20)),
		Artist:    randomString(20 + rand.IntN(20)),
		Price:     rand.IntN(100000),
		CreatedAt: randomTime(),
		UpdatedAt: randomTime(),
	}
}

// randomAlbums returns a slice containing n randomly generated Albums.
func randomAlbums(n int) []Album {
	albs := make([]Album, n)
	for i := range albs {
		albs[i] = randomAlbum()
	}
	return albs
}
