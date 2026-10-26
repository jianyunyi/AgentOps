import type { ReactNode } from "react";
import type { Metadata } from "next";
import Script from "next/script";

import "./globals.css";
import { ConsoleShell } from "../components/layout/console-shell";

export const metadata: Metadata = {
  title: "AgentOps | Autonomous traffic control",
  description: "Observe, govern, and investigate live AI Agent operations.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="zh-CN">
      <head><Script src="/runtime-config.js" strategy="beforeInteractive" /></head>
      <body>
        <ConsoleShell>{children}</ConsoleShell>
      </body>
    </html>
  );
}
