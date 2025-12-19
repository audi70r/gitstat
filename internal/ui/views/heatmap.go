package views

import (
	"fmt"
	"time"

	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
	"github.com/audi70r/gitstat/internal/ui/components"
)

var weekdayNames = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

// HeatmapView displays work hours heatmap
type HeatmapView struct {
	root *tview.Flex
	text *tview.TextView
}

// NewHeatmapView creates a new heatmap view
func NewHeatmapView() *HeatmapView {
	v := &HeatmapView{}
	v.setup()
	return v
}

func (v *HeatmapView) setup() {
	v.text = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	v.root = tview.NewFlex().
		AddItem(nil, 2, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).
			AddItem(v.text, 0, 1, false).
			AddItem(nil, 1, 0, false), 0, 1, false).
		AddItem(nil, 2, 0, false)
}

// Refresh updates the view with new data
func (v *HeatmapView) Refresh(repo *stats.Repository, tz *time.Location) {
	heatmap := repo.GetHeatmap(tz)
	peakDay, peakHour, totalCommits := components.GetHeatmapStats(heatmap.Matrix)

	if tz == nil {
		tz = time.Local
	}

	// Calculate weekday totals
	weekdayTotals := make([]int, 7)
	for day := 0; day < 7; day++ {
		for hour := 0; hour < 24; hour++ {
			weekdayTotals[day] += heatmap.Matrix[day][hour]
		}
	}

	// Calculate hour totals
	hourTotals := make([]int, 24)
	for hour := 0; hour < 24; hour++ {
		for day := 0; day < 7; day++ {
			hourTotals[hour] += heatmap.Matrix[day][hour]
		}
	}

	// Find busiest weekday and hour
	busiestDay := 0
	for i, total := range weekdayTotals {
		if total > weekdayTotals[busiestDay] {
			busiestDay = i
		}
	}

	busiestHour := 0
	for i, total := range hourTotals {
		if total > hourTotals[busiestHour] {
			busiestHour = i
		}
	}

	// Render heatmap grid
	heatmapGrid := components.RenderHeatmap(heatmap.Matrix, heatmap.MaxValue)

	// Calculate work hours vs off hours
	var workHours, offHours int
	for day := 0; day < 7; day++ {
		for hour := 0; hour < 24; hour++ {
			commits := heatmap.Matrix[day][hour]
			if day < 5 && hour >= 9 && hour < 18 {
				workHours += commits
			} else {
				offHours += commits
			}
		}
	}

	workPct := 0.0
	if totalCommits > 0 {
		workPct = float64(workHours) / float64(totalCommits) * 100
	}

	content := fmt.Sprintf(`[::b]Work Hours Heatmap[-:-:-]

  Timezone: [cyan]%s[-]

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

%s

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Peak Activity[-:-:-]

  Peak Time:          [green]%s[-] at [green]%02d:00[-] ([cyan]%d[-] commits)
  Busiest Day:        [green]%s[-] ([cyan]%d[-] commits total)
  Busiest Hour:       [green]%02d:00[-] ([cyan]%d[-] commits total)

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Work Patterns[-:-:-]

  Work Hours (Mon-Fri, 9-18):   [cyan]%d[-] commits (%.1f%%)
  Off Hours:                    [cyan]%d[-] commits (%.1f%%)

  Pattern:            %s

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Weekday Breakdown[-:-:-]

  Mon: [cyan]%4d[-]  Tue: [cyan]%4d[-]  Wed: [cyan]%4d[-]  Thu: [cyan]%4d[-]
  Fri: [cyan]%4d[-]  Sat: [cyan]%4d[-]  Sun: [cyan]%4d[-]

`,
		tz.String(),
		heatmapGrid,
		weekdayNames[peakDay], peakHour, heatmap.Matrix[peakDay][peakHour],
		weekdayNames[busiestDay], weekdayTotals[busiestDay],
		busiestHour, hourTotals[busiestHour],
		workHours, workPct,
		offHours, 100-workPct,
		getWorkPattern(workPct),
		weekdayTotals[0], weekdayTotals[1], weekdayTotals[2], weekdayTotals[3],
		weekdayTotals[4], weekdayTotals[5], weekdayTotals[6],
	)

	v.text.SetText(content)
}

func getWorkPattern(workPct float64) string {
	if workPct >= 80 {
		return "[green]Highly structured (mostly work hours)[-]"
	} else if workPct >= 60 {
		return "[cyan]Balanced (mix of work and off hours)[-]"
	} else if workPct >= 40 {
		return "[yellow]Flexible (significant off-hours work)[-]"
	}
	return "[red]Non-traditional (mostly off-hours)[-]"
}

// Root returns the root primitive
func (v *HeatmapView) Root() tview.Primitive {
	return v.root
}
