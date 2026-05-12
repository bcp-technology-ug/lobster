package orchestration

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/go-connections/nat"
	"go.yaml.in/yaml/v3"
)

// composeSpec is the minimal Compose file subset needed for container lifecycle
// management in v0.1. Only fields used by the orchestrator are unmarshalled.
type composeSpec struct {
	Services map[string]*composeSvcDef `yaml:"services"`
}

type composeSvcDef struct {
	Image       string                 `yaml:"image"`
	Command     interface{}            `yaml:"command"`     // string | []string
	Entrypoint  interface{}            `yaml:"entrypoint"`  // string | []string
	Environment interface{}            `yaml:"environment"` // map | []string
	Ports       []interface{}          `yaml:"ports"`       // string | map
	DependsOn   interface{}            `yaml:"depends_on"`  // []string | map
	Healthcheck *composeSvcHealthcheck `yaml:"healthcheck"`
	Labels      interface{}            `yaml:"labels"` // map | []string
	ExtraHosts  []string               `yaml:"extra_hosts"`
	Restart     string                 `yaml:"restart"`
	User        string                 `yaml:"user"`
	WorkingDir  string                 `yaml:"working_dir"`
	Networks    interface{}            `yaml:"networks"` // ignored in v0.1 (single default)
	Volumes     []interface{}          `yaml:"volumes"`  // ignored in v0.1
	EnvFile     interface{}            `yaml:"env_file"` // ignored in v0.1
}

type composeSvcHealthcheck struct {
	Test        interface{} `yaml:"test"` // []string | string
	Interval    string      `yaml:"interval"`
	Timeout     string      `yaml:"timeout"`
	Retries     int         `yaml:"retries"`
	Disable     bool        `yaml:"disable"`
	StartPeriod string      `yaml:"start_period"`
}

// resolvedService is the Docker API-ready service definition produced by
// merging one or more Compose file definitions.
type resolvedService struct {
	Image        string
	Env          []string // "KEY=value"
	Cmd          []string
	Entrypoint   []string
	PortBindings nat.PortMap
	ExposedPorts nat.PortSet
	DependsOn    []string
	Labels       map[string]string
	ExtraHosts   []string
	User         string
	WorkingDir   string
}

// parseComposeFiles parses one or more Compose files with last-wins merge
// semantics and returns the resolved service definitions.
func parseComposeFiles(paths []string) (map[string]*resolvedService, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no compose files specified")
	}
	merged := map[string]*composeSvcDef{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read compose file %q: %w", p, err)
		}
		var spec composeSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parse compose file %q: %w", p, err)
		}
		for name, svc := range spec.Services {
			if existing, ok := merged[name]; ok {
				mergeServiceDef(existing, svc)
			} else {
				merged[name] = svc
			}
		}
	}

	result := make(map[string]*resolvedService, len(merged))
	for name, def := range merged {
		rs, err := resolveServiceDef(def)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		result[name] = rs
	}
	return result, nil
}

// mergeServiceDef applies non-zero fields from src onto dst.
func mergeServiceDef(dst, src *composeSvcDef) {
	if src.Image != "" {
		dst.Image = src.Image
	}
	if src.Command != nil {
		dst.Command = src.Command
	}
	if src.Entrypoint != nil {
		dst.Entrypoint = src.Entrypoint
	}
	if src.Environment != nil {
		dst.Environment = src.Environment
	}
	if src.Restart != "" {
		dst.Restart = src.Restart
	}
	if len(src.Ports) > 0 {
		dst.Ports = append(dst.Ports, src.Ports...)
	}
	if src.DependsOn != nil {
		dst.DependsOn = src.DependsOn
	}
	if len(src.ExtraHosts) > 0 {
		dst.ExtraHosts = src.ExtraHosts
	}
	if src.Labels != nil {
		dst.Labels = src.Labels
	}
	if src.Healthcheck != nil {
		dst.Healthcheck = src.Healthcheck
	}
	if src.User != "" {
		dst.User = src.User
	}
	if src.WorkingDir != "" {
		dst.WorkingDir = src.WorkingDir
	}
}

