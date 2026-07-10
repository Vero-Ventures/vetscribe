// VetScribe frontend — hand-written vanilla JavaScript, loaded as a classic
// deferred script (<script defer>). No build step, no bundler, no transpile.
// Ported from the former Preact web/src/app.tsx.
//
// Responsibilities:
//   1. Fetch /healthz and reflect live backend status in the DOM (the real-browser
//      assertion target for the chromedp e2e).
//   2. Hold the MediaRecorder capture scaffolding used by later milestones (M4).

"use strict";

/** Update the health-status element, or surface an error element on failure. */
async function refreshHealth() {
  const statusEl = document.querySelector('[data-testid="health-status"]');
  try {
    const res = await fetch("/healthz", { headers: { Accept: "application/json" } });
    if (!res.ok) {
      throw new Error(`status ${res.status}`);
    }
    const data = await res.json();
    const version = (data.build && data.build.version) || "unknown";
    if (statusEl) {
      statusEl.textContent = `Backend: ${data.status} (v${version})`;
    }
  } catch (err) {
    const main = document.querySelector("main") || document.body;
    let errEl = document.querySelector('[data-testid="health-error"]');
    if (!errEl) {
      errEl = document.createElement("p");
      errEl.setAttribute("data-testid", "health-error");
      main.appendChild(errEl);
    }
    const message = err instanceof Error ? err.message : "unknown error";
    errEl.textContent = `Backend error: ${message}`;
    if (statusEl) {
      statusEl.remove();
    }
  }
}

// --- MediaRecorder capture scaffolding (wired up in a later milestone) ----------
//
// Kept deliberately minimal: it proves the unavoidable browser API surface is
// reachable from a plain asset without a framework. No UI is bound yet.

/** @type {MediaRecorder | null} */
let recorder = null;
/** @type {Blob[]} */
let chunks = [];

/**
 * Begin capturing microphone audio. Returns the active MediaRecorder.
 * @returns {Promise<MediaRecorder>}
 */
async function startCapture() {
  if (!navigator.mediaDevices || !window.MediaRecorder) {
    throw new Error("MediaRecorder API unavailable in this browser");
  }
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  chunks = [];
  recorder = new MediaRecorder(stream);
  recorder.addEventListener("dataavailable", (event) => {
    if (event.data && event.data.size > 0) {
      chunks.push(event.data);
    }
  });
  recorder.start();
  return recorder;
}

/**
 * Stop the active capture and resolve with the recorded audio blob.
 * @returns {Promise<Blob>}
 */
function stopCapture() {
  return new Promise((resolve, reject) => {
    if (!recorder) {
      reject(new Error("no active recording"));
      return;
    }
    recorder.addEventListener(
      "stop",
      () => {
        const type = recorder && recorder.mimeType ? recorder.mimeType : "audio/webm";
        const blob = new Blob(chunks, { type });
        recorder = null;
        resolve(blob);
      },
      { once: true },
    );
    recorder.stop();
    for (const track of recorder.stream.getTracks()) {
      track.stop();
    }
  });
}

// Expose the capture API for later milestones without a module bundler.
window.vetscribe = { startCapture, stopCapture };

document.addEventListener("DOMContentLoaded", refreshHealth);
