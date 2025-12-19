package views

import (
	"fmt"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// FilesView displays top changed files
type FilesView struct {
	root    *tview.Flex
	table   *tview.Table
	info    *tview.TextView
	sortCol int
	sortAsc bool
	columns []string
}

// NewFilesView creates a new files view
func NewFilesView() *FilesView {
	v := &FilesView{
		sortCol: 2, // Default sort by changes
		sortAsc: false,
		columns: []string{"#", "File", "Changes", "Touches", "Authors", "+Lines", "-Lines"},
	}
	v.setup()
	return v
}

func (v *FilesView) setup() {
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

func (v *FilesView) renderHeader() {
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
func (v *FilesView) Refresh(repo *stats.Repository) {
	// Clear existing data rows
	for row := v.table.GetRowCount() - 1; row > 0; row-- {
		v.table.RemoveRow(row)
	}

	// Get sorted files
	sortBy := []string{"", "path", "changes", "touches", "authors", "changes", "changes"}[v.sortCol]
	if sortBy == "" {
		sortBy = "changes"
	}
	files := repo.GetTopFiles(sortBy, v.sortAsc, 50)

	// Render data
	for i, file := range files {
		row := i + 1

		// Truncate long paths
		displayPath := file.Path
		if len(displayPath) > 50 {
			displayPath = "..." + displayPath[len(displayPath)-47:]
		}

		// Color based on directory
		dir := filepath.Dir(file.Path)
		pathColor := getDirColor(dir)

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorDarkGray).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 1, tview.NewTableCell(displayPath).
			SetTextColor(pathColor).
			SetExpansion(1))

		v.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", file.TotalChanges)).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", file.TouchCount)).
			SetAlign(tview.AlignRight))

		authorCount := len(file.Authors)
		authorColor := tcell.ColorWhite
		if authorCount >= 5 {
			authorColor = tcell.ColorRed
		} else if authorCount >= 3 {
			authorColor = tcell.ColorYellow
		}
		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", authorCount)).
			SetTextColor(authorColor).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("+%d", file.Additions)).
			SetTextColor(tcell.ColorGreen).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("-%d", file.Deletions)).
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignRight))
	}

	// Update info
	v.info.SetText(fmt.Sprintf("[yellow]%d[-] files shown (of %d) | Sort: [green]%s[-] | [s] cycle column, [r] reverse",
		len(files), len(repo.FileStats), v.columns[v.sortCol]))

	v.renderHeader()
}

func getDirColor(dir string) tcell.Color {
	// Simple hash-based coloring for directories
	colors := []tcell.Color{
		tcell.ColorLightCyan,
		tcell.ColorLightGreen,
		tcell.ColorLightYellow,
		tcell.ColorLightBlue,
		tcell.ColorWhite,
	}

	hash := 0
	for _, c := range dir {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}

	return colors[hash%len(colors)]
}

// CycleSortColumn cycles through sort columns
func (v *FilesView) CycleSortColumn() {
	v.sortCol = (v.sortCol + 1) % len(v.columns)
	if v.sortCol == 0 {
		v.sortCol = 1
	}
}

// ReverseSortOrder reverses the sort order
func (v *FilesView) ReverseSortOrder() {
	v.sortAsc = !v.sortAsc
}

// Root returns the root primitive
func (v *FilesView) Root() tview.Primitive {
	return v.root
}

// GetFocusable returns the focusable component
func (v *FilesView) GetFocusable() tview.Primitive {
	return v.table
}
