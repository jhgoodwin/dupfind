package mincopies

import (
	"fmt"
	"os"
	"strings"
)

// threeCopiesA, threeCopiesB, threeCopiesC are identical functions that
// should be detected as exact duplicates when min-copies=3.
func threeCopiesA(path string) (int, error) {
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
