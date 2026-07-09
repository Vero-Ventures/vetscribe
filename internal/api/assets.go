package api

import "embed"

// frontendAssets embeds the built frontend. The build pipeline compiles the Preact
// app into web/dist before `go build`; a committed placeholder keeps `go build`
// working before the frontend is built.
//
//go:embed all:dist
var frontendAssets embed.FS

const frontendRoot = "dist"
