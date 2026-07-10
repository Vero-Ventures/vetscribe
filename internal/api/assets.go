package api

import "embed"

// templateFS embeds the server-rendered HTML templates. The page is rendered by
// Go's html/template at request time, so build metadata is injected server-side and
// the page works before any JavaScript runs.
//
//go:embed templates/*.tmpl
var templateFS embed.FS

// staticAssets embeds the hand-written, no-build frontend assets (vanilla JS + CSS).
// They are real committed source files, so the embed is always valid — there is no
// generated build output and no placeholder needed.
//
//go:embed assets/app.js assets/app.css
var staticAssets embed.FS
