package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ProgressView displays scanning progress
type ProgressView struct {
	root        *tview.Flex
	progressBar *tview.TextView
	statusText  *tview.TextView
	countText   *tview.TextView
	total       int
	current     int
}

// NewProgressView creates a new progress view
func NewProgressView() *ProgressView {
	p := &ProgressView{}
	p.setup()
	return p
}

func (p *ProgressView) setup() {
	// Title
	title := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[::b]Scanning Repository[-:-:-]")
	title.SetBackgroundColor(tcell.ColorDarkBlue)

	// Progress bar container
	p.progressBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Status text
	p.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Count text
	p.countText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Progress container
	progressContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(p.statusText, 2, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(p.progressBar, 3, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(p.countText, 1, 0, false).
		AddItem(nil, 0, 1, false)

	// Center the progress area
	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(progressContainer, 60, 0, false).
		AddItem(nil, 0, 1, false)

	p.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(centered, 0, 1, false)

	p.SetProgress(0, 0)
}

// SetTotal sets the total number of commits to process
func (p *ProgressView) SetTotal(total int) {
	p.total = total
}

// SetProgress updates the progress display
func (p *ProgressView) SetProgress(current, total int) {
	p.current = current
	if total > 0 {
		p.total = total
	}

	// Calculate percentage
	var pct float64
	if p.total > 0 {
		pct = float64(current) / float64(p.total) * 100
	}

	// Build progress bar
	barWidth := 50
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	p.progressBar.SetText(fmt.Sprintf("[green]%s[-]\n%.1f%%", bar, pct))

	// Update count
	if p.total > 0 {
		p.countText.SetText(fmt.Sprintf("[yellow]%d[-] / [yellow]%d[-] commits processed", current, p.total))
	} else {
		p.countText.SetText(fmt.Sprintf("[yellow]%d[-] commits processed", current))
	}
}

// SetStatus updates the status message
func (p *ProgressView) SetStatus(status string) {
	p.statusText.SetText("[white]" + status + "[-]")
}

// Root returns the root primitive
func (p *ProgressView) Root() tview.Primitive {
	return p.root
}
