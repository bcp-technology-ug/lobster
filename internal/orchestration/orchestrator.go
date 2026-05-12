// Package orchestration provides Docker SDK-backed stack lifecycle management.
// It implements the Orchestrator contract expected by internal/api/stacksvc.
package orchestration

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

// Docker Compose-compatible label constants. Using standard Compose labels means
// containers are visible to docker-compose tooling for debugging.
const (
	labelProject   = "com.docker.compose.project"
	labelService   = "com.docker.compose.service"
	labelManagedBy = "com.lobster.managed-by"
	labelWorkspace = "com.lobster.workspace"
	managedByValue = "lobster"
)

// Setup holds the resolved compose configuration for a stack.
type Setup struct {
	// ComposeFiles is the ordered list of Compose YAML file paths.
	// Later files merge over earlier files (last-wins on simple fields).
	ComposeFiles []string

	// ProjectName is the Compose project name used to namespace containers
	// and networks. Falls back to the workspace ID when empty.
	ProjectName string

	// WaitTimeout is the upper bound for readiness polling when
	// EnsureStack is called with wait_for_readiness=true.
	WaitTimeout time.Duration

	// Profiles is the list of Compose profiles to activate.
	Profiles []string
}

// ConfigProvider resolves compose setup for a workspace and optional profile.
// workspaceID and profileName come directly from the RPC request.
type ConfigProvider func(ctx context.Context, workspaceID, profileName string) (*Setup, error)

// DockerOrchestrator implements the stacksvc.Orchestrator interface using the
// Docker Engine API. Container lifecycle uses Compose-compatible label semantics
// so containers appear correctly in docker-compose tooling.
type DockerOrchestrator struct {
	cli      *client.Client
	configFn ConfigProvider
}

// New creates a DockerOrchestrator. Set dockerHost to "" to inherit DOCKER_HOST
// from the environment. cfgFn resolves compose setup for each workspace+profile.
func New(dockerHost string, cfgFn ConfigProvider) (*DockerOrchestrator, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if dockerHost != "" {
		opts = append(opts, client.WithHost(dockerHost))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &DockerOrchestrator{cli: cli, configFn: cfgFn}, nil
}

// Close releases the underlying Docker client connection.
func (o *DockerOrchestrator) Close() error {
	return o.cli.Close()
}
