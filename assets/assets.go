package assets

import "embed"

//go:embed favicon.ico *.css *.html *.svg
var FS embed.FS
