import { describe, expect, it } from "vitest";
import { canonicalRequest, hashBody, newNonce, signRequest } from "../src/signing.js";

const secret = Buffer.from("01234567890123456789012345678901");

describe("AgentScope HMAC v1 signing", () => {
  it("matches the frozen cross-language protocol vector", () => {
    const body = Buffer.from("hello");
    const bodyHash = hashBody(body);
    expect(bodyHash).toBe("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824");
    expect(canonicalRequest("POST", "/api/v1/ingest/events", 1700000000, "nonce-fixed", bodyHash)).toBe("v1\nPOST\n/api/v1/ingest/events\n1700000000\nnonce-fixed\n2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824");
    expect(signRequest(secret, "POST", "/api/v1/ingest/events", body, 1700000000, "nonce-fixed")).toBe("v1=f7151e596375c3562ff93158cd2dd289b4a17bdf23f5b8149902b8bb8f4b3cb3");
  });

  it("generates unique lowercase hexadecimal nonces", () => {
    const first = newNonce();
    const second = newNonce();
    expect(first).not.toBe(second);
    expect(first).toMatch(/^[0-9a-f]{64}$/);
    expect(second).toMatch(/^[0-9a-f]{64}$/);
  });
});
