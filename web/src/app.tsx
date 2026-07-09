import { useEffect, useState } from "preact/hooks";

interface Health {
  status: string;
  build: { version: string; commit: string; date: string };
}

export function App() {
  const [health, setHealth] = useState<Health | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/healthz")
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(`status ${r.status}`))))
      .then((data: Health) => setHealth(data))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "unknown error"));
  }, []);

  return (
    <main>
      <h1>VetScribe</h1>
      <p data-testid="tagline">Veterinary visit transcription and SOAP drafting.</p>
      {error && <p data-testid="health-error">Backend error: {error}</p>}
      {health && (
        <p data-testid="health-status">
          Backend: {health.status} (v{health.build.version})
        </p>
      )}
    </main>
  );
}
