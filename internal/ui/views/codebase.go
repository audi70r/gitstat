package views

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// CodebaseView displays overall codebase statistics
type CodebaseView struct {
	root *tview.Flex
	text *tview.TextView
}

// NewCodebaseView creates a new codebase view
func NewCodebaseView() *CodebaseView {
	v := &CodebaseView{}
	v.setup()
	return v
}

func (v *CodebaseView) setup() {
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
func (v *CodebaseView) Refresh(repo *stats.Repository) {
	cbStats := repo.GetCodebaseStats()

	totalChanges := cbStats.TotalAdditions + cbStats.TotalDeletions
	var addPct, delPct float64
	if totalChanges > 0 {
		addPct = float64(cbStats.TotalAdditions) / float64(totalChanges) * 100
		delPct = float64(cbStats.TotalDeletions) / float64(totalChanges) * 100
	}

	// Build visual bar
	barWidth := 50
	addBar := int(addPct / 100 * float64(barWidth))
	delBar := barWidth - addBar

	addBarStr := ""
	delBarStr := ""
	for i := 0; i < addBar; i++ {
		addBarStr += "█"
	}
	for i := 0; i < delBar; i++ {
		delBarStr += "█"
	}

	// Calculate churn rate (refactoring indicator)
	churnIndicator := getChurnIndicator(cbStats.RefactoredPercent)

	content := fmt.Sprintf(`[::b]Codebase Changes Overview[-:-:-]

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Summary[-:-:-]

  Total Commits:      [cyan]%d[-]
  Total Authors:      [cyan]%d[-]
  Files Modified:     [cyan]%d[-]

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Codebase Size[-:-:-]

  Current Size:       [cyan]%s[-] lines
  Churn Rate:         [%s]%.1f%%[-] of codebase touched
  Churn Level:        %s

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Lines Changed[-:-:-]

  [green]+ Additions:[-]        [green]%s[-] lines
  [red]- Deletions:[-]        [red]%s[-] lines
  [white]= Total Changes:[-]    [white]%s[-] lines

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Change Distribution[-:-:-]

  [green]%s[-][red]%s[-]

  [green]%.1f%% additions[-]  |  [red]%.1f%% deletions[-]

[yellow]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[-]

  [::b]Activity Metrics[-:-:-]

  Net Change:         [%s]%+d[-] lines
  Avg per Commit:     [cyan]%.1f[-] lines
  Avg per Author:     [cyan]%.1f[-] lines

`,
		repo.TotalCommits,
		repo.TotalAuthors,
		cbStats.FilesModified,
		formatNumber(cbStats.CodebaseSize),
		getChurnColor(cbStats.RefactoredPercent),
		cbStats.RefactoredPercent,
		churnIndicator,
		formatNumber(cbStats.TotalAdditions),
		formatNumber(cbStats.TotalDeletions),
		formatNumber(totalChanges),
		addBarStr,
		delBarStr,
		addPct,
		delPct,
		getNetColor(cbStats.TotalAdditions-cbStats.TotalDeletions),
		cbStats.TotalAdditions-cbStats.TotalDeletions,
		safeDivide(float64(totalChanges), float64(repo.TotalCommits)),
		safeDivide(float64(totalChanges), float64(repo.TotalAuthors)),
	)

	v.text.SetText(content)
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func getNetColor(net int) string {
	if net > 0 {
		return "green"
	} else if net < 0 {
		return "red"
	}
	return "white"
}

func getChurnColor(pct float64) string {
	if pct >= 100 {
		return "red"
	} else if pct >= 50 {
		return "yellow"
	}
	return "cyan"
}

func getChurnIndicator(pct float64) string {
	if pct >= 200 {
		return "[red]Very High[-] (major rewrite)"
	} else if pct >= 100 {
		return "[red]High[-] (significant refactoring)"
	} else if pct >= 50 {
		return "[yellow]Moderate[-] (active development)"
	} else if pct >= 20 {
		return "[cyan]Normal[-] (healthy activity)"
	}
	return "[green]Low[-] (stable codebase)"
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// Root returns the root primitive
func (v *CodebaseView) Root() tview.Primitive {
	return v.root
}
