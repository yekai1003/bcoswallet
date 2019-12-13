package util

import (
	"crypto/rand"
	"io"
)

type UUID []byte

var rander = rand.Reader // random function

func randomBits(b []byte) {
	if _, err := io.ReadFull(rander, b); err != nil {
		panic(err.Error()) // rand should never fail
	}
}
func NewRandom() UUID {
	uuid := make([]byte, 16)
	randomBits([]byte(uuid))
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	return uuid
}
