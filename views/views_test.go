package views_test

import (
	"io"
	"testing"

	"github.com/ninedraft/substream/views"
)

func TestViews(t *testing.T) {
	t.Parallel()
	t.Log(
		"Rendering views should not return an error.",
	)

	testView(t, views.Track)
}

func testView[E any](t *testing.T, view views.View[E]) {
	t.Helper()

	t.Run(string(view)+".Render", func(t *testing.T) {
		t.Parallel()

		if err := view.Render(io.Discard, new(E)); err != nil {
			t.Errorf("render: %v", err)
		}
	})
}
