package host

import "slices"

func limitProcesses(list []Process, limit int, sortBy string) []Process {
	slices.SortStableFunc(list, func(a, b Process) int {
		av, bv := a.CPUPercent, b.CPUPercent
		if sortBy == "mem" {
			av, bv = a.MemPercent, b.MemPercent
		}
		switch {
		case av > bv:
			return -1
		case av < bv:
			return 1
		case a.PID < b.PID:
			return -1
		case a.PID > b.PID:
			return 1
		default:
			return 0
		}
	})
	if limit > len(list) {
		limit = len(list)
	}
	if limit < 0 {
		limit = 0
	}
	out := make([]Process, limit)
	copy(out, list[:limit])
	return out
}
