package main

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/NimbleMarkets/ntcharts/heatmap"
	tea "github.com/charmbracelet/bubbletea"
)

const STATE_WALL_OF_FAME = "wall_of_fame"

type WallOfFameModel struct {
	List    list.Model
	HeatMap heatmap.Model
	Data    WallOfFame
}

type WallOfFame struct {
	UpdatedBy      string              `yaml:"updated_by"`
	UpdatedAt      time.Time           `yaml:"updated_at"`
	Entries        []WallOfFameEntry   `yaml:"entries,omitempty"`
	ByYear         map[int]int         `yaml:"by_year,omitempty"`
	ByYearAndMonth map[int]map[int]int `yaml:"by_year_and_month,omitempty"`
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
func (i WallOfFameEntry) FilterValue() string {
	return i.Applicant + "|" + i.ObjectName + "|" + i.ObjectClass
}

func (m *Model) InitWallOfFameModel() {

	// initialize basic model; heatmap will be configured based on loaded data
	m.WallOfFame = WallOfFameModel{
		Data: WallOfFame{},
	}

	// load persisted data and populate list + heatmap
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

		// determine year range and maximum value to set proper scaling
		minYear, maxYear := 2018, 2026
		maxValue := 0
		if len(data.ByYearAndMonth) > 0 {
			first := true
			for y, months := range data.ByYearAndMonth {
				if first {
					minYear = y
					maxYear = y
					first = false
				} else {
					if y < minYear {
						minYear = y
					}
					if y > maxYear {
						maxYear = y
					}
				}
				for _, v := range months {
					if v > maxValue {
						maxValue = v
					}
				}
			}
		}

		// ensure sensible range
		if maxValue < 1 {
			maxValue = 1
		}

		logger.Printf("Wall of fame year range: %d - %d, max monthly value: %d\n", minYear, maxYear, maxValue)
		// recreate heatmap with value range based on data
		m.WallOfFame.HeatMap = heatmap.New(20, 10, heatmap.WithValueRange(0, float64(maxValue)))
		// Set expected X/Y ranges (months, years)
		m.WallOfFame.HeatMap.SetXYRange(float64(1), float64(12), float64(minYear), float64(maxYear))
		// Also set the displayed/view ranges so points map inside the canvas
		m.WallOfFame.HeatMap.SetViewXYRange(float64(1), float64(12), float64(minYear), float64(maxYear))
		// make sure the heatmap uses the expected value range for color mapping
		m.WallOfFame.HeatMap.SetValueRange(0, float64(maxValue))

		// X = month (1..12), Y = year (minYear..maxYear)
		// Show month names on X axis and year numbers on Y axis
		m.WallOfFame.HeatMap.XLabelFormatter = func(i int, v float64) string {
			months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
			idx := int(math.Round(v))
			if idx >= 1 && idx <= 12 {
				return months[idx-1]
			}
			return fmt.Sprintf("%.0f", v)
		}
		m.WallOfFame.HeatMap.YLabelFormatter = func(i int, v float64) string {
			return fmt.Sprintf("%.0f", v)
		}

		// show every month and every year tick
		m.WallOfFame.HeatMap.SetXStep(1)
		m.WallOfFame.HeatMap.SetYStep(1)

		// Ensure one visual column per month: compute needed canvas width/height
		years := maxYear - minYear + 1
		// compute max label width for Y axis (years)
		labelW := 0
		for y := minYear; y <= maxYear; y++ {
			s := fmt.Sprintf("%.0f", float64(y))
			if len(s) > labelW {
				labelW = len(s)
			}
		}
		// graph width should be exactly 12 to have one column per month
		// linechart reserves (labelW + 1) columns for Y axis when YStep>0
		desiredWidth := labelW + 1 + 12
		// when XStep>0 the linechart uses last 2 rows for X axis labels
		desiredHeight := years + 2
		if desiredHeight < 3 {
			desiredHeight = 3
		}
		m.WallOfFame.HeatMap.Resize(desiredWidth, desiredHeight)

		// push a heat point for every year/month pair; missing values -> 0
		for y := minYear; y <= maxYear; y++ {
			for month := 1; month <= 12; month++ {
				val := 0
				if months, ok := data.ByYearAndMonth[y]; ok {
					if v, ok2 := months[month]; ok2 {
						val = v
					}
				}
				// note: X=month, Y=year to render like GitHub commit heatmap
				logger.Printf("Adding heatmap point for %d/%02d: %d\n", month, y, val)
				m.WallOfFame.HeatMap.Push(heatmap.NewHeatPoint(float64(month), float64(y), float64(val)))
			}
		}

	} else {
		logger.Printf("Failed to load wall of fame: %v\n", err)
		// fallback default range
		m.WallOfFame.HeatMap.SetXYRange(float64(1), float64(12), float64(2018), float64(2026))
	}

	m.WallOfFame.HeatMap.Draw()
}

