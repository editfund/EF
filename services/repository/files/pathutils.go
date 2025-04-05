package files

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var fileNameComponentSanitizeRegexp = regexp.MustCompile(`(?i)\.\.|[<>:\"/\\|?*\x{0000}-\x{001F}]|^(con|prn|aux|nul|com\d|lpt\d)$`)

// SanitizePath cleans and validates a file path
func SanitizePath(inputPath string) (string, error) {
	// Normalize path separators
	s := strings.ReplaceAll(inputPath, "\\", "/")
	// Clean the path
	s = path.Clean(s)
	// Split the path components
	pathComponents := strings.Split(s, "/")
	// Sanitize each path component
	var sanitizedComponents []string
	for _, component := range pathComponents {
		// Trim whitespace and apply regex sanitization
		sanitizedComponent := strings.TrimSpace(fileNameComponentSanitizeRegexp.ReplaceAllString(component, "_"))

		// Skip empty components after sanitization
		if sanitizedComponent != "" {
			sanitizedComponents = append(sanitizedComponents, sanitizedComponent)
		}
	}
	// Check if we have any components left after sanitization
	if len(sanitizedComponents) == 0 {
		return "", fmt.Errorf("path became empty after sanitization")
	}
	// Reconstruct the path
	reconstructedPath := path.Join(sanitizedComponents...)
	return reconstructedPath, nil
}
