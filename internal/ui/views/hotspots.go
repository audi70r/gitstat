package views

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// HotspotsView displays high-risk files
type HotspotsView struct {
	root    *tview.Flex
	table   *tview.Table
	info    *tview.TextView
	sortCol int
	sortAsc bool
	columns []string
}

// NewHotspotsView creates a new hotspots view
func NewHotspotsView() *HotspotsView {
	v := &HotspotsView{
		sortCol: 5, // Default sort by risk score
		sortAsc: false,
		columns: []string{"#", "File", "Churn%", "Touches", "Authors", "Risk"},
	}
	v.setup()
	return v
}

func (v *HotspotsView) setup() {
	v.table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ')

	v.info = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	v.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.table, 0, 1, true).
		AddItem(v.info, 1, 0, false)

	v.renderHeader()
}

func (v *HotspotsView) renderHeader() {
	for col, name := range v.columns {
		cell := tview.NewTableCell(name).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)

		if col == v.sortCol {
			arrow := "▼"
			if v.sortAsc {
				arrow = "▲"
			}
			cell.SetText(name + arrow)
		}

		v.table.SetCell(0, col, cell)
	}
}

// Refresh updates the view with new data
func (v *HotspotsView) Refresh(repo *stats.Repository) {
	// Clear existing data rows
	for row := v.table.GetRowCount() - 1; row > 0; row-- {
		v.table.RemoveRow(row)
	}

	// Get hotspots and apply sorting
	hotspots := repo.GetHotspots(50)

	// Sort based on selected column
	sort.Slice(hotspots, func(i, j int) bool {
		var cmp bool
		switch v.sortCol {
		case 1: // File path
			cmp = hotspots[i].Path < hotspots[j].Path
		case 2: // Churn
			cmp = hotspots[i].ChurnScore < hotspots[j].ChurnScore
		case 3: // Touches
			cmp = hotspots[i].TouchCount < hotspots[j].TouchCount
		case 4: // Authors
			cmp = hotspots[i].AuthorCount < hotspots[j].AuthorCount
		case 5: // Risk
			cmp = hotspots[i].RiskScore < hotspots[j].RiskScore
		default:
			cmp = hotspots[i].RiskScore < hotspots[j].RiskScore
		}
		if v.sortAsc {
			return cmp
		}
		return !cmp
	})

	// Render data
	for i, spot := range hotspots {
		row := i + 1

		// Truncate long paths
		displayPath := spot.Path
		if len(displayPath) > 50 {
			displayPath = "..." + displayPath[len(displayPath)-47:]
		}

		// Risk color
		riskColor := getRiskColor(spot.RiskScore)

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorDarkGray).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 1, tview.NewTableCell(displayPath).
			SetExpansion(1))

		v.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.1f%%", spot.ChurnScore)).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", spot.TouchCount)).
			SetAlign(tview.AlignRight))

		authorColor := tcell.ColorWhite
		if spot.AuthorCount >= 5 {
			authorColor = tcell.ColorRed
		} else if spot.AuthorCount >= 3 {
			authorColor = tcell.ColorYellow
		}
		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", spot.AuthorCount)).
			SetTextColor(authorColor).
			SetAlign(tview.AlignRight))

		// Risk score with visual bar
		riskBar := getRiskBar(spot.RiskScore)
		v.table.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%.0f %s", spot.RiskScore, riskBar)).
			SetTextColor(riskColor).
			SetAlign(tview.AlignRight))
	}

	// Count high-risk files
	highRisk := 0
	for _, spot := range hotspots {
		if spot.RiskScore >= 50 {
			highRisk++
		}
	}

	// Update info
	v.info.SetText(fmt.Sprintf("[yellow]%d[-] hotspots | [red]%d[-] high-risk | Sort: [green]%s[-] | [s] cycle, [r] reverse",
		len(hotspots), highRisk, v.columns[v.sortCol]))

	v.renderHeader()
}

func getRiskColor(score float64) tcell.Color {
	if score >= 70 {
		return tcell.ColorRed
	} else if score >= 50 {
		return tcell.ColorOrange
	} else if score >= 30 {
		return tcell.ColorYellow
	}
	return tcell.ColorGreen
}

func getRiskBar(score float64) string {
	filled := int(score / 20) // 0-5 blocks
	if filled > 5 {
		filled = 5
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < 5; i++ {
		bar += "░"
	}
	return bar
}

// CycleSortColumn cycles through sort columns
func (v *HotspotsView) CycleSortColumn() {
	v.sortCol = (v.sortCol + 1) % len(v.columns)
	if v.sortCol == 0 {
		v.sortCol = 1
	}
}

// ReverseSortOrder reverses the sort order
func (v *HotspotsView) ReverseSortOrder() {
	v.sortAsc = !v.sortAsc
}

// Root returns the root primitive
func (v *HotspotsView) Root() tview.Primitive {
	return v.root
}

// GetFocusable returns the focusable component
func (v *HotspotsView) GetFocusable() tview.Primitive {
	return v.table
}
