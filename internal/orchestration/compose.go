package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"go.yaml.in/yaml/v3"
)

// expandComposeVars expands ${VAR} and ${VAR:-default} references in compose
// file content using os.Getenv, with bash-style default values.
func expandComposeVars(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":-"); idx >= 0 {
			name, def := key[:idx], key[idx+2:]
			if v := os.Getenv(name); v != "" {
				return v
			}
			return def
		}
		return os.Getenv(key)
	})
}

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
	Volumes     []interface{}          `yaml:"volumes"`
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
	Binds        []string                // Docker bind-mount specs with resolved absolute paths
	Restart      string                  // Docker restart policy name (e.g. "unless-stopped")
	Healthcheck  *container.HealthConfig // nil = use image default
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
		if err := yaml.Unmarshal([]byte(expandComposeVars(string(data))), &spec); err != nil {
			return nil, fmt.Errorf("parse compose file %q: %w", p, err)
		}
		composeDir := filepath.Dir(filepath.Clean(p))
		for name, svc := range spec.Services {
			// Resolve relative host paths to absolute before merging so that
			// path resolution is always relative to the declaring file's directory.
			svc.Volumes = resolveVolumeHostPaths(svc.Volumes, composeDir)
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
		dst.Ports = src.Ports // later file replaces, per compose override semantics
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
	if len(src.Volumes) > 0 {
		dst.Volumes = append(dst.Volumes, src.Volumes...)
	}
}

func resolveServiceDef(def *composeSvcDef) (*resolvedService, error) {
	rs := &resolvedService{
		Image:      def.Image,
		User:       def.User,
		WorkingDir: def.WorkingDir,
		ExtraHosts: def.ExtraHosts,
		Restart:    def.Restart,
	}
	rs.Env = resolveEnv(def.Environment)
	rs.Cmd = resolveStringOrSlice(def.Command)
	rs.Entrypoint = resolveStringOrSlice(def.Entrypoint)
	rs.DependsOn = resolveDependsOn(def.DependsOn)
	rs.Labels = resolveLabels(def.Labels)
	rs.Binds = resolveBinds(def.Volumes)
	rs.Healthcheck = resolveHealthcheck(def.Healthcheck)

	var err error
	rs.PortBindings, rs.ExposedPorts, err = resolvePorts(def.Ports)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

// resolveHealthcheck converts a compose healthcheck definition to a Docker
// HealthConfig.  Only the disable case is handled — a disable:true entry
// produces {Test:["NONE"]} which instructs Docker to clear the image default.
// Custom test commands are also forwarded verbatim so they override the image.
func resolveHealthcheck(hc *composeSvcHealthcheck) *container.HealthConfig {
	if hc == nil {
		return nil // use image default
	}
	if hc.Disable {
		return &container.HealthConfig{Test: []string{"NONE"}}
	}
	test := resolveStringOrSlice(hc.Test)
	if len(test) == 0 {
		return nil
	}
	cfg := &container.HealthConfig{Test: test}
	if hc.Retries > 0 {
		cfg.Retries = hc.Retries
	}
	return cfg
}

// resolveVolumeHostPaths rewrites relative host-side paths in volume entries
// to absolute paths using composeDir as the base directory. Only entries in
// short-string format ("HOST:CONTAINER" or "HOST:CONTAINER:MODE") are
// modified; map-form or named-volume entries are passed through unchanged.
func resolveVolumeHostPaths(vols []interface{}, composeDir string) []interface{} {
	if len(vols) == 0 {
		return vols
	}
	out := make([]interface{}, len(vols))
	for i, v := range vols {
		s, ok := v.(string)
		if !ok {
			out[i] = v
			continue
		}
		out[i] = resolveBindHostPath(s, composeDir)
	}
	return out
}

// resolveBindHostPath resolves a relative host path in a bind-mount spec to an
// absolute path. Named volumes (no path separator in the host portion) are
// returned unchanged.
func resolveBindHostPath(spec, composeDir string) string {
	parts := strings.SplitN(spec, ":", 3)
	if len(parts) < 2 {
		return spec
	}
	host := parts[0]
	// Named volumes contain no slash and don't start with . or /
	if !filepath.IsAbs(host) && !strings.HasPrefix(host, ".") &&
		!strings.Contains(host, "/") && !strings.Contains(host, "\\") {
		return spec
	}
	if !filepath.IsAbs(host) {
		host = filepath.Join(composeDir, host)
	}
	// Ensure the result is absolute even when composeDir itself was relative.
	if abs, err := filepath.Abs(host); err == nil {
		host = abs
	}
	parts[0] = host
	return strings.Join(parts, ":")
}

// resolveBinds extracts Docker bind-mount strings from compose volume entries.
// Named volumes and map-form entries are skipped; only bind mounts with
// absolute host paths (after resolveVolumeHostPaths has run) are returned.
func resolveBinds(vols []interface{}) []string {
	var binds []string
	for _, v := range vols {
		s, ok := v.(string)
		if !ok {
			continue
		}
		parts := strings.SplitN(s, ":", 2)
		if len(parts) < 2 {
			continue
		}
		// Only absolute host paths are bind mounts; named volumes are relative
		if !filepath.IsAbs(parts[0]) {
			continue
		}
		binds = append(binds, s)
	}
	return binds
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
