package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	adminv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/admin"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// doctorStatus represents the outcome level of a single health check.
type doctorStatus int

const (
	doctorOK   doctorStatus = iota // check passed
	doctorWarn                     // advisory — not blocking
	doctorFail                     // error — exit non-zero
)

// doctorCheck is a single diagnostic result emitted by `lobster doctor`.
type doctorCheck struct {
	Group  string `json:"group"`
	Label  string `json:"label"`
	status doctorStatus
	Status string `json:"status"` // "ok" | "warning" | "error"
	Detail string `json:"detail"`
}

// mkCheck constructs a doctorCheck, deriving the string Status from s.
func mkCheck(group, label string, s doctorStatus, detail string) doctorCheck {
	statusStr := "ok"
	switch s {
	case doctorWarn:
		statusStr = "warning"
	case doctorFail:
		statusStr = "error"
	}
	return doctorCheck{Group: group, Label: label, status: s, Status: statusStr, Detail: detail}
}

func newDoctorCommand(v *viper.Viper) *cobra.Command {
	var (
		format     string
		daemonAddr string
		authToken  string
		caFile     string
		certFile   string
		keyFile    string
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check lobster environment and configuration health",
		Long: "Inspect tool dependencies, configuration files, feature paths, persistence wiring,\n" +
			"and optional integrations. Exits non-zero when any check reports an error.",
		// Override the root PersistentPreRunE so doctor can run even when
		// lobster.yaml is missing or malformed — config errors are surfaced as
		// check results rather than pre-run failures.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cfgFile := ""
			if f := cmd.InheritedFlags().Lookup("config"); f != nil {
				cfgFile = f.Value.String()
			}
			_ = initViper(v, cfgFile) // best-effort; errors are reported as check items
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			cwd, _ := os.Getwd()
			var checks []doctorCheck

			// 1. Tool dependencies (docker, docker compose, git).
			checks = append(checks, checkDoctorTools()...)

			// 2. lobster.yaml discovery — walk up to git root AND down into subtree.
			configPaths := checkDoctorConfig(cwd, v, &checks)

			// 3. Feature files (uses config loaded above).
			checkDoctorFeatures(v, &checks)

			// 4. SQLite persistence path.
			checkDoctorPersistence(v, &checks)

			// 5. Docker Compose file existence.
			checkDoctorCompose(v, &checks)

			// 6. .lobster/ directory structure.
			checkDoctorLobsterDir(cwd, configPaths, v, &checks)

			// 7. Migrations directory.
			checkDoctorMigrations(v, &checks)

			// 8. Daemon reachability (only when mode=daemon or --executor-addr provided).
			addr := daemonAddr
			if addr == "" {
				addr = v.GetString("execution.executor_addr")
			}
			if v.GetString("execution.mode") == "daemon" || addr != "" {
				checks = append(checks, checkDoctorDaemon(ctx, addr, authToken, caFile, certFile, keyFile))
			}

			// 9. Integration adapters (Keycloak, etc.).
			checkDoctorIntegrations(ctx, v, &checks)

			// Determine overall exit status.
			hasError := false
			for _, c := range checks {
				if c.status == doctorFail {
					hasError = true
					break
				}
			}

			if strings.EqualFold(format, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(checks)
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), renderDoctorChecks(checks))

			if hasError {
				return &ExitError{Code: ExitConfigError}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json")
	cmd.Flags().StringVar(&daemonAddr, "executor-addr", "", "lobsterd address to probe (host:port)")
	cmd.Flags().StringVar(&authToken, "auth-token", "", "bearer token for daemon connection")
	cmd.Flags().StringVar(&caFile, "tls-ca-file", "", "TLS CA certificate file for daemon connection")
	cmd.Flags().StringVar(&certFile, "tls-cert-file", "", "TLS client certificate file for daemon connection")
	cmd.Flags().StringVar(&keyFile, "tls-key-file", "", "TLS client key file for daemon connection")

	return cmd
}

// ---------------------------------------------------------------------------
// Individual check implementations
// ---------------------------------------------------------------------------

func checkDoctorTools() []doctorCheck {
	const group = "Tools"
	var results []doctorCheck

	// docker
	if _, err := exec.LookPath("docker"); err != nil {
		results = append(results, mkCheck(group, "docker", doctorFail, "not found in PATH"))
	} else {
		out, _ := exec.Command("docker", "version", "--format", "{{.Client.Version}}").Output()
		ver := strings.TrimSpace(string(out))
		if ver == "" {
			ver = "installed"
		}
		results = append(results, mkCheck(group, "docker", doctorOK, ver))
	}

	// docker compose (v2 plugin)
	out, err := exec.Command("docker", "compose", "version", "--short").Output()
	if err != nil {
		results = append(results, mkCheck(group, "docker compose", doctorFail,
			"not available (requires Docker Compose v2 plugin)"))
	} else {
		results = append(results, mkCheck(group, "docker compose", doctorOK,
			strings.TrimSpace(string(out))))
	}

	// git
	if _, err := exec.LookPath("git"); err != nil {
		results = append(results, mkCheck(group, "git", doctorFail, "not found in PATH"))
	} else {
		out, _ := exec.Command("git", "--version").Output()
		ver := strings.TrimSpace(string(out))
		if ver == "" {
			ver = "installed"
		}
		results = append(results, mkCheck(group, "git", doctorOK, ver))
	}

	return results
}

// checkDoctorConfig discovers lobster.yaml files by walking up to the git root
// and down into the directory subtree (max 4 levels), then validates the active
// config loaded by viper. It appends check results to checks and returns the
// list of discovered config file paths.
func checkDoctorConfig(cwd string, v *viper.Viper, checks *[]doctorCheck) []string {
	const group = "Configuration"

	upPaths := findLobsterYAMLUp(cwd)
	downPaths := findLobsterYAMLDown(cwd, upPaths)
	all := append(upPaths, downPaths...)

	// Deduplicate by absolute path.
	seen := make(map[string]bool)
	var unique []string
	for _, p := range all {
		abs, _ := filepath.Abs(p)
		if !seen[abs] {
			seen[abs] = true
			unique = append(unique, p)
		}
	}

	if len(unique) == 0 {
		*checks = append(*checks, mkCheck(group, "lobster.yaml", doctorWarn,
			"not found in directory tree — run `lobster init` to create one"))
		return nil
	}

	cfgUsed := v.ConfigFileUsed()
	cfgUsedAbs := ""
	if cfgUsed != "" {
		cfgUsedAbs, _ = filepath.Abs(cfgUsed)
	}

	for _, p := range unique {
		abs, _ := filepath.Abs(p)
		rel, err := filepath.Rel(cwd, abs)
		if err != nil {
			rel = p
		}
		label := "lobster.yaml"
		if cfgUsedAbs != "" && abs == cfgUsedAbs {
			label = "lobster.yaml (active)"
		}
		*checks = append(*checks, mkCheck(group, label, doctorOK, rel))
	}

	// Validate key fields in the active config.
	if cfgUsed != "" {
		if project := v.GetString("project"); project == "" {
			*checks = append(*checks, mkCheck(group, "project name", doctorWarn,
				"not set in lobster.yaml"))
		} else {
			*checks = append(*checks, mkCheck(group, "project name", doctorOK, project))
		}
	}

	return unique
}

func checkDoctorFeatures(v *viper.Viper, checks *[]doctorCheck) {
	const group = "Features"

	paths := v.GetStringSlice("features.paths")
	if len(paths) == 0 {
		*checks = append(*checks, mkCheck(group, "features.paths", doctorWarn,
			"no paths configured in lobster.yaml"))
		return
	}
	for _, glob := range paths {
		matches, err := filepath.Glob(glob)
		if err != nil {
			*checks = append(*checks, mkCheck(group, glob, doctorFail,
				fmt.Sprintf("invalid glob: %v", err)))
			continue
		}
		if len(matches) == 0 {
			*checks = append(*checks, mkCheck(group, glob, doctorWarn, "no files matched"))
		} else {
			*checks = append(*checks, mkCheck(group, glob, doctorOK,
				fmt.Sprintf("%d file(s) found", len(matches))))
		}
	}
}

func checkDoctorPersistence(v *viper.Viper, checks *[]doctorCheck) {
	const group = "Persistence"

	sqlitePath := v.GetString("persistence.sqlite.path")
	if sqlitePath == "" {
		sqlitePath = ".lobster/lobster.db"
	}
	dir := filepath.Dir(sqlitePath)

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			*checks = append(*checks, mkCheck(group, "sqlite dir", doctorWarn,
				fmt.Sprintf("%s does not exist (will be created on first run)", dir)))
		} else {
			*checks = append(*checks, mkCheck(group, "sqlite dir", doctorFail,
				fmt.Sprintf("%s: %v", dir, err)))
		}
		return
	}
	if !info.IsDir() {
		*checks = append(*checks, mkCheck(group, "sqlite dir", doctorFail,
			fmt.Sprintf("%s is not a directory", dir)))
		return
	}

	// Probe writability with a temporary file.
	tmp, err := os.CreateTemp(dir, ".lobster-doctor-probe-*")
	if err != nil {
		*checks = append(*checks, mkCheck(group, "sqlite dir", doctorFail,
			fmt.Sprintf("%s not writable: %v", dir, err)))
		return
	}
	_ = tmp.Close()
	_ = os.Remove(tmp.Name())

	*checks = append(*checks, mkCheck(group, "sqlite path", doctorOK, sqlitePath))
}

