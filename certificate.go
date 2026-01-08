package main

import (
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Certificate struct {
	ID          uuid.UUID             `yaml:"id"`
	Date        time.Time             `yaml:"date"`
	Applicant   string                `yaml:"applicant"`
	ObjectName  string                `yaml:"object_name"`
	ObjectClass string                `yaml:"object_class,omitempty"`
	Avg         float32               `yaml:"avg,omitempty"`
	Rank        string                `yaml:"rank,omitempty"`
	Reviewers   []string              `yaml:"reviewers"`
	Questions   []CertificateQuestion `yaml:"questions"`
}

type CertificateQuestion struct {
	Question  string                `yaml:"question"`
	Weight    float32               `yaml:"weight,omitempty"`
	Responses []CertificateResponse `yaml:"responses,omitempty"`
}

type CertificateResponse struct {
	Name    string `yaml:"name"`
	Value   int    `yaml:"value"`
	Comment string `yaml:"comment,omitempty"`
}

type CertificateSummaryEx struct {
	applicant   string                             `yaml:"applicant"`
	objectName  string                             `yaml:"object_name"`
	objectClass string                             `yaml:"object_class,omitempty"`
	avg         float32                            `yaml:"avg"`
	level       string                             `yaml:"level"`
	entries     map[string]CertificateSummaryEntry `yaml:"entries,omitempty"`
}

type CertificateSummaryEntry struct {
	min float32 `yaml:"min"`
	max float32 `yaml:"max"`
	avg float32 `yaml:"avg"`
	sum float32 `yaml:"sum"`
}

func (m *Model) CreateCertificate() *Certificate {

	if m.State != STATE_SUMMARY {
		return nil
	}

	certificate := Certificate{
		ID:          uuid.New(),
		Date:        time.Now(),
		Applicant:   m.applicantName,
		ObjectName:  m.objectName,
		ObjectClass: m.objectClass,
		Reviewers:   make([]string, 0),
		Questions:   make([]CertificateQuestion, 0),
	}

	for _, k := range m.SortedReviewerKeys {
		reviewer := m.Reviewers[k]
		reviewerName := reviewer.Name
		certificate.Reviewers = append(certificate.Reviewers, reviewerName)
	}

	type envelope struct {
		key    string
		weight float32
	}

	formKeysGrouped := make(map[string][]envelope)
	for _, g := range m.Cfg.Evaluation {
		groupKey := g.Key
		formKeysGrouped[g.Title] = []envelope{}
		for _, fc := range g.Fields {
			fieldKey := BuildFieldKey(groupKey, fc.Key)
			var appendFieldKey bool = false
			if fc.Type == "range" && strings.HasSuffix(fieldKey, "_rating") {
				appendFieldKey = true
			} else if fc.Type == "text" && strings.HasSuffix(fieldKey, "_comment") {
				appendFieldKey = true
			}

			if appendFieldKey {
				formKeysGrouped[g.Title] = append(formKeysGrouped[g.Title], envelope{key: fieldKey, weight: fc.Weight})
			}
		}
	}

	for groupTitle, formKeyGroup := range formKeysGrouped {

		if len(formKeyGroup) != 2 {
			panic("expected rating & comment field per group")
		}

		certificateQuestion := CertificateQuestion{
			Question:  groupTitle,
			Weight:    1.0, // TODO: support per-question weights
			Responses: []CertificateResponse{},
		}

		var fcRatingKey, fcCommentKey string
		for _, fk := range formKeyGroup {
			if strings.HasSuffix(fk.key, "_rating") {
				fcRatingKey = fk.key
				certificateQuestion.Weight = fk.weight
			} else if strings.HasSuffix(fk.key, "_comment") {
				fcCommentKey = fk.key
			}
		}

		for _, reviewerIdx := range m.SortedReviewerKeys {
			form := m.Evaluation.Forms[reviewerIdx]
			reviewerName := m.GetReviewerName(reviewerIdx)
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

	return &certificate
}

func roundToOneDecimal(val float32) float32 {
	return float32(math.Round(float64(val*10))) / 10
}

// summarizeEvaluations computes min, max and weighted average for each
// evaluation field across all reviewers. Results are stored in
// `m.Summaries[fieldKey]` with keys "min","max","avg","count".
func (m *Model) SummarizeCertificate(cert Certificate) CertificateSummaryEx {

	summary := CertificateSummaryEx{
		applicant:   cert.Applicant,
		objectName:  cert.ObjectName,
		objectClass: cert.ObjectClass,
		entries:     make(map[string]CertificateSummaryEntry),
	}

	var avgTotal float32 = 0.0

	for _, question := range cert.Questions {
		var minVal float32 = math.MaxFloat32
		var maxVal float32 = -math.MaxFloat32
		var sumVal float32 = 0

		if question.Weight == 0.0 {
			question.Weight = m.Cfg.GetWeightForField(question.Question, "rating")
		}

		for _, response := range question.Responses {
			valWeighted := float32(response.Value) * question.Weight
			minVal = minf(minVal, valWeighted)
			maxVal = maxf(maxVal, valWeighted)
			sumVal += valWeighted
		}

		avgVal := sumVal / float32(len(question.Responses))
		avgTotal += avgVal

		summary.entries[question.Question] = CertificateSummaryEntry{
			min: minVal,
			max: maxVal,
			avg: avgVal,
			sum: sumVal,
		}
	}

	achivedLevel := ""
	avgResult := roundToOneDecimal(avgTotal / float32(len(cert.Questions)))

	for _, level := range m.Cfg.SkillLevels {
		if avgResult >= level.MinPoints {
			achivedLevel = level.Name
		}
	}

	summary.avg = avgResult
	summary.level = achivedLevel

	return summary
}

// / GetRanks computes the overall and class-specific ranks for the given
// / certificate summary based on the Wall of Fame data.
// / If the Wall of Fame data cannot be loaded, it returns default ranks (1, 1).
func (certSummary CertificateSummaryEx) GetRanks() (overall int, class int) {
	overallRank := 1
	classRank := 1

	// Try to load the wall of fame data. If loading fails, return default ranks (1,1).
	if wof, err := LoadWallOfFame(); err == nil {
		// overall: count entries with Avg > certSummary.avg
		overallCount := 0
		classCount := 0

		for _, e := range wof.Entries {
			if e.Avg > certSummary.avg {
				overallCount++
			}
			// compare class (both may be empty)
			if e.ObjectClass == certSummary.objectClass && e.Avg > certSummary.avg {
				classCount++
			}
		}

		overallRank = overallCount + 1
		classRank = classCount + 1
	}

	return overallRank, classRank
}

func LoadCertificate(stateFile string) (Certificate, error) {
	var cert Certificate

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return cert, err
	}

	if err := yaml.Unmarshal(data, &cert); err != nil {
		return cert, err
	}

	// Backwards compatibility: older saved certificates may not have had the
	// exported `ID` field written to YAML (it was previously unexported). If
	// the unmarshaled ID is zero, try to infer it from the filename which is
	// the UUID used when the file was created.
	if cert.ID == uuid.Nil {
		base := filepath.Base(stateFile)
		idStr := strings.TrimSuffix(base, filepath.Ext(base))
		if id, err := uuid.Parse(idStr); err == nil {
			cert.ID = id
		}
	}

	return cert, nil
}

func SaveCertificate(path string, cert Certificate) error {

	if buff, err := yaml.Marshal(cert); err != nil {
		logger.Fatalf("Failed to marshal certification state to YAML: %v\n", err)
		return err
	} else {
		if err := os.WriteFile(path, buff, 0644); err != nil {
			logger.Fatalf("Failed to write certification state file: %v\n", err)
			return err
		} else {
			logger.Printf("Certificate saved at `%s`.\n", path)
		}
		return nil
	}
}

func (m *Model) SaveCertificateByConvention(certificate Certificate) error {

	currentPath := path.Join(getCertificatesPath(), certificate.Date.Format("2006"), certificate.Date.Format("01"))
	currentCertificatePath := path.Join(currentPath, certificate.ID.String()+".yaml")

	os.MkdirAll(currentPath, os.ModePerm)

	// Determine source image: prefer user-selected `m.objectImage` if present
	// and exists; otherwise fall back to the app asset `assets/designer.png`.
	sourceImage := strings.TrimSpace(m.objectImage)
	if sourceImage == "" {
		// try app asset
		appAsset := filepath.Join(getAppBasePath(), "assets", "designer.png")
		if _, err := os.Stat(appAsset); err == nil {
			sourceImage = appAsset
			logger.Printf("no selected image; using app asset %s", appAsset)
		} else {
			logger.Printf("no selected image and app asset not found: %s", appAsset)
			sourceImage = ""
		}
	} else {
		// ensure selected image actually exists; fall back if not
		if _, err := os.Stat(sourceImage); err != nil {
			logger.Printf("selected image not found: %s; attempting fallback asset", sourceImage)
			appAsset := filepath.Join(getAppBasePath(), "assets", "designer.png")
			if _, err := os.Stat(appAsset); err == nil {
				sourceImage = appAsset
				logger.Printf("falling back to app asset %s", appAsset)
			} else {
				logger.Printf("fallback asset not found: %s", appAsset)
				sourceImage = ""
			}
		}
	}

	// If we have a source image (either user-selected or app asset), copy it
	// next to the certificate using the certificate ID as basename while
	// preserving the original extension.
	if sourceImage != "" {
		ext := filepath.Ext(sourceImage)
		imgDest := filepath.Join(currentPath, certificate.ID.String()+ext)

		in, err := os.Open(sourceImage)
		if err != nil {
			logger.Printf("failed to open source image %s: %v", sourceImage, err)
		} else {
			defer in.Close()
			out, err := os.Create(imgDest)
			if err != nil {
				logger.Printf("failed to create destination image %s: %v", imgDest, err)
			} else {
				if _, err := io.Copy(out, in); err != nil {
					logger.Printf("failed to copy image to %s: %v", imgDest, err)
				} else {
					_ = out.Close()
					// make readable
					_ = os.Chmod(imgDest, 0644)
					logger.Printf("copied image %s to %s", sourceImage, imgDest)
				}
			}
		}
	}

	return SaveCertificate(currentCertificatePath, certificate)
}
