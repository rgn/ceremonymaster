package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

type Reviewer struct {
	Name      string
	FormKey   string
	Idx       int
	Completed bool
}

func BuildReviewerKey(prefix string, reviewer Reviewer) string {
	return fmt.Sprintf("%s_reviewer_%d", prefix, reviewer.Idx)
}

func (mainModel *Model) BuildReviewers() {
	reviewerRe := regexp.MustCompile(`reviewer_(\d+)`)

	for _, g := range mainModel.Cfg.DataCollection {
		groupKey := g.Key
		for _, fc := range g.Fields {
			fieldKey := BuildFieldKey(groupKey, fc.Key)
			if sm := reviewerRe.FindStringSubmatch(fieldKey); sm != nil {
				if n, err := strconv.Atoi(sm[1]); err == nil {
					r := Reviewer{
						Name:    mainModel.DataEntry.Form.GetString(fieldKey),
						FormKey: fieldKey,
						Idx:     n,
					}

					mainModel.Reviewers[r.Idx] = &r
				}
			}
		}
	}

	keys := make([]int, 0, len(mainModel.Reviewers))

	for _, r := range mainModel.Reviewers {
		keys = append(keys, r.Idx)
	}

	// iterate reviewers in numeric oder for deterministic behavior
	sort.Ints(keys)

	mainModel.SortedReviewerKeys = keys
}

func (m *Model) GetNextReviewerIdx() int {
	next := -1

	for _, k := range m.SortedReviewerKeys {
		r := m.Reviewers[k]
		logger.Println("Checking reviewer idx:", r.Idx, "completed:", r.Completed)
		if r.Completed {
			continue
		}
		next = r.Idx
		break
	}

	return next
}

func (mainModel *Model) GetReviewerName(reviewerIdx int) string {

	reviewerName := mainModel.Reviewers[reviewerIdx].Name
	if reviewerName == "" {
		reviewerName = mainModel.Reviewers[reviewerIdx].FormKey
		if reviewerName == "" {
			reviewerName = "Reviewer " + strconv.Itoa(reviewerIdx)
		}
	}

	return reviewerName
}
