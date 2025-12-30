package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type DataEntryModel struct {
	Form      *huh.Form
	Reviewers []string
}

func (m *Model) InitDataEntryModel() {
	logger.Println("Init print model")

	groups := m.buildGroups(m.Cfg.DataCollection)

	m.DataEntry = DataEntryModel{
		Form: huh.NewForm(groups...).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(true),
	}
}

func (mainModel *Model) UpdateDataEntryModel(msg tea.Msg) []tea.Cmd {

	m := &mainModel.DataEntry

	cmds := []tea.Cmd{}

	if mainModel.State != STATE_DATA_ENTRY {
		return cmds
	}

	Form, cmd := mainModel.DataEntry.Form.Update(msg)

	if f, ok := Form.(*huh.Form); ok {
		mainModel.DataEntry.Form = f
	}

	m.Reviewers = []string{}
	for _, v := range mainModel.Cfg.DataCollection {
		groupKey := v.Key
		for _, fc := range v.Fields {
			fieldKey := BuildFieldKey(groupKey, fc.Key)
			if strings.HasPrefix(fieldKey, "reviewer_") {
				var reviewer = m.Form.GetString(fieldKey)
				if reviewer != "" {
					m.Reviewers = append(m.Reviewers, reviewer)
				}
			}
		}
	}

	sort.Strings(m.Reviewers)

	// If the Form just completed, collect results and transition to
	// the review State while initializing the evaluation Form.
	if m.Form.State == huh.StateCompleted {

		mainModel.applicantName = m.Form.GetString("data_entry_applicant_name")
		mainModel.objectName = m.Form.GetString("data_entry_object_description")
		mainModel.objectClass = m.Form.GetString("data_entry_object_class")
		mainModel.objectImage = m.Form.GetString("data_entry_object_image")
		mainModel.BuildReviewers()
		mainModel.BuildReviewerForms()

		// Transition to evaluation and initialize the evaluation Form.
		mainModel.State = "evaluation"
		// Do not append the Form's quit command to avoid exiting the app.
	} else {
		// Only append the Form cmd while it is still active.
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (mainModel *Model) ViewDataEntry() (header string, body string, footer string) {

	s := mainModel.Styles
	m := mainModel.DataEntry

	header = "Datenerfassung"

	switch m.Form.State {
	case huh.StateCompleted:
		var b strings.Builder
		body = s.Status.Margin(0, 1).Padding(1, 2).Width(48).Render(b.String()) + "\n\n"
	default:
		// Form (left side)
		v := strings.TrimSuffix(m.Form.View(), "\n\n")
		renderedForm := mainModel.Lg.NewStyle().Margin(1, 0).Render(v)

		errors := m.Form.Errors()
		if len(errors) > 0 {
			header = mainModel.appErrorBoundaryView(mainModel.errorView(m.Form))
		}

		// Status (right side)
		var status string
		{
			var (
				objectDescription string
				objectClass       string
				buildInfo         = "(None)"
				jobDescription    string
			)

			if m.Form.GetString("data_entry_applicant_name") != "" {
				applicantName := m.Form.GetString("data_entry_applicant_name")
				buildInfo = s.Highlight.Render(applicantName)
			}

			if m.Form.GetString("data_entry_object_description") != "" {
				objectDescription := m.Form.GetString("data_entry_object_description")
				buildInfo += fmt.Sprintf(" beantragt die Zertifizierung von %s", s.Highlight.Render(objectDescription))
			}

			if m.Form.GetString("data_entry_object_class") != "" {
				objectClass := m.Form.GetString("data_entry_object_class")
				buildInfo += fmt.Sprintf(" (Klasse: %s)", s.Highlight.Render(objectClass))
			}

			if objectDescription != "" || objectClass != "" {
				buildInfo += "."
			}

			if len(m.Reviewers) > 0 {
				jobDescription += "\n\nBegutachtet durch:\n\t- " + strings.Join(m.Reviewers, "\n\t- ")
			}

			const statusWidth = 28
			statusMarginLeft := mainModel.width - statusWidth - lipgloss.Width(renderedForm) - s.Status.GetMarginRight()
			status = s.Status.
				Height(lipgloss.Height(renderedForm)).
				Width(statusWidth).
				MarginLeft(statusMarginLeft).
				Render(s.StatusHeader.Render("Zertifizierungsantrag") + "\n\n" +
					buildInfo +
					jobDescription)
		}

		body = lipgloss.JoinHorizontal(lipgloss.Left, renderedForm, status)
		body = lipgloss.JoinVertical(lipgloss.Top, []string{body}...)

		footer = mainModel.appBoundaryView(m.Form.Help().ShortHelpView(m.Form.KeyBinds()))
		if len(errors) > 0 {
			footer = mainModel.appErrorBoundaryView("")
		}
	}

	return header, body, footer
}
