package views

import (
	"fmt"
	"sort"

	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
	"github.com/audi70r/gitstat/internal/ui/components"
)

// TimelineView displays commits over time
type TimelineView struct {
	root *tview.Flex
	text *tview.TextView
}

// NewTimelineView creates a new timeline view
func NewTimelineView() *TimelineView {
	v := &TimelineView{}
	v.setup()
	return v
}

func (v *TimelineView) setup() {
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
func (v *TimelineView) Refresh(repo *stats.Repository) {
	timeline := repo.GetTimeline(7)

	if len(timeline.Values) == 0 {
		v.text.SetText("[yellow]No commit data available[-]")
		return
	}

	// Calculate stats
	var total, maxVal, minVal int
	minVal = timeline.Values[0]
	for _, val := range timeline.Values {
		total += val
		if val > maxVal {
			maxVal = val
		}
		if val < minVal {
			minVal = val
		}
	}
	avg := float64(total) / float64(len(timeline.Values))

	// Get first and last dates
	firstDate := timeline.Labels[0]
	lastDate := timeline.Labels[len(timeline.Labels)-1]

	// Generate sparkline
	sparkWidth := 70
	sparkline := components.RenderSparklineWithWidth(timeline.Values, sparkWidth)

	// Weekly aggregation
	weeklyValues := aggregateWeekly(timeline.Labels, timeline.Values)
	weeklySparkline := components.RenderSparkline(weeklyValues)

	// Find peak day
	peakIdx := 0
	for i, val := range timeline.Values {
		if val > timeline.Values[peakIdx] {
			peakIdx = i
		}
	}
	peakDate := ""
	if peakIdx < len(timeline.Labels) {
		peakDate = timeline.Labels[peakIdx]
	}

	content := fmt.Sprintf(`[::b]Commits Over Time[-:-:-]

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Daily Activity (sparkline)[-:-:-]

  [green]%s[-]

  %s to %s

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Weekly Activity (sparkline)[-:-:-]

  [cyan]%s[-]

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Statistics[-:-:-]

  Period:             [cyan]%d[-] days
  Total Commits:      [cyan]%d[-]
  Average per Day:    [cyan]%.2f[-]
  Peak Day:           [green]%d[-] commits on [green]%s[-]
  Minimum Day:        [red]%d[-] commits
  Maximum Day:        [green]%d[-] commits

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]7-Day Rolling Average[-:-:-]

  Current:            [cyan]%.2f[-] commits/day
  Trend:              %s

`,
		sparkline,
		firstDate, lastDate,
		weeklySparkline,
		len(timeline.Values),
		total,
		avg,
		maxVal, peakDate,
		minVal,
		maxVal,
		timeline.RollingAvg[len(timeline.RollingAvg)-1],
		getTrendIndicator(timeline.RollingAvg),
	)

	v.text.SetText(content)
}

func aggregateWeekly(labels []string, values []int) []int {
	if len(values) == 0 {
		return nil
	}

	// Group by week
	weeks := make(map[string]int)
	weekOrder := make([]string, 0)

	for i, label := range labels {
		// Get week number from date
		weekKey := label[:7] // YYYY-MM as approximation
		if _, exists := weeks[weekKey]; !exists {
			weekOrder = append(weekOrder, weekKey)
		}
		weeks[weekKey] += values[i]
	}

	// Sort weeks
	sort.Strings(weekOrder)

	result := make([]int, len(weekOrder))
	for i, week := range weekOrder {
		result[i] = weeks[week]
	}

	return result
}

func getTrendIndicator(rollingAvg []float64) string {
	if len(rollingAvg) < 14 {
		return "[gray]Insufficient data[-]"
	}

	// Compare last week's average to previous week
	recent := rollingAvg[len(rollingAvg)-7:]
	previous := rollingAvg[len(rollingAvg)-14 : len(rollingAvg)-7]

	var recentSum, prevSum float64
	for _, v := range recent {
		recentSum += v
	}
	for _, v := range previous {
		prevSum += v
	}

	recentAvg := recentSum / 7
	prevAvg := prevSum / 7

	diff := recentAvg - prevAvg
	pctChange := 0.0
	if prevAvg > 0 {
		pctChange = diff / prevAvg * 100
	}

	if pctChange > 10 {
		return fmt.Sprintf("[green]↑ +%.1f%%[-] (increasing)", pctChange)
	} else if pctChange < -10 {
		return fmt.Sprintf("[red]↓ %.1f%%[-] (decreasing)", pctChange)
	}
	return fmt.Sprintf("[yellow]→ %.1f%%[-] (stable)", pctChange)
}

// Root returns the root primitive
func (v *TimelineView) Root() tview.Primitive {
	return v.root
}