func resolveServiceDef(def *composeSvcDef) (*resolvedService, error) {
	rs := &resolvedService{
		Image:      def.Image,
		User:       def.User,
		WorkingDir: def.WorkingDir,
		ExtraHosts: def.ExtraHosts,
	}
	rs.Env = resolveEnv(def.Environment)
	rs.Cmd = resolveStringOrSlice(def.Command)
	rs.Entrypoint = resolveStringOrSlice(def.Entrypoint)
	rs.DependsOn = resolveDependsOn(def.DependsOn)
	rs.Labels = resolveLabels(def.Labels)

	var err error
	rs.PortBindings, rs.ExposedPorts, err = resolvePorts(def.Ports)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func resolveEnv(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case map[string]interface{}:
		env := make([]string, 0, len(t))
		for k, val := range t {
			if val == nil {
				env = append(env, k)
			} else {
				env = append(env, fmt.Sprintf("%s=%v", k, val))
			}
		}
		return env
	case []interface{}:
		env := make([]string, 0, len(t))
		for _, item := range t {
			env = append(env, fmt.Sprintf("%v", item))
		}
		return env
	}
	return nil
}

func resolveStringOrSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		return strings.Fields(t)
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, item := range t {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	}
	return nil
}

func resolveDependsOn(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []interface{}:
		deps := make([]string, 0, len(t))
		for _, item := range t {
			deps = append(deps, fmt.Sprintf("%v", item))
		}
		return deps
	case map[string]interface{}:
		deps := make([]string, 0, len(t))
		for name := range t {
			deps = append(deps, name)
		}
		return deps
	}
	return nil
}

func resolveLabels(v interface{}) map[string]string {
	if v == nil {
		return nil
	}
	result := map[string]string{}
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			result[k] = fmt.Sprintf("%v", val)
		}
	case []interface{}:
		for _, item := range t {
			s := fmt.Sprintf("%v", item)
			if idx := strings.Index(s, "="); idx > 0 {
				result[s[:idx]] = s[idx+1:]
			} else {
				result[s] = ""
			}
		}
	}
	return result
}

func resolvePorts(ports []interface{}) (nat.PortMap, nat.PortSet, error) {
	portMap := nat.PortMap{}
	portSet := nat.PortSet{}
	for _, p := range ports {
		switch t := p.(type) {
		case string:
			exposed, bindings, err := nat.ParsePortSpecs([]string{t})
			if err != nil {
				return nil, nil, fmt.Errorf("port %q: %w", t, err)
			}
			for ep := range exposed {
				portSet[ep] = struct{}{}
			}
			for port, binding := range bindings {
				portMap[port] = append(portMap[port], binding...)
			}
		case map[string]interface{}:
			// {target: 80, published: "8080", protocol: "tcp"}
			target := fmt.Sprintf("%v", t["target"])
			proto := "tcp"
			if pr, ok := t["protocol"]; ok {
				proto = fmt.Sprintf("%v", pr)
			}
			containerPort := nat.Port(target + "/" + proto)
			published := ""
			if pub, ok := t["published"]; ok {
				published = fmt.Sprintf("%v", pub)
			}
			portSet[containerPort] = struct{}{}
			portMap[containerPort] = []nat.PortBinding{{HostPort: published}}
		default:
			s := fmt.Sprintf("%v", p)
			exposed, bindings, err := nat.ParsePortSpecs([]string{s})
			if err != nil {
				return nil, nil, fmt.Errorf("port %q: %w", s, err)
			}
			for ep := range exposed {
				portSet[ep] = struct{}{}
			}
			for port, binding := range bindings {
				portMap[port] = append(portMap[port], binding...)
			}
		}
	}
	return portMap, portSet, nil
}

// topologicalOrder returns service names in dependency order (dependencies first).
// Cycles are handled gracefully by the visited check.
func topologicalOrder(services map[string]*resolvedService) []string {
	visited := map[string]bool{}
	var result []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		if svc, ok := services[name]; ok {
			for _, dep := range svc.DependsOn {
				visit(dep)
			}
		}
		result = append(result, name)
	}

	for name := range services {
		visit(name)
	}
	return result
}
