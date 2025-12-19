package views

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// PullRequestsView displays PR/merge statistics
type PullRequestsView struct {
	root      *tview.Flex
	summary   *tview.TextView
	table     *tview.Table
	info      *tview.TextView
	sortCol   int
	sortAsc   bool
	columns   []string
	repoStats *stats.Repository
	showPRs   bool // Toggle between author view and PR list
}

// NewPullRequestsView creates a new pull requests view
func NewPullRequestsView() *PullRequestsView {
	v := &PullRequestsView{
		sortCol: 1, // Default sort by merges
		sortAsc: false,
		columns: []string{"#", "Author", "Merges", "Changes", "PRs"},
		showPRs: false,
	}
	v.setup()
	return v
}

func (v *PullRequestsView) setup() {
	// Summary panel at top
	v.summary = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	v.summary.SetBorder(true).SetTitle(" Summary ")

	// Main table
	v.table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ')

	// Info bar
	v.info = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	v.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.summary, 7, 0, false).
		AddItem(v.table, 0, 1, true).
		AddItem(v.info, 1, 0, false)

	v.renderHeader()
}

func (v *PullRequestsView) renderHeader() {
	v.table.Clear()

	if v.showPRs {
		// PR list columns
		prColumns := []string{"#", "PR", "Branch", "Merged By", "Size", "Files", "Date"}
		for col, name := range prColumns {
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
	} else {
		// Author view columns
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
}

// Refresh updates the view with new data
func (v *PullRequestsView) Refresh(repo *stats.Repository) {
	v.repoStats = repo
	v.renderHeader()

	prStats := repo.PRStats
	if prStats == nil {
		v.summary.SetText("[red]No PR/merge data available[-]")
		return
	}

	// Update summary
	v.updateSummary(prStats)

	// Clear existing data rows
	for row := v.table.GetRowCount() - 1; row > 0; row-- {
		v.table.RemoveRow(row)
	}

	if v.showPRs {
		v.renderPRList(prStats)
	} else {
		v.renderAuthorView(prStats)
	}
}

func (v *PullRequestsView) updateSummary(prStats *stats.PRStatistics) {
	// Calculate averages
	avgSize := 0
	if prStats.TotalMerges > 0 {
		totalChanges := 0
		for _, pr := range prStats.PRList {
			totalChanges += pr.Additions + pr.Deletions
		}
		avgSize = totalChanges / prStats.TotalMerges
	}

	// Find busiest day
	busiestDay := ""
	maxMerges := 0
	for day, count := range prStats.DailyMerges {
		if count > maxMerges {
			maxMerges = count
			busiestDay = day
		}
	}

	var content string
	content += fmt.Sprintf("  [cyan]Total Merges:[-]      %d\n", prStats.TotalMerges)
	content += fmt.Sprintf("  [cyan]Identified PRs:[-]    %d (with PR# in message)\n", prStats.TotalPRs)
	content += fmt.Sprintf("  [cyan]Contributors:[-]      %d\n", len(prStats.MergesByAuthor))
	content += fmt.Sprintf("  [cyan]Avg PR Size:[-]       %d lines\n", avgSize)
	if busiestDay != "" {
		content += fmt.Sprintf("  [cyan]Busiest Day:[-]       %s (%d merges)\n", busiestDay, maxMerges)
	}

	v.summary.SetText(content)
}

func (v *PullRequestsView) renderAuthorView(prStats *stats.PRStatistics) {
	// Get sorted authors
	sortBy := []string{"", "name", "merges", "changes", ""}[v.sortCol]
	if sortBy == "" {
		sortBy = "merges"
	}
	authors := v.repoStats.GetPRLeaderboard(sortBy, v.sortAsc)

	for i, author := range authors {
		row := i + 1

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorDarkGray).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 1, tview.NewTableCell(author.Name).
			SetExpansion(1))

		v.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", author.MergeCount)).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", author.TotalChanges)).
			SetAlign(tview.AlignRight))

		prCount := len(author.PRNumbers)
		prText := fmt.Sprintf("%d", prCount)
		if prCount > 0 && prCount <= 3 {
			// Show PR numbers if few
			prText = fmt.Sprintf("%v", author.PRNumbers)
		}
		v.table.SetCell(row, 4, tview.NewTableCell(prText).
			SetTextColor(tcell.ColorAqua).
			SetAlign(tview.AlignRight))
	}

	// Update info
	toggleText := "[t] show PR list"
	v.info.SetText(fmt.Sprintf("[yellow]%d[-] contributors | %s | [s] sort, [r] reverse",
		len(authors), toggleText))
}

