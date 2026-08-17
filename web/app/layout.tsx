import type { ReactNode } from "react";

import "./globals.css";

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <header className="app-header">
          <span className="brand">AgentScope</span>
          <span className="subtitle">AI Agent governance console</span>
        </header>
        <main className="app-main">{children}</main>
      </body>
    </html>
  );
}
