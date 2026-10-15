package host

import (
	"sync"
	"time"
)

const (
	historyWindow = time.Hour
	historyMax    = 360
)

type history struct {
	mu     sync.Mutex
	points []Point
}

func (h *history) add(point Point, now time.Time) []Point {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := now.Add(-historyWindow)
	next := make([]Point, 0, len(h.points)+1)
	for _, item := range h.points {
		at, err := time.Parse(time.RFC3339, item.At)
		if err != nil || at.Before(cutoff) {
			continue
		}
		next = append(next, item)
	}
	next = append(next, point)
	if len(next) > historyMax {
		next = next[len(next)-historyMax:]
	}
	h.points = next
	out := make([]Point, len(next))
	copy(out, next)
	return out
}
