import type { ReactNode } from "react";
import type { Metadata } from "next";

import "./globals.css";
import { ConsoleShell } from "../components/layout/console-shell";

export const metadata: Metadata = {
  title: "AgentOps | Autonomous traffic control",
  description: "Observe, govern, and investigate live AI Agent operations.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>
        <ConsoleShell>{children}</ConsoleShell>
      </body>
    </html>
  );
}
