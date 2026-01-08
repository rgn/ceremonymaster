package main

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	tea "github.com/charmbracelet/bubbletea"
)

const STATE_WALL_OF_FAME = "wall_of_fame"

type WallOfFameModel struct {
	List list.Model
	Data WallOfFame
}

type WallOfFame struct {
	UpdatedBy string            `yaml:"updated_by"`
	UpdatedAt time.Time         `yaml:"updated_at"`
	Entries   []WallOfFameEntry `yaml:"entries,omitempty"`
}

type WallOfFameEntry struct {
	Applicant   string  `yaml:"applicant"`
	ObjectName  string  `yaml:"object_name"`
	ObjectClass string  `yaml:"object_class,omitempty"`
	Avg         float32 `yaml:"avg"`
	Rank        string  `yaml:"rank"`
}

type listHelpKeyMap struct {
	list.KeyMap
}

func (i WallOfFameEntry) Title() string {
	return fmt.Sprintf("%s %.1f für %s", i.Applicant, i.Avg, i.ObjectName)
}
func (i WallOfFameEntry) Description() string { return i.Applicant }
func (i WallOfFameEntry) FilterValue() string { return i.Applicant + "|" + i.ObjectName + "|" + i.ObjectClass }

func (m *Model) InitWallOfFameModel() {

	m.WallOfFame = WallOfFameModel{
		Data: WallOfFame{},
	}

	if data, err := LoadWallOfFame(); err == nil {
		m.WallOfFame.Data = data

		logger.Println("Loaded wall of fame with ", len(data.Entries), " entries.")

		items := make([]list.Item, len(data.Entries))
		for i, entry := range data.Entries {
			items[i] = entry
		}

		wallOfFameList := list.New(items, list.DefaultDelegate{}, m.width, m.height)
		wallOfFameList.Title = m.Styles.Highlight.Render("Wall of Fame")
		wallOfFameList.SetShowHelp(false)

		m.WallOfFame.List = wallOfFameList
	} else {
		logger.Printf("Failed to load wall of fame: %v\n", err)
	}
}

func (mainModel *Model) UpdateWallOfFameModel(msg tea.Msg) []tea.Cmd {
	cmds := []tea.Cmd{}

	if mainModel.State != STATE_WALL_OF_FAME {
		return cmds
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := mainModel.Styles.Base.GetFrameSize()
		mainModel.WallOfFame.List.SetSize(msg.Width-h, msg.Height-v)
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if mainModel.WallOfFame.List.FilterState() == list.Filtering {
			break
		}
	}

	newListModel, cmd := mainModel.WallOfFame.List.Update(msg)
	mainModel.WallOfFame.List = newListModel
	cmds = append(cmds, cmd)

	return cmds
}

func (m *Model) ViewWallOfFame() (string, string, string) {

	header := "🏆 Wall of Fame 🏆"

	body := m.WallOfFame.List.View()
	footer := m.appBoundaryView(m.WallOfFame.List.Help.View(m.WallOfFame.List))

	return header, body, footer
}

func (m *WallOfFameModel) GetStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.HiddenBorder())
	// Height(m.List.Height()).
	// Width(m.List.Width())
}

func (m *Model) UpdateWallOfFame() {

	logger.Println("Updating wall of fame...")

	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	wallOfFame := WallOfFame{
		UpdatedBy: currentUser.Username,
		UpdatedAt: time.Now(),
		Entries:   []WallOfFameEntry{},
	}

	for _, certificateFilePath := range GetCertificateFilePaths() {
		logger.Println("Processing certificate: ", certificateFilePath)
		if cert, err := LoadCertificate(certificateFilePath); err == nil {
			certSummary := m.SummarizeCertificate(cert)
			logger.Println("Applicant ", certSummary.applicant, "object ", certSummary.objectName, " avg score: ", certSummary.avg, " rank: ", certSummary.level, " entries: ", len(certSummary.entries))
			wallOfFame.Entries = append(wallOfFame.Entries, WallOfFameEntry{
				Applicant:   certSummary.applicant,
				ObjectName:  certSummary.objectName,
				ObjectClass: certSummary.objectClass,
				Avg:         certSummary.avg,
				Rank:        certSummary.level,
			})
		} else {
			logger.Printf("Failed to load certificate %s: %v", certificateFilePath, err)
		}
	}

	sort.SliceStable(wallOfFame.Entries, func(i, j int) bool {
		return wallOfFame.Entries[i].Avg > wallOfFame.Entries[j].Avg
	})

	wallOfFamePath := path.Join(getDataPath(), "wall_of_fame.yaml")
	if buff, err := yaml.Marshal(wallOfFame); err != nil {
		logger.Fatalf("Failed to marshal wall of fame to YAML: %v\n", err)
		return
	} else {
		if err := os.WriteFile(wallOfFamePath, buff, 0644); err != nil {
			logger.Fatalf("Failed to write wall of fame state file: %v\n", err)
			return
		} else {
			logger.Printf("Wall of fame saved at `%s`.\n", wallOfFamePath)
		}
		return
	}
}

func LoadWallOfFame() (WallOfFame, error) {
	var wallOfFame WallOfFame

	wallOfFamePath := path.Join(getDataPath(), "wall_of_fame.yaml")
	data, err := os.ReadFile(wallOfFamePath)
	if err != nil {
		return wallOfFame, err
	}

	if err := yaml.Unmarshal(data, &wallOfFame); err != nil {
		logger.Fatalf("Failed to unmarshal wall of fame from YAML: %v\n", err)
		return wallOfFame, err
	}

	return wallOfFame, nil
}
