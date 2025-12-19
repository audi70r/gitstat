package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// OwnershipView displays directory ownership with visual breakdown
type OwnershipView struct {
	root      *tview.Flex
	list      *tview.List
	detail    *tview.TextView
	info      *tview.TextView
	dirs      []*stats.DirStats
	sortCol   int
	sortAsc   bool
	columns   []string
	repoStats *stats.Repository
}

// NewOwnershipView creates a new ownership view
func NewOwnershipView() *OwnershipView {
	v := &OwnershipView{
		sortCol: 1, // Default sort by changes
		sortAsc: false,
		columns: []string{"path", "changes", "authors"},
	}
	v.setup()
	return v
}

func (v *OwnershipView) setup() {
	// Directory list on the left
	v.list = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	v.list.SetBorder(true).SetTitle(" Directories ")

	// Detail view on the right
	v.detail = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	v.detail.SetBorder(true).SetTitle(" Ownership Details ")

	// Info bar at bottom
	v.info = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Layout: list on left, details on right
	content := tview.NewFlex().
		AddItem(v.list, 35, 0, true).
		AddItem(v.detail, 0, 1, false)

	v.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(v.info, 1, 0, false)

	// Handle list selection
	v.list.SetChangedFunc(func(idx int, main, secondary string, shortcut rune) {
		if idx >= 0 && idx < len(v.dirs) {
			v.showDirectoryDetails(v.dirs[idx])
		}
	})
}

// Refresh updates the view with new data
func (v *OwnershipView) Refresh(repo *stats.Repository) {
	v.repoStats = repo
	v.list.Clear()

	// Get sorted directories
	sortBy := v.columns[v.sortCol]
	v.dirs = repo.GetOwnership(sortBy, v.sortAsc)

	// Populate list
	for _, dir := range v.dirs {
		dirName := dir.Path
		if dirName == "." {
			dirName = "(root files)"
		}

		// Secondary text with quick stats
		authorCount := len(dir.Authors)
		secondary := fmt.Sprintf("%s changes, %d authors", formatChanges(dir.TotalChanges), authorCount)

		v.list.AddItem(dirName, secondary, 0, nil)
	}

	// Select first item
	if len(v.dirs) > 0 {
		v.list.SetCurrentItem(0)
		v.showDirectoryDetails(v.dirs[0])
	}

	// Update info
	v.info.SetText(fmt.Sprintf("[yellow]%d[-] directories | [s] sort by: [green]%s[-] | [r] reverse order",
		len(v.dirs), v.columns[v.sortCol]))
}

func (v *OwnershipView) showDirectoryDetails(dir *stats.DirStats) {
	var sb strings.Builder

	dirName := dir.Path
	if dirName == "." {
		dirName = "(root files)"
	}

	sb.WriteString(fmt.Sprintf("[::b]%s[-:-:-]\n\n", dirName))

	// Stats summary
	sb.WriteString(fmt.Sprintf("[yellow]━━━ Overview ━━━[-]\n\n"))
	sb.WriteString(fmt.Sprintf("  Total Changes:  [cyan]%s[-] lines\n", formatChanges(dir.TotalChanges)))
	sb.WriteString(fmt.Sprintf("  Total Touches:  [cyan]%d[-] commits\n", dir.TouchCount))
	sb.WriteString(fmt.Sprintf("  Contributors:   [cyan]%d[-] authors\n", len(dir.Authors)))

	// Ownership breakdown
	if len(dir.Authors) > 0 {
		sb.WriteString(fmt.Sprintf("\n[yellow]━━━ Ownership Breakdown ━━━[-]\n\n"))

		// Sort authors by share
		authors := make([]*stats.DirAuthorStats, 0, len(dir.Authors))
		for _, a := range dir.Authors {
			authors = append(authors, a)
		}
		sort.Slice(authors, func(i, j int) bool {
			return authors[i].Share > authors[j].Share
		})

		// Calculate max name length for alignment
		maxNameLen := 0
		for _, a := range authors {
			if len(a.Name) > maxNameLen {
				maxNameLen = len(a.Name)
			}
		}
		if maxNameLen > 20 {
			maxNameLen = 20
		}

		// Display each author with a visual bar
		barWidth := 30
		for i, author := range authors {
			name := author.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}

			// Ownership bar
			filled := int(author.Share / 100 * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}

			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			// Color based on ownership
			barColor := getOwnershipColor(author.Share)

			// Rank indicator
			rank := ""
			switch i {
			case 0:
				rank = "[gold]★[-] " // Primary owner
			case 1:
				rank = "[silver]☆[-] "
			case 2:
				rank = "[#CD7F32]☆[-] " // Bronze
			default:
				rank = "  "
			}

			sb.WriteString(fmt.Sprintf("  %s%-*s [%s]%s[-] [white]%5.1f%%[-] (%d commits)\n",
				rank, maxNameLen, name, barColor, bar, author.Share, author.Commits))
		}

		// Ownership concentration indicator
		sb.WriteString(fmt.Sprintf("\n[yellow]━━━ Analysis ━━━[-]\n\n"))

		if len(authors) > 0 {
			topOwnership := authors[0].Share
			concentrationIndicator := getConcentrationIndicator(topOwnership, len(authors))
			sb.WriteString(fmt.Sprintf("  Ownership Type:   %s\n", concentrationIndicator))

			// Bus factor estimation
			busFactor := estimateBusFactor(authors)
			busfactorColor := "red"
			if busFactor >= 3 {
				busfactorColor = "green"
			} else if busFactor >= 2 {
				busfactorColor = "yellow"
			}
			sb.WriteString(fmt.Sprintf("  Bus Factor:       [%s]%d[-] (contributors with >10%% ownership)\n",
				busfactorColor, busFactor))
		}
	}

	v.detail.SetText(sb.String())
	v.detail.SetTitle(fmt.Sprintf(" %s ", dirName))
}

func getOwnershipColor(share float64) string {
	if share >= 60 {
		return "green"
	} else if share >= 30 {
		return "cyan"
	} else if share >= 10 {
		return "yellow"
	}
	return "gray"
}

func getConcentrationIndicator(topShare float64, authorCount int) string {
	if topShare >= 80 {
		return "[yellow]Single Owner[-] (one person owns >80%)"
	} else if topShare >= 60 {
		return "[cyan]Concentrated[-] (primary owner with >60%)"
	} else if authorCount <= 2 {
		return "[green]Shared[-] (2 primary contributors)"
	} else if topShare >= 40 {
		return "[green]Collaborative[-] (lead contributor <60%)"
	}
	return "[blue]Distributed[-] (many contributors)"
}

func estimateBusFactor(authors []*stats.DirAuthorStats) int {
	count := 0
	for _, a := range authors {
		if a.Share >= 10 {
			count++
		}
	}
	return count
}

func formatChanges(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// CycleSortColumn cycles through sort columns
func (v *OwnershipView) CycleSortColumn() {
	v.sortCol = (v.sortCol + 1) % len(v.columns)
}

// ReverseSortOrder reverses the sort order
func (v *OwnershipView) ReverseSortOrder() {
	v.sortAsc = !v.sortAsc
}

// Root returns the root primitive
func (v *OwnershipView) Root() tview.Primitive {
	return v.root
}

// GetFocusable returns the focusable component
func (v *OwnershipView) GetFocusable() tview.Primitive {
	return v.list
}
