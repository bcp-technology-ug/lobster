package coverage

import (
	"os"
	"regexp"
	"strings"
)

// rpcPattern matches `rpc MethodName(...)` lines inside a service block.
// We strip comments before applying this so multi-line and inline comments
// do not confuse the match.
var rpcPattern = regexp.MustCompile(`\brpc\s+(\w+)\s*\(`)

// servicePattern matches `service ServiceName {` blocks.
var servicePattern = regexp.MustCompile(`\bservice\s+(\w+)\s*\{`)

// blockCommentPattern strips /* ... */ style comments (non-greedy).
var blockCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)

// lineCommentPattern strips // ... to end-of-line comments.
var lineCommentPattern = regexp.MustCompile(`//[^\n]*`)

// ScanProtoGlob parses all .proto files matching the given glob and returns
// one CoverageItem per rpc method found.
func ScanProtoGlob(glob string) ([]CoverageItem, error) {
	paths, err := expandGlob(glob)
	if err != nil {
		return nil, err
	}
	var items []CoverageItem
	for _, p := range paths {
		got, err := scanProtoFile(p)
		if err != nil {
			return nil, err
		}
		items = append(items, got...)
	}
	return items, nil
}

// scanProtoFile extracts all rpc definitions from a single .proto file.
func scanProtoFile(path string) ([]CoverageItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(data)

	// Strip comments so they don't confuse the patterns.
	src = blockCommentPattern.ReplaceAllString(src, " ")
	src = lineCommentPattern.ReplaceAllString(src, " ")

	var items []CoverageItem

	// Walk the file finding service blocks.
	// Strategy: split on `service X {` boundaries then scan each block for rpcs.
	serviceLocs := servicePattern.FindAllStringSubmatchIndex(src, -1)
	if len(serviceLocs) == 0 {
		return nil, nil
	}

	for i, loc := range serviceLocs {
		serviceName := src[loc[2]:loc[3]]

		// The service block runs from loc[1] to the start of the next service
		// block (or end of file).
		blockStart := loc[1]
		blockEnd := len(src)
		if i+1 < len(serviceLocs) {
			blockEnd = serviceLocs[i+1][0]
		}
		block := src[blockStart:blockEnd]

		// Find matching closing brace for this service block.
		depth := 1
		end := -1
		for j, ch := range block {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					end = j
					break
				}
			}
		}
		if end > 0 {
			block = block[:end]
		}

		// Find all rpc methods inside this service block.
		rpcLocs := rpcPattern.FindAllStringSubmatch(block, -1)
		for _, m := range rpcLocs {
			methodName := strings.TrimSpace(m[1])
			id := serviceName + "." + methodName
			items = append(items, CoverageItem{
				ID:      id,
				Kind:    KindRPC,
				Label:   serviceName + "." + methodName,
				Service: serviceName,
				Method:  methodName,
			})
		}
	}
	return items, nil
}
