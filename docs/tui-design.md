# TUI Design System

This document captures the visual design language, layout principles, and implementation decisions
for the lobster interactive terminal UI. Any future TUI work should reference and extend these
conventions rather than introduce competing patterns.

---

## Core principles

**1. Everything is centered.**
All content is placed in the horizontal center of the terminal. Nothing clings to the left edge.
Use `TUICenter(m.width, content)` to achieve this for any rendered block.

**2. Maximum content width of 128 columns.**
On wide terminals content would become hard to read if it stretched the full width. Cards cap at
128 columns and add 3-character side margins, giving breathing room on any screen size.

**3. The card is the unit of composition.**
Every tab pane renders its content inside a single rounded-border card (`TUICardStyle`). The card
supplies the border, padding, and max-width in one consistent container. Sub-models do not draw
their own borders.

**4. Structured visual hierarchy: logo → tabs → card → footer.**
The screen is always divided into these four rows in that order with single blank-line separators
between them. This means your eyes know exactly where to look regardless of which tab is active.

**5. Color carries semantic meaning, not decoration.**
- Violet / indigo (`colorPrimary`, `colorSecondary`): chrome, active state, headings
- Cyan-teal: accent only (reserved for future graph/timeline elements)
- Colored background badges: status cells exclusively — each status has one badge, used nowhere else
- Muted gray: secondary text, hints, placeholder copy
- Never use `colorPrimary` for error states or `colorError` for headings.

---

## Color palette

All colors are `lipgloss.AdaptiveColor` with separate light-terminal and dark-terminal values.

| Token           | Dark terminal | Light terminal | Semantic meaning         |
|-----------------|---------------|----------------|--------------------------|
| `colorPrimary`  | `#A78BFA`     | `#6D28D9`      | Brand violet — chrome    |
| `colorSecondary`| `#818CF8`     | `#4F46E5`      | Indigo — sub-headings    |
| `colorSuccess`  | `#4ADE80`     | `#15803D`      | Pass / healthy           |
| `colorWarning`  | `#FCD34D`     | `#B45309`      | Degraded / pending       |
| `colorError`    | `#FCA5A5`     | `#B91C1C`      | Failure                  |
| `colorMuted`    | `#9CA3AF`     | `#6B7280`      | Secondary / hints        |
| `colorBorder`   | `#374151`     | `#D1D5DB`      | Subtle border             |
| `colorHighlight`| `#1E1B4B`     | `#EDE9FE`      | Inline code bg            |

---

## Status badges

Status values in table cells use pill-shaped background badges rather than plain colored text.
This ensures status is readable at a glance and does not rely on foreground color alone (accessible
for low-contrast modes).

| Badge var        | Background (dark)       | Text      | Used for                   |
|------------------|-------------------------|-----------|----------------------------|
| `BadgeRunning`   | `#155E75` cyan-dark     | `#E0FFFF` | Run: running               |
| `BadgePassed`    | `#14532D` green-dark    | `#D1FAE5` | Run: passed / healthy      |
| `BadgeFailed`    | `#7F1D1D` red-dark      | `#FEE2E2` | Run: failed / unhealthy    |
| `BadgePending`   | `#78350F` amber-dark    | `#FEF3C7` | Run: pending / starting    |
| `BadgeCancelled` | `#1F2937` neutral-dark  | `#F3F4F6` | Run: cancelled / unknown   |

`Padding(0, 1)` on all badges gives one character of horizontal breathing room inside the pill.

---

## Tab bar

Tabs use circled digit glyphs (①②③④) as a compact prefix before the tab name. This makes
keyboard shortcuts visually self-documenting without requiring separate help text.

- **Active tab**: `TUITabActive` — filled violet background, white bold text, 2-char horizontal padding
- **Inactive tab**: `TUITabInactive` — muted foreground, same padding, no background
- Tabs are joined with two spaces of separation and the entire row is centered

Tab keyboard bindings:
- `tab` / `shift+tab` — cycle forward/backward
- `1`–`4` — jump to numbered tab directly
- `h` / `l` — vim-style left/right (lobby level)

---

## Typography

