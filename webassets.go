// Package webassets embeds the built frontend (frontend/ -> `npm run
// build` -> dist/) into the compiled binary, so the .exe is fully
// self-contained: no separate dist/ folder needs to ship alongside it.
// `dist/` must exist at build time — run `npm run build` (or `npm
// install`, which does it via postinstall) before `go build`/`go test`
// on this package or anything that imports it.
package webassets

import "embed"

//go:embed all:dist
var Dist embed.FS
