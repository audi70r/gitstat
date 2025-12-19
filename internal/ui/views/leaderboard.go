package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// LeaderboardView displays author statistics
type LeaderboardView struct {
	root    *tview.Flex
	table   *tview.Table
	info    *tview.TextView
	sortCol int
	sortAsc bool
	columns []string
}

// NewLeaderboardView creates a new leaderboard view
func NewLeaderboardView() *LeaderboardView {
	v := &LeaderboardView{
		sortCol: 2, // Default sort by commits
		sortAsc: false,
		columns: []string{"#", "Author", "Commits", "Additions", "Deletions", "Net", "Files"},
	}
	v.setup()
	return v
}

func (v *LeaderboardView) setup() {
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

func (v *LeaderboardView) renderHeader() {
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
func (v *LeaderboardView) Refresh(repo *stats.Repository) {
	// Clear existing data rows
	for row := v.table.GetRowCount() - 1; row > 0; row-- {
		v.table.RemoveRow(row)
	}

	// Get sorted leaderboard
	sortBy := []string{"", "name", "commits", "additions", "deletions", "net", ""}[v.sortCol]
	if sortBy == "" {
		sortBy = "commits"
	}
	authors := repo.GetLeaderboard(sortBy, v.sortAsc)

	// Render data
	for i, author := range authors {
		row := i + 1
		net := author.Additions - author.Deletions
		netStr := fmt.Sprintf("%+d", net)
		filesCount := len(author.FilesTouched)

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorDarkGray).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 1, tview.NewTableCell(author.Name).
			SetExpansion(1))

		v.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", author.Commits)).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("+%d", author.Additions)).
			SetTextColor(tcell.ColorGreen).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("-%d", author.Deletions)).
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignRight))

		netColor := tcell.ColorWhite
		if net > 0 {
			netColor = tcell.ColorGreen
		} else if net < 0 {
			netColor = tcell.ColorRed
		}
		v.table.SetCell(row, 5, tview.NewTableCell(netStr).
			SetTextColor(netColor).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%d", filesCount)).
			SetAlign(tview.AlignRight))
	}

	// Update info
	v.info.SetText(fmt.Sprintf("[yellow]%d[-] authors | Sort: [green]%s[-] | [s] cycle column, [r] reverse",
		len(authors), v.columns[v.sortCol]))

	v.renderHeader()
}

// CycleSortColumn cycles through sort columns
func (v *LeaderboardView) CycleSortColumn() {
	v.sortCol = (v.sortCol + 1) % len(v.columns)
	if v.sortCol == 0 || v.sortCol == 6 {
		v.sortCol = 1 // Skip rank and files columns
	}
}

// ReverseSortOrder reverses the sort order
func (v *LeaderboardView) ReverseSortOrder() {
	v.sortAsc = !v.sortAsc
}

// Root returns the root primitive
func (v *LeaderboardView) Root() tview.Primitive {
	return v.root
}

// GetFocusable returns the focusable component
func (v *LeaderboardView) GetFocusable() tview.Primitive {
	return v.table
}
