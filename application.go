package main

import (
	"fmt"
	"os"
	"path"
)

var (
	APPLICATION_PATH             string
	CERTIFICATE_PATH             string
	APPLICATION_LOG_PATH         = "logs"
	APPLICATION_CERTIFICATE_PATH = "certificates"
	CONFIGURATION_FILE           = "config.yaml"
)

func initApplication() (Configuration, func()) {

	if homedir, err := os.UserHomeDir(); err != nil {
		fmt.Println("Failed to determine user home directory: ", err)
		os.Exit(1)
	} else {
		APPLICATION_PATH = path.Join(homedir, ".ceremonymaster")
	}

	if _, err := os.Stat(APPLICATION_PATH); err != nil {
		err := os.Mkdir(APPLICATION_PATH, os.FileMode(0755))
		check(err)
		fmt.Println("Created application directory at ", APPLICATION_PATH)
	}

	configurationFile := getConfigurationFilePath()
	if _, err := os.Stat(configurationFile); err != nil {
		defaultCfg := defaultConfiguration()
		err := saveConfiguration(configurationFile, defaultCfg)
		check(err)
		fmt.Println("Created default configuration file at ", configurationFile)
	}

	// use the expanded applicationPath when initializing the logger
	initLogger(getLogPath())

	cfg, err := loadConfiguration(configurationFile)
	check(err)

	CERTIFICATE_PATH = cfg.CertificatePath

	return cfg, func() { closeLogger() }
}

func getLogPath() string {
	s := path.Join(APPLICATION_PATH, APPLICATION_LOG_PATH)
	return s
}

func getCertificatesPath() string {

	basePath := APPLICATION_PATH
	if CERTIFICATE_PATH != "" {
		if _, err := os.Stat(CERTIFICATE_PATH); err == nil {
			basePath = CERTIFICATE_PATH
		} else {
			logger.Fatalf("Configured certificate `%s` path not found.\n", CERTIFICATE_PATH)
			os.Exit(1)
		}
	}

	s := path.Join(basePath, APPLICATION_CERTIFICATE_PATH)
	if _, err := os.Stat(s); err != nil {
		err := os.Mkdir(s, os.FileMode(0755))
		check(err)
	}
	return s
}

func getConfigurationFilePath() string {
	s := path.Join(APPLICATION_PATH, CONFIGURATION_FILE)
	return s
}
