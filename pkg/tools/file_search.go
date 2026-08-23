package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	defaultFindFilesLimit = 50
	maxFindFilesLimit     = 200
)

// FindFilesTool provides a predictable, cross-platform way for the agent to
// locate files before reading or editing them. It deliberately does not follow
// directory symlinks, which keeps searches fast and avoids escaping a trusted
// workspace through a link.
type FindFilesTool struct {
	workspace string
	restrict  bool
	patterns  []*regexp.Regexp
}

func NewFindFilesTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *FindFilesTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &FindFilesTool{workspace: workspace, restrict: restrict, patterns: patterns}
}

func (t *FindFilesTool) Name() string { return "find_files" }

func (t *FindFilesTool) Description() string {
	return "Find files and folders recursively by name, extension, or path fragment before reading or editing them. Searches the current workspace by default, supports macOS and Windows-style path separators, skips generated/dependency folders, and returns paths ready for read_file or edit_file."
}

func (t *FindFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Case-insensitive file or folder name, extension, or path fragment. Supports * and ? wildcards.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional directory to search. Relative paths use the current workspace; absolute paths are allowed only when the existing file safety policy permits them.",
			},
			"kind": map[string]any{
				"type":        "string",
				"enum":        []string{"file", "directory", "any"},
				"description": "What to return. Defaults to file.",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum results to return, from 1 to 200. Defaults to 50.",
			},
		},
		"required": []string{"query"},
	}
}

type findFileResult struct {
	Path         string `json:"path"`
	AbsolutePath string `json:"absolute_path,omitempty"`
	Kind         string `json:"kind"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
}

func (t *FindFilesTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return ErrorResult("query is required: provide a file name, extension, or path fragment")
	}

	rootInput, _ := args["path"].(string)
	rootInput = strings.TrimSpace(rootInput)
	if rootInput == "" {
		rootInput = "."
	}
	root, err := validatePathWithAllowPaths(rootInput, t.workspace, t.restrict, t.patterns)
	if err != nil {
		return ErrorResult(err.Error())
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to inspect search path: %v", err))
	}
	if !rootInfo.IsDir() {
		return ErrorResult("search path is not a directory")
	}

	kind, _ := args["kind"].(string)
	if kind == "" {
		kind = "file"
	}
	if kind != "file" && kind != "directory" && kind != "any" {
		return ErrorResult("kind must be file, directory, or any")
	}

	limit := defaultFindFilesLimit
	if _, exists := args["max_results"]; exists {
		parsed, err := getInt64Arg(args, "max_results", int64(defaultFindFilesLimit))
		if err != nil {
			return ErrorResult(err.Error())
		}
		if parsed <= 0 {
			return ErrorResult("max_results must be greater than 0")
		}
		if parsed > maxFindFilesLimit {
			parsed = maxFindFilesLimit
		}
		limit = int(parsed)
	}

	query = normalizeSearchText(query)
	results := make([]findFileResult, 0, limit)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Permission-denied children should not make an otherwise useful
			// search fail. The root itself was already checked above.
			if path == root || os.IsPermission(walkErr) {
				return nil
			}
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path != root && entry.IsDir() && shouldSkipFindDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		isDir := entry.IsDir()
		if (kind == "file" && isDir) || (kind == "directory" && !isDir) {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil || !matchesFindQuery(relative, entry.Name(), query) {
			return nil
		}

		result := findFileResult{
			Path: searchResultPath(path, t.workspace),
			Kind: map[bool]string{true: "directory", false: "file"}[isDir],
		}
		if !t.restrict || isAllowedPath(path, t.patterns) {
			result.AbsolutePath = filepath.Clean(path)
		}
		if !isDir {
			if info, infoErr := entry.Info(); infoErr == nil {
				result.SizeBytes = info.Size()
			}
		}
		results = append(results, result)
		if len(results) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll && err != context.Canceled && err != context.DeadlineExceeded {
		return ErrorResult(fmt.Sprintf("file search failed: %v", err))
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	if len(results) == 0 {
		return NewToolResult(fmt.Sprintf("No files or folders matching %q were found under %s.", query, filepath.ToSlash(rootInput)))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "Found %d result(s) for %q under %s:\n", len(results), query, filepath.ToSlash(rootInput))
	for _, result := range results {
		fmt.Fprintf(&output, "%s: %s", strings.ToUpper(result.Kind), result.Path)
		if result.AbsolutePath != "" {
			fmt.Fprintf(&output, " (absolute: %s)", result.AbsolutePath)
		}
		if result.SizeBytes > 0 {
			fmt.Fprintf(&output, " [%d bytes]", result.SizeBytes)
		}
		output.WriteByte('\n')
	}
	if len(results) == limit {
		fmt.Fprintf(&output, "\n[RESULT LIMIT: showing the first %d matches. Narrow the query or path for more.]", limit)
	}
	return NewToolResult(output.String())
}

func normalizeSearchText(value string) string {
	return strings.ToLower(strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/"))
}

func matchesFindQuery(relativePath, baseName, query string) bool {
	relative := normalizeSearchText(relativePath)
	base := normalizeSearchText(baseName)
	if strings.Contains(relative, query) || strings.Contains(base, query) {
		return true
	}
	if strings.ContainsAny(query, "*?") {
		if matched, _ := filepath.Match(query, base); matched {
			return true
		}
		if matched, _ := filepath.Match(query, relative); matched {
			return true
		}
	}
	return false
}

func searchResultPath(path, workspace string) string {
	if workspace != "" {
		if absWorkspace, err := filepath.Abs(workspace); err == nil {
			if relative, relErr := filepath.Rel(absWorkspace, path); relErr == nil && filepath.IsLocal(relative) {
				return filepath.ToSlash(relative)
			}
		}
	}
	return filepath.Clean(path)
}

func shouldSkipFindDirectory(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".svn", ".hg", "node_modules", "vendor", ".venv", "venv", "__pycache__", ".cache", "build", "dist", "target":
		return true
	default:
		return strings.HasPrefix(name, ".") && name != ".config"
	}
}
