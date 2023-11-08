package libscan_test

import (
	"context"
	"io"
	"testing"
	"testing/fstest"

	"github.com/ninedraft/substream/internal/libscan"
	"github.com/stretchr/testify/require"
)

func TestScanner_Stream(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"foo.txt":     {Data: []byte("foo\n")},
		"sub/bar.txt": {Data: []byte("bar\n")},
		"baz.dat":     {Data: []byte("baz\n")},
	}

	scanner := &libscan.Scanner{
		FS:      fsys,
		Glob:    "*.txt",
		Shuffle: false,
	}

	t.Run("context stops looping", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		got, input := io.Pipe()

		buf := make([]byte, 1)
		go func() {
			defer cancel()
			for {
				_, err := got.Read(buf)
				cancel()
				if err != nil {
					return
				}
			}
		}()

		err := scanner.Stream(ctx, input)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("file contents are copied to dst", func(t *testing.T) {
		t.Parallel()

		gotStream, input := io.Pipe()

		go func() {
			defer input.Close()
			_ = scanner.Stream(context.Background(), input)
		}()

		got := make([]byte, 1024)
		_, errGot := io.ReadFull(gotStream, got)

		require.NoError(t, errGot)
		require.Containsf(t, string(got), "foo", "got %q", got)
		require.Containsf(t, string(got), "bar", "got %q", got)
	})
}
