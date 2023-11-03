package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/ninedraft/substream/assets"
	"github.com/ninedraft/substream/streamer"
)

func main() {
	addr := "localhost:8080"
	flag.StringVar(&addr, "addr", addr, "address to listen on")

	file := "music.mp3"
	flag.StringVar(&file, "file", file, "path to music file")

	flag.Parse()

	music, errMusic := os.Open(file)
	if errMusic != nil {
		panic("opening music file: " + errMusic.Error())
	}
	defer music.Close()

	bitrate, errBitrate := readBitrate(music)
	if errBitrate != nil {
		panic("reading bitrate: " + errBitrate.Error())
	}

	log.Printf("track bitrate: %d", bitrate)

	_, errSeek := music.Seek(0, io.SeekStart)
	if errSeek != nil {
		panic("seeking music file: " + errSeek.Error())
	}

	streamer := streamer.New()

	streamChunk := func(buf []byte) error {
		n, errRead := music.Read(buf)

		switch {
		case errors.Is(errRead, io.EOF):
			_, _ = streamer.Write(buf[:n])
			return nil
		case errRead != nil:
			return fmt.Errorf("reading music file: %w", errRead)
		}

		_, errWrite := streamer.Write(buf[:n])
		if errWrite != nil {
			return fmt.Errorf("writing to streamer: %w", errWrite)
		}

		return nil
	}

	ctx := context.Background()
	go func() {
		buf := make([]byte, 16<<10)
		defer streamer.Close()

		_ = streamChunk(buf)

		r := streamRate(len(buf), bitrate)
		log.Printf("streaming %v buffers %d bytes per second", r.Limit(), bitrate/8)
		for {
			if err := r.Wait(ctx); err != nil {
				log.Println("rate limiter:", err)
				return
			}

			if err := streamChunk(buf); err != nil {
				log.Println("streaming chunk:", err)
				return
			}
		}
	}()

	go func() {
		for range time.Tick(10 * time.Second) {
			log.Printf("streaming to %d clients", streamer.NClients())
		}
	}()

	currentTrack := atomic.Pointer[trackData]{}
	currentTrack.Store(&trackData{
		Title:  "Music for Programming",
		Artist: "MFP",
		Length: time.Minute,
	})

	currentTrackCover := atomic.Pointer[trackCover]{}

	mux := http.NewServeMux()
	mux.HandleFunc("/music",
		func(w http.ResponseWriter, r *http.Request) {
			tick := time.Now()
			log.Println("new client", r.RemoteAddr)

			defer func() {
				log.Println("client disconnected", r.RemoteAddr, time.Since(tick))
			}()

			w.Header().Set("Content-Type", "audio/mpeg")
			streamer.ServeHTTP(w, r)
		})

	assetsFS := http.FileServer(http.FS(assets.FS))
	mux.Handle("/", assetsFS)

	mux.HandleFunc("/music/track",
		func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if q.Get("next") == "true" {
				<-r.Context().Done()
				return
			}

			view := currentTrack.Load()
			if view == nil {
				return
			}

			w.Header().Set("Content-Type", "text/html")
			errRender := view.Render(w)
			if errRender != nil {
				log.Println("rendering track:", errRender)
			}
		})

	mux.HandleFunc("/music/cover",
		func(w http.ResponseWriter, r *http.Request) {

			cover := currentTrackCover.Load()
			if cover == nil {
				return
			}

			w.Header().Set("Content-Type", cover.contentType)
			http.ServeContent(w, r, "cover", cover.updatedAt, bytes.NewReader(cover.data))
		})

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL, r.RemoteAddr)

		mux.ServeHTTP(w, r)
	}

	_ = http.ListenAndServe("localhost:9080", handler)
}
