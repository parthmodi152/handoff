package web

import "embed"

//go:embed templates/layouts/*.html templates/pages/*.html static/css/*.css static/js/*.js
var Files embed.FS
