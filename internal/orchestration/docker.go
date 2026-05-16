package orchestration

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
)

const (
	healthyStatus = "healthy"
	pollInterval  = 500 * time.Millisecond
)

// EnsureStack starts the compose stack and optionally waits for all services
// to report healthy before returning.
func (o *DockerOrchestrator) EnsureStack(ctx context.Context, req *stackv1.EnsureStackRequest) (*stackv1.EnsureStackResponse, error) {
	setup, err := o.configFn(ctx, req.WorkspaceId, req.ProfileName)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "resolve stack config: %v", err)
	}
	if setup.ProjectName == "" {
		setup.ProjectName = req.WorkspaceId
	}

	services, err := parseComposeFiles(setup.ComposeFiles)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse compose files: %v", err)
	}

	if err := o.ensureNetwork(ctx, setup.ProjectName); err != nil {
		return nil, status.Errorf(codes.Internal, "ensure network: %v", err)
	}

	for _, name := range topologicalOrder(services) {
		if err := o.ensureContainer(ctx, setup.ProjectName, req.WorkspaceId, name, services[name]); err != nil {
			return nil, status.Errorf(codes.Internal, "start service %q: %v", name, err)
		}
	}

	if req.WaitForReadiness {
		timeout := setup.WaitTimeout
		if req.WaitTimeout != nil {
			if d := req.WaitTimeout.AsDuration(); d > 0 {
				timeout = d
			}
		}
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		if err := o.waitAllHealthy(ctx, setup.ProjectName, timeout); err != nil {
			return nil, status.Errorf(codes.DeadlineExceeded, "wait healthy: %v", err)
		}
	}

	stack, err := o.buildStackProto(ctx, setup.ProjectName, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build stack: %v", err)
	}
	return &stackv1.EnsureStackResponse{Stack: stack}, nil
}

// TeardownStack stops and removes all containers belonging to the project,
// then removes the project network.
func (o *DockerOrchestrator) TeardownStack(ctx context.Context, req *stackv1.TeardownStackRequest) (*stackv1.TeardownStackResponse, error) {
	setup, err := o.configFn(ctx, req.WorkspaceId, "")
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "resolve stack config: %v", err)
	}
	if setup.ProjectName == "" {
		setup.ProjectName = req.WorkspaceId
	}

	containers, err := o.listProjectContainers(ctx, setup.ProjectName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list containers: %v", err)
	}

	for _, c := range containers {
		_ = o.cli.ContainerStop(ctx, c.ID, container.StopOptions{})
		_ = o.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{
			RemoveVolumes: req.RemoveVolumes,
			Force:         true,
		})
	}

	// Network removal is best-effort; it may be shared or already removed.
	_ = o.removeNetwork(ctx, setup.ProjectName+"_default")

	return &stackv1.TeardownStackResponse{
		WorkspaceId:    req.WorkspaceId,
		TerminalStatus: stackv1.StackStatus_STACK_STATUS_TEARDOWN,
	}, nil
}

// GetStackLogs multiplexes container logs from all project services (or a
// single service if service_name is set) onto the gRPC server stream.
func (o *DockerOrchestrator) GetStackLogs(req *stackv1.GetStackLogsRequest, stream stackv1.StackService_GetStackLogsServer) error {
	ctx := stream.Context()

	setup, err := o.configFn(ctx, req.WorkspaceId, "")
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "resolve stack config: %v", err)
	}
	if setup.ProjectName == "" {
		setup.ProjectName = req.WorkspaceId
	}

	containers, err := o.listProjectContainers(ctx, setup.ProjectName)
	if err != nil {
		return status.Errorf(codes.Internal, "list containers: %v", err)
	}

	if req.ServiceName != "" {
		filtered := containers[:0]
		for _, c := range containers {
			if c.Labels[labelService] == req.ServiceName {
				filtered = append(filtered, c)
			}
		}
		containers = filtered
	}

	if len(containers) == 0 {
		return nil
	}

	var mu sync.Mutex // guards stream.Send
	var wg sync.WaitGroup
	var firstErr error

	for _, c := range containers {
		c := c
		svcName := c.Labels[labelService]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := o.streamContainerLogs(ctx, c.ID, svcName, req, stream, &mu); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
					return
				}
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// --- internal helpers ---

