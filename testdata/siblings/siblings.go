package siblings

import "fmt"

// dispatch contains case clauses that are structurally similar (same pattern
// of fmt.Sprintf + return). The siblings filter should prevent them from
// being reported as near-duplicates since they belong to the same switch.
func dispatch(cmd string) string {
	switch cmd {
	case "run":
		return fmt.Sprintf("running %s", cmd)
	case "stop":
		return fmt.Sprintf("stopping %s", cmd)
	case "status":
		return fmt.Sprintf("status %s", cmd)
	default:
		return fmt.Sprintf("unknown %s", cmd)
	}
}
