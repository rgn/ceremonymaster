package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxWidth         = 120
	maxHeight        = 20
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
	State     string
	PrevState string
	Lg        *lipgloss.Renderer
	Styles    *Styles
	width     int
	height    int

	applicantName      string
	objectName         string
	objectClass        string
	objectImage        string
	SortedReviewerKeys []int
	Reviewers          map[int]*Reviewer

	Cfg        Configuration
	WallOfFame WallOfFameModel
	DataEntry  DataEntryModel
	Evaluation EvaluationModel
	Summary    SummaryModel

	// Values holds pointers to the backing variables for each field keyed by field key.
	Values map[string]any
	// Menu state is embedded (defined in menu.go)
	Menu MenuState
	// Print view
	PrintIndex int
	PrintList  []CertificateSummary
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

func InitModel(cfg Configuration) Model {

	m := Model{
		Cfg:                cfg,
		Lg:                 lipgloss.DefaultRenderer(),
		State:              STATE_MENU,
		SortedReviewerKeys: []int{},
		Reviewers:          make(map[int]*Reviewer),
		Values:             make(map[string]any),
		width:              maxWidth,
		height:             maxHeight,
	}

	m.Styles = NewStyles(m.Lg)

	m.InitMenuModel()
	m.InitWallOfFameModel()
	m.InitDataEntryModel()
	m.InitEvaluationModel()
	m.InitSummaryModel()

	return m
}

func (m Model) Init() tea.Cmd {

	// Initialize the Form matching the current State so its internal
	if m.State == STATE_EVALUATION {
		return m.Evaluation.Form.Init()
	}

	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// record previous state for updaters to make entry/transition decisions
	prev := m.State
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.Styles.Base.GetHorizontalFrameSize()
		m.height = min(msg.Height, maxHeight) - m.Styles.Base.GetVerticalFrameSize()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "esc":
			m.State = STATE_MENU
			return m, tea.ClearScreen
		case "q":
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	// persist prev state into the model so the updaters can use it
	m.PrevState = prev

	// Dispatch to state-specific updaters
	cmds = append(cmds, m.UpdateMenuModel(msg)...)
	cmds = append(cmds, m.UpdateDataEntryModel(msg)...)
	cmds = append(cmds, m.UpdatePrintModel(msg)...)
	cmds = append(cmds, m.UpdateEvaluationModel(msg)...)
	cmds = append(cmds, m.UpdateSummaryModel(msg)...)
	cmds = append(cmds, m.UpdateWallOfFameModel(msg)...)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	s := m.Styles

	var (
		currentHeader = "Ceremony Master"
		header        string
		body          string
		footer        string
		rightPane     string
	)

	if m.State == STATE_MENU {
		header, body, footer = m.ViewMenu()
	}

	if m.State == STATE_DATA_ENTRY {
		header, body, footer = m.ViewDataEntry()
	}

	if m.State == STATE_EVALUATION {
		header, body, footer = m.ViewEvaluation()
	}

	if m.State == STATE_SUMMARY {
		header, body, footer = m.ViewSummary()
	}

	if m.State == STATE_PRINT {
		header, body, footer = m.ViewPrint()
	}

	if m.State == STATE_WALL_OF_FAME {
		header, body, footer = m.ViewWallOfFame()
	}

	if len(header) > 0 {
		currentHeader = "Ceremony Master - " + header
	}

	leftWidth := lipgloss.Width(body)
	rightWidth := lipgloss.Width(rightPane)
	avail := m.width - leftWidth
	if avail < rightWidth {
		avail = rightWidth
	}
	rightAligned := lipgloss.PlaceHorizontal(avail, lipgloss.Right, rightPane)
	body = lipgloss.JoinHorizontal(lipgloss.Left, body, rightAligned)

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

	if _, err := tea.NewProgram(InitModel(cfg)).Run(); err != nil {
		logger.Printf("application error: %v", err)

		os.Exit(1)
	}
}
