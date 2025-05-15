package filesystem

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/navicore/mcpterm-go/pkg/tools/core"
)

// GrepInput represents parameters for the grep tool
type GrepInput struct {
	Pattern    string `json:"pattern"`               // The pattern to search for
	Path       string `json:"path,omitempty"`        // Directory to search in (optional, defaults to current directory)
	Include    string `json:"include,omitempty"`     // File pattern to include (e.g., "*.go")
	Exclude    string `json:"exclude,omitempty"`     // File pattern to exclude
	Recursive  bool   `json:"recursive,omitempty"`   // Whether to search recursively (default true)
	IgnoreCase bool   `json:"ignore_case,omitempty"` // Case insensitive search
	MaxFiles   int    `json:"max_files,omitempty"`   // Maximum number of files to search
	MaxResults int    `json:"max_results,omitempty"` // Maximum number of results to return
}

// GrepMatch represents a single match in a file
type GrepMatch struct {
	LineNumber int    `json:"line_number"` // Line number where the match was found
	LineText   string `json:"line_text"`   // The text of the matched line
}

// GrepFileResult represents the grep results for a single file
type GrepFileResult struct {
	FilePath string      `json:"file_path"` // Path to the file
	Matches  []GrepMatch `json:"matches"`   // Matches found in the file
}

// GrepResult represents the complete grep results
type GrepResult struct {
	Pattern      string           `json:"pattern"`             // The pattern that was searched for
	TotalMatches int              `json:"total_matches"`       // Total number of matches found
	FilesMatched int              `json:"files_matched"`       // Number of files with matches
	Results      []GrepFileResult `json:"results"`             // Results for each file
	Error        string           `json:"error,omitempty"`     // Error message, if any
	Truncated    bool             `json:"truncated,omitempty"` // Whether results were truncated
}

// GrepTool implements a tool for searching file contents
type GrepTool struct {
	core.BaseToolImpl
}

// NewGrepTool creates a new grep tool
func NewGrepTool() *GrepTool {
	tool := &GrepTool{}
	tool.BaseToolImpl = *core.NewBaseTool(
		"grep",
		"Search file contents for specified patterns",
		"filesystem",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regular expression pattern to search for",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory to search in (defaults to current directory)",
				},
				"include": map[string]interface{}{
					"type":        "string",
					"description": "File pattern to include (e.g., '*.go')",
				},
				"exclude": map[string]interface{}{
					"type":        "string",
					"description": "File pattern to exclude",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to search recursively (default true)",
				},
				"ignore_case": map[string]interface{}{
					"type":        "boolean",
					"description": "Case insensitive search",
				},
				"max_files": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of files to search",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return",
				},
			},
			"required": []string{"pattern"},
		},
	)
	return tool
}

// Execute implements the Tool interface
func (t *GrepTool) Execute(input json.RawMessage) (interface{}, error) {
	var params GrepInput
	if err := json.Unmarshal(input, &params); err != nil {
		return GrepResult{
			Error: fmt.Sprintf("Invalid input: %v", err),
		}, fmt.Errorf("invalid input for grep tool: %w", err)
	}

	// Validate pattern
	if params.Pattern == "" {
		return GrepResult{
			Error: "Pattern is required",
		}, fmt.Errorf("pattern is required")
	}

	// Set default path if not provided
	searchPath := params.Path
	if searchPath == "" {
		// Use current directory if not specified
		var err error
		searchPath, err = os.Getwd()
		if err != nil {
			return GrepResult{
				Error: fmt.Sprintf("Failed to get current directory: %v", err),
			}, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Set defaults
	if params.Recursive == false {
		// Default to recursive search
		params.Recursive = true
	}

	if params.MaxFiles <= 0 {
		// Default max files
		params.MaxFiles = 100
	}

	if params.MaxResults <= 0 {
		// Default max results
		params.MaxResults = 1000
	}

	// Compile the pattern
	var pattern *regexp.Regexp
	var err error
	if params.IgnoreCase {
		// Case insensitive
		pattern, err = regexp.Compile("(?i)" + params.Pattern)
	} else {
		pattern, err = regexp.Compile(params.Pattern)
	}

	if err != nil {
		return GrepResult{
			Pattern: params.Pattern,
			Error:   fmt.Sprintf("Invalid pattern: %v", err),
		}, fmt.Errorf("invalid pattern: %w", err)
	}

	// Perform the search
	result := GrepResult{
		Pattern:      params.Pattern,
		TotalMatches: 0,
		FilesMatched: 0,
		Results:      []GrepFileResult{},
	}

	// Check if the search path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		result.Error = fmt.Sprintf("Search failed: path %s does not exist", searchPath)
		return result, fmt.Errorf("search failed: path %s does not exist", searchPath)
	}

	err = t.searchFiles(searchPath, params, pattern, &result)
	if err != nil {
		result.Error = fmt.Sprintf("Search failed: %v", err)
		return result, fmt.Errorf("search failed: %w", err)
	}

	return result, nil
}

// searchFiles searches files in the given path
func (t *GrepTool) searchFiles(rootPath string, params GrepInput, pattern *regexp.Regexp, result *GrepResult) error {
	fileCount := 0

	walkFn := func(path string, info os.FileInfo, err error) error {
		// Skip errors in accessing files
		if err != nil {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// Skip recursive search if not enabled
			if !params.Recursive && path != rootPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file count limit
		if fileCount >= params.MaxFiles {
			result.Truncated = true
			return filepath.SkipAll
		}

		// Check file name patterns
		if params.Include != "" {
			matched, err := filepath.Match(params.Include, filepath.Base(path))
			if err != nil {
				return nil // Skip on error
			}
			if !matched {
				return nil
			}
		}

		if params.Exclude != "" {
			matched, err := filepath.Match(params.Exclude, filepath.Base(path))
			if err != nil {
				return nil // Skip on error
			}
			if matched {
				return nil
			}
		}

		// Process the file
		fileCount++
		matches, err := t.searchInFile(path, pattern, params.MaxResults-result.TotalMatches)
		if err != nil {
			return nil // Skip on error
		}

		// Add results if we found matches
		if len(matches) > 0 {
			result.FilesMatched++
			result.TotalMatches += len(matches)

			fileResult := GrepFileResult{
				FilePath: path,
				Matches:  matches,
			}
			result.Results = append(result.Results, fileResult)

			// Check if we've hit the max results
			if result.TotalMatches >= params.MaxResults {
				result.Truncated = true
				return filepath.SkipAll
			}
		}

		return nil
	}

	return filepath.Walk(rootPath, walkFn)
}

// searchInFile searches for the pattern in a single file
func (t *GrepTool) searchInFile(filePath string, pattern *regexp.Regexp, maxMatches int) ([]GrepMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	matches := []GrepMatch{}
	lineNum := 0
	scanner := bufio.NewScanner(file)

	// Handle binary files gracefully
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if we have a match
		if pattern.MatchString(line) {
			matches = append(matches, GrepMatch{
				LineNumber: lineNum,
				LineText:   line,
			})

			// Check if we've hit the max matches for this file
			if len(matches) >= maxMatches {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// If it's likely a binary file, just skip it
		if err == io.ErrUnexpectedEOF || strings.Contains(err.Error(), "invalid UTF-8") {
			return nil, nil
		}
		return nil, err
	}

	return matches, nil
}