func checkDoctorCompose(v *viper.Viper, checks *[]doctorCheck) {
	const group = "Docker Compose"

	files := v.GetStringSlice("compose.files")
	if len(files) == 0 {
		*checks = append(*checks, mkCheck(group, "compose.files", doctorWarn,
			"no compose files configured"))
		return
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			*checks = append(*checks, mkCheck(group, f, doctorFail, "file not found"))
		} else {
			*checks = append(*checks, mkCheck(group, f, doctorOK, "exists"))
		}
	}
}

func checkDoctorLobsterDir(cwd string, configPaths []string, v *viper.Viper, checks *[]doctorCheck) {
	const group = ".lobster directory"

	// Use the directory of the active config file as the base, then fall back
	// to the first discovered config path, and finally to cwd.
	base := cwd
	if cfgUsed := v.ConfigFileUsed(); cfgUsed != "" {
		base = filepath.Dir(cfgUsed)
	} else if len(configPaths) > 0 {
		if abs, err := filepath.Abs(configPaths[0]); err == nil {
			base = filepath.Dir(abs)
		}
	}

	lobsterDir := filepath.Join(base, ".lobster")
	if _, err := os.Stat(lobsterDir); err != nil {
		*checks = append(*checks, mkCheck(group, ".lobster/", doctorWarn,
			"not found — run `lobster init` to scaffold"))
		return
	}
	*checks = append(*checks, mkCheck(group, ".lobster/", doctorOK, lobsterDir))

	// Plans / blob directory.
	blobDir := v.GetString("persistence.plans.blob_dir")
	if blobDir == "" {
		blobDir = filepath.Join(lobsterDir, "plans")
	}
	if _, err := os.Stat(blobDir); err != nil {
		*checks = append(*checks, mkCheck(group, "plans dir", doctorWarn,
			fmt.Sprintf("%s not found", blobDir)))
	} else {
		*checks = append(*checks, mkCheck(group, "plans dir", doctorOK, blobDir))
	}
}

