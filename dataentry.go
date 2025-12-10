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

	groups := m.buildGroups(m.Cfg.DataCollection)

	m.DataEntry = DataEntryModel{
		Form: huh.NewForm(groups...).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(true),
	}
}

func (m *Model) UpdateDataEntryModel(msg tea.Msg) []tea.Cmd {

	cmds := []tea.Cmd{}

	if m.State != STATE_DATA_ENTRY {
		return cmds
	}

	Form, cmd := m.DataEntry.Form.Update(msg)

	if f, ok := Form.(*huh.Form); ok {
		m.DataEntry.Form = f
	}

	m.DataEntry.Reviewers = []string{}
	for _, v := range m.Cfg.DataCollection {
		groupKey := v.Key
		for _, fc := range v.Fields {
			fieldKey := BuildFieldKey(groupKey, fc.Key)
			if strings.HasPrefix(fieldKey, "reviewer_") {
				var reviewer = m.DataEntry.Form.GetString(fieldKey)
				if reviewer != "" {
					m.DataEntry.Reviewers = append(m.DataEntry.Reviewers, reviewer)
				}
			}
		}
	}

	sort.Strings(m.DataEntry.Reviewers)

	// If the Form just completed, collect results and transition to
	// the review State while initializing the evaluation Form.
	if m.DataEntry.Form.State == huh.StateCompleted {

		m.applicantName = m.DataEntry.Form.GetString("data_entry_applicant_name")
		m.objectName = m.DataEntry.Form.GetString("data_entry_object_description")
		m.objectImage = m.DataEntry.Form.GetString("data_entry_object_image")

		// Transition to evaluation and initialize the evaluation Form.
		m.State = "evaluation"
		// Do not append the Form's quit command to avoid exiting the app.
	} else {
		// Only append the Form cmd while it is still active.
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (m *Model) ViewDataEntry() (header string, body string, footer string) {
	s := m.Styles

	header = "Datenerfassung"

	switch m.DataEntry.Form.State {
	case huh.StateCompleted:
		var b strings.Builder
		body = s.Status.Margin(0, 1).Padding(1, 2).Width(48).Render(b.String()) + "\n\n"
	default:
		// Form (left side)
		v := strings.TrimSuffix(m.DataEntry.Form.View(), "\n\n")
		renderedForm := m.Lg.NewStyle().Margin(1, 0).Render(v)

		errors := m.DataEntry.Form.Errors()
		if len(errors) > 0 {
			header = m.appErrorBoundaryView(m.errorView(m.DataEntry.Form))
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

			if m.DataEntry.Form.GetString("data_entry_applicant_name") != "" {
				applicantName := m.DataEntry.Form.GetString("data_entry_applicant_name")
				buildInfo = m.Styles.Highlight.Render(applicantName)
			}

			if m.DataEntry.Form.GetString("data_entry_object_description") != "" {
				objectDescription := m.DataEntry.Form.GetString("data_entry_object_description")
				buildInfo += fmt.Sprintf(" beantragt die Zertifizierung von %s", m.Styles.Highlight.Render(objectDescription))
			}

			if m.DataEntry.Form.GetString("data_entry_object_class") != "" {
				objectClass := m.DataEntry.Form.GetString("data_entry_object_class")
				buildInfo += fmt.Sprintf(" (Klasse: %s)", m.Styles.Highlight.Render(objectClass))
			}

			if objectDescription != "" || objectClass != "" {
				buildInfo += "."
			}

			if len(m.DataEntry.Reviewers) > 0 {
				jobDescription += "\n\nBegutachtet durch:\n\t- " + strings.Join(m.DataEntry.Reviewers, "\n\t- ")
			}

			const statusWidth = 28
			statusMarginLeft := m.width - statusWidth - lipgloss.Width(renderedForm) - s.Status.GetMarginRight()
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

		footer = m.appBoundaryView(m.DataEntry.Form.Help().ShortHelpView(m.DataEntry.Form.KeyBinds()))
		if len(errors) > 0 {
			footer = m.appErrorBoundaryView("")
		}
	}

	return header, body, footer
}
