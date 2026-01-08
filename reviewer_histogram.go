package main

import (
	"fmt"
	"sort"
)

// BuildReviewerHistogram scans all certificates and returns a map of reviewer -> count
func BuildReviewerHistogram() (map[string]int, error) {
	counts := make(map[string]int)

	for _, p := range GetCertificateFilePaths() {
		cert, err := LoadCertificate(p)
		if err != nil {
			// skip unreadable certificate but continue
			logger.Printf("warning: failed to load certificate %s: %v", p, err)
			continue
		}
		for _, r := range cert.Reviewers {
			counts[r]++
		}
	}

	return counts, nil
}

// RenderReviewerHistogram returns a textual histogram scaled to maxBarWidth.
// It sorts reviewers by count descending.
func RenderReviewerHistogram(hist map[string]int, maxBarWidth int) string {
	if maxBarWidth < 10 {
		maxBarWidth = 10
	}

	// collect and sort
	type kv struct {
		Name  string
		Count int
	}
	var list []kv
	maxCount := 0
	for k, v := range hist {
		list = append(list, kv{Name: k, Count: v})
		if v > maxCount {
			maxCount = v
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count == list[j].Count {
			return list[i].Name < list[j].Name
		}
		return list[i].Count > list[j].Count
	})

	// build lines
	var out string
	for _, e := range list {
		// only include reviewers that appear more than once
		if e.Count <= 1 {
			continue
		}
		barLen := 0
		if maxCount > 0 {
			barLen = int(float64(e.Count) / float64(maxCount) * float64(maxBarWidth))
		}
		if barLen < 0 {
			barLen = 0
		}
		out += fmt.Sprintf("%-20s |%s %d\n", e.Name, string(repeatRune('█', barLen)), e.Count)
	}
	return out
}

// repeatRune returns a string with count times r.
func repeatRune(r rune, count int) []rune {
	if count <= 0 {
		return []rune{}
	}
	res := make([]rune, count)
	for i := 0; i < count; i++ {
		res[i] = r
	}
	return res
}
