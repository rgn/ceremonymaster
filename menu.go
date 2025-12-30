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
	Index int
	Items []MenuItem
}

type MenuItem struct {
	Title       string
	TargetState string
	Hidden      bool
	Callback    func(m *Model) []tea.Cmd
}

func (m *Model) InitMenuModel() {

	m.Menu = MenuState{
		Index: 0,
		Items: []MenuItem{
			{Title: "Zertifizierung starten...", TargetState: STATE_DATA_ENTRY, Callback: func(m *Model) []tea.Cmd {
				cmds := []tea.Cmd{}
				m.InitDataEntryModel()
				if m.DataEntry.Form != nil {
					cmds = append(cmds, m.DataEntry.Form.Init())
					// clear screen when entering data entry
					cmds = append(cmds, tea.ClearScreen)
				}
				return cmds
			}},
			{Title: "Zertifikat drucken", TargetState: STATE_PRINT, Callback: func(m *Model) []tea.Cmd {
				// clear screen when entering print view
				m.InitPrintModel()
				return []tea.Cmd{tea.ClearScreen}
			}},
			{Title: "Wall of Fame", TargetState: STATE_WALL_OF_FAME, Callback: func(m *Model) []tea.Cmd {
				m.InitWallOfFameModel()
				return []tea.Cmd{tea.ClearScreen}
			}},
			{Title: "Wall of Fame aktualisieren", Hidden: false, Callback: func(m *Model) []tea.Cmd {
				m.UpdateWallOfFame()
				return []tea.Cmd{}
			}},
			{Title: "Beenden", Callback: func(m *Model) []tea.Cmd { return []tea.Cmd{tea.Quit} }},
		},
	}
}

func (m *Model) RegisterMenuItem(title string, hidden bool, callback func()) {
	m.Menu.Items = append(m.Menu.Items, MenuItem{Title: title})
}

func (m *Model) SetNextMenuItem(reversed bool) {
	n := len(m.Menu.Items)
	if n == 0 {
		return
	}
	start := m.Menu.Index
	if start < -1 || start >= n {
		start = -1
		if reversed {
			start = n
		}
	}
	for step := 1; step <= n; step++ {
		idx := (start + step) % n
		if reversed {
			idx = (start - step + n) % n
		}

		if !m.Menu.Items[idx].Hidden {
			m.Menu.Index = idx
			break
		}
	}
}

func (m *Model) UpdateMenuModel(msg tea.Msg) []tea.Cmd {

	cmds := []tea.Cmd{}

	if m.State != STATE_MENU {
		return cmds
	}

	logger.Println("Updating menu...")
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.SetNextMenuItem(true)
		case "down":
			m.SetNextMenuItem(false)
		case "enter":

			// guard against invalid index to avoid panic
			if len(m.Menu.Items) == 0 {
				return cmds
			}
			if m.Menu.Index < 0 || m.Menu.Index >= len(m.Menu.Items) {
				m.Menu.Index = 0
			}

			menuItem := m.Menu.Items[m.Menu.Index]

			if menuItem.Callback != nil {
				cmds = append(cmds, menuItem.Callback(m)...)
			}

			if menuItem.TargetState != "" {
				logger.Println("Switching from state", m.State, "to state:", menuItem.TargetState)
				logger.Println(len(m.PrintList), "certificates available for printing")
				m.State = menuItem.TargetState
			} else {
				m.Menu.Index = 0
				m.State = STATE_MENU
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

	for i, opt := range m.Menu.Items {

		if opt.Hidden {
			continue
		}

		if i == m.Menu.Index {
			fmt.Fprintf(&b, "> %s\n", s.Highlight.Render(opt.Title))
		} else {
			fmt.Fprintf(&b, "  %s\n", opt.Title)
		}
	}

	footer = s.ShortKeyStyle.Inline(true).Render("↑/↓") + " " + s.DescStyle.Inline(true).Render("Navigieren, ")
	footer += s.ShortKeyStyle.Inline(true).Render("Enter") + " " + s.DescStyle.Inline(true).Render("Auswählen, ")
	footer += s.ShortKeyStyle.Inline(true).Render("q") + " " + s.DescStyle.Inline(true).Render("Beenden")

	body = b.String()
	footer = m.appBoundaryView(footer)
	return header, body, footer
}
