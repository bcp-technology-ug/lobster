// Package adminsvc implements the gRPC AdminService server.
package adminsvc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/admin"
	configv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/config"
	"github.com/bcp-technology-ug/lobster/internal/store"
)

const (
	apiPackage = "lobster.v1"
	apiVersion = "v0.1"
)

// Capability names reported by GetCapabilities.
var capabilities = []struct {
	name    string
	enabled bool
}{
	{"run.sync", true},
	{"run.async", true},
	{"run.stream_events", true},
	{"run.cancel", true},
	{"plan", true},
	{"stack.ensure", true},
	{"stack.logs", true},
	{"integrations", true},
	{"admin.health", true},
	{"admin.capabilities", true},
	{"admin.config_summary", true},
}

// Service implements adminv1.AdminServiceServer.
type Service struct {
	adminv1.UnimplementedAdminServiceServer

	store      *store.Store
	version    string
	configFunc func() *adminv1.ConfigSummary
}

// New creates a Service.
// version is the daemon binary version string (e.g. "v0.1.0").
// configFunc returns the sanitised effective config on demand; may return nil.
func New(st *store.Store, version string, configFunc func() *adminv1.ConfigSummary) *Service {
	if configFunc == nil {
		configFunc = func() *adminv1.ConfigSummary { return nil }
	}
	return &Service{
		store:      st,
		version:    version,
		configFunc: configFunc,
	}
}

// GetHealth returns liveness and readiness status.
func (s *Service) GetHealth(ctx context.Context, _ *adminv1.GetHealthRequest) (*adminv1.GetHealthResponse, error) {
	live := true
	ready := s.store != nil && s.store.DB() != nil

	if ready {
		if err := s.store.DB().PingContext(ctx); err != nil {
			ready = false
		}
	}

	return &adminv1.GetHealthResponse{
		Health: &adminv1.HealthStatus{
			Live:       live,
			Ready:      ready,
			Version:    s.version,
			ObservedAt: timestamppb.Now(),
		},
	}, nil
}

// GetCapabilities returns the set of server capabilities.
func (s *Service) GetCapabilities(_ context.Context, _ *adminv1.GetCapabilitiesRequest) (*adminv1.GetCapabilitiesResponse, error) {
	caps := make([]*adminv1.Capability, 0, len(capabilities))
	for _, c := range capabilities {
		caps = append(caps, &adminv1.Capability{
			Name:    c.name,
			Enabled: c.enabled,
		})
	}
	return &adminv1.GetCapabilitiesResponse{
		ApiPackage:   apiPackage,
		ApiVersion:   apiVersion,
		Capabilities: caps,
	}, nil
}

// GetConfigSummary returns the sanitised effective configuration.
func (s *Service) GetConfigSummary(_ context.Context, _ *adminv1.GetConfigSummaryRequest) (*adminv1.GetConfigSummaryResponse, error) {
	cfg := s.configFunc()
	if cfg == nil {
		return nil, status.Error(codes.Unavailable, "config summary not available")
	}
	return &adminv1.GetConfigSummaryResponse{Config: cfg}, nil
}

// MakeConfigSummary builds a ConfigSummary from component values.
// Transport credentials are intentionally omitted.
func MakeConfigSummary(workspaceID, activeProfile string, exec *configv1.ExecutionConfig, compose *configv1.ComposeConfig, persistence *configv1.PersistenceConfig) *adminv1.ConfigSummary {
	return &adminv1.ConfigSummary{
		WorkspaceId:   workspaceID,
		ActiveProfile: activeProfile,
		Execution:     exec,
		Compose:       compose,
		Persistence:   persistence,
	}
}
