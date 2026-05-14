package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── colour tokens ────────────────────────────────────────────────────────────

// logoShell is the main lobster body colour — rich red-orange.
var logoShell = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FF6B35"})

// logoAccent is warm amber, used for antennae and eyes so they pop.
var logoAccent = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"})

// logoText is the bold brand orange for the ANSI Shadow block letters.
var logoText = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#9A3412", Dark: "#F97316"})

// ─── small logo (--version, init) ────────────────────────────────────────────
//
//	  ≋     ≋
//	 (◕  ◡  ◕)
//	═╡ ▓▓▓▓▓ ╞═
//	 ╲ ▓▓▓▓▓ ╱
//	  ╙─────╜
var logoSmallLines = [5]string{
	"   ≋     ≋",
	"  (◕  ◡  ◕)",
	" ═╡ ▓▓▓▓▓ ╞═",
	"  ╲ ▓▓▓▓▓ ╱",
	"   ╙─────╜",
}

// ─── big art (daemon banner, above the text logo) ────────────────────────────
//
//	  ≋≋≋≋              ≋≋≋≋
//	 /    \  (◕  ◡  ◕) /    \
//	< ════╡  ────────  ╞════ >
//	 \    ╚══▓▓▓▓▓▓▓══╝    /
//	      ╔══▓▓▓▓▓▓▓══╗
//	      ║  ▓▓▓▓▓▓▓  ║
//	      ║  ▓▓▓▓▓▓▓  ║
//	     ╔╝  ▓▓▓▓▓▓▓  ╚╗
//	    ╔╝   ▓▓▓▓▓▓▓   ╚╗
//	    ╙────╜   ╙────╜
var logoBigArtLines = [10]string{
	"  ≋≋≋≋              ≋≋≋≋",
	" /    \\  (◕  ◡  ◕) /    \\",
	"< ════╡  ────────  ╞════ >",
	" \\    ╚══▓▓▓▓▓▓▓══╝    /",
	"      ╔══▓▓▓▓▓▓▓══╗",
	"      ║  ▓▓▓▓▓▓▓  ║",
	"      ║  ▓▓▓▓▓▓▓  ║",
	"     ╔╝  ▓▓▓▓▓▓▓  ╚╗",
	"    ╔╝   ▓▓▓▓▓▓▓   ╚╗",
	"    ╙────╜   ╙────╜",
}

// ─── text logo — ANSI Shadow block letters ────────────────────────────────────
//
//	██╗      ██████╗ ██████╗ ███████╗████████╗███████╗██████╗
//	██║     ██╔═══██╗██╔══██╗██╔════╝╚══██╔══╝██╔════╝██╔══██╗
//	██║     ██║   ██║██████╔╝███████╗   ██║   █████╗  ██████╔╝
//	██║     ██║   ██║██╔══██╗╚════██║   ██║   ██╔══╝  ██╔══██╗
//	███████╗╚██████╔╝██████╔╝███████║   ██║   ███████╗██║  ██║
//	╚══════╝ ╚═════╝ ╚═════╝ ╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝
const logoTextBlock = `██╗      ██████╗ ██████╗ ███████╗████████╗███████╗██████╗ 
██║     ██╔═══██╗██╔══██╗██╔════╝╚══██╔══╝██╔════╝██╔══██╗
██║     ██║   ██║██████╔╝███████╗   ██║   █████╗  ██████╔╝
██║     ██║   ██║██╔══██╗╚════██║   ██║   ██╔══╝  ██╔══██╗
███████╗╚██████╔╝██████╔╝███████║   ██║   ███████╗██║  ██║
╚══════╝ ╚═════╝ ╚═════╝ ╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝`

// ─── Public render functions ──────────────────────────────────────────────────

// LogoSmall returns the 5-line cute ASCII lobster icon in two-tone colour.
// Antennae and eyes glow amber; claws and body are lobster red-orange.
// Used for --version output and lobster init.
func LogoSmall() string {
	lines := make([]string, len(logoSmallLines))
	for i, l := range logoSmallLines {
		if i < 2 {
			lines[i] = logoAccent.Render(l)
		} else {
			lines[i] = logoShell.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

// LogoText returns "LOBSTER" in 6-line ANSI Shadow block letters.
func LogoText() string {
	return logoText.Render(logoTextBlock)
}

// LogoBanner returns the full help/root banner: the small lobster icon
// horizontally joined to the right of the ANSI Shadow text logo, so that
// running bare `lobster` or `lobster --help` shows both the character and
// the bold wordmark side-by-side.
//
//	╔═══════════════════════════════════════════════════╗
//	║  ██╗      ██████╗ ██████╗ ...   ║   ≋     ≋      ║
//	║  ██║     ██╔═══██╗...           ║  (◕  ◡  ◕)     ║
//	║  ...                            ║ ═╡ ▓▓▓▓▓ ╞═    ║
//	║                                 ║  ╲ ▓▓▓▓▓ ╱     ║
//	║                                 ║   ╙─────╜       ║
//	╚═══════════════════════════════════════════════════╝
func LogoBanner() string {
	textBlock := logoText.Render(logoTextBlock)
	// Build the small art with an extra top blank line so it vertically
	// centres alongside the 6-line text block.
	smallLines := make([]string, len(logoSmallLines))
	for i, l := range logoSmallLines {
		if i < 2 {
			smallLines[i] = logoAccent.Render(l)
		} else {
			smallLines[i] = logoShell.Render(l)
		}
	}
	art := "\n" + strings.Join(smallLines, "\n") // pad top to match 6 lines

	artBlock := lipgloss.NewStyle().
		PaddingLeft(4).
		Render(art)

	return lipgloss.JoinHorizontal(lipgloss.Top, textBlock, artBlock)
}

// LogoTUI returns a compact single-line wordmark for the TUI header.
func LogoTUI() string {
	brackets := logoShell.Render("─<")
	name := logoText.Render("  LOBSTER  ")
	bracketsR := logoShell.Render(">─")
	return brackets + name + bracketsR
}

// LogoBig returns the full daemon banner: the detailed lobster art followed
// by the ANSI Shadow text logo. Used for lobsterd start output.
func LogoBig() string {
	artLines := make([]string, len(logoBigArtLines))
	for i, l := range logoBigArtLines {
		if i < 2 {
			artLines[i] = logoAccent.Render(l)
		} else {
			artLines[i] = logoShell.Render(l)
		}
	}
	art := strings.Join(artLines, "\n")
	return art + "\n\n" + logoText.Render(logoTextBlock)
}

// LogoVersion returns the small logo followed by the name and version line.
// version should be the bare version string, e.g. "v0.1.0" or "dev".
func LogoVersion(name, version string) string {
	header := LogoSmall()
	ver := fmt.Sprintf("\n%s  %s\n",
		logoText.Render(name),
		StyleMuted.Render(version),
	)
	return header + ver
}
