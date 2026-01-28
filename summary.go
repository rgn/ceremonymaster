package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	MIN = "min"
	MAX = "max"
	AVG = "avg"
	SUM = "sum"
)

type SummaryModel struct {
	level        string
	avt_total    float32
	sum_total    float32
	table        table.Model
	field_titles map[string]string
	// summaries holds computed aggregates (min/max/avg/count) per evaluation field.
	summaries    map[string]map[string]float32
	overall_rank int
	class_rank   int
	Certificate  *Certificate
}

func (m *Model) InitSummaryModel() {

	columns := []table.Column{
		{Title: "Bewertungsparameter", Width: 30},
		{Title: "Ø", Width: 8},
		{Title: "Min", Width: 8},
		{Title: "Max", Width: 8},
	}

	field_titles := make(map[string]string)

	for _, g := range m.Cfg.Evaluation {
		for _, f := range g.Fields {
			field_titles[f.Key] = f.Title
		}
	}

	m.Summary = SummaryModel{
		avt_total:    0.0,
		sum_total:    0.0,
		overall_rank: -1,
		class_rank:   -1,
		field_titles: field_titles,
		table: table.New(
			table.WithColumns(columns),
			table.WithFocused(true),
			table.WithHeight(len(m.Cfg.Evaluation)+1),
		),
		summaries: make(map[string]map[string]float32),
	}
}

type sumEnvelope struct {
	key    string
	weigth float32
}

// summarizeEvaluations computes min, max and weighted average for each
// evaluation field across all reviewers. Results are stored in
// `m.summaries[fieldKey]` with keys "min","max","avg","count".
func (m *Model) summarizeEvaluations() {

	formKeysGrouped := make(map[string][]sumEnvelope)

	for _, g := range m.Cfg.Evaluation {
		groupKey := g.Key
		formKeysGrouped[g.Title] = []sumEnvelope{}
		for _, fc := range g.Fields {
			sumEnv := sumEnvelope{
				key:    BuildFieldKey(groupKey, fc.Key),
				weigth: fc.Weight,
			}
			formKeysGrouped[g.Title] = append(formKeysGrouped[g.Title], sumEnv)
		}
	}

	for k, fromKeyGroup := range formKeysGrouped {

		groupKey := k
		var minVal float32 = math.MaxFloat32
		var maxVal float32 = -math.MaxFloat32
		var sumVal float32 = 0

		for _, form := range m.Evaluation.Forms {
			for _, k := range fromKeyGroup {
				if strings.HasSuffix(k.key, "_rating") {
					val := form.GetString(k.key)
					iVal, _ := strconv.Atoi(val)
					iValWeighted := float32(iVal) * k.weigth

					minVal = minf(minVal, iValWeighted)
					maxVal = maxf(maxVal, iValWeighted)
					sumVal += iValWeighted
				}
			}
		}

		m.Summary.summaries[groupKey] = map[string]float32{
			MIN: float32(minVal),
			MAX: float32(maxVal),
			AVG: float32(sumVal) / float32(len(m.Evaluation.Forms)),
			SUM: float32(sumVal),
		}
	}
}

func (m *Model) UpdateSummaryModel(msg tea.Msg) []tea.Cmd {

	cmds := []tea.Cmd{}

	if m.State != STATE_SUMMARY {
		return cmds
	}

	logger.Println("Update summary")

	rows := []table.Row{}

	if m.Summary.Certificate == nil {

		m.Summary.Certificate = m.CreateCertificate()
		summary := m.SummarizeCertificate(*m.Summary.Certificate)

		for title, summaryEntry := range summary.entries {
			rows = append(rows, table.Row{
				title,
				fmt.Sprintf("%.2f", summaryEntry.avg),
				fmt.Sprintf("%.0f", summaryEntry.min),
				fmt.Sprintf("%.0f", summaryEntry.max),
			})
		}

		m.Summary.table.SetRows(rows)
		m.Summary.level = summary.level
		m.Summary.avt_total = summary.avg
		m.Summary.overall_rank, m.Summary.class_rank = summary.GetRanks()

		m.SaveCertificateByConvention(*m.Summary.Certificate)
		m.UpdateWallOfFame()
	}

	return cmds
}

func (m *Model) ViewSummary() (header string, body string, footer string) {

	s := m.Styles
	header = "Zusammenfassung"

	var b strings.Builder

	fmt.Fprintf(&b, "\nDeine Bewertungen für %s ist %s\n", s.Highlight.Render(m.objectName), s.Highlight.Render(fmt.Sprintf("%.2f", m.Summary.avt_total)))
	fmt.Fprintf(&b, "\nHerzlichen Glückwunsch %s zum %s\n", s.Highlight.Render(m.applicantName), s.Highlight.Render(m.Summary.level))
	fmt.Fprintf(&b, "\nDu erreich damit den Rang %d in der Bestenliste und Rang %d für %s.\n", m.Summary.overall_rank, m.Summary.class_rank, m.objectClass)

	b.WriteString(s.Base.Render(m.Summary.table.View()))

	body = b.String() + "\n\n"
	footer = m.appBoundaryView("Drücken Sie 'q' oder 'Esc', um die Anwendung zu beenden.")

	return header, body, footer
}
