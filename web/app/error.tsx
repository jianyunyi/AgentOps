"use client";

export default function GlobalError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return <main role="alert"><h1>Something went wrong</h1><p>The console could not complete this request.</p><button onClick={() => reset()}>Try again</button></main>;
}