func checkDoctorMigrations(v *viper.Viper, checks *[]doctorCheck) {
	const group = "Migrations"

	dir := v.GetString("persistence.migrations.dir")
	if dir == "" {
		dir = "migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		*checks = append(*checks, mkCheck(group, "migrations dir", doctorWarn,
			fmt.Sprintf("%s not found (embedded migrations used if available)", dir)))
	} else {
		*checks = append(*checks, mkCheck(group, "migrations dir", doctorOK, dir))
	}
}

func checkDoctorDaemon(ctx context.Context, addr, authToken, caFile, certFile, keyFile string) doctorCheck {
	const group = "Daemon"

	if addr == "" {
		return mkCheck(group, "lobsterd", doctorWarn,
			"no executor address configured (set execution.executor_addr or --executor-addr)")
	}

	conn, err := dialDaemon(ctx, addr, authToken, caFile, certFile, keyFile)
	if err != nil {
		return mkCheck(group, "lobsterd", doctorFail,
			fmt.Sprintf("connection to %s failed: %v", addr, err))
	}
	defer conn.Close()

	client := adminv1.NewAdminServiceClient(conn)
	resp, err := client.GetHealth(ctx, &adminv1.GetHealthRequest{})
	if err != nil {
		return mkCheck(group, "lobsterd", doctorFail,
			fmt.Sprintf("health check at %s failed: %v", addr, err))
	}

	h := resp.GetHealth()
	detail := fmt.Sprintf("%s — live=%v ready=%v version=%s",
		addr, h.GetLive(), h.GetReady(), h.GetVersion())
	if !h.GetLive() || !h.GetReady() {
		return mkCheck(group, "lobsterd", doctorWarn, detail)
	}
	return mkCheck(group, "lobsterd", doctorOK, detail)
}

