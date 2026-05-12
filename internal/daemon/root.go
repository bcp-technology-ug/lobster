package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bcp-technology/lobster/internal/api"
	"github.com/bcp-technology/lobster/internal/api/adminsvc"
	"github.com/bcp-technology/lobster/internal/api/middleware"
	"github.com/bcp-technology/lobster/internal/config"
	"github.com/bcp-technology/lobster/internal/integrations"
	"github.com/bcp-technology/lobster/internal/integrations/keycloak"
	"github.com/bcp-technology/lobster/internal/orchestration"
	"github.com/bcp-technology/lobster/internal/runner"
	"github.com/bcp-technology/lobster/internal/steps"
	"github.com/bcp-technology/lobster/internal/steps/builtin"
	"github.com/bcp-technology/lobster/internal/store"

	adminv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/admin"
	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	configv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	serviceName = "lobsterd"
	version     = "dev"
)

// NewRootCommand builds the lobsterd command tree.
func NewRootCommand() *cobra.Command {
	v := viper.New()
	var cfgFile string

	root := &cobra.Command{
		Use:   "lobsterd",
		Short: "Lobster daemon process",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initViper(v, cfgFile)
		},
	}
	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (defaults to ./lobster.yaml)")
	root.AddCommand(newStartCommand(v))
	return root
}

func newStartCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start daemon services",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			storeCfg, migrationMode, err := buildStoreConfigFromInputs(cmd, v)
			if err != nil {
				return err
			}
			grpcListen := valueString(cmd, v, "daemon.listen", "listen")
			httpListen := valueString(cmd, v, "daemon.http_listen", "http-listen")
			if grpcListen == "" {
				return fmt.Errorf("listen address is required")
			}
			if httpListen == "" {
				return fmt.Errorf("http-listen address is required")
			}

			workspaceID := valueString(cmd, v, "workspace.selected", "workspace")

			// Auth config from flags / env.
			authCfg := middleware.AuthConfig{
				JWKSUrl:            valueString(cmd, v, "transport.auth.jwks_url", "jwks-url"),
				StaticToken:        valueString(cmd, v, "transport.auth.static_bearer_token", "static-token"),
				AllowInsecureLocal: valueBool(cmd, v, "transport.allow_insecure_local", "insecure-local"),
				ExplicitLocalMode:  valueBool(cmd, v, "transport.allow_insecure_local", "insecure-local"),
			}

			st, err := store.Open(ctx, storeCfg)
			if err != nil {
				return fmt.Errorf("initialize store: %w", err)
			}
			defer func() { _ = st.Close() }()

			// --- step registry ---
			reg := steps.NewRegistry()
			if regErr := builtin.Register(reg); regErr != nil {
				return fmt.Errorf("register builtin steps: %w", regErr)
			}

			// --- orchestration ConfigProvider ---
			orchCfgFn := orchestration.ConfigProvider(func(_ context.Context, _, profileName string) (*orchestration.Setup, error) {
				// Profile-specific compose overrides can be added here in future.
				_ = profileName
				return &orchestration.Setup{
					ComposeFiles: v.GetStringSlice("compose.files"),
					ProjectName:  v.GetString("compose.project_name"),
					WaitTimeout:  v.GetDuration("compose.wait.timeout"),
					Profiles:     v.GetStringSlice("compose.profiles"),
				}, nil
			})

			// --- Docker orchestrator ---
			orch, orchErr := orchestration.New(v.GetString("transport.docker_host"), orchCfgFn)
			if orchErr != nil {
				return fmt.Errorf("init docker orchestrator: %w", orchErr)
			}
			defer func() { _ = orch.Close() }()

			// --- runner ConfigProvider ---
			runCfgFn := runner.ConfigProvider(func(_ context.Context, _, profileName string) (*runner.RunConfig, error) {
				_ = profileName
				return &runner.RunConfig{
					BaseURL:        v.GetString("http.base_url"),
					DefaultHeaders: v.GetStringMapString("http.default_headers"),
					Variables:      v.GetStringMapString("variables.suite"),
					FeaturePaths:   v.GetStringSlice("features.paths"),
					StepTimeout:    v.GetDuration("execution.step_timeout"),
					RunTimeout:     v.GetDuration("execution.timeout"),
					SoftAssert:     v.GetBool("execution.soft_assert"),
					FailFast:       v.GetBool("execution.fail_fast"),
					KeepStack:      v.GetBool("execution.keep_stack"),
				}, nil
			})

			runnerImpl := runner.New(runCfgFn, orch, reg, st)
			daemonHooks := steps.NewHookRegistry()
			builtin.RegisterHooks(daemonHooks)
			runnerImpl = runnerImpl.WithHooks(daemonHooks)
			plannerImpl := runner.NewPlanner(runCfgFn, st)

			// --- integrations ---
			intReg := integrations.NewRegistry()
			if v.GetBool("integrations.keycloak.enabled") {
				adminPassword := os.Getenv(v.GetString("integrations.keycloak.admin_password_env"))
				kcAdapter := keycloak.New("keycloak-primary", keycloak.Config{
					BaseURL:       v.GetString("integrations.keycloak.base_url"),
					AdminUser:     v.GetString("integrations.keycloak.admin_user"),
					AdminPassword: adminPassword,
					Realm:         v.GetString("integrations.keycloak.realm"),
				})
				if regErr := intReg.Register(kcAdapter); regErr != nil {
					return fmt.Errorf("register keycloak adapter: %w", regErr)
				}
			}

			// Build the gRPC server and all service implementations.
			cfgSummaryFn := func() *adminv1.ConfigSummary {
				migMode := migrationModeProto(v.GetString("compose.migrations.mode"))
				return adminsvc.MakeConfigSummary(
					workspaceID,
					valueString(cmd, v, "profile", "profile"),
					&configv1.ExecutionConfig{
						SoftAssert:  v.GetBool("execution.soft_assert"),
						FailFast:    v.GetBool("execution.fail_fast"),
						KeepStack:   v.GetBool("execution.keep_stack"),
						RunTimeout:  durationpb.New(v.GetDuration("execution.timeout")),
						StepTimeout: durationpb.New(v.GetDuration("execution.step_timeout")),
					},
					&configv1.ComposeConfig{
						Files:         v.GetStringSlice("compose.files"),
						ProjectName:   v.GetString("compose.project_name"),
						Profiles:      v.GetStringSlice("compose.profiles"),
						MigrationMode: migMode,
						WaitTimeout:   durationpb.New(v.GetDuration("compose.wait.timeout")),
					},
					&configv1.PersistenceConfig{
						SqlitePath:  storeCfg.SQLitePath,
						JournalMode: storeCfg.JournalMode,
					},
				)
			}
			srv, err := api.Build(st, api.Config{
				Auth:              authCfg,
				Version:           version,
				WorkspaceID:       workspaceID,
				ActiveProfile:     valueString(cmd, v, "profile", "profile"),
				ConfigSummaryFunc: cfgSummaryFn,
			}, api.Services{
				Runner:       runnerImpl,
				Planner:      plannerImpl,
				Orchestrator: orch,
				Validator:    integrations.NewValidator(intReg),
			})
			if err != nil {
				return fmt.Errorf("build API server: %w", err)
			}

			// Start gRPC listener.
			grpcLis, err := net.Listen("tcp", grpcListen)
			if err != nil {
				return fmt.Errorf("listen gRPC: %w", err)
			}

			// Build the HTTP/JSON gateway mux and healthz handler.
			gatewayMux, err := api.GatewayMuxFor(ctx, grpcListen)
			if err != nil {
				return fmt.Errorf("build gateway: %w", err)
			}

			httpMux := http.NewServeMux()
			httpMux.Handle("/api/", gatewayMux)
			httpMux.HandleFunc("/healthz", healthzHandler(storeCfg.SQLitePath, migrationMode))

			errCh := make(chan error, 2)

			// gRPC server goroutine.
			go func() {
				errCh <- api.ServeGRPC(ctx, srv.GRPCServer, grpcLis)
			}()

			// HTTP gateway goroutine.
			go func() {
				errCh <- api.ServeHTTP(ctx, httpMux, httpListen)
			}()

			// Wait for first error or context cancellation.
			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				srv.GRPCServer.GracefulStop()
				return nil
			}
		},
	}

	addDaemonFlags(cmd.Flags())
	return cmd
}