| Style var               | Weight  | Color          | Usage                           |
|-------------------------|---------|----------------|---------------------------------|
| `TUILogoStyle`          | Bold    | `colorPrimary` | Top-of-screen wordmark          |
| `TUICardHeaderStyle`    | Bold    | `colorPrimary` | Pane title inside card          |
| `StyleHeading`          | Bold    | `colorPrimary` | CLI command output headings     |
| `StyleSubheading`       | Bold    | `colorSecondary`| Section sub-titles             |
| `StyleMuted`            | Normal  | `colorMuted`   | Secondary copy, hints           |
| `TUIFooterKeyStyle`     | Bold    | `colorSecondary`| Key names in hint bars         |

Key hint bars always use `renderKeyHint(key, action)` which outputs **bold-secondary key** followed
by muted action text. Multiple hints are joined with `  ·  ` (muted separator).

---

## Layout skeleton

```
                    (blank line)
            🦞  lobster
            workspace: <id>          ← only when non-empty

            ①  Live Runs  ②  History  ③  Stack  ④  Admin

   ╭──────────────────────────────────────────────────────────╮
   │  Run History                                             │
   │                                                          │
   │  ID        Status     Workspace    Scenarios  …          │
   │  ────────  ─────────  ───────────  ─────────  …          │
   │  abc12345  ◼ passed   lobster-ci   12/12      …          │
   │  …                                                        │
   │                                                          │
   │  ↵ detail   / filter   r refresh                         │
   ╰──────────────────────────────────────────────────────────╯

            tab/shift+tab  ·  1–4  ·  r  ·  ctrl+c
```

The lobby model owns the logo row, tab row, and footer row. Sub-models own only the card content.

---

## Responsive column sizing

Tables inside cards must not overflow the card. On every `tea.WindowSizeMsg` the pane model
recalculates its card width and distributes the available inner width as follows:

1. `cardWidth = min(terminalWidth - 6, 128)` — subtract margin, cap at max
2. `tableInner = cardWidth - 6` — subtract card border (2) + card padding (4)
3. Fixed-width columns (ID, Status, Duration, Created) get deterministic widths
4. One fluid "workspace" or "service" column absorbs the remainder

This means the table never clips text on narrow terminals (floor at 60 cols) and never leaves
dead whitespace on ultra-wide terminals.

---

## WindowSizeMsg propagation

The lobby model propagates a reduced `tea.WindowSizeMsg` to each sub-model on every resize:

```go
inner := tea.WindowSizeMsg{Width: m.width, Height: m.height - 7}
// 7 = logo(2) + tab-bar(1) + blank-lines(3) + footer(1)
```

Sub-models must handle `WindowSizeMsg` and update both table height and viewport dimensions.
Viewports should be sized to `height - 12` to account for card chrome plus header/footer rows.

---

## Viewport content

Detail views (run detail, plan detail) use a `bubbles/viewport` for scrollable content. Key rules:

- Viewport width = `cardWidth - 10` (card border + padding + inner gutter)
- Viewport height = `max(3, height - 12)`
- Always call `viewport.SetContent` after resize so lipgloss-rendered content reflows correctly
- Header and hint footer live outside the viewport, inside the card

---

## Icon constants

Icons defined in `theme.go` and used across CLI and TUI output:

```
✓  IconCheck    — success confirmation
✗  IconCross    — failure / error label
⚠  IconWarning  — caution / degraded
ℹ  IconInfo     — informational
→  IconArrow    — direction / action
·  IconDot      — list bullet
🚀 IconRocket   — coming-soon placeholder
🦞            — wordmark glyph in lobby header
```

---

## Dos and don'ts

| Do | Don't |
|----|-------|
| Use `TUICenter` for every rendered block in a pane | Left-align directly to terminal edge |
| Use `TUICardStyle.Width(cw - 6).Render(content)` | Add a second border inside a card |
| Use badges for all status cells | Use plain `StyleSuccess.Render("passed")` in table cells |
| Use `renderKeyHint(key, action)` for footer bars | Write `[bracket]` style hints |
| Derive table column widths from `cardWidth` at runtime | Hard-code column widths |
| Keep the four-zone layout (logo/tabs/card/footer) intact | Insert additional fixed rows |
| Add new tab badge styles to `theme.go` alongside existing ones | Define one-off styles inline |
