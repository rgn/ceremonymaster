package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxWidth         = 80
	STATE_DATA_ENTRY = "data_entry"
	STATE_EVALUATION = "evaluation"
	STATE_SUMMARY    = "summary"
	STATE_DONE       = "done"
)

var (
	red    = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	indigo = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
)

var (
	logout *log.Logger
)

type Model struct {
	State  string
	Lg     *lipgloss.Renderer
	Styles *Styles
	width  int

	applicantName string
	objectName    string
	objectImage   string

	Cfg        Configuration
	DataEntry  DataEntryModel
	Evaluation EvaluationModel
	Summary    SummaryModel

	// Values holds pointers to the backing variables for each field keyed by field key.
	Values map[string]any
}

func (m Model) GetString(key string) string {
	v, ok := m.Values[key].(string)
	if !ok {
		return ""
	}
	return v
}

func BuildFieldKey(groupKey, fieldKey string) string {
	return fmt.Sprintf("%s_%s", groupKey, fieldKey)
}

func NewModel(cfg Configuration) Model {

	m := Model{
		Cfg:    cfg,
		Lg:     lipgloss.DefaultRenderer(),
		State:  STATE_DATA_ENTRY,
		Values: make(map[string]any),
		width:  maxWidth,
	}

	m.Styles = NewStyles(m.Lg)

	m.InitDataEntryModel()
	m.InitEvaluationModel()
	m.InitSummaryModel()

	return m
}

// buildGroups constructs huh.Groups from GroupConfig entries. If reviewers
// are provided and a group's name is "Wertung" it will expand that group into
// one per reviewer, suffixing keys with the reviewer key to avoid collisions.
func (m *Model) buildGroups(groupCfgs []GroupConfig) []*huh.Group {
	var res []*huh.Group

	for _, gcfg := range groupCfgs {
		groupKey := gcfg.Key
		// Default handling for non-Wertung groups (or Wertung when no reviewers)
		var fields []huh.Field
		for _, fc := range gcfg.Fields {
			fcKey := BuildFieldKey(groupKey, fc.Key)
			switch fc.Type {
			case "range":
				v := ""
				m.Values[fc.Key] = &v
				sel := huh.NewSelect[string]().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				sel = sel.Options(
					huh.NewOption[string]("", "0"),
					huh.NewOption[string]("⭐", "1"),
					huh.NewOption[string]("⭐⭐", "2"),
					huh.NewOption[string]("⭐⭐⭐", "3"),
					huh.NewOption[string]("⭐⭐⭐⭐", "4"),
					huh.NewOption[string]("⭐⭐⭐⭐⭐", "5"),
				)
				if fc.Mandatory {
					sel = sel.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, sel)
			case "input":
				v := ""
				m.Values[fc.Key] = &v
				inp := huh.NewInput().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				if fc.Mandatory {
					inp = inp.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, inp)
			case "select":
				v := ""
				m.Values[fc.Key] = &v
				sel := huh.NewSelect[string]().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				if len(fc.Options) > 0 {
					sel = sel.Options(huh.NewOptions[string](fc.Options...)...)
				}
				if fc.Mandatory {
					sel = sel.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, sel)
			case "text":
				v := ""
				m.Values[fc.Key] = &v
				txt := huh.NewText().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				if fc.Mandatory {
					txt = txt.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, txt)
			case "filepicker":
				v := ""
				m.Values[fc.Key] = &v
				fp := huh.NewFilePicker().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description).
					AllowedTypes(fc.Options)
				if fc.Mandatory {
					fp = fp.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, fp)
			case "confirm":
				b := false
				m.Values[fc.Key] = &b
				conf := huh.NewConfirm().
					Key(fc.Key).
					Value(&b).
					Title(fc.Title)
				if fc.Description != "" {
					conf = conf.Description(fc.Description)
				}
				if fc.Affirmative != "" {
					conf = conf.Affirmative(fc.Affirmative)
				}
				if fc.Negative != "" {
					conf = conf.Negative(fc.Negative)
				}
				if fc.RequireYes {
					conf = conf.Validate(func(v bool) error {
						if !v {
							return fmt.Errorf("Welp, finish up then")
						}
						return nil
					})
				}
				fields = append(fields, conf)
			case "multiselect":
				var vs []string
				m.Values[fc.Key] = &vs
				ms := huh.NewMultiSelect[string]().
					Key(fcKey).
					Value(&vs).
					Title(fc.Title).
					Description(fc.Description)
				if len(fc.Options) > 0 {
					ms = ms.Options(huh.NewOptions[string](fc.Options...)...)
				}
				if fc.Mandatory {
					ms = ms.Validate(func(s []string) error {
						if len(s) == 0 {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, ms)
			default:
				// unknown field type: ignore
			}
		}

		group := huh.NewGroup(fields...).
			Title(gcfg.Title).
			Description(gcfg.Description + "\n")

		res = append(res, group)
	}

	return res
}

func (m Model) Init() tea.Cmd {
	// Initialize the Form matching the current State so its internal
	// components are prepared before the first Update/View cycle.
	if m.State == "evaluation" && !m.Evaluation.FormInitialized {
		return m.Evaluation.Form.Init()
	}

	if m.DataEntry.Form != nil {
		return m.DataEntry.Form.Init()
	}

	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.Styles.Base.GetHorizontalFrameSize()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "esc", "q":
			if m.State == STATE_SUMMARY {
				m.CreateCertificate()
			}
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	cmds = append(cmds, m.UpdateDataEntryModel(msg)...)
	cmds = append(cmds, m.UpdateEvaluationModel(msg)...)
	cmds = append(cmds, m.UpdateSummaryModel(msg)...)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	s := m.Styles

	var (
		currentHeader = "Ceremony Master"
		header        string
		body          string
		footer        string
	)

	if m.State == STATE_DATA_ENTRY {
		header, body, footer = m.ViewDataEntry()
	}

	if m.State == STATE_EVALUATION {
		header, body, footer = m.ViewEvaluation()
	}

	if m.State == STATE_SUMMARY {
		header, body, footer = m.ViewSummary()
	}

	if len(header) > 0 {
		currentHeader = "Ceremony Master - " + header
	}
	return s.Base.Render(m.appBoundaryView(currentHeader) + "\n" + body + "\n\n" + footer)
}

func (m Model) errorView(Form *huh.Form) string {
	var s string
	for _, err := range Form.Errors() {
		s += err.Error()
	}
	return s
}

func (m Model) appBoundaryView(text string) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		m.Styles.HeaderText.Render(text),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceForeground(indigo),
	)
}

func (m Model) appErrorBoundaryView(text string) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		m.Styles.ErrorHeaderText.Render(text),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceForeground(red),
	)
}

func check(e error) {
	if e != nil {
		fmt.Println("Fatal error: ", e)
	}
}

func main() {

	cfg, cleanUpCallback := initApplication()

	defer cleanUpCallback()

	if _, err := tea.NewProgram(NewModel(cfg)).Run(); err != nil {
		logger.Printf("application error: %v", err)

		os.Exit(1)
	}
}
