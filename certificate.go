package main

import (
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Certificate struct {
	id         uuid.UUID             `yaml:"id"`
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
	return Certificate{}, nil
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