func (mainModel *Model) UpdateWallOfFameModel(msg tea.Msg) []tea.Cmd {
	cmds := []tea.Cmd{}

	if mainModel.State != STATE_WALL_OF_FAME {
		return cmds
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := mainModel.Styles.Base.GetFrameSize()
		// reserve some space for the heatmap under the list
		heatmapHeight := 10
		listHeight := msg.Height - v - heatmapHeight
		if listHeight < 5 {
			listHeight = msg.Height - v // fallback to full height if too small
			heatmapHeight = 0
		}
		mainModel.WallOfFame.List.SetSize(msg.Width-h, listHeight)
		if mainModel.WallOfFame.HeatMap.Width() > 0 || heatmapHeight > 0 {
			mainModel.WallOfFame.HeatMap.Resize(msg.Width-h, heatmapHeight)
		}
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

	// Always render the simple labeled month×year grid for predictable layout
	hmView := renderSimpleHeatGrid(m.WallOfFame.Data)

	body := lipgloss.JoinVertical(lipgloss.Top, m.WallOfFame.List.View(), hmView)

	footer := m.appBoundaryView(m.WallOfFame.List.Help.View(m.WallOfFame.List))

	return header, body, footer
}

// renderSimpleHeatGrid draws a labeled month x year grid using lipgloss background
// blocks. It guarantees month numbers on the top and years on the left.
func renderSimpleHeatGrid(data WallOfFame) string {
	// determine year range
	minYear, maxYear := 2018, 2026
	if len(data.ByYearAndMonth) > 0 {
		first := true
		for y := range data.ByYearAndMonth {
			if first {
				minYear = y
				maxYear = y
				first = false
				continue
			}
			if y < minYear {
				minYear = y
			}
			if y > maxYear {
				maxYear = y
			}
		}
	}

	// find max value for scaling
	maxValue := 1
	for _, months := range data.ByYearAndMonth {
		for _, v := range months {
			if v > maxValue {
				maxValue = v
			}
		}
	}

	// green color scale (low->dark to high->bright)
	cs := []lipgloss.Color{
		lipgloss.Color("#08330B"),
		lipgloss.Color("#0B5A13"),
		lipgloss.Color("#0D7F1A"),
		lipgloss.Color("#19A21F"),
		lipgloss.Color("#4DD64A"),
		lipgloss.Color("#9FF59F"),
	}
	// build header with month numbers (1..12)
	var b strings.Builder
	// left padding for year labels
	labelW := len(fmt.Sprintf("%d", maxYear))
	b.WriteString(strings.Repeat(" ", labelW+1))

	// base width per month and pixel scaling
	baseMonthW := 3
	pixelScale := 2 // double size
	colW := baseMonthW * pixelScale

	for m := 1; m <= 12; m++ {
		ms := fmt.Sprintf("%d", m)
		padLeft := (colW - len(ms)) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		padRight := colW - len(ms) - padLeft
		b.WriteString(strings.Repeat(" ", padLeft) + ms + strings.Repeat(" ", padRight))
	}
	b.WriteString("\n")

	// rows per year (each year uses pixelScale rows to double vertical size)
	for y := minYear; y <= maxYear; y++ {
		for row := 0; row < pixelScale; row++ {
			if row == 0 {
				yearLabel := fmt.Sprintf("%*d ", labelW, y)
				b.WriteString(yearLabel)
			} else {
				b.WriteString(strings.Repeat(" ", labelW+1))
			}
			for mth := 1; mth <= 12; mth++ {
				val := 0
				if months, ok := data.ByYearAndMonth[y]; ok {
					if v, ok2 := months[mth]; ok2 {
						val = v
					}
				}
				// zero -> transparent (unstyled space)
				if val == 0 {
					b.WriteString(strings.Repeat(" ", colW))
				} else {
					// map value to color index
					idx := 0
					if maxValue > 0 {
						idx = int(math.Round((float64(val) / float64(maxValue)) * float64(len(cs)-1)))
					}
					if idx < 0 {
						idx = 0
					}
					if idx >= len(cs) {
						idx = len(cs) - 1
					}
					style := lipgloss.NewStyle().Background(cs[idx]).Foreground(lipgloss.Color("#000000"))
					block := strings.Repeat(" ", colW)
					b.WriteString(style.Render(block))
				}
			}
			b.WriteString("\n")
		}
	}

	return b.String()
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
		UpdatedBy:      currentUser.Username,
		UpdatedAt:      time.Now(),
		Entries:        []WallOfFameEntry{},
		ByYear:         make(map[int]int),
		ByYearAndMonth: make(map[int]map[int]int),
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

			wallOfFame.ByYear[cert.Date.Year()] += 1
			if _, ok := wallOfFame.ByYearAndMonth[cert.Date.Year()]; !ok {
				wallOfFame.ByYearAndMonth[cert.Date.Year()] = make(map[int]int)
			}
			wallOfFame.ByYearAndMonth[cert.Date.Year()][int(cert.Date.Month())] += 1
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