func (o *DockerOrchestrator) ensureNetwork(ctx context.Context, projectName string) error {
	networkName := projectName + "_default"
	f := filters.NewArgs(filters.Arg("name", networkName))
	list, err := o.cli.NetworkList(ctx, network.ListOptions{Filters: f})
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}
	for _, n := range list {
		if n.Name == networkName {
			return nil // already exists
		}
	}
	_, err = o.cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			labelProject:   projectName,
			labelManagedBy: managedByValue,
		},
	})
	return err
}

func (o *DockerOrchestrator) removeNetwork(ctx context.Context, networkName string) error {
	f := filters.NewArgs(filters.Arg("name", networkName))
	list, err := o.cli.NetworkList(ctx, network.ListOptions{Filters: f})
	if err != nil {
		return err
	}
	for _, n := range list {
		if n.Name == networkName {
			return o.cli.NetworkRemove(ctx, n.ID)
		}
	}
	return nil
}

func (o *DockerOrchestrator) ensureContainer(ctx context.Context, projectName, workspaceID, serviceName string, svc *resolvedService) error {
	existing, err := o.findContainer(ctx, projectName, serviceName)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.State == "running" {
			return nil
		}
		return o.cli.ContainerStart(ctx, existing.ID, container.StartOptions{})
	}

	containerName := fmt.Sprintf("%s-%s-1", projectName, serviceName)
	networkName := projectName + "_default"

	// Pull the image if it is not already present locally.
	if err := o.ensureImage(ctx, svc.Image); err != nil {
		return fmt.Errorf("pull image %q: %w", svc.Image, err)
	}

	labels := map[string]string{
		labelProject:   projectName,
		labelService:   serviceName,
		labelManagedBy: managedByValue,
		labelWorkspace: workspaceID,
	}
	for k, v := range svc.Labels {
		labels[k] = v
	}

	cfg := &container.Config{
		Image:        svc.Image,
		Env:          svc.Env,
		Labels:       labels,
		ExposedPorts: svc.ExposedPorts,
		Healthcheck:  svc.Healthcheck,
	}
	if len(svc.Cmd) > 0 {
		cfg.Cmd = svc.Cmd
	}
	if len(svc.Entrypoint) > 0 {
		cfg.Entrypoint = svc.Entrypoint
	}
	if svc.User != "" {
		cfg.User = svc.User
	}
	if svc.WorkingDir != "" {
		cfg.WorkingDir = svc.WorkingDir
	}

	hostCfg := &container.HostConfig{
		NetworkMode:   container.NetworkMode(networkName),
		PortBindings:  svc.PortBindings,
		ExtraHosts:    svc.ExtraHosts,
		Binds:         svc.Binds,
		RestartPolicy: resolveRestartPolicy(svc.Restart),
	}

	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {Aliases: []string{serviceName}},
		},
	}

	resp, err := o.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, containerName)
	if err != nil {
		return fmt.Errorf("create container %q: %w", containerName, err)
	}
	return o.cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
}

// resolveRestartPolicy maps a Compose restart string to a Docker RestartPolicy.
func resolveRestartPolicy(policy string) container.RestartPolicy {
	switch policy {
	case "always":
		return container.RestartPolicy{Name: container.RestartPolicyAlways}
	case "unless-stopped":
		return container.RestartPolicy{Name: container.RestartPolicyUnlessStopped}
	case "on-failure":
		return container.RestartPolicy{Name: container.RestartPolicyOnFailure}
	default:
		return container.RestartPolicy{Name: container.RestartPolicyDisabled}
	}
}

// ensureImage pulls the image if it is not already present in the local daemon.
func (o *DockerOrchestrator) ensureImage(ctx context.Context, image string) error {
	if image == "" {
		return nil
	}
	_, err := o.cli.ImageInspect(ctx, image)
	if err == nil {
		return nil // already present
	}
	rc, pullErr := o.cli.ImagePull(ctx, image, dockerimage.PullOptions{})
	if pullErr != nil {
		return pullErr
	}
	_, _ = io.Copy(io.Discard, rc)
	return rc.Close()
}