func (v *PullRequestsView) renderPRList(prStats *stats.PRStatistics) {
	// Get sorted PRs
	sortBy := []string{"", "", "", "name", "size", "files", "date"}
	if v.sortCol < len(sortBy) && sortBy[v.sortCol] != "" {
		// Use the sort
	}
	prs := v.repoStats.GetPRList("date", v.sortAsc, 100)

	// Sort locally based on column
	switch v.sortCol {
	case 1: // PR number
		sort.Slice(prs, func(i, j int) bool {
			if v.sortAsc {
				return prs[i].PRNumber < prs[j].PRNumber
			}
			return prs[i].PRNumber > prs[j].PRNumber
		})
	case 4: // Size
		sort.Slice(prs, func(i, j int) bool {
			si := prs[i].Additions + prs[i].Deletions
			sj := prs[j].Additions + prs[j].Deletions
			if v.sortAsc {
				return si < sj
			}
			return si > sj
		})
	case 5: // Files
		sort.Slice(prs, func(i, j int) bool {
			if v.sortAsc {
				return prs[i].FilesCount < prs[j].FilesCount
			}
			return prs[i].FilesCount > prs[j].FilesCount
		})
	case 6: // Date
		sort.Slice(prs, func(i, j int) bool {
			if v.sortAsc {
				return prs[i].MergedAt.Before(prs[j].MergedAt)
			}
			return prs[i].MergedAt.After(prs[j].MergedAt)
		})
	}

	for i, pr := range prs {
		row := i + 1

		v.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorDarkGray).
			SetAlign(tview.AlignRight))

		// PR number or "merge"
		prText := "merge"
		prColor := tcell.ColorWhite
		if pr.PRNumber > 0 {
			prText = fmt.Sprintf("#%d", pr.PRNumber)
			prColor = tcell.ColorAqua
		}
		v.table.SetCell(row, 1, tview.NewTableCell(prText).
			SetTextColor(prColor))

		// Branch (truncate if long)
		branch := pr.Branch
		if len(branch) > 25 {
			branch = branch[:22] + "..."
		}
		v.table.SetCell(row, 2, tview.NewTableCell(branch).
			SetExpansion(1))

		// Merged by (truncate)
		mergedBy := pr.MergedBy
		if len(mergedBy) > 15 {
			mergedBy = mergedBy[:12] + "..."
		}
		v.table.SetCell(row, 3, tview.NewTableCell(mergedBy))

		// Size
		size := pr.Additions + pr.Deletions
		sizeColor := tcell.ColorWhite
		if size > 1000 {
			sizeColor = tcell.ColorRed
		} else if size > 500 {
			sizeColor = tcell.ColorYellow
		}
		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", size)).
			SetTextColor(sizeColor).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%d", pr.FilesCount)).
			SetAlign(tview.AlignRight))

		v.table.SetCell(row, 6, tview.NewTableCell(pr.MergedAt.Format("2006-01-02")).
			SetTextColor(tcell.ColorDarkGray))
	}

	// Update info
	toggleText := "[t] show by author"
	v.info.SetText(fmt.Sprintf("[yellow]%d[-] merges | %s | [s] sort, [r] reverse",
		len(prs), toggleText))
}

// ToggleView switches between author view and PR list
func (v *PullRequestsView) ToggleView() {
	v.showPRs = !v.showPRs
	v.sortCol = 1
	v.sortAsc = false
	if v.repoStats != nil {
		v.Refresh(v.repoStats)
	}
}

// CycleSortColumn cycles through sort columns
func (v *PullRequestsView) CycleSortColumn() {
	if v.showPRs {
		// PR list: 7 columns
		v.sortCol = (v.sortCol + 1) % 7
		if v.sortCol == 0 || v.sortCol == 2 || v.sortCol == 3 {
			v.sortCol++ // Skip rank, branch, merged by
		}
	} else {
		// Author view
		v.sortCol = (v.sortCol + 1) % len(v.columns)
		if v.sortCol == 0 || v.sortCol == 4 {
			v.sortCol = 1 // Skip rank and PRs columns
		}
	}
}

// ReverseSortOrder reverses the sort order
func (v *PullRequestsView) ReverseSortOrder() {
	v.sortAsc = !v.sortAsc
}

// Root returns the root primitive
func (v *PullRequestsView) Root() tview.Primitive {
	return v.root
}

// GetFocusable returns the focusable component
func (v *PullRequestsView) GetFocusable() tview.Primitive {
	return v.table
}
