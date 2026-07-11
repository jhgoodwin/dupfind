package falsepos

import (
	"fmt"
	"os"
	"strings"
)

// doSomething and doSomethingElse are semantically unrelated but
// share the same Go error-handling structure. dupfind should flag
// them as exact or near duplicates — but they are NOT copies.

func doSomething(path string) (int, error) {
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

func doSomethingElse(name string) (int, error) {
	value, err := lookupName(name)
	if err != nil {
		return 0, fmt.Errorf("lookup %s: %w", name, err)
	}
	total := 0
	for _, item := range splitParts(value) {
		if len(item) > 0 {
			total++
		}
	}
	return total, nil
}

func lookupName(name string) (string, error) {
	return name, nil
}

func splitParts(s string) []string {
	return strings.Split(s, ",")
}