func (o *DockerOrchestrator) findContainer(ctx context.Context, projectName, serviceName string) (*container.Summary, error) {
	f := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", labelProject, projectName)),
		filters.Arg("label", fmt.Sprintf("%s=%s", labelService, serviceName)),
	)
	list, err := o.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

func (o *DockerOrchestrator) listProjectContainers(ctx context.Context, projectName string) ([]container.Summary, error) {
	f := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", labelProject, projectName)),
	)
	return o.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
}

func (o *DockerOrchestrator) waitAllHealthy(ctx context.Context, projectName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		containers, err := o.listProjectContainers(ctx, projectName)
		if err != nil {
			return err
		}
		allReady := true
		for _, c := range containers {
			info, err := o.cli.ContainerInspect(ctx, c.ID)
			if err != nil {
				return err
			}
			if info.State == nil {
				allReady = false
				break
			}
			if info.State.Health == nil {
				// Container without healthcheck: healthy if running.
				if info.State.Status != "running" {
					allReady = false
					break
				}
				continue
			}
			if info.State.Health.Status != healthyStatus {
				allReady = false
				break
			}
		}
		if allReady {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("stack not healthy after %v", timeout)
}

func (o *DockerOrchestrator) buildStackProto(ctx context.Context, projectName, workspaceID string) (*stackv1.Stack, error) {
	containers, err := o.listProjectContainers(ctx, projectName)
	if err != nil {
		return nil, err
	}
	stack := &stackv1.Stack{
		StackId:     projectName,
		WorkspaceId: workspaceID,
		ProjectName: projectName,
		Status:      stackv1.StackStatus_STACK_STATUS_HEALTHY,
	}
	allHealthy := true
	for _, c := range containers {
		info, err := o.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}
		comp := &stackv1.StackComponent{
			Name:        c.Labels[labelService],
			Image:       c.Image,
			ContainerId: c.ID[:12],
			Status:      info.State.Status,
		}
		health := stackv1.ServiceHealth_SERVICE_HEALTH_STARTING
		if info.State.Health != nil {
			switch info.State.Health.Status {
			case "healthy":
				health = stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY
			case "unhealthy":
				health = stackv1.ServiceHealth_SERVICE_HEALTH_UNHEALTHY
				allHealthy = false
			default:
				health = stackv1.ServiceHealth_SERVICE_HEALTH_STARTING
				allHealthy = false
			}
		} else if info.State.Status == "running" {
			health = stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY
		} else {
			allHealthy = false
		}
		comp.Health = health
		stack.Services = append(stack.Services, comp)
	}
	if !allHealthy {
		stack.Status = stackv1.StackStatus_STACK_STATUS_DEGRADED
	}
	return stack, nil
}

func (o *DockerOrchestrator) streamContainerLogs(
	ctx context.Context,
	containerID, serviceName string,
	req *stackv1.GetStackLogsRequest,
	stream stackv1.StackService_GetStackLogsServer,
	mu *sync.Mutex,
) error {
	tail := "all"
	if req.TailLines > 0 {
		tail = fmt.Sprintf("%d", req.TailLines)
	}
	rc, err := o.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     req.Follow,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		return fmt.Errorf("container logs: %w", err)
	}
	defer rc.Close()

	pr, pw := io.Pipe()
	go func() {
		_, copyErr := stdcopy.StdCopy(pw, pw, rc)
		pw.CloseWithError(copyErr)
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		line := scanner.Text()
		observedAt := time.Now()
		text := line
		// Docker timestamp prefix: "2024-01-01T00:00:00.000000000Z text"
		if len(line) > 31 && line[30] == ' ' {
			if t, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(line[:30])); parseErr == nil {
				observedAt = t
				text = line[31:]
			}
		}
		mu.Lock()
		sendErr := stream.Send(&stackv1.LogLine{
			ServiceName: serviceName,
			Line:        text,
			ObservedAt:  timestamppb.New(observedAt),
		})
		mu.Unlock()
		if sendErr != nil {
			return sendErr
		}
	}
	return scanner.Err()
}
