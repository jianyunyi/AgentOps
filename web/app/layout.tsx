import type { ReactNode } from "react";
import type { Metadata } from "next";

import "./globals.css";
import { ConsoleShell } from "../components/layout/console-shell";

export const metadata: Metadata = {
  title: "AgentScope | AI operations control",
  description: "Observe, govern, and audit every AI Agent execution.",
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
