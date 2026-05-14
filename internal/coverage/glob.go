package coverage

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// expandGlob resolves a glob pattern to a list of matching file paths.
// Unlike filepath.Glob, it supports the "**" double-star wildcard, which
// matches zero or more path segments recursively.
func expandGlob(pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		return filepath.Glob(pattern)
	}

	// Split on the first "**" segment to get the root directory and the
	// suffix pattern that must match the remainder of the path.
	parts := strings.SplitN(pattern, "**", 2)
	root := filepath.Clean(parts[0])
	suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))

	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If the root directory doesn't exist, return no matches (not an error).
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if suffix == "" {
			matches = append(matches, path)
			return nil
		}
		// Match the trailing portion against the suffix using filepath.Match.
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		// Try matching the suffix against the full relative path so that
		// "v1/*.proto" works correctly when rel is "example/v1/svc.proto".
		ok, err := filepath.Match(suffix, filepath.Base(path))
		if err != nil {
			return nil
		}
		if ok {
			matches = append(matches, path)
			return nil
		}
		// Also try matching the full relative path against the suffix.
		ok, err = filepath.Match(suffix, rel)
		if err != nil {
			return nil
		}
		if ok {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}
