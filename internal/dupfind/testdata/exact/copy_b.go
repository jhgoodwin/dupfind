package exact

import (
	"fmt"
	"os"
	"strings"
)

// processFile is an exact copy of the one in copy_a.go.
func processFile(path string) (int, error) {
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
