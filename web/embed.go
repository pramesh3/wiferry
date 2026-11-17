// Package web holds WiFerry's embedded browser UI. Everything is served from
// the binary so it works without internet.
package web

import "embed"

// FS contains the HTML, CSS and JS assets.
//
//go:embed *.html *.css *.js
var FS embed.FS
