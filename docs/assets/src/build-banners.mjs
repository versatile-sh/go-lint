import { createRequire } from "node:module";
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const opentype = require("opentype.js");
const here = dirname(fileURLToPath(import.meta.url));
const out = join(here, "..");

const font = (pkg, file) => {
  const buf = readFileSync(join(here, "node_modules/@fontsource", pkg, "files", file));
  return opentype.parse(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
};
const serif = font("instrument-serif", "instrument-serif-latin-400-normal.woff");
const italic = font("instrument-serif", "instrument-serif-latin-400-italic.woff");
const mono = font("jetbrains-mono", "jetbrains-mono-latin-400-normal.woff");
const monoMedium = font("jetbrains-mono", "jetbrains-mono-latin-500-normal.woff");

const THEMES = {
  dark: { bg: "#0e1116", fg: "#f5f5f2", muted: "#8a919c", accent: "#c6f432", accentText: "#c6f432", line: "#f5f5f2", card: "#12161d", glow: "#ffffff" },
  light: { bg: "#f5f5f2", fg: "#0e1116", muted: "#5b626d", accent: "#c6f432", accentText: "#4d6b00", line: "#0e1116", card: "#ffffff", glow: "#c6f432" },
};
let T;

const width = (f, s, size) => f.getAdvanceWidth(s, size);
const r = (n) => Math.round(n * 10) / 10;

let glyphs;
const glyphId = (f, glyph, size) => {
  const key = `${f.names.fullName.en}|${size}|${glyph.index}`;
  if (!glyphs.has(key)) {
    glyphs.set(key, { id: `g${glyphs.size}`, d: glyph.getPath(0, 0, size).toPathData(size >= 30 ? 0 : 1) });
  }
  return glyphs.get(key);
};
const defs = () => [...glyphs.values()].filter((g) => g.d).map((g) => `<path id="${g.id}" d="${g.d}"/>`).join("");

const text = (f, s, x, y, size, fill) => {
  const uses = [];
  f.forEachGlyph(s, x, y, size, { kerning: true }, (glyph, gx, gy) => {
    const g = glyphId(f, glyph, size);
    if (g.d) uses.push(`<use href="#${g.id}" x="${r(gx)}" y="${r(gy)}"/>`);
  });
  return `<g fill="${fill}">${uses.join("")}</g>`;
};

const runs = (parts, x, y) => {
  let cx = x;
  return parts
    .map(([f, s, size, fill]) => {
      const p = text(f, s, cx, y, size, fill);
      cx += width(f, s, size);
      return p;
    })
    .join("");
};

const fit = (f, s, size, max) => {
  if (width(f, s, size) > max) throw new Error(`"${s}" overflows its box`);
  return s;
};

const frame = (w, h, label, body) => {
  const grid = [];
  for (let x = 64; x < w; x += 64) grid.push(`M${x} 0V${h}`);
  for (let y = 64; y < h; y += 64) grid.push(`M0 ${y}H${w}`);
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}" role="img" aria-label="${label}"><title>${label}</title><defs><radialGradient id="l" cx="0.5" cy="0" r="0.75"><stop offset="0" stop-color="${T.glow}" stop-opacity="0.09"/><stop offset="1" stop-color="${T.glow}" stop-opacity="0"/></radialGradient><clipPath id="c"><rect width="${w}" height="${h}" rx="16"/></clipPath>${defs()}</defs><g clip-path="url(#c)"><rect width="${w}" height="${h}" fill="${T.bg}"/><path d="${grid.join("")}" stroke="${T.line}" stroke-opacity="0.04" fill="none"/><rect width="${w}" height="${h}" fill="url(#l)"/>${body}</g><rect x="0.5" y="0.5" width="${w - 1}" height="${h - 1}" rx="15.5" fill="none" stroke="${T.line}" stroke-opacity="0.18"/></svg>\n`;
};

const wordmark = (x, y) => runs([[mono, "versatile", 18, T.fg], [mono, ".sh", 18, T.accentText]], x, y);

const HERO_LABEL = "go-lint: one lint kit for every Versatile Go repository. A generated golangci-lint config, a drift check and seven analyzers for the layout, the clock seam, configuration, errors, tests and comments.";

const hero = () => {
  glyphs = new Map();
  const w = 1280;
  const h = 480;
  const tags = ["golangci-lint config", "drift check", "7 analyzers", "safe fixes only", "go run, no install"];
  let tx = 72;
  const pills = tags
    .map((t) => {
      const tw = width(mono, t, 15);
      const p = `<rect x="${tx}" y="388" width="${(tw + 32).toFixed(1)}" height="36" rx="18" fill="none" stroke="${T.line}" stroke-opacity="0.18"/>${text(mono, t, tx + 16, 411, 15, T.muted)}`;
      tx += tw + 44;
      return p;
    })
    .join("");
  const license = "open source  ·  GPL-3.0-or-later";
  const body = [
    wordmark(72, 87),
    text(mono, license, w - 72 - width(mono, license, 14), 86, 14, T.muted),
    text(serif, "go-lint", 68, 214, 112, T.fg),
    runs([[serif, "One lint kit for ", 46, T.fg], [italic, "every Go repository", 46, T.accentText], [serif, ".", 46, T.fg]], 72, 284),
    text(serif, "The same rules, the same config, checked the same way everywhere.", 72, 334, 34, T.muted),
    `<path d="M72 364H${w - 72}" stroke="${T.line}" stroke-opacity="0.1"/>`,
    pills,
  ].join("");
  return frame(w, h, HERO_LABEL, body);
};

const FLOW_LABEL = "How go-lint runs. Your repository's check command calls vlint through go run at a pinned version. vlint golangci writes .golangci.yml from the shared config plus your overlay, and vlint golangci -check fails when the committed file drifts. vlint ./... runs the seven analyzers: layout, clockseam, envroot, errstring, noskipenv, doccomment and adrref. golangci-lint then runs with the generated config.";

const diagram = () => {
  glyphs = new Map();
  const w = 1200;
  const h = 470;
  const top = 64;
  const boxH = 360;
  const pad = 28;
  const side = 300;
  const mid = 420;
  const gap = (w - 2 * 40 - 2 * side - mid) / 2;
  const repo = 40;
  const kit = repo + side + gap;
  const outs = kit + mid + gap;
  const box = (x, y, bw, bh) => `<rect x="${x}" y="${y}" width="${bw}" height="${bh}" rx="12" fill="${T.card}" stroke="${T.line}" stroke-opacity="0.18"/>`;
  const card = (x, bw, title, sub, rows) =>
    [
      box(x, top, bw, boxH),
      text(monoMedium, fit(monoMedium, title, 26, bw - 2 * pad), x + pad, top + 50, 26, T.fg),
      text(monoMedium, fit(monoMedium, sub, 16, bw - 2 * pad), x + pad, top + 80, 16, T.muted),
      `<path d="M${x + pad} ${top + 104}H${x + bw - pad}" stroke="${T.line}" stroke-opacity="0.1"/>`,
      ...rows.map(([s, fill], i) => text(monoMedium, fit(monoMedium, s, 18, bw - 2 * pad), x + pad, top + 146 + i * 30, 18, fill)),
    ].join("");
  const arrow = (x1, x2, y, color, label) =>
    `<path d="M${x1 + 6} ${y}H${x2 - 8}" stroke="${color}" stroke-width="1.5"/><path d="M${x2 - 16} ${y - 5}L${x2 - 6} ${y}L${x2 - 16} ${y + 5}Z" fill="${color}"/>` +
    text(monoMedium, label, (x1 + x2) / 2 - width(monoMedium, label, 16) / 2, y - 14, 16, color);
  const body = [
    card(repo, side, "Your repo", "make check", [["go.mod", T.fg], ["lint/", T.muted], ["  golangci.local.yml", T.muted], [".golangci.yml", T.fg]]),
    card(kit, mid, "vlint", "go run …/cmd/vlint@vX", [["layout", T.accentText], ["clockseam · envroot", T.fg], ["errstring · noskipenv", T.fg], ["doccomment · adrref", T.fg], ["golangci [-check]", T.fg], ["fix: errorlint + fmt", T.muted]]),
    card(outs, side, "Results", "fail the check", [["findings", T.fg], [".golangci.yml", T.fg], ["drift error", T.fg], ["golangci-lint run", T.muted]]),
    arrow(repo + side, kit, top + boxH / 2, T.muted, "./..."),
    arrow(kit + mid, outs, top + boxH / 2, T.accentText, "zero"),
  ].join("");
  return frame(w, h, FLOW_LABEL, body);
};

for (const [name, theme] of Object.entries(THEMES)) {
  T = theme;
  writeFileSync(join(out, `banner-hero-${name}.svg`), hero());
  writeFileSync(join(out, `diagram-flow-${name}.svg`), diagram());
}
console.log(`wrote banner-hero and diagram-flow, light and dark, to ${out}`);
