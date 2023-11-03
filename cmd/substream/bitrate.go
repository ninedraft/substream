package main

import (
	"io"
	"slices"

	"github.com/tcolgate/mp3"
	"golang.org/x/time/rate"
)

var twoChannels = []mp3.FrameChannelMode{
	mp3.Stereo, mp3.JointStereo, mp3.DualChannel,
}

func readBitrate(re io.Reader) (int, error) {
	decoder := mp3.NewDecoder(re)

	frame := new(mp3.Frame)
	errDecode := decoder.Decode(frame, new(int))
	if errDecode != nil {
		return 0, errDecode
	}

	header := frame.Header()
	bitrate := header.BitRate()

	if slices.Contains(twoChannels, header.ChannelMode()) {
		bitrate *= 2
	}

	return int(bitrate), nil
}

// returns a rate limiter that limits the stream to the given bitrate
// it regulates N sends per second, where N bitrate / 8 / bufSize
func streamRate(bufSize int, bitrate int) *rate.Limiter {
	byteRate := float64(bitrate / 8)
	bs := float64(bufSize)
	limit := 1.05 * rate.Limit(byteRate/bs)

	return rate.NewLimiter(limit, 2)
}
