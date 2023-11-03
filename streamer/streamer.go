package streamer

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/ninedraft/substream/broadcast"
)

type Streamer struct {
	buf         *bytes.Buffer
	broadcaster *broadcast.Broadcaster[*bytes.Buffer]
}

func New() *Streamer {
	return &Streamer{
		broadcaster: broadcast.New(&bytes.Buffer{}),
	}
}

var _ http.Handler = (*Streamer)(nil)

func (streamer *Streamer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	flush := func() error { return nil }

	controller := http.NewResponseController(rw)
	if controller != nil {
		flush = controller.Flush
	}

	streamer.broadcaster.Listen(func(buf *bytes.Buffer) error {
		p := buf.Bytes()

		if _, err := rw.Write(p); err != nil {
			return fmt.Errorf("writing to response writer: %w", err)
		}

		if err := flush(); err != nil {
			return fmt.Errorf("flushing response writer: %w", err)
		}

		return nil
	})
}

func (s *Streamer) Write(p []byte) (n int, err error) {
	err = s.broadcaster.Update(func(buf *bytes.Buffer) *bytes.Buffer {
		buf.Reset()
		n, _ = buf.Write(p)
		return buf
	})

	return n, err
}

func (s *Streamer) NClients() int {
	return s.broadcaster.NSubscribers()
}

func (s *Streamer) Close() error {
	return s.broadcaster.Close()
}
