package views

import "time"

const Track View[TrackData] = "track.html"

type TrackData struct {
	Title  string
	Artist string
	Length time.Duration
}
