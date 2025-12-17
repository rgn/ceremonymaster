package main

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

const (
	MIN = "min"
	MAX = "max"
	AVG = "avg"
	SUM = "sum"
)

type SummaryModel struct {
	Rank        string
	AvgTotal    float32
	SumTotal    float32
	Table       table.Model
	FieldTitles map[string]string
	// Summaries holds computed aggregates (min/max/avg/count) per evaluation field.
	Summaries map[string]map[string]float32
}

func (m *Model) InitSummaryModel() {

	columns := []table.Column{
		{Title: "Bewertungsparameter", Width: 30},
		{Title: "Ø", Width: 8},
		{Title: "Min", Width: 8},
		{Title: "Max", Width: 8},
	}

	fieldTitles := make(map[string]string)

	for _, g := range m.Cfg.Evaluation {
		for _, f := range g.Fields {
			fieldTitles[f.Key] = f.Title
		}
	}

	m.Summary = SummaryModel{
		AvgTotal:    0.0,
		SumTotal:    0.0,
		FieldTitles: fieldTitles,
		Table: table.New(
			table.WithColumns(columns),
			table.WithFocused(true),
			table.WithHeight(len(m.Cfg.Evaluation)+1),
		),
		Summaries: make(map[string]map[string]float32),
	}
}

type sumEnvelope struct {
	key    string
	weigth float32
}

// summarizeEvaluations computes min, max and weighted average for each
// evaluation field across all reviewers. Results are stored in
// `m.Summaries[fieldKey]` with keys "min","max","avg","count".
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

		m.Summary.Summaries[groupKey] = map[string]float32{
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

	m.summarizeEvaluations()

	var AvgTotal float32 = 0.0
	rows := []table.Row{}

	for groupTitle, summary := range m.Summary.Summaries {
		// title, ok := m.Summary.FieldTitles[fieldKey]
		// if !ok || title == "" {
		// 	// fallback: use the key itself or a placeholder so UI isn't blank
		// 	title = fieldKey
		// }

		AvgTotal += summary[AVG]

		rows = append(rows, table.Row{
			groupTitle,
			fmt.Sprintf("%.2f", summary[AVG]),
			fmt.Sprintf("%.0f", summary[MIN]),
			fmt.Sprintf("%.0f", summary[MAX]),
		})
	}

	m.Summary.Table.SetRows(rows)

	Rank := ""
	avgResult := AvgTotal / float32(len(m.Cfg.Evaluation))
	for _, level := range m.Cfg.SkillLevels {
		if avgResult >= level.MinPoints {
			Rank = level.Name
		}
	}
	m.Summary.Rank = Rank
	m.Summary.AvgTotal = avgResult

	return cmds
}

func (m *Model) ViewSummary() (header string, body string, footer string) {

	s := m.Styles
	header = "Zusammenfassung"

	var b strings.Builder

	fmt.Fprintf(&b, "\nDeine Bewertungen für %s ist %s\n", s.Highlight.Render(m.objectName), s.Highlight.Render(fmt.Sprintf("%.2f", m.Summary.AvgTotal)))
	fmt.Fprintf(&b, "\nHerzlichen Glückwunsch %s zum %s\n", s.Highlight.Render(m.applicantName), s.Highlight.Render(m.Summary.Rank))

	b.WriteString(s.Base.Render(m.Summary.Table.View()))

	body = b.String() + "\n\n"
	footer = m.appBoundaryView("Drücken Sie 'q' oder 'Esc', um die Anwendung zu beenden.")

	return header, body, footer
}

func (m *Model) CreateCertificate() {

	if m.State != STATE_SUMMARY {
		return
	}

	certificate := Certificate{
		ID:         uuid.New(),
		Date:       time.Now(),
		Applicant:  m.applicantName,
		ObjectName: m.objectName,
		Reviewers:  make([]string, 0),
		Questions:  make([]CertificateQuestion, 0),
	}

	for _, reviewer := range m.Evaluation.Reviewers {
		reviewerName := m.DataEntry.Form.GetString(reviewer.key)
		if strings.TrimSpace(reviewerName) == "" {
			reviewerName = reviewer.key
		}
		certificate.Reviewers = append(certificate.Reviewers, reviewerName)
	}

	formKeysGrouped := make(map[string][]string)
	for _, g := range m.Cfg.Evaluation {
		groupKey := g.Key
		formKeysGrouped[g.Title] = []string{}
		for _, fc := range g.Fields {
			fieldKey := BuildFieldKey(groupKey, fc.Key)
			var appendFieldKey bool = false
			if fc.Type == "range" && strings.HasSuffix(fieldKey, "_rating") {
				appendFieldKey = true
			} else if fc.Type == "text" && strings.HasSuffix(fieldKey, "_comment") {
				appendFieldKey = true
			}

			if appendFieldKey {
				formKeysGrouped[g.Title] = append(formKeysGrouped[g.Title], fieldKey)
			}
		}
	}

	for groupTitle, formKeyGroup := range formKeysGrouped {

		if len(formKeyGroup) != 2 {
			panic("expected rating & comment field per group")
		}

		certificateQuestion := CertificateQuestion{
			Question:  groupTitle,
			Responses: []CertificateResponse{},
		}

		var fcRatingKey, fcCommentKey string
		for _, fk := range formKeyGroup {
			if strings.HasSuffix(fk, "_rating") {
				fcRatingKey = fk
			} else if strings.HasSuffix(fk, "_comment") {
				fcCommentKey = fk
			}
		}

		for reviewerIdx, form := range m.Evaluation.Forms {

			reviewerName := m.getReviewerName(reviewerIdx)
			commentVal := form.GetString(fcCommentKey)
			ratingVal := 0
			if rv, err := strconv.Atoi(form.GetString(fcRatingKey)); err == nil {
				ratingVal = rv
			}

			response := CertificateResponse{
				Name:    reviewerName,
				Value:   ratingVal,
				Comment: commentVal,
			}

			certificateQuestion.Responses = append(certificateQuestion.Responses, response)
		}

		certificate.Questions = append(certificate.Questions, certificateQuestion)
	}

	currentPath := path.Join(getCertificatesPath(), certificate.Date.Format("2006"), certificate.Date.Format("01"))
	currentCertificatePath := path.Join(currentPath, certificate.ID.String()+".yaml")

	os.MkdirAll(currentPath, os.ModePerm)

	saveCertificate(currentCertificatePath, certificate)
}
