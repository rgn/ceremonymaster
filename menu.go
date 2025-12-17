package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	STATE_MENU  = "menu"
	STATE_PRINT = "print"
)

type MenuState struct {
	Index   int
	Options []string
}

func (m *Model) InitMenuModel() {

	m.Menu = MenuState{
		Index:   0,
		Options: []string{"1) Zertifizierung starten...", "2) Zertifikat drucken", "3) Beenden"},
	}
}

func (m *Model) UpdateMenuModel(msg tea.Msg) []tea.Cmd {

	cmds := []tea.Cmd{}

	if m.State != STATE_MENU {
		return cmds
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Menu.Index > 0 {
				m.Menu.Index--
			}
		case "down", "j":
			if m.Menu.Index < len(m.Menu.Options)-1 {
				m.Menu.Index++
			}
		case "enter":
			switch m.Menu.Index {
			case 0:
				// Start data entry
				m.State = STATE_DATA_ENTRY
				if m.DataEntry.Form != nil {
					cmds = append(cmds, m.DataEntry.Form.Init())
					// clear screen when entering data entry
					cmds = append(cmds, tea.ClearScreen)
				}
			case 1:
				// Go to print view
				m.State = STATE_PRINT
				m.InitPrintModel()
				// clear screen when entering print view
				cmds = append(cmds, tea.ClearScreen)
			case 2:
				// Quit application
				cmds = append(cmds, tea.Quit)
			}
		}
	}

	return cmds
}

func (m Model) ViewMenu() (header string, body string, footer string) {
	s := m.Styles
	header = "Hauptmenü"

	var b strings.Builder

	fmt.Fprintf(&b, "\n")

	for i, opt := range m.Menu.Options {
		if i == m.Menu.Index {
			fmt.Fprintf(&b, "> %s\n", s.Highlight.Render(opt))
		} else {
			fmt.Fprintf(&b, "  %s\n", opt)
		}
	}

	body = b.String()
	footer = m.appBoundaryView("Navigiere mit Pfeiltasten, Bestätige mit Enter")
	return header, body, footer
}
