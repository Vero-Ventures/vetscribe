//go:build e2e

// Package api's end-to-end browser test. Guarded by the `e2e` build tag so
// chromedp and its dependency tree stay out of the default build graph, the
// govulncheck surface, the race/coverage suite, and the shipped binary.
//
// Run against a already-running server:
//
//	VETSCRIBE_BASE_URL=http://localhost:8080 go test -tags e2e ./internal/api/ -run TestE2E
//
// If VETSCRIBE_BASE_URL is unset it defaults to http://localhost:8080. The test
// skips (rather than fails) when no Chrome/Chromium binary is available locally,
// but CI installs one so the assertion runs there.
package api

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func baseURL() string {
	if v := os.Getenv("VETSCRIBE_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func chromeAvailable() bool {
	for _, name := range []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome",
	} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}

// TestE2EBackendHealth loads the served page in a real headless browser and
// asserts the JS-driven backend-health element reports "ok" — the real-browser
// backend-health assertion that replaces the former Playwright smoke test.
func TestE2EBackendHealth(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no Chrome/Chromium binary found; skipping browser e2e")
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(
		context.Background(),
		append(
			chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-gpu", true),
			// CI runners have a small /dev/shm; without this Chrome crashes on
			// startup and never connects (page never loads).
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var heading, status string
	err := chromedp.Run(
		ctx,
		chromedp.Navigate(baseURL()+"/"),
		chromedp.WaitVisible(`h1`, chromedp.ByQuery),
		chromedp.Text(`h1`, &heading, chromedp.ByQuery),
		// app.js flips this element to "Backend: ok (v...)" once /healthz responds.
		chromedp.WaitVisible(`[data-testid="health-status"]`, chromedp.ByQuery),
		chromedp.Poll(
			`document.querySelector('[data-testid="health-status"]').textContent.includes('ok')`,
			nil,
			chromedp.WithPollingTimeout(15*time.Second),
		),
		chromedp.Text(`[data-testid="health-status"]`, &status, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("chromedp run: %v", err)
	}

	if !strings.Contains(heading, "VetScribe") {
		t.Fatalf("expected heading to contain VetScribe, got %q", heading)
	}
	if !strings.Contains(status, "ok") {
		t.Fatalf("expected health-status to contain 'ok', got %q", status)
	}
}
