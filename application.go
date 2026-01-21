package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	APPLICATION_PATH     string
	DATA_PATH            string
	CERTIFICATE_PATH     = "certificates"
	APPLICATION_LOG_PATH = "logs"
	CONFIGURATION_FILE   = "config.yaml"
)

// HomeConfig represents the optional configuration file at ~/.ceremonymaster
type HomeConfig struct {
	DataPath string `yaml:"data_path"`           // mandatory: path must exist and be an accessible folder
	LogLevel string `yaml:"log_level,omitempty"` // optional: default = info
}

// expandPath expands environment variables and tilde in a path
func expandPath(p string) string {
	// First expand environment variables like $HOME, $USER, etc.
	p = os.ExpandEnv(p)

	// Then expand tilde if present at the beginning
	if strings.HasPrefix(p, "~/") || p == "~" {
		if homedir, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				p = homedir
			} else {
				p = filepath.Join(homedir, p[2:])
			}
		}
	}

	return p
}

// loadHomeConfig loads the ~/.ceremonymaster YAML configuration file
func loadHomeConfig(filePath string) (HomeConfig, error) {
	var cfg HomeConfig

	data, err := os.ReadFile(filePath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read home config file: %w", err)
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse home config file: %w", err)
	}

	// Validate mandatory DataPath field
	if cfg.DataPath == "" {
		return cfg, fmt.Errorf("data_path is mandatory in home config file")
	}

	// Expand path variables and tilde
	cfg.DataPath = expandPath(cfg.DataPath)

	// Check if DataPath exists and is accessible
	info, err := os.Stat(cfg.DataPath)
	if err != nil {
		return cfg, fmt.Errorf("data_path '%s' does not exist or is not accessible: %w", cfg.DataPath, err)
	}

	if !info.IsDir() {
		return cfg, fmt.Errorf("data_path '%s' is not a directory", cfg.DataPath)
	}

	// Set default LogLevel if not provided
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}

func initApplication() (Configuration, func()) {

	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to determine user home directory: ", err)
		os.Exit(1)
	}

	homeCeremonyMaster := path.Join(homedir, ".ceremonymaster")

	// Check if ~/.ceremonymaster exists and whether it's a file or directory
	info, err := os.Stat(homeCeremonyMaster)
	if err == nil {
		// ~/.ceremonymaster exists
		if !info.IsDir() {
			// Case 1: ~/.ceremonymaster is a file - load as YAML config
			fmt.Println("Loading configuration from ", homeCeremonyMaster)
			homeCfg, err := loadHomeConfig(homeCeremonyMaster)
			if err != nil {
				fmt.Printf("Failed to load home config file: %v\n", err)
				os.Exit(1)
			}
			// Use the DataPath from home config as APPLICATION_PATH
			APPLICATION_PATH = homeCfg.DataPath
			fmt.Printf("Using data path from config: %s\n", APPLICATION_PATH)
			// TODO: Use homeCfg.LogLevel for logger configuration if needed
		} else {
			// Case 2: ~/.ceremonymaster is a directory (already implemented)
			APPLICATION_PATH = homeCeremonyMaster
		}
	} else if os.IsNotExist(err) {
		// ~/.ceremonymaster doesn't exist - create directory (default behavior)
		APPLICATION_PATH = homeCeremonyMaster
		err := os.Mkdir(APPLICATION_PATH, os.FileMode(0755))
		check(err)
		fmt.Println("Created application directory at ", APPLICATION_PATH)
	} else {
		// Some other error occurred while checking
		fmt.Printf("Failed to check ~/.ceremonymaster: %v\n", err)
		os.Exit(1)
	}

	// use the expanded applicationPath when initializing the logger
	initLogger(getLogPath())

	configurationFile := getConfigurationFilePath()
	if _, err := os.Stat(configurationFile); err != nil {
		defaultCfg := defaultConfiguration()
		err := saveConfiguration(configurationFile, defaultCfg)
		check(err)
		fmt.Println("Created default configuration file at ", configurationFile)
	}

	getCertificatesPath()
	getTemplatesPath()

	cfg, err := loadConfiguration(configurationFile)
	check(err)

	DATA_PATH = cfg.DataPath

	return cfg, func() { closeLogger() }
}

func getLogPath() string {
	s := path.Join(getDataPath(), APPLICATION_LOG_PATH)
	return s
}

func getDataPath() string {
	var basePath = APPLICATION_PATH

	if DATA_PATH != "" {
		if _, err := os.Stat(DATA_PATH); err == nil {
			basePath = DATA_PATH
		} else {
			logger.Fatalf("Configured data `%s` path not found.\n", DATA_PATH)
			os.Exit(1)
		}
	}

	return basePath
}

func getTemplatesPath() string {
	// Ensure templates directory exists in application path. If the user hasn't
	// customized templates, copy the provided default template bundled in the
	// repository's templates/ directory into the application directory so users
	// can edit it there.
	templatesPath := path.Join(getDataPath(), "templates")
	if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
		if err := os.MkdirAll(templatesPath, os.FileMode(0755)); err == nil {
			// try to copy default template from working directory if present
			src := path.Join(getAppBasePath(), "templates", "certificate.html")
			dst := path.Join(templatesPath, "certificate.html")
			if data, err := os.ReadFile(src); err == nil {
				_ = os.WriteFile(dst, data, 0644)
			}
		}
	}

	return templatesPath
}

func getCertificatesPath() string {

	certificatePath := path.Join(getDataPath(), CERTIFICATE_PATH)
	if _, err := os.Stat(certificatePath); err != nil {
		err := os.Mkdir(certificatePath, os.FileMode(0755))
		check(err)
	}

	return certificatePath
}

func getConfigurationFilePath() string {
	s := path.Join(APPLICATION_PATH, CONFIGURATION_FILE)
	return s
}
