package coverage

import (
	"fmt"
	"os"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// openAPIDoc is the minimal structure we need from an OpenAPI 3.x document.
type openAPIDoc struct {
	Paths map[string]pathItem `yaml:"paths"`
}

// pathItem maps HTTP method names to operation objects.
type pathItem map[string]interface{}

// httpMethods are the standard HTTP verbs we look for in path items.
var httpMethods = []string{"get", "post", "put", "patch", "delete", "head", "options"}

// ScanOpenAPI parses an OpenAPI YAML file at path and returns one CoverageItem
// per path+method combination found.
func ScanOpenAPI(path string) ([]CoverageItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read openapi file %q: %w", path, err)
	}

	var doc openAPIDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi file %q: %w", path, err)
	}

	var items []CoverageItem
	for apiPath, pathObj := range doc.Paths {
		for _, method := range httpMethods {
			if _, ok := pathObj[method]; ok {
				methodUpper := strings.ToUpper(method)
				id := methodUpper + ":" + apiPath
				items = append(items, CoverageItem{
					ID:         id,
					Kind:       KindHTTP,
					Label:      methodUpper + " " + apiPath,
					HTTPMethod: methodUpper,
					HTTPPath:   apiPath,
				})
			}
		}
	}
	return items, nil
}
