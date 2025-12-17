package main

import (
	"fmt"
	"os"
	"path"
)

var (
	APPLICATION_PATH     string
	DATA_PATH            string
	CERTIFICATE_PATH     = "certificates"
	APPLICATION_LOG_PATH = "logs"
	CONFIGURATION_FILE   = "config.yaml"
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
