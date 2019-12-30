package controllers

import (
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" + "0123456789"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func totalValueForKeys(m map[string]int32) int32 {
	var total int32 = 0
	for _, value := range m {
		total += value
	}
	return total
}
