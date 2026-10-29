---
name: "Lucid-CI Platform"
description: "Autonomous AI-Native DevSecOps Command Center"
colors:
  canvas-base: "#09090b"
  surface-elevated: "#18181b"
  surface-hover: "#27272a"
  border-subtle: "#27272a"
  border-strong: "#3f3f46"
  text-primary: "#f4f4f5"
  text-secondary: "#a1a1aa"
  text-muted: "#71717a"
  semantic-danger: "#ef4444"
  semantic-warning: "#f59e0b"
  semantic-success: "#10b981"
  semantic-info: "#3b82f6"
typography:
  display:
    fontFamily: "Inter, -apple-system, BlinkMacSystemFont, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: "2rem"
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "Inter, -apple-system, BlinkMacSystemFont, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 600
    lineHeight: "1.5rem"
    letterSpacing: "-0.015em"
  body:
    fontFamily: "Inter, -apple-system, BlinkMacSystemFont, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: "1.25rem"
    letterSpacing: "0em"
  code:
    fontFamily: "JetBrains Mono, Menlo, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: "1.125rem"
    letterSpacing: "0em"
rounded:
  xs: "2px"
  sm: "4px"
  md: "6px"
  lg: "8px"
  pill: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
components:
  badge-danger:
    backgroundColor: "rgba(239, 68, 68, 0.1)"
    textColor: "{colors.semantic-danger}"
    rounded: "{rounded.md}"
    padding: "2px 8px"
  badge-success:
    backgroundColor: "rgba(16, 185, 129, 0.1)"
    textColor: "{colors.semantic-success}"
    rounded: "{rounded.md}"
    padding: "2px 8px"
  button-primary:
    backgroundColor: "{colors.surface-elevated}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.md}"
    padding: "6px 12px"
---

## Overview

Lucid-CI's interface operates strictly in **Operate Mode**: an authenticated DevSecOps intelligence command center engineered for developers and security engineers. The UI prioritizes high-density data presentation, scanability, and earned familiarity over decorative visual noise.

Inspired by industry-standard systems (Linear, Vercel, GitHub), the interface eliminates floaty card paddings, exaggerated drop-shadows, and gratuitous transitions. Instead, it provides tight, information-rich tables, flush split-pane workbenches, interactive AST syntax graphs, and git-native inline patch diffs.

---

## Colors

The palette is rooted in a pure, neutral monochromatic dark spectrum, reserving chromatic saturation exclusively for security semantics:

### Neutral Foundations
- **Canvas Base (`#09090b` / `zinc-950`):** Pure background for the root canvas and code viewports.
- **Surface Elevated (`#18181b` / `zinc-900`):** Component cards, sidebars, toolbars, and table row groupings.
- **Surface Hover (`#27272a` / `zinc-800`):** Interactive row and button hover states.
- **Borders (`#27272a` subtle, `#3f3f46` strong):** 1px sharp structural lines separating regions without drop shadows.
- **Text Layers:** Primary (`#f4f4f5`), Secondary (`#a1a1aa`), Muted (`#71717a`).

### Security Semantic Accents
- **Danger / Sinks / Critical (`#ef4444` / Red):** Taint sinks, critical vulnerabilities, failed sandboxes, deleted lines.
- **Warning / Sources / High (`#f59e0b` / Amber):** Untrusted input sources, high-severity findings, in-progress scans.
- **Success / Sanitizers (`#10b981` / Emerald):** Sanitization functions, passing checks, verified gVisor containment, added lines.
- **Info / Tokens (`#3b82f6` / Blue):** AST nodes, PR numbers, commit SHAs, line numbers.

---

## Typography

Typography establishes an unambiguous visual hierarchy with tight vertical rhythmic scales:

| Role | Font Family | Size | Weight | Tracking | Usage |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Display** | `Inter` | 24px (1.5rem) | 600 | -0.02em | Page headers, overview numbers |
| **Headline** | `Inter` | 16px (1rem) | 600 | -0.015em | Panel titles, modal titles |
| **Body UI** | `Inter` | 13px (0.8125rem) | 400 / 500 | 0em | Table text, labels, button text |
| **Code / Data**| `JetBrains Mono` | 12px (0.75rem) | 400 / 500 | 0em | AST nodes, commit SHAs, code diffs, line gutters |
| **Micro Tag** | `JetBrains Mono` | 11px (0.6875rem) | 500 | 0.02em | Status pill labels, CWE badges |

---

## Layout

- **Workbench Grid:** 12-column layout with 4 columns for findings navigation and 8 columns for interactive AST flow / diff inspection.
- **Flush Gutters:** Elimination of excessive margins (using 12px to 16px gutters) allowing dense telemetry visualization.
- **Split Diff Panels:** Monospaced diff tables with fixed-width line number gutters (`w-12`), explicit `+` and `-` markers, and syntax-preserving horizontal overflow.

---

## Elevation & Depth

- **Border-Led Hierarchy:** Surfaces are defined by 1px solid borders (`border-zinc-800`), not blurred drop-shadows.
- **Backdrop Blur:** Subtle `backdrop-blur-md` with `bg-zinc-950/80` for fixed headers and sticky navigation bars.
- **Layer Stacking:** Canvas (`#09090b`) $\to$ Container (`#18181b`) $\to$ Hover/Overlay (`#27272a`).

---

## Shapes

- **Base Radius:** 6px (`rounded-md`) for buttons, form controls, and table rows; 8px (`rounded-lg`) for major container panels.
- **Micro Badges:** 4px (`rounded-sm`) for inline code tags and CWE chips.
- **Pills:** Full rounded (`rounded-full`) reserved strictly for live state indicators and tag pills.

---

## Components

### 1. AST Flow Graph (`ASTFlowGraph.tsx`)
- Dark-themed node cards (`bg-zinc-950`, border `zinc-800`).
- Status headers: `SOURCE` (amber), `TAINT` (indigo), `SINK` (rose), `SANITIZER` (emerald).
- Animated red edges (`#ef4444`) with smooth Bezier curvature.
- MiniMap with semantic node dot coloring.

### 2. Inline Diff Viewer (`DiffViewer.tsx`)
- Monospaced line-by-line unified diff.
- Rose background for deletions (`bg-rose-950/25`), emerald background for additions (`bg-emerald-950/25`).
- One-click copy patch with visual confirmation.

### 3. Metric Ribbons (`DashboardView.tsx`)
- Flush horizontal bar displaying measured latency (`< 21ms`), gVisor `runsc` containment, and prompt defense (`0.0% ASR`).
- Monospaced commit hashes and tabular numeric alignment.

---

## Do's and Don'ts

### Do
- Use monospaced fonts (`JetBrains Mono`) for all hashes, lines, and AST nodes.
- Maintain high information density and tabular alignment.
- Keep border lines sharp, subtle, and 1px wide.
- Use color strictly for security state and data flow meaning.

### Don't
- Don't use colorful gradient cards or decorative shadows.
- Don't use large, floaty 24px+ paddings inside operational data tables.
- Don't use display serif fonts anywhere on the dashboard.
- Don't hide critical telemetry behind hover-only tooltips.
