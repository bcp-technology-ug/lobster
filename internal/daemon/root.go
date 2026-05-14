package daemon

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/api"
	"github.com/bcp-technology-ug/lobster/internal/api/adminsvc"
	"github.com/bcp-technology-ug/lobster/internal/api/middleware"
	"github.com/bcp-technology-ug/lobster/internal/config"
	"github.com/bcp-technology-ug/lobster/internal/integrations"
	"github.com/bcp-technology-ug/lobster/internal/integrations/keycloak"
	lobsterlog "github.com/bcp-technology-ug/lobster/internal/log"
	"github.com/bcp-technology-ug/lobster/internal/orchestration"
	"github.com/bcp-technology-ug/lobster/internal/runner"
	"github.com/bcp-technology-ug/lobster/internal/steps"
	"github.com/bcp-technology-ug/lobster/internal/steps/builtin"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/ui"

	adminv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/admin"
	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	configv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/config"
	integrationsv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/integrations"
	integrationstore "github.com/bcp-technology-ug/lobster/gen/sqlc/integrations"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		Use:     "lobsterd",
		Short:   "Lobster daemon process",
		Version: version,
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

			// Print startup banner.
			fmt.Fprintln(cmd.OutOrStdout(), ui.LogoBig())
			fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("lobsterd  "+version))
			fmt.Fprintln(cmd.OutOrStdout())

			// Retrieve the logger injected by main and attach it to the start context.
			logger := lobsterlog.FromContext(cmd.Context())
			ctx = lobsterlog.WithLogger(ctx, logger)

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
			// --auth-mode flag: "token" (default) or "none" (insecure local)
			authMode := valueString(cmd, v, "transport.auth.mode", "auth-mode")
			insecureLocal := valueBool(cmd, v, "transport.allow_insecure_local", "insecure-local") || authMode == "none"
			authCfg := middleware.AuthConfig{
				JWKSUrl:            valueString(cmd, v, "transport.auth.jwks_url", "jwks-url"),
				StaticToken:        valueString(cmd, v, "transport.auth.static_bearer_token", "static-token"),
				AllowInsecureLocal: insecureLocal,
				ExplicitLocalMode:  insecureLocal,
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
					BaseURL:            v.GetString("http.base_url"),
					DefaultHeaders:     v.GetStringMapString("http.default_headers"),
					Variables:          v.GetStringMapString("variables.suite"),
					FeaturePaths:       v.GetStringSlice("features.paths"),
					StepTimeout:        v.GetDuration("execution.step_timeout"),
					RunTimeout:         v.GetDuration("execution.timeout"),
					SoftAssert:         v.GetBool("execution.soft_assert"),
					FailFast:           v.GetBool("execution.fail_fast"),
					KeepStack:          v.GetBool("execution.keep_stack"),
					QuarantineEnabled:  v.GetBool("quarantine.enabled"),
					QuarantineTag:      v.GetString("quarantine.tag"),
					QuarantineBlocking: v.GetBool("quarantine.blocking_in_main_ci"),
				}, nil
			})

			runnerImpl := runner.New(runCfgFn, orch, reg, st)
			daemonHooks := steps.NewHookRegistry()
			builtin.RegisterHooks(daemonHooks)
			runnerImpl = runnerImpl.WithHooks(daemonHooks)
			// Apply concurrency cap: prefer daemon.max_concurrent_runs, then
			// execution.max_concurrent_runs, default 0 (unlimited).
			maxConcurrent := v.GetInt("daemon.max_concurrent_runs")
			if maxConcurrent == 0 {
				maxConcurrent = v.GetInt("execution.max_concurrent_runs")
			}
			if maxConcurrent > 0 {
				runnerImpl = runnerImpl.WithMaxConcurrentRuns(maxConcurrent)
			}
			// Apply retention config.
			daemonRetention := store.RetentionConfig{
				MaxRuns: int64(v.GetInt("persistence.retention.max_runs")),
				MaxAge:  v.GetDuration("persistence.retention.max_age"),
			}
			if daemonRetention.MaxRuns > 0 || daemonRetention.MaxAge > 0 {
				runnerImpl = runnerImpl.WithRetention(daemonRetention)
			}
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
				if regErr := kcAdapter.RegisterSteps(reg); regErr != nil {
					return fmt.Errorf("register keycloak steps: %w", regErr)
				}
				// Persist adapter record so the integrations API can list/get it.
				now := time.Now().UTC().Format(time.RFC3339Nano)
				if upsertErr := st.Integrations.UpsertIntegrationAdapter(ctx, integrationstore.UpsertIntegrationAdapterParams{
					AdapterID: kcAdapter.ID(),
					Name:      kcAdapter.ID(),
					Type:      kcAdapter.Kind(),
					State:     int64(integrationsv1.AdapterState_ADAPTER_STATE_READY),
					UpdatedAt: now,
				}); upsertErr != nil {
					return fmt.Errorf("persist keycloak adapter: %w", upsertErr)
				}
				for _, cap := range []string{"auth", "user_management"} {
					if upsertErr := st.Integrations.UpsertIntegrationAdapterCapability(ctx, integrationstore.UpsertIntegrationAdapterCapabilityParams{
						AdapterID: kcAdapter.ID(),
						Name:      cap,
						Enabled:   1,
					}); upsertErr != nil {
						return fmt.Errorf("persist keycloak capability %q: %w", cap, upsertErr)
					}
				}
			}
			runnerImpl = runnerImpl.WithAdapterRegistry(intReg)

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
				Logger:            logger,
				Version:           version,
				WorkspaceID:       workspaceID,
				ActiveProfile:     valueString(cmd, v, "profile", "profile"),
				ConfigSummaryFunc: cfgSummaryFn,
			}, api.Services{
				Runner:       runnerImpl,
				Planner:      plannerImpl,
				Orchestrator: orch,
				Validator:    integrations.NewValidator(intReg),
				Notifier:     intReg,
			})
			if err != nil {
				return fmt.Errorf("build API server: %w", err)
			}

			// Start gRPC listener — optionally with TLS.
			grpcLis, err := buildGRPCListener(cmd, v, grpcListen)
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

			errCh := make(chan error, 3)

			// gRPC server goroutine.
			go func() {
				errCh <- api.ServeGRPC(ctx, srv.GRPCServer, grpcLis)
			}()

			// HTTP gateway goroutine.
			go func() {
				errCh <- api.ServeHTTP(ctx, httpMux, httpListen)
			}()

			// Optional Wish SSH server goroutine.
			sshAddr := valueString(cmd, v, "daemon.ssh_listen", "ssh-listen")
			if sshAddr != "" {
				hostKeyPath := valueString(cmd, v, "daemon.ssh_host_key", "ssh-host-key")
				if hostKeyPath == "" {
					hostKeyPath = ".lobster/ssh_host_key"
				}
				sshConn, sshConnErr := grpcDialInsecure(ctx, grpcListen)
				if sshConnErr != nil {
					logger.Warn("ssh server: failed to create gRPC client — SSH server disabled",
						zap.Error(sshConnErr))
				} else {
					sshCfg := SSHServerConfig{
						Addr:               sshAddr,
						HostKeyPath:        hostKeyPath,
						AuthorizedKeysPath: valueString(cmd, v, "daemon.ssh_authorized_keys", "ssh-authorized-keys"),
						StaticToken:        valueString(cmd, v, "transport.auth.static_bearer_token", "static-token"),
						WorkspaceID:        workspaceID,
					}
					go func() {
						errCh <- StartSSHServer(ctx, sshCfg, sshConn)
					}()
				}
			}

			// Wait for first error or context cancellation.
			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				// Signal NOT_SERVING before stopping so in-flight health checks
				// and load balancers see the server going away gracefully.
				srv.HealthSrv.Shutdown()
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
	fs.String("db-path", "", "SQLite database path (alias for --sqlite-path)")
	fs.String("migrations-dir", "migrations", "directory containing SQL migrations")
	fs.String("migration-mode", "auto", "migration mode: auto|external|disabled")
	fs.String("journal-mode", "", "SQLite journal mode override")
	fs.String("synchronous", "", "SQLite synchronous pragma override")
	fs.Duration("busy-timeout", 0, "SQLite busy timeout (for example: 5s)")
	fs.String("jwks-url", "", "JWKS endpoint URL for token validation")
	fs.String("static-token", "", "static bearer token for development use")
	fs.Bool("insecure-local", false, "skip auth token checks (local development only)")
	fs.String("auth-mode", "token", "auth mode: token|none (none disables token checks for local development)")
	fs.String("tls-cert-file", "", "TLS server certificate file")
	fs.String("tls-key-file", "", "TLS server private key file")
	fs.String("tls-client-ca-file", "", "trusted CA bundle for mTLS client certificate verification")
	fs.String("ssh-listen", "", "Wish SSH server listen address (e.g. :2222), empty to disable")
	fs.String("ssh-host-key", ".lobster/ssh_host_key", "path to ED25519 SSH host key (auto-generated if missing)")
	fs.String("ssh-authorized-keys", "", "path to SSH authorized_keys file for public key auth")
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
				Path:        valueStringFallback(cmd, v, "persistence.sqlite.path", "sqlite-path", "db-path"),
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

