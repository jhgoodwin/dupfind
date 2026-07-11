package near

import "fmt"

// unrelated is a function that shares some Go boilerplate (error handling)
// with the variants but is semantically unrelated. It gives IDF enough
// data to down-weight common patterns.
func unrelated(items []string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("empty list")
	}
	result := items[0]
	for _, item := range items[1:] {
		result = result + " " + item
	}
	return result, nil
}