func checkDoctorIntegrations(ctx context.Context, v *viper.Viper, checks *[]doctorCheck) {
	const group = "Integrations"

	kcURL := v.GetString("integrations.keycloak.url")
	if kcURL == "" {
		return // not configured — skip silently
	}

	hc := &http.Client{Timeout: 5 * time.Second}
	healthURL := strings.TrimRight(kcURL, "/") + "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		*checks = append(*checks, mkCheck(group, "keycloak", doctorFail,
			fmt.Sprintf("invalid URL %q: %v", kcURL, err)))
		return
	}

	resp, err := hc.Do(req)
	if err != nil {
		*checks = append(*checks, mkCheck(group, "keycloak", doctorFail,
			fmt.Sprintf("%s unreachable: %v", kcURL, err)))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 400 {
		*checks = append(*checks, mkCheck(group, "keycloak", doctorOK,
			fmt.Sprintf("%s (HTTP %d)", kcURL, resp.StatusCode)))
	} else {
		*checks = append(*checks, mkCheck(group, "keycloak", doctorWarn,
			fmt.Sprintf("%s returned HTTP %d", kcURL, resp.StatusCode)))
	}
}

// ---------------------------------------------------------------------------
// Directory-search helpers
// ---------------------------------------------------------------------------

// findLobsterYAMLUp walks from dir up to the git repository root (or
// filesystem root if not in a repo), collecting any lobster.yaml files found
// along the way.
func findLobsterYAMLUp(dir string) []string {
	gitRoot := doctorGitRoot()

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}

	var found []string
	cur := abs
	for {
		candidate := filepath.Join(cur, "lobster.yaml")
		if _, err := os.Stat(candidate); err == nil {
			found = append(found, candidate)
		}
		if gitRoot != "" && cur == gitRoot {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break // filesystem root reached
		}
		cur = parent
	}
	return found
}

// findLobsterYAMLDown recursively scans dir (up to 4 levels deep) for
// lobster.yaml files, skipping hidden directories, vendor, and node_modules.
// Files already present in exclude (by absolute path) are omitted.
func findLobsterYAMLDown(dir string, exclude []string) []string {
	excludeAbs := make(map[string]bool, len(exclude))
	for _, p := range exclude {
		abs, _ := filepath.Abs(p)
		excludeAbs[abs] = true
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}

	var found []string
	_ = filepath.Walk(absDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			// Skip hidden dirs (except the scan root), vendor, and node_modules.
			if p != absDir && (strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules") {
				return filepath.SkipDir
			}
			// Limit recursion to 4 levels below the scan root.
			rel, _ := filepath.Rel(absDir, p)
			if depth := len(strings.Split(rel, string(filepath.Separator))); depth > 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "lobster.yaml" {
			abs, _ := filepath.Abs(p)
			if !excludeAbs[abs] {
				found = append(found, p)
			}
		}
		return nil
	})
	return found
}

// doctorGitRoot returns the git repository root directory, or an empty string
// if the working directory is not inside a git repository.
func doctorGitRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ---------------------------------------------------------------------------
// Text renderer
// ---------------------------------------------------------------------------

func renderDoctorChecks(checks []doctorCheck) string {
	// Collect groups in insertion order.
	var groups []string
	groupMap := make(map[string][]doctorCheck)
	for _, c := range checks {
		if _, ok := groupMap[c.Group]; !ok {
			groups = append(groups, c.Group)
		}
		groupMap[c.Group] = append(groupMap[c.Group], c)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(ui.StyleHeading.Render(ui.IconInfo + "  lobster doctor"))
	b.WriteString("\n\n")

	for _, g := range groups {
		b.WriteString(ui.StyleSubheading.Render(g))
		b.WriteString("\n")
		for _, c := range groupMap[g] {
			var icon string
			switch c.status {
			case doctorOK:
				icon = ui.StyleSuccess.Render(ui.IconCheck)
			case doctorWarn:
				icon = ui.StyleWarning.Render(ui.IconWarning)
			default:
				icon = ui.StyleError.Render(ui.IconCross)
			}
			b.WriteString(fmt.Sprintf("  %s  %-24s%s\n", icon, c.Label, ui.StyleMuted.Render(c.Detail)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
