package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// CertificateSummary is a lightweight view of a certificate used for listing.
type CertificateSummary struct {
	Path       string
	Name       string
	Date       time.Time
	Applicant  string
	ObjectName string
}

// findLatestCertificates scans the certificates directory under the application
// certificates path and returns the most recent `limit` certificates sorted
// descending by date where possible. If the YAML contains a `date` field it
// will be used; otherwise the file mod time is used.
func findLatestCertificates(limit int) ([]CertificateSummary, error) {
	base := getCertificatesPath()
	var summaries []CertificateSummary

	_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(p) != ".yaml" {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}

		// Try to unmarshal date/applicant/object_name from YAML into small struct
		var meta struct {
			Date       time.Time `yaml:"date"`
			Applicant  string    `yaml:"applicant"`
			ObjectName string    `yaml:"object_name"`
		}
		var usedDate time.Time
		if err := yaml.Unmarshal(data, &meta); err == nil {
			usedDate = meta.Date
		}

		if usedDate.IsZero() {
			if fi, err := os.Stat(p); err == nil {
				usedDate = fi.ModTime()
			}
		}

		name := filepath.Base(p)
		summaries = append(summaries, CertificateSummary{
			Path:       p,
			Name:       name,
			Date:       usedDate,
			Applicant:  meta.Applicant,
			ObjectName: meta.ObjectName,
		})

		return nil
	})

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Date.After(summaries[j].Date)
	})

	if len(summaries) > limit {
		summaries = summaries[:limit]
	}
	return summaries, nil
}

func (m *Model) InitPrintModel() {
	list, err := findLatestCertificates(5)
	if err != nil {
		logger.Printf("Failed to load certificate list: %v", err)
		m.PrintList = []CertificateSummary{}
		return
	}
	m.PrintList = list
	if m.PrintIndex >= len(m.PrintList) {
		m.PrintIndex = 0
	}
}

func (m *Model) UpdatePrintModel(msg tea.Msg) []tea.Cmd {
	cmds := []tea.Cmd{}

	if m.State != STATE_PRINT {
		return cmds
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.State = STATE_MENU
			cmds = append(cmds, tea.ClearScreen)
		case "up", "k":
			if m.PrintIndex > 0 {
				m.PrintIndex--
			}
		case "down", "j":
			if m.PrintIndex < len(m.PrintList)-1 {
				m.PrintIndex++
			}
		case "enter":
			// only trigger generation when the user presses Enter while already
			// focused in the print view. If we just transitioned into the print
			// view (PrevState != STATE_PRINT), ignore the Enter that opened the
			// view from the menu.
			if m.PrevState != STATE_PRINT {
				break
			}
			if len(m.PrintList) == 0 {
				break
			}
			sel := m.PrintList[m.PrintIndex]

			cert, err := loadCertificate(sel.Path)
			if err != nil {
				logger.Printf("Failed to load certificate %s: %v", sel.Path, err)
				break
			}

			// use the YAML filename (without extension) as the output base name
			outputBase := strings.TrimSuffix(sel.Name, filepath.Ext(sel.Name))
			out, err := GenerateCertificatePDF(cert, filepath.Dir(sel.Path), outputBase, m.Cfg.SkillLevels)
			if err != nil {
				logger.Printf("Failed to generate certificate PDF/HTML: %v", err)
				break
			}

			logger.Printf("Generated certificate output: %s", out)

			if err := openFile(out); err != nil {
				logger.Printf("Failed to open generated file: %v", err)
			}
		}
	}

	return cmds
}

func (m *Model) ViewPrint() (string, string, string) {
	//s := m.Styles

	header := "Zertifikat drucken"

	if len(m.PrintList) == 0 {
		body := "Keine Zertifikate gefunden."
		footer := m.appBoundaryView("Drücken Sie 'esc' oder 'q' zum Zurückkehren")
		return header, body, footer
	}

	var b strings.Builder

	fmt.Fprintf(&b, "\n")

	for i, s := range m.PrintList {
		marker := "  "
		if i == m.PrintIndex {
			marker = "> "
		}
		label := s.Name
		if s.Applicant != "" || s.ObjectName != "" {
			label = fmt.Sprintf("%s - %s (%s)", s.Date.Format("2006-01-02"), s.Applicant, s.ObjectName)
		} else {
			label = fmt.Sprintf("%s - %s", s.Date.Format("2006-01-02"), s.Name)
		}
		fmt.Fprintf(&b, "%s%s\n", marker, label)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, []string{b.String()}...)

	footer := m.appBoundaryView("Navigiere mit Pfeiltasten, Bestätige mit Enter")

	return header, body, footer
}

// openFile opens the provided path with the OS default handler.
func openFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
