import { describe, expect, it } from "vitest";
import { MAX_CUSTOM_CSS_LEN, sanitizeCustomCss } from "./theme-sanitize";

// err unwraps a rejection; it throws (failing the test) when the input is
// accepted, so the matrix can stay declarative.
function err(css: string): string {
  const r = sanitizeCustomCss(css);
  if (r.ok) {
    throw new Error(`expected rejection, input was accepted: ${css.slice(0, 60)}`);
  }
  return r.error;
}

describe("sanitizeCustomCss — @import rules", () => {
  it("rejects @import with a url()", () => {
    expect(err(`@import url("https://evil.example.com/track.css");`)).toBe(
      "customCss rejected: @import rules are not allowed",
    );
  });

  it("rejects a bare @import string", () => {
    expect(err(`@import "https://evil.example.com/x.css";`)).toContain("@import rules are not allowed");
  });

  it("rejects @import case-insensitively", () => {
    expect(err(`@IMPORT "x.css";`)).toBe("customCss rejected: @import rules are not allowed");
  });

  it("rejects @import anywhere in the stylesheet", () => {
    expect(err(`.a { color: red; }\n@media all { @import "x.css"; }`)).toContain(
      "@import rules are not allowed",
    );
  });
});

describe("sanitizeCustomCss — external url() references", () => {
  it("rejects https url() and names the reference", () => {
    expect(err(`.a { background: url("https://fonts.example.com/x.woff2"); }`)).toBe(
      "customCss rejected: external url() reference 'https://fonts.example.com/x.woff2' is not allowed",
    );
  });

  it("rejects http url() with single quotes", () => {
    expect(err(`.a { background: url('http://evil.example.com/x.png'); }`)).toContain(
      "external url() reference 'http://evil.example.com/x.png' is not allowed",
    );
  });

  it("rejects protocol-relative url()", () => {
    expect(err(`.a { background: url(//cdn.example.com/x.png); }`)).toContain(
      "external url() reference '//cdn.example.com/x.png' is not allowed",
    );
  });

  it("rejects uppercase URL(HTTP://...) case-insensitively, naming the original", () => {
    expect(err(`.a { background: URL(HTTP://evil.example.com/x.png); }`)).toContain(
      "external url() reference 'HTTP://evil.example.com/x.png' is not allowed",
    );
  });

  it("trims whitespace inside url() before checking", () => {
    expect(err(`.a { background: url(  https://x.example/y.png  ); }`)).toContain(
      "external url() reference 'https://x.example/y.png' is not allowed",
    );
  });
});

describe("sanitizeCustomCss — allowed url() forms", () => {
  it("accepts inline data: URIs", () => {
    expect(
      sanitizeCustomCss(`.a { background: url(data:image/png;base64,iVBORw0KGgo=); }`),
    ).toEqual({ ok: true });
  });

  it("accepts relative and absolute-path url()", () => {
    expect(sanitizeCustomCss(`.a { background: url(/static/tile.png); }`)).toEqual({ ok: true });
    expect(sanitizeCustomCss(`.a { background: url(../img/tile.png); }`)).toEqual({ ok: true });
    expect(sanitizeCustomCss(`.a { background: url("tile.png"); }`)).toEqual({ ok: true });
  });

  it("accepts mixed allowed references in one stylesheet", () => {
    const css = [
      `.a { background: url(data:image/svg+xml;utf8,<svg></svg>), url(/static/t.png); }`,
      `.b { cursor: url(tile.png), auto; }`,
    ].join("\n");
    expect(sanitizeCustomCss(css)).toEqual({ ok: true });
  });
});

describe("sanitizeCustomCss — size cap", () => {
  it("accepts a stylesheet exactly at the cap", () => {
    expect(sanitizeCustomCss(".".repeat(MAX_CUSTOM_CSS_LEN))).toEqual({ ok: true });
  });

  it("rejects a stylesheet one character over the cap", () => {
    expect(err(".".repeat(MAX_CUSTOM_CSS_LEN + 1))).toBe(
      `customCss rejected: exceeds the maximum length of ${MAX_CUSTOM_CSS_LEN} characters`,
    );
  });

  it("counts bytes like the server (Go len), not UTF-16 code units", () => {
    // "é" is 2 UTF-8 bytes: 16384 chars = 32768 bytes (ok), +1 char = 32770 (rejected).
    expect(sanitizeCustomCss("é".repeat(MAX_CUSTOM_CSS_LEN / 2))).toEqual({ ok: true });
    expect(err("é".repeat(MAX_CUSTOM_CSS_LEN / 2 + 1))).toContain(
      `exceeds the maximum length of ${MAX_CUSTOM_CSS_LEN} characters`,
    );
  });

  it("checks the size cap before any other rule", () => {
    const over = "@import \"x.css\";" + ".".repeat(MAX_CUSTOM_CSS_LEN);
    expect(err(over)).toContain("exceeds the maximum length");
  });
});

describe("sanitizeCustomCss — brace balance", () => {
  it("rejects a missing closing brace", () => {
    expect(err(`.a { color: red;`)).toBe("customCss rejected: unbalanced braces");
  });

  it("rejects an extra closing brace", () => {
    expect(err(`.a }`)).toBe("customCss rejected: unbalanced braces");
  });

  it("accepts balanced nested braces", () => {
    expect(sanitizeCustomCss(`@media all { .a { color: red; } }`)).toEqual({ ok: true });
  });
});

describe("sanitizeCustomCss — HTML delimiters", () => {
  it("rejects <script> breakout attempts", () => {
    expect(err(`<script>alert(1)</script>`)).toBe(
      'customCss rejected: HTML delimiter "<script" is not allowed',
    );
  });

  it("rejects <style> delimiters case-insensitively", () => {
    expect(err(`<STYLE>.a{}</STYLE>`)).toBe(
      'customCss rejected: HTML delimiter "<style" is not allowed',
    );
    expect(err(`<!-- </STYLE> -->`)).toBe(
      'customCss rejected: HTML delimiter "</style" is not allowed',
    );
    expect(err(`</ScRiPt>`)).toBe(
      'customCss rejected: HTML delimiter "</script" is not allowed',
    );
  });
});

describe("sanitizeCustomCss — rule order mirrors the server", () => {
  it("@import is reported before external url()", () => {
    expect(err(`@import url(https://evil.example.com/x.css);`)).toContain(
      "@import rules are not allowed",
    );
  });

  it("external url() is reported before unbalanced braces", () => {
    expect(err(`.a { background: url(https://evil.example.com/x.png);`)).toContain(
      "external url() reference",
    );
  });

  it("unbalanced braces are reported before HTML delimiters", () => {
    expect(err(`.a { content: "<style>";`)).toBe("customCss rejected: unbalanced braces");
  });
});

describe("sanitizeCustomCss — general", () => {
  it("accepts a typical override stylesheet", () => {
    const css = [
      ":root { --accent: #10b981; }",
      ".dashboard-card { border-radius: 12px; padding: 1rem; }",
      ".sidebar .logo { filter: hue-rotate(90deg); }",
    ].join("\n");
    expect(sanitizeCustomCss(css)).toEqual({ ok: true });
  });

  it("accepts an empty stylesheet", () => {
    expect(sanitizeCustomCss("")).toEqual({ ok: true });
  });
});
