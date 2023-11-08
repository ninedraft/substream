package libscan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"path"

	"golang.org/x/time/rate"
)

// Scanner searches for files in the filesystem matching the glob pattern.
// Then it streams the content of the files to the io.Writer.
type Scanner struct {
	FS      fs.FS
	Dir     string
	Glob    string
	BufSize int
	Shuffle bool
	Limiter *rate.Limiter

	OnNext func(ctx context.Context, filename string) error
}

// Stream copies content of the files matching the glob pattern to the dst.
// It loops infinitely until the context is canceled or an error occurs.
func (scanner *Scanner) Stream(ctx context.Context, dst io.Writer) error {
	files, errFiles := scanner.findAll(ctx)
	if errFiles != nil {
		return fmt.Errorf("finding files: %w", errFiles)
	}

	bufSize := scanner.BufSize
	if bufSize <= 0 {
		bufSize = 16 * 1024
	}
	buf := make([]byte, bufSize)

	copyFile := func(filename string) error {
		file, errFile := scanner.FS.Open(filename)
		if errFile != nil {
			return fmt.Errorf("opening file: %w", errFile)
		}
		defer file.Close()

		errCopy := scanner.copy(ctx, dst, file, buf)
		if errCopy != nil {
			return fmt.Errorf("copying file: %w", errCopy)
		}

		return nil
	}

	n := len(files)
	if n == 0 {
		return nil
	}

	for i := 0; ctx.Err() == nil; i = (i + 1) % n {
		if i == 0 && scanner.Shuffle {
			shuffle(files)
		}

		filename := files[i]

		if scanner.OnNext != nil {
			errNext := scanner.OnNext(ctx, filename)
			if errNext != nil {
				return fmt.Errorf("next callback: %w", errNext)
			}
		}

		errCopy := copyFile(filename)

		switch {
		case errors.Is(errCopy, fs.ErrNotExist):
			continue
		case errors.Is(errCopy, io.EOF):
			continue
		case errCopy != nil:
			return fmt.Errorf("file %q: %w", filename, errCopy)
		}
	}

	return ctx.Err()
}

func (scanner *Scanner) findAll(ctx context.Context) ([]string, error) {
	if _, err := path.Match("", scanner.Glob); err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}

	dir := scanner.Dir
	if dir == "" {
		dir = "."
	}

	var files []string

	errWalk := fs.WalkDir(scanner.FS, dir, func(fpath string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		ok, _ := path.Match(scanner.Glob, d.Name())
		if ok {
			files = append(files, fpath)
		}

		return nil
	})

	if errWalk != nil {
		return nil, fmt.Errorf("walking dir %q: %w", dir, errWalk)
	}

	return files, nil
}

func (scanner *Scanner) copy(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) error {
	if scanner.Limiter == nil {
		_, err := io.CopyBuffer(dst, src, buf)
		return err
	}

	streamChunk := func(buf []byte) error {
		n, errRead := src.Read(buf)

		log.Printf("read %d bytes, err=%v", n, errRead)

		switch {
		case errors.Is(errRead, io.EOF):
			_, errWrite := dst.Write(buf[:n])
			if errWrite != nil {
				return fmt.Errorf("writing: %w", errWrite)
			}
			return nil
		case errRead != nil:
			return fmt.Errorf("reading: %w", errRead)
		}

		if err := scanner.Limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter: %w", err)
		}

		_, errWrite := dst.Write(buf[:n])
		if errWrite != nil {
			return fmt.Errorf("writing: %w", errWrite)
		}

		log.Printf("wrote %d bytes", n)

		return nil
	}

	for {
		if err := streamChunk(buf); err != nil {
			return fmt.Errorf("streaming chunk: %w", err)
		}
	}
}

func shuffle[E any](slice []E) {
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}
