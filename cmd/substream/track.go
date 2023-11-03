package main

import (
	"html/template"
	"io"
	"time"
)

type trackCover struct {
	contentType string
	data        []byte
	updatedAt   time.Time
}

type trackData struct {
	Title  string
	Artist string
	Length time.Duration
}

var trackView = template.Must(template.New("track").Parse(`
<div class="track">
	<div class="track__title">{{ .Title }}</div>
	<div class="track__artist">{{ .Artist }}</div>
	<div class="track__length">{{ .Length }}</div>
</div>
`))

func (t *trackData) Render(dst io.Writer) error {
	return trackView.Execute(dst, t)
}
