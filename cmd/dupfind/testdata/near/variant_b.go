package near

import (
	"fmt"
	"os"
	"strings"
)

// countLines is a near-copy of the one in variant_a.go.
// The only difference is > 0 changed to >= 1.
func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) >= 1 {
			count++
		}
	}
	return count, nil
}
