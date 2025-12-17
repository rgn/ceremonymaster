package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	DataPath       string             `yaml:"data_path,omitempty"`
	DataCollection []GroupConfig      `yaml:"datacollection"`
	Evaluation     []GroupConfig      `yaml:"evaluation"`
	SkillLevels    []SkillLevelConfig `yaml:"skilllevels"`
}

type SkillLevelConfig struct {
	Level       int     `yaml:"level"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	MinPoints   float32 `yaml:"min_points"`
}

type GroupConfig struct {
	Key         string        `yaml:"key"`
	Title       string        `yaml:"name"`
	Description string        `yaml:"description"`
	Fields      []FieldConfig `yaml:"fields"`
}

type FieldConfig struct {
	Type        string   `yaml:"type"` // "select" or "range" or "input" or "text" or "filepicker" or "confirm"
	Key         string   `yaml:"key"`  // *MUST BE* <type>_<identifier>
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	Mandatory   bool     `yaml:"mandatory"`
	Options     []string `yaml:"options,omitempty"`     // for select
	Affirmative string   `yaml:"affirmative,omitempty"` // for confirm
	Negative    string   `yaml:"negative,omitempty"`    // for confirm
	RequireYes  bool     `yaml:"require_yes,omitempty"` // if true, confirm validation fails when false
	// Weight is an optional multiplier used when summarizing numeric
	// evaluation fields across reviewers. If omitted, a weight of 1.0
	// is assumed.
	Weight float32 `yaml:"weight,omitempty"` // for range
}

func defaultConfiguration() Configuration {
	return Configuration{
		SkillLevels: []SkillLevelConfig{
			{
				Level:       0,
				Name:        "Junior Cake Engineer 👷",
				Description: "",
				MinPoints:   0.0,
			},
			{
				Level:       1,
				Name:        "Cake Engineer",
				Description: "",
				MinPoints:   1.0,
			},
			{
				Level:       2,
				Name:        "Senior Cake Engineer 🤠",
				Description: "",
				MinPoints:   2.0,
			},
			{
				Level:       3,
				Name:        "Cake Consultant 🥸",
				Description: "",
				MinPoints:   3.0,
			},
			{
				Level:       4,
				Name:        "Senior Cake Consultant 🧐",
				Description: "",
				MinPoints:   4.0,
			},
			{
				Level:       4,
				Name:        "Principal Cake Architect 🤯",
				Description: "",
				MinPoints:   4.6,
			},
		},
		DataCollection: []GroupConfig{
			{
				Key:         "data_entry",
				Title:       "Zertifizierungsantrag",
				Description: "Wer will sich womit zertifizieren lassen?",
				Fields: []FieldConfig{
					{
						Type:        "input",
						Key:         "applicant_name",
						Title:       "Name des Antragstellers",
						Description: "Bitte geben Sie den vollständigen Namen des Antragstellers ein.",
						Mandatory:   true,
					},
					{
						Type:        "text",
						Key:         "object_description",
						Title:       "Zertifizierungsobjekt",
						Description: "Was soll zertifiziert werden?",
						Mandatory:   true,
					},
					{
						Type:        "select",
						Key:         "object_class",
						Title:       "Zertifizierungsobjektklasse",
						Description: "Welcher Art ist das Objekt?",
						Options:     []string{"Kuchen", "Torte", "Sonstiges"},
						Mandatory:   true,
					},
					{
						Type:        "filepicker",
						Key:         "object_image",
						Title:       "Zertifizierungsobjektbild",
						Description: "Bitte wählen Sie ein Bild des Zertifizierungsobjekts aus.",
						Options:     []string{".png", ".jpg"},
						Mandatory:   false,
					},
					{
						Type:        "confirm",
						Key:         "approval",
						Title:       "Antrag vollständig erfasst?",
						Description: "Sind Sie der Meinung, dass der Antrag vollständig erfasst ist?",
						Affirmative: "Ja, Freigabe erteilen",
						Negative:    "Nein, keine Freigabe",
						RequireYes:  true,
					},
				},
			},
			{
				Key:         "reviewer",
				Title:       "Review Panel",
				Description: "Wer wird den Zertifzierungsantrag prüfen?",
				Fields: []FieldConfig{
					{
						Type:        "input",
						Key:         "1",
						Title:       "Zertifizierer (1)",
						Description: "Gib einen Namen des Zertifizierers ein.",
					},
					{
						Type:        "input",
						Key:         "2",
						Title:       "Zertifizierer (2)",
						Description: "Gib einen Namen des Zertifizierers ein.",
					},
					{
						Type:        "input",
						Key:         "3",
						Title:       "Zertifizierer (3)",
						Description: "Gib einen Namen des Zertifizierers ein.",
					},
					{
						Type:        "input",
						Key:         "4",
						Title:       "Zertifizierer (4)",
						Description: "Gib einen Namen des Zertifizierers ein.",
					},
					{
						Type:        "confirm",
						Key:         "approval",
						Title:       "Zertifizierer erfasst?",
						Description: "Sind alle Zertifizierenden erfasst?",
						Affirmative: "Ja!",
						Negative:    "Oh nö...",
						RequireYes:  true,
					},
				},
			},
		},
		Evaluation: []GroupConfig{
			{
				Key:         "appearance",
				Title:       "Optikbewertung",
				Description: "Wie sieht das Zertifizierungsobjekt aus?",
				Fields: []FieldConfig{
					{
						Type:        "range",
						Key:         "rating",
						Title:       "Aussehen",
						Description: "Wie sieht das Zertifizierungsobjekt aus?",
						Mandatory:   true,
						Weight:      1.0,
					},
					{
						Type:        "text",
						Key:         "comment",
						Title:       "Kommentar",
						Description: "Was sind deine Gedanken dazu?",
						Mandatory:   false,
					},
				},
			},
			{
				Key:         "smell",
				Title:       "Geruchsbewertung",
				Description: "Wie hat das Zertifizierungsobjekt gerochen?",
				Fields: []FieldConfig{
					{
						Type:        "range",
						Key:         "rating",
						Title:       "Geruch",
						Description: "Wie gut riecht das Zertifizierungsobjekt?",
						Mandatory:   true,
						Weight:      1.0,
					},
					{
						Type:        "text",
						Key:         "comment",
						Title:       "Kommentar",
						Description: "Was sind deine Gedanken dazu?",
						Mandatory:   false,
					},
				},
			},
			{
				Key:         "taste",
				Title:       "Geschmacksbewertung",
				Description: "Wie hat das Zertifizierungsobjekt geschmeckt?",
				Fields: []FieldConfig{
					{
						Type:        "range",
						Key:         "rating",
						Title:       "Geschmack",
						Description: "Wie gut schmeckt das Zertifizierungsobjekt?",
						Mandatory:   true,
						Weight:      1.0,
					},
					{
						Type:        "text",
						Key:         "comment",
						Title:       "Kommentar",
						Description: "Was sind deine Gedanken dazu?",
						Mandatory:   false,
					},
				},
			},
			{
				Key:         "innovation",
				Title:       "Innovationsbewertung",
				Description: "Wie innovativ ist das Zertifizierungsobjekt?",
				Fields: []FieldConfig{
					{
						Type:        "range",
						Key:         "rating",
						Title:       "Innovationsgrad",
						Description: "Wie innovativ ist das Zertifizierungsobjekt?",
						Mandatory:   true,
						Weight:      1.0,
					},
					{
						Type:        "text",
						Key:         "comment",
						Title:       "Kommentar",
						Description: "Was sind deine Gedanken dazu?",
						Mandatory:   false,
					},
				},
			},
			{
				Key:         "complexity",
				Title:       "Schwierigkeitsbewertung",
				Description: "Wie komplex ist das Zertifizierungsobjekt?",
				Fields: []FieldConfig{
					{
						Type:        "range",
						Key:         "rating",
						Title:       "Komplexitätsgrad",
						Description: "Wie komplex ist das Zertifizierungsobjekt?",
						Mandatory:   true,
						Weight:      1.0,
					},
					{
						Type:        "text",
						Key:         "comment",
						Title:       "Kommentar",
						Description: "Was sind deine Gedanken dazu?",
						Mandatory:   false,
					},
				},
			},
			{
				Key:         "presentation",
				Title:       "Präsentationsbewertung",
				Description: "Wie wurde das Zertifizierungsobjekt präsentiert?",
				Fields: []FieldConfig{
					{
						Type:        "range",
						Key:         "rating",
						Title:       "Präsentationsformat",
						Description: "Wie wurde das Zertifizierungsobjekt präsentiert?",
						Mandatory:   true,
						Weight:      1.0,
					},
					{
						Type:        "text",
						Key:         "comment",
						Title:       "Kommentar",
						Description: "Was sind deine Gedanken dazu?",
						Mandatory:   false,
					},
				},
			},
		},
	}
}

/** Load configuration from YAML file */
func loadConfiguration(path string) (Configuration, error) {

	if _, err := os.Stat(path); err != nil {
		logger.Println("Configuration file not found, using default configuration.")
		return defaultConfiguration(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Println("Failed to read configuration file, using default configuration:", err)
		return defaultConfiguration(), err
	}

	fmt.Println("Configuration file read: ", path)

	var configuration Configuration
	if err := yaml.Unmarshal(data, &configuration); err != nil {
		logger.Println("error unmarshaling configuration from YAML, using default configuration:", err)
		return defaultConfiguration(), err
	}

	return configuration, nil
}

func saveConfiguration(path string, config Configuration) error {

	logger.Println("Saving configuration to: ", path)

	data, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatalf("error marshaling configuration to YAML: %v", err)
		return err
	}

	logger.Println("Configuration data marshaled to YAML.")
	if err := os.WriteFile(path, data, 0644); err != nil {
		logger.Fatalf("Failed to write configuration file: %v", err)
		return err
	}

	logger.Println("Configuration file written successfully.", path)

	return nil
}