// valueStringFallback is like valueString but checks an additional flag alias
// (aliasFlag) when the primary flag is not set.
func valueStringFallback(cmd *cobra.Command, v *viper.Viper, key, flagName, aliasFlag string) string {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		return strings.TrimSpace(f.Value.String())
	}
	if a := cmd.Flags().Lookup(aliasFlag); a != nil && a.Changed {
		return strings.TrimSpace(a.Value.String())
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

// buildGRPCListener creates a TCP listener, optionally wrapping it with TLS.
// When --tls-cert-file and --tls-key-file are both provided, TLS is enabled.
// When --tls-client-ca-file is also provided, mutual TLS (mTLS) is enforced.
// If no cert file is provided the listener is plaintext (dev mode).
func buildGRPCListener(cmd *cobra.Command, v *viper.Viper, addr string) (net.Listener, error) {
	certFile := valueString(cmd, v, "transport.tls.cert_file", "tls-cert-file")
	keyFile := valueString(cmd, v, "transport.tls.key_file", "tls-key-file")
	clientCAFile := valueString(cmd, v, "transport.tls.client_ca_file", "tls-client-ca-file")

	if certFile == "" || keyFile == "" {
		// Plaintext — safe for localhost / in-cluster loopback.
		return net.Listen("tcp", addr)
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS certificate: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if clientCAFile != "" {
		pool, err := loadClientCAPool(clientCAFile)
		if err != nil {
			return nil, fmt.Errorf("load client CA: %w", err)
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(lis, tlsCfg), nil
}

// loadClientCAPool parses a PEM-encoded CA bundle from the given file path.
func loadClientCAPool(caFile string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(caFile) // #nosec G304 — operator-controlled config path
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no valid certificates found in %q", caFile)
	}
	return pool, nil
}

// grpcDialInsecure opens a plaintext gRPC client connection to addr.
// Used by the SSH server to call back into the local daemon without TLS so that
// it works regardless of whether the gRPC listener has TLS enabled — the SSH
// tunnel provides the transport-layer security instead.
func grpcDialInsecure(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	// Use passthrough so the address is resolved verbatim (avoids DNS issues
	// with localhost on macOS/Docker Desktop).
	if !strings.Contains(addr, "://") {
		addr = "passthrough:///" + addr
	}
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	// Probe connectivity so we fail fast rather than deferring to first RPC.
	conn.Connect()
	_ = dialCtx // timeout context used for the Connect probe only
	return conn, nil
}
