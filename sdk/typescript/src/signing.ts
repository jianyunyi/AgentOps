import { createHash, createHmac, randomBytes } from "node:crypto";

export const SIGNING_VERSION = "v1";

export function hashBody(body: Buffer): string {
  return createHash("sha256").update(body).digest("hex");
}

export function canonicalRequest(method: string, path: string, timestamp: number, nonce: string, bodyHash: string): string {
  return [SIGNING_VERSION, method, path, String(timestamp), nonce, bodyHash].join("\n");
}

export function signRequest(secret: Buffer, method: string, path: string, body: Buffer, timestamp: number, nonce: string): string {
  const digest = createHmac("sha256", secret).update(canonicalRequest(method, path, timestamp, nonce, hashBody(body))).digest("hex");
  return `${SIGNING_VERSION}=${digest}`;
}

export function newNonce(): string {
  return randomBytes(32).toString("hex");
}
