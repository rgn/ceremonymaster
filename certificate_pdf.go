package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// GenerateCertificatePDF renders the certificate using an HTML template located
// in the application's templates directory and then converts it to PDF.
// The template is editable by the user at templates/certificate.html.
// It prefers to use `wkhtmltopdf` if installed; otherwise it writes the HTML
// next to the YAML so users can manually convert.
func GenerateCertificatePDF(cert Certificate, basePath string, outputBaseName string, skillLevels []SkillLevelConfig) (string, error) {
	var tplPath = filepath.Join(getAppBasePath(), "templates", "certificate.html")
	localTplPath := filepath.Join(getTemplatesPath(), "certificate.html")

	logger.Println("Local certificate template at: ", localTplPath)
	if _, err := os.Stat(localTplPath); err == nil {
		tplPath = localTplPath
	}

	logger.Println("Using certificate template at: ", tplPath)

	funcMap := template.FuncMap{
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
		"substr": func(s string, start, length int) string {
			if start < 0 || start >= len(s) || length <= 0 {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"stars": func(n int) string {
			if n <= 0 {
				return ""
			}
			return strings.Repeat("⭐", n)
		},
		"initials": func(s string) string {
			s = strings.TrimSpace(s)
			if s == "" {
				return ""
			}
			// collect up to two runes
			var runes []rune
			for _, r := range s {
				if len(runes) >= 2 {
					break
				}
				if unicode.IsSpace(r) {
					continue
				}
				runes = append(runes, unicode.ToUpper(r))
			}
			return string(runes)
		},
	}

	tpl, err := template.New(filepath.Base(tplPath)).Funcs(funcMap).ParseFiles(tplPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", tplPath, err)
	}

	// determine output base name
	var name string
	if outputBaseName != "" {
		name = outputBaseName
	} else {
		name = cert.ID.String()
	}

	// prepare per-question summaries (avg, min, max)
	type questionSummary struct {
		Question string
		Avg      float64
		Min      int
		Max      int
		Count    int
	}

	var summaries []questionSummary
	overallSum := 0
	overallCount := 0
	for _, q := range cert.Questions {
		min := 1 << 30
		max := -1 << 30
		sum := 0
		count := 0
		for _, r := range q.Responses {
			v := r.Value
			sum += v
			count++
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		overallSum += sum
		overallCount += count
		if count == 0 {
			min = 0
			max = 0
		}
		avg := 0.0
		if count > 0 {
			avg = float64(sum) / float64(count)
		}
		summaries = append(summaries, questionSummary{Question: q.Question, Avg: avg, Min: min, Max: max, Count: count})
	}

	overallAvg := 0.0
	if overallCount > 0 {
		overallAvg = float64(overallSum) / float64(overallCount)
	}

	// determine rank from configured skill levels (mirrors summary.go logic)
	rank := ""
	// skillLevels may be empty; in that case leave rank empty
	for _, level := range skillLevels {
		if overallAvg >= float64(level.MinPoints) {
			rank = level.Name
		}
	}

	// prepare template data with optional ImageFile, summaries, overall average and rank
	data := struct {
		Certificate
		ImageFile  string
		Summaries  []questionSummary
		OverallAvg float64
		Rank       string
	}{
		Certificate: cert,
		ImageFile:   "",
		Summaries:   summaries,
		OverallAvg:  overallAvg,
		Rank:        rank,
	}

	// if a PNG with the same base name exists in basePath, reference it
	pngSrc := filepath.Join(basePath, name+".png")
	if _, err := os.Stat(pngSrc); err == nil {
		// template will reference basename only (relative path)
		data.ImageFile = name + ".png"
	}

	var htmlBuf bytes.Buffer
	if err := tpl.Execute(&htmlBuf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Ensure basePath exists
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create output path: %w", err)
	}

	htmlOut := path.Join(basePath, name+".html")
	pdfOut := path.Join(basePath, name+".pdf")

	if err := os.WriteFile(htmlOut, htmlBuf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write html file: %w", err)
	}

	// If wkhtmltopdf is available, use it to create the PDF
	if _, err := exec.LookPath("wkhtmltopdf"); err == nil {
		cmd := exec.Command("wkhtmltopdf", "-q", htmlOut, pdfOut)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("wkhtmltopdf failed: %v - %s", err, string(out))
		}
		return pdfOut, nil
	}

	// wkhtmltopdf not found: return path to HTML so user can convert manually
	return htmlOut, nil
}

// getAppBasePath returns the repository root path where templates and data live.
// It uses the binary's working directory as a best-effort location.
func getAppBasePath() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