func healthzHandler(sqlitePath, migrationMode string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":%q,"version":%q,"status":"ready","sqlite_path":%q,"migration_mode":%q,"checked_at_utc":%q}`+"\n",
			serviceName, version, sqlitePath, migrationMode,
			time.Now().UTC().Format(time.RFC3339))
	}
}

func addDaemonFlags(fs *pflag.FlagSet) {
	fs.String("listen", ":9443", "daemon gRPC listen address")
	fs.String("http-listen", ":8080", "daemon HTTP/JSON gateway listen address")
	fs.String("workspace", "", "workspace selection for monorepo isolation")
	fs.String("profile", "", "active config profile name")
	fs.String("sqlite-path", "", "explicit SQLite path override")
	fs.String("migrations-dir", "migrations", "directory containing SQL migrations")
	fs.String("migration-mode", "auto", "migration mode: auto|external|disabled")
	fs.String("journal-mode", "", "SQLite journal mode override")
	fs.String("synchronous", "", "SQLite synchronous pragma override")
	fs.Duration("busy-timeout", 0, "SQLite busy timeout (for example: 5s)")
	fs.String("jwks-url", "", "JWKS endpoint URL for token validation")
	fs.String("static-token", "", "static bearer token for development use")
	fs.Bool("insecure-local", false, "skip auth token checks (local development only)")
}

func buildStoreConfigFromInputs(cmd *cobra.Command, v *viper.Viper) (store.Config, string, error) {
	migrationMode := valueString(cmd, v, "compose.migrations.mode", "migration-mode")
	if _, err := store.ParseMigrationMode(migrationMode); err != nil {
		return store.Config{}, "", err
	}

	cfg, err := config.StoreConfigFromInput(config.StoreAdapterInput{
		Workspace:     valueString(cmd, v, "workspace.selected", "workspace"),
		MigrationsDir: valueString(cmd, v, "persistence.migrations.dir", "migrations-dir"),
		Profile: config.Profile{
			Compose: config.ComposeConfig{MigrationMode: migrationMode},
			Persistence: config.PersistenceConfig{SQLite: config.SQLiteConfig{
				Path:        valueString(cmd, v, "persistence.sqlite.path", "sqlite-path"),
				JournalMode: valueString(cmd, v, "persistence.sqlite.journal_mode", "journal-mode"),
				Synchronous: valueString(cmd, v, "persistence.sqlite.synchronous", "synchronous"),
				BusyTimeout: valueDuration(cmd, v, "persistence.sqlite.busy_timeout", "busy-timeout"),
			}},
		},
	})
	if err != nil {
		return store.Config{}, "", err
	}

	// Daemon always requires a store; fall back to a sensible local default
	// when neither an explicit sqlite-path nor a workspace is configured.
	if cfg.SQLitePath == "" {
		cfg.SQLitePath = store.DefaultSQLitePath()
	}

	return cfg, migrationMode, nil
}

func valueString(cmd *cobra.Command, v *viper.Viper, key, flagName string) string {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		return strings.TrimSpace(f.Value.String())
	}
	return strings.TrimSpace(v.GetString(key))
}

func valueBool(cmd *cobra.Command, v *viper.Viper, key, flagName string) bool {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		b, _ := cmd.Flags().GetBool(flagName)
		return b
	}
	return v.GetBool(key)
}

func valueDuration(cmd *cobra.Command, v *viper.Viper, key, flagName string) time.Duration {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		d, _ := cmd.Flags().GetDuration(flagName)
		return d
	}
	return v.GetDuration(key)
}

func initViper(v *viper.Viper, cfgFile string) error {
	v.SetEnvPrefix("LOBSTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	if strings.TrimSpace(cfgFile) != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("read config file: %w", err)
		}
		return nil
	}

	v.SetConfigName("lobster")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}
	return nil
}

// migrationModeProto converts the string config value to the proto enum.
func migrationModeProto(s string) commonv1.MigrationMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto":
		return commonv1.MigrationMode_MIGRATION_MODE_AUTO
	case "external":
		return commonv1.MigrationMode_MIGRATION_MODE_EXTERNAL
	case "disabled":
		return commonv1.MigrationMode_MIGRATION_MODE_DISABLED
	default:
		return commonv1.MigrationMode_MIGRATION_MODE_AUTO
	}
}
