# ─────────────────────────────────────────────────────────────────────────────
# Lobster demo recording
#
# Shows: cd into tests/ → lobster run against shell-steps.feature → all 14
# scenario results visible → final PASSED summary.
#
# Usage (run from project root):
#   vhs docs/demo.tape
#
# Output: docs/demo.gif
# Requires: vhs (brew install vhs), lobster on PATH (make install)
# ─────────────────────────────────────────────────────────────────────────────

Output docs/demo.gif

Set Width        860
Set Height       500
Set FontSize     13
Set Framerate    10
Set PlaybackSpeed 1

Set Theme "Catppuccin Mocha"
Set Margin       20
Set MarginFill   "#1e1e2e"

# ─── Setup (hidden) ──────────────────────────────────────────────────────────
Hide
Type "cd /Users/bencoeppicus/Documents/bcp-technology/opensource/lobster/tests"
Enter
Sleep 300ms
Type "export PS1='~/tests ❯ '"
Enter
Sleep 300ms
Type "clear"
Enter
Sleep 300ms
Show

# ─── Run a single feature file so every result is visible ────────────────────
Sleep 1s
Type "lobster run --features 'features/shell-steps.feature' --no-compose --ci"
Enter

# Run completes in ~100ms; wait a moment then linger on the full output
Sleep 2s

# ─── Hold on the summary ─────────────────────────────────────────────────────
Sleep 4s
