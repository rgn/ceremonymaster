package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// reviewer represents a reviewer key and its numeric index (e.g. reviewer_1 -> 1).
type reviewer struct {
	key             string
	idx             int
	reviewCompleted bool
}

type EvaluationModel struct {
	FormInitialized   bool
	Form              *huh.Form
	Forms             map[int]*huh.Form // per reviewer idx
	ActiveReviewerIdx int

	Reviewers map[int]reviewer
	//ReviewerLookup        map[string]reviewer
	ReviewerReverseLookup map[int]string

	// Results holds the finalized values after Form completion.
	Results map[string]any
}

func (m *Model) InitEvaluationModel() {

	m.Evaluation = EvaluationModel{
		ActiveReviewerIdx: math.MaxInt32,
		Form:              nil,
		Forms:             make(map[int]*huh.Form),
		Reviewers:         make(map[int]reviewer),
		//ReviewerLookup:        make(map[string]reviewer),
		ReviewerReverseLookup: make(map[int]string),
		Results:               make(map[string]any),
	}

	reviewerRe := regexp.MustCompile(`reviewer_(\d+)`)
	for _, g := range m.Cfg.DataCollection {
		groupKey := g.Key
		for _, fc := range g.Fields {
			fieldKey := BuildFieldKey(groupKey, fc.Key)
			if sm := reviewerRe.FindStringSubmatch(fieldKey); sm != nil {
				if n, err := strconv.Atoi(sm[1]); err == nil {
					r := reviewer{key: fieldKey, idx: n} // reviewerName
					m.Evaluation.Reviewers[n] = r
				}
			}
		}
	}

	minId := 9999999
	for _, r := range m.Evaluation.Reviewers {
		minId = min(minId, r.idx)
	}

	m.Evaluation.ActiveReviewerIdx = minId

	for _, reviewer := range m.Evaluation.Reviewers {
		reviewerEvaluationGroups := m.buildGroups(m.Cfg.Evaluation)
		reviewerForm := huh.NewForm(reviewerEvaluationGroups...).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(true)

		m.Evaluation.Forms[reviewer.idx] = reviewerForm

	}

	m.Evaluation.FormInitialized = false
	m.Evaluation.Form = m.Evaluation.Forms[m.Evaluation.ActiveReviewerIdx]
}

func buildReviewerKey(prefix string, revreviewer reviewer) string {
	return fmt.Sprintf("%s_reviewer_%d", prefix, revreviewer.idx)
}

func (m *Model) getReviewerName(reviewerIdx int) string {

	reviewerKey := m.Evaluation.Reviewers[reviewerIdx].key
	reviewerName := m.DataEntry.Form.GetString(reviewerKey)
	if strings.TrimSpace(reviewerName) == "" {
		reviewerName = reviewerKey
	}

	return reviewerName
}

func (m *EvaluationModel) getNextReviewerIdx() int {
	next := -1
	// iterate reviewers in numeric order for determinism
	keys := make([]int, 0, len(m.Reviewers))
	for _, r := range m.Reviewers {
		keys = append(keys, r.idx)
	}
	sort.Ints(keys)
	for _, k := range keys {
		r := m.Reviewers[k]
		if r.reviewCompleted {
			continue
		}
		next = k
		break
	}
	return next
}

func (m *EvaluationModel) hasPendingReviews() bool {
	for _, r := range m.Reviewers {
		if !r.reviewCompleted {
			return true
		}
	}
	return false
}

