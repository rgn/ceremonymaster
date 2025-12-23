package main

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type EvaluationModel struct {
	Form              *huh.Form
	Forms             map[int]*huh.Form // per reviewer idx
	ActiveReviewerIdx int

	// Results holds the finalized values after Form completion.
	Results map[string]any
}

func (mainModel *Model) InitEvaluationModel() {

	mainModel.Evaluation = EvaluationModel{
		ActiveReviewerIdx: math.MaxInt32,
		Form:              nil,
		Forms:             make(map[int]*huh.Form),
		Results:           make(map[string]any),
	}
}

func (mainModel *Model) BuildReviewerForms() {

	m := &mainModel.Evaluation

	logger.Println("Initialize evaluation forms for", len(mainModel.Reviewers), "reviewers.")

	for _, reviewer := range mainModel.Reviewers {
		logger.Println("Initializing evaluation form for reviewer:", reviewer.Name, "idx:", reviewer.Idx)
		reviewerEvaluationGroups := mainModel.buildGroups(mainModel.Cfg.Evaluation)
		reviewerForm := huh.NewForm(reviewerEvaluationGroups...).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(true)

		m.Forms[reviewer.Idx] = reviewerForm
	}
	minId := 9999999
	for _, r := range mainModel.Reviewers {
		minId = min(minId, r.Idx)
	}

	m.ActiveReviewerIdx = minId
}

func (mainModel *Model) UpdateEvaluationModel(msg tea.Msg) []tea.Cmd {

	cmds := []tea.Cmd{}

	if mainModel.State != STATE_EVALUATION {
		return cmds
	}

	m := &mainModel.Evaluation
	var cmd tea.Cmd

	if m.Forms[m.ActiveReviewerIdx] == nil {
		logger.Println("No form found for active reviewer idx:", m.ActiveReviewerIdx)
		return cmds
	}

	Form, cmd := m.Forms[m.ActiveReviewerIdx].Update(msg)
	if f, ok := Form.(*huh.Form); ok {
		m.Form = f
	}

	if m.Forms[m.ActiveReviewerIdx].State == huh.StateCompleted {

		reviewer := mainModel.Reviewers[m.ActiveReviewerIdx]
		reviewer.Completed = true
		logger.Println("Form completed for reviewer:", reviewer.Name, "idx:", m.ActiveReviewerIdx)

		// collect current evaluation field values into Results with suffix
		for _, g := range mainModel.Cfg.Evaluation {
			for _, fc := range g.Fields {
				k := fc.Key
				// check for keys specific to the current reviewer
				// add value to a reviewer specific result key
				outKey := BuildReviewerKey(k, *reviewer)
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

		// find next reviewer without a review
		next := mainModel.GetNextReviewerIdx()
		if next != -1 && next != m.ActiveReviewerIdx {
			logger.Println("Switching to next reviewer: ", next)

			m.ActiveReviewerIdx = next

			// initialize the new Form
			cmds = append(cmds, m.Forms[m.ActiveReviewerIdx].Init())
		} else {
			logger.Println("No more reviewers left without completed review.")
			// no more reviewers -> finish
			mainModel.State = STATE_SUMMARY
		}
	} else {
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (mainModel *Model) ViewEvaluation() (header string, body string, footer string) {

	s := mainModel.Styles
	m := mainModel.Evaluation

	reviewerName := mainModel.Reviewers[m.ActiveReviewerIdx].Name

	header += fmt.Sprintf("Review by %s - %d/%d", s.Highlight.Render(reviewerName), m.ActiveReviewerIdx, len(mainModel.Reviewers))
	if m.Forms[m.ActiveReviewerIdx] == nil {
		return header, "", ""
	}

	switch m.Forms[m.ActiveReviewerIdx].State {
	case huh.StateCompleted:
		// nothing
	default:

		v := strings.TrimSuffix(m.Form.View(), "\n\n")
		renderedForm := mainModel.Lg.NewStyle().Margin(1, 0).Render(v)

		errors := m.Form.Errors()
		if len(errors) > 0 {
			header = mainModel.appErrorBoundaryView(mainModel.errorView(m.Form))
		}

		const statusWidth = 28
		sb := strings.Builder{}
		if m.Results != nil {
			for k, v := range m.Results {
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

		statusMarginLeft := mainModel.width - statusWidth - lipgloss.Width(renderedForm) - s.Status.GetMarginRight()
		status := s.Status.
			Height(lipgloss.Height(renderedForm)).
			Width(statusWidth).
			MarginLeft(statusMarginLeft).
			Render(s.StatusHeader.Render("Ergebnis") + "\n\n" + sb.String())

		body = lipgloss.JoinHorizontal(lipgloss.Left, renderedForm, status)
		body = lipgloss.JoinVertical(lipgloss.Top, []string{body}...)

		footer = mainModel.appBoundaryView(m.Form.Help().ShortHelpView(m.Form.KeyBinds()))
		if len(errors) > 0 {
			footer = mainModel.appErrorBoundaryView("")
		}

		body = lipgloss.JoinHorizontal(lipgloss.Left, renderedForm)

		footer = mainModel.appBoundaryView(m.Form.Help().ShortHelpView(m.Form.KeyBinds()))
		if len(errors) > 0 {
			footer = mainModel.appErrorBoundaryView("")
		}
	}

	return header, body, footer
}
