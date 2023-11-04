package main

import (
	"time"
)

type trackCover struct {
	contentType string
	data        []byte
	updatedAt   time.Time
}
