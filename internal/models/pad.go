package models

import (
	"fmt"
	"regexp"
	"strings"
)

// Pad represents a single pad document.
type Pad struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	ParentPath string `json:"parent_path,omitempty"`
	UpdatedAt  int64  `json:"updated_at"`
	CreatedAt  int64  `json:"created_at"`
}

// ChildPad is a lightweight representation for listing children.
type ChildPad struct {
	Path      string `json:"path"`
	UpdatedAt int64  `json:"updated_at"`
}

var (
	// validSegment matches lowercase alphanumeric, hyphens, and underscores.
	validSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

	// reservedPaths are blocked from being created as pads.
	reservedPaths = map[string]bool{
		"api":           true,
		"static":        true,
		"healthz":       true,
		"manifest.json": true,
		"sw.js":         true,
		"favicon.ico":   true,
	}

	maxDepth         = 10
	maxSegmentLength = 64
	maxPathLength    = 512
)

// NormalizePath lowercases, strips trailing slashes, and collapses double slashes.
func NormalizePath(path string) string {
	path = strings.ToLower(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	// Collapse multiple slashes.
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// Trim again after collapse.
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	return path
}

// ValidatePath checks if a path is valid for use as a pad path.
// Returns an error describing the validation failure, or nil if valid.
func ValidatePath(path string) error {
	// Root path is always valid.
	if path == "" {
		return nil
	}

	if len(path) > maxPathLength {
		return fmt.Errorf("path exceeds maximum length of %d characters", maxPathLength)
	}

	segments := strings.Split(path, "/")

	if len(segments) > maxDepth {
		return fmt.Errorf("path exceeds maximum depth of %d levels", maxDepth)
	}

	// Check if the first segment is a reserved path.
	if reservedPaths[segments[0]] {
		return fmt.Errorf("path '%s' is reserved", segments[0])
	}

	for _, seg := range segments {
		if seg == "" {
			return fmt.Errorf("path contains empty segment")
		}
		if len(seg) > maxSegmentLength {
			return fmt.Errorf("segment '%s' exceeds maximum length of %d characters", seg, maxSegmentLength)
		}
		if !validSegment.MatchString(seg) {
			return fmt.Errorf("segment '%s' contains invalid characters (allowed: lowercase alphanumeric, hyphen, underscore)", seg)
		}
	}

	return nil
}

// ParentPath computes the parent path from a given path.
// Root path returns "" (self). "mypage" returns "". "mypage/hello" returns "mypage".
func ParentPath(path string) string {
	if path == "" {
		return ""
	}
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return ""
	}
	return path[:idx]
}

// IsReservedPath checks if the first segment of a path is reserved.
func IsReservedPath(path string) bool {
	if path == "" {
		return false
	}
	segments := strings.Split(path, "/")
	return reservedPaths[segments[0]]
}
