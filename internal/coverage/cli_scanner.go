package coverage

// CLICommands is the canonical list of all lobster CLI commands and subcommands.
// Each entry produces one CoverageItem with KindCLI. The IDs match the tag
// convention: "cli:<command>" or "cli:<command>:<subcommand>".
//
// Update this list whenever a new command is added to the CLI.
var CLICommands = []struct {
	ID    string
	Label string
}{
	{"cli:run", "lobster run"},
	{"cli:run:watch", "lobster run watch"},
	{"cli:run:status", "lobster run status"},
	{"cli:run:cancel", "lobster run cancel"},
	{"cli:validate", "lobster validate"},
	{"cli:lint", "lobster lint"},
	{"cli:plan", "lobster plan"},
	{"cli:init", "lobster init"},
	{"cli:config", "lobster config"},
	{"cli:runs", "lobster runs"},
	{"cli:plans", "lobster plans"},
	{"cli:stack", "lobster stack"},
	{"cli:integrations", "lobster integrations"},
	{"cli:admin:health", "lobster admin health"},
	{"cli:admin:capabilities", "lobster admin capabilities"},
	{"cli:admin:config", "lobster admin config"},
	{"cli:doctor", "lobster doctor"},
	{"cli:tui", "lobster tui"},
	{"cli:coverage", "lobster coverage"},
}

// ScanCLI returns one CoverageItem per entry in CLICommands.
func ScanCLI() []CoverageItem {
	items := make([]CoverageItem, 0, len(CLICommands))
	for _, c := range CLICommands {
		items = append(items, CoverageItem{
			ID:         c.ID,
			Kind:       KindCLI,
			Label:      c.Label,
			CLICommand: c.ID,
		})
	}
	return items
}
