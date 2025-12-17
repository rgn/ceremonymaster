package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Certificate struct {
	ID         uuid.UUID             `yaml:"id"`
	Date       time.Time             `yaml:"date"`
	Applicant  string                `yaml:"applicant"`
	ObjectName string                `yaml:"object_name"`
	Reviewers  []string              `yaml:"reviewers"`
	Questions  []CertificateQuestion `yaml:"questions"`
}

type CertificateQuestion struct {
	Question  string                `yaml:"question"`
	Responses []CertificateResponse `yaml:"responses,omitempty"`
}

type CertificateResponse struct {
	Name    string `yaml:"name"`
	Value   int    `yaml:"value"`
	Comment string `yaml:"comment,omitempty"`
}

func loadCertificate(stateFile string) (Certificate, error) {
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

func saveCertificate(path string, cert Certificate) error {

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