func (mainModel *Model) UpdateEvaluationModel(msg tea.Msg) []tea.Cmd {

	cmds := []tea.Cmd{}

	if mainModel.State != STATE_EVALUATION {
		return cmds
	}

	m := &mainModel.Evaluation
	var cmd tea.Cmd

	if m.Forms[m.ActiveReviewerIdx] == nil {
		return cmds
	}

	Form, cmd := m.Forms[m.ActiveReviewerIdx].Update(msg)
	if f, ok := Form.(*huh.Form); ok {
		m.Form = f
	}

	if m.Forms[m.ActiveReviewerIdx].State == huh.StateCompleted {
		revIdx := m.ActiveReviewerIdx
		rev := m.Reviewers[revIdx]

		// collect current evaluation field values into Results with suffix
		for _, g := range mainModel.Cfg.Evaluation {
			for _, fc := range g.Fields {
				k := fc.Key
				// check for keys specific to the current reviewer
				// add value to a reviewer specific result key
				outKey := buildReviewerKey(k, rev)
				if p, ok := mainModel.Values[k]; ok {
					switch v := p.(type) {
					case *string:
						m.Results[outKey] = *v
						// reset backing pointer for next run
						*v = ""
					case *bool:
						m.Results[outKey] = *v
						*v = false
					case *int:
						m.Results[outKey] = *v
						*v = 0
					case *[]string:
						m.Results[outKey] = *v
						*v = nil
					default:
						m.Results[outKey] = v
					}
				}
			}
		}

		// mark reviewer completed in the map
		if r, ok := m.Reviewers[revIdx]; ok {
			r.reviewCompleted = true
			m.Reviewers[revIdx] = r
		}

		// find next reviewer without a review
		next := m.getNextReviewerIdx()

		if next != -1 {
			m.ActiveReviewerIdx = next

			// initialize the new Form
			cmds = append(cmds, m.Forms[m.ActiveReviewerIdx].Init())
		} else {
			// no more reviewers -> finish
			mainModel.State = STATE_SUMMARY
		}
	} else {
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (m *Model) ViewEvaluation() (header string, body string, footer string) {
	s := m.Styles

	reviewerName := m.getReviewerName(m.Evaluation.ActiveReviewerIdx)

	header += fmt.Sprintf("Review by %s - %d/%d", s.Highlight.Render(reviewerName), m.Evaluation.ActiveReviewerIdx, len(m.Evaluation.Reviewers))

	switch m.Evaluation.Forms[m.Evaluation.ActiveReviewerIdx].State {
	case huh.StateCompleted:
		// nothing
	default:

		v := strings.TrimSuffix(m.Evaluation.Form.View(), "\n\n")
		renderedForm := m.Lg.NewStyle().Margin(1, 0).Render(v)

		errors := m.Evaluation.Form.Errors()
		if len(errors) > 0 {
			header = m.appErrorBoundaryView(m.errorView(m.Evaluation.Form))
		}

		const statusWidth = 28
		sb := strings.Builder{}
		if m.Evaluation.Results != nil {
			for k, v := range m.Evaluation.Results {
				switch t := v.(type) {
				case string:
					sb.WriteString(fmt.Sprintf("%s: %s\n", k, t))
				case bool, int:
					sb.WriteString(fmt.Sprintf("%s: %v\n", k, t))
				case []string:
					sb.WriteString(fmt.Sprintf("%s: %s\n", k, strings.Join(t, ", ")))
				default:
					sb.WriteString(fmt.Sprintf("%s: %v\n", k, t))
				}
			}
		}

		statusMarginLeft := m.width - statusWidth - lipgloss.Width(renderedForm) - s.Status.GetMarginRight()
		status := s.Status.
			Height(lipgloss.Height(renderedForm)).
			Width(statusWidth).
			MarginLeft(statusMarginLeft).
			Render(s.StatusHeader.Render("Ergebnis") + "\n\n" + sb.String())

		body = lipgloss.JoinHorizontal(lipgloss.Left, renderedForm, status)
		body = lipgloss.JoinVertical(lipgloss.Top, []string{body}...)

		footer = m.appBoundaryView(m.Evaluation.Form.Help().ShortHelpView(m.Evaluation.Form.KeyBinds()))
		if len(errors) > 0 {
			footer = m.appErrorBoundaryView("")
		}

		body = lipgloss.JoinHorizontal(lipgloss.Left, renderedForm)

		footer = m.appBoundaryView(m.DataEntry.Form.Help().ShortHelpView(m.DataEntry.Form.KeyBinds()))
		if len(errors) > 0 {
			footer = m.appErrorBoundaryView("")
		}
	}

	return header, body, footer
}
