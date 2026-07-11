package near

import (
	"fmt"
	"os"
	"strings"
)

// countLines reads a file and counts non-empty lines.
func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) > 0 {
			count++
		}
	}
	return count, nil
}
