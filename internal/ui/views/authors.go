package views

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/stats"
)

// AuthorMerge represents a merged author identity
type AuthorMerge struct {
	PrimaryEmail string
	PrimaryName  string
	AliasEmails  []string
	AliasNames   []string
}

// AuthorsView allows managing and merging author identities
type AuthorsView struct {
	root        *tview.Flex
	list        *tview.List
	detail      *tview.TextView
	info        *tview.TextView
	authors     []*stats.AuthorStats
	merges      map[string]string // email -> primary email
	selected    map[string]bool   // selected emails for batch operations
	repoStats   *stats.Repository
	onMerge     func(merges map[string]string)
	selectedIdx int
}

// NewAuthorsView creates a new authors management view
func NewAuthorsView(onMerge func(merges map[string]string)) *AuthorsView {
	v := &AuthorsView{
		merges:   make(map[string]string),
		selected: make(map[string]bool),
		onMerge:  onMerge,
	}
	v.setup()
	return v
}

func (v *AuthorsView) setup() {
	// Instructions header
	instructions := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]MERGE AUTHORS:[-] [Space] select  [m] merge selected  [a] apply  [c] clear")

	// Authors list
	v.list = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	v.list.SetBorder(true).SetTitle(" Authors ")

	// Detail/merge panel
	v.detail = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	v.detail.SetBorder(true).SetTitle(" Author Details ")

	// Info bar
	v.info = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Layout
	content := tview.NewFlex().
		AddItem(v.list, 45, 0, true).
		AddItem(v.detail, 0, 1, false)

	v.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(instructions, 1, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(v.info, 1, 0, false)

	// Handle selection
	v.list.SetChangedFunc(func(idx int, main, secondary string, shortcut rune) {
		v.selectedIdx = idx
		if idx >= 0 && idx < len(v.authors) {
			v.showAuthorDetails(v.authors[idx])
		}
	})

	// Input handler for merging
	v.list.SetInputCapture(v.handleInput)
}

func (v *AuthorsView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case ' ':
			// Toggle selection for batch operations
			v.toggleSelection()
			return nil
		case 'm', 'M':
			// Mark for merge: if selected items exist, merge them
			// Otherwise, mark current author for sequential merging
			if len(v.selected) >= 2 {
				v.mergeSelected()
			} else {
				v.markForMerge()
			}
			return nil
		case 'c', 'C':
			// Clear all selections and merges
			v.clearAll()
			return nil
		case 'a', 'A':
			// Apply merges
			v.applyMerges()
			return nil
		}
	}
	return event
}

func (v *AuthorsView) markForMerge() {
	if v.selectedIdx < 0 || v.selectedIdx >= len(v.authors) {
		return
	}

	selected := v.authors[v.selectedIdx]

	// Check if there's already a primary
	var primaryEmail string
	for email, target := range v.merges {
		if email == target {
			primaryEmail = email
			break
		}
	}

	if primaryEmail == "" {
		// No primary yet - mark this as primary
		v.merges[selected.Email] = selected.Email
	} else if selected.Email != primaryEmail {
		// Add as alias of the primary
		v.merges[selected.Email] = primaryEmail
	}

	v.refreshList()
}

func (v *AuthorsView) toggleSelection() {
	if v.selectedIdx < 0 || v.selectedIdx >= len(v.authors) {
		return
	}

	author := v.authors[v.selectedIdx]
	if v.selected[author.Email] {
		delete(v.selected, author.Email)
	} else {
		v.selected[author.Email] = true
	}

	v.refreshList()
}

func (v *AuthorsView) mergeSelected() {
	if len(v.selected) < 2 {
		// Need at least 2 selected to merge
		return
	}

	// Find the author with most commits among selected - that's the primary
	var primaryEmail string
	maxCommits := 0
	for email := range v.selected {
		for _, a := range v.authors {
			if a.Email == email && a.Commits > maxCommits {
				maxCommits = a.Commits
				primaryEmail = email
			}
		}
	}

	if primaryEmail == "" {
		return
	}

	// Set up merges
	for email := range v.selected {
		if email == primaryEmail {
			v.merges[email] = email // Primary points to itself
		} else {
			v.merges[email] = primaryEmail // Aliases point to primary
		}
	}

	// Clear selections
	v.selected = make(map[string]bool)
	v.refreshList()
}

func (v *AuthorsView) clearAll() {
	v.selected = make(map[string]bool)
	v.merges = make(map[string]string)
	v.refreshList()
}

func (v *AuthorsView) applyMerges() {
	if len(v.merges) == 0 {
		return
	}
	if v.onMerge != nil {
		// Clear local state after applying
		mergesCopy := make(map[string]string)
		for k, val := range v.merges {
			mergesCopy[k] = val
		}
		v.merges = make(map[string]string)
		v.selected = make(map[string]bool)
		v.onMerge(mergesCopy)
	}
}

// Refresh updates the view with new data
func (v *AuthorsView) Refresh(repo *stats.Repository) {
	v.repoStats = repo
	v.refreshList()
}

func (v *AuthorsView) refreshList() {
	v.list.Clear()

	// Get authors sorted by commits
	v.authors = v.repoStats.GetLeaderboard("commits", false)

	// Find pending merges for display
	var primaryEmail string
	for email, target := range v.merges {
		if email == target {
			primaryEmail = email
		}
	}

	for _, author := range v.authors {
		status := ""
		mainText := author.Name

		// Check if selected
		if v.selected[author.Email] {
			mainText = fmt.Sprintf("[blue]◉ %s[-]", author.Name)
			status = " [blue]SELECTED[-]"
		}

		// Check if part of merge
		if email, ok := v.merges[author.Email]; ok {
			if email == author.Email {
				status = " [green]★ PRIMARY[-]"
				mainText = fmt.Sprintf("[green]%s[-]", author.Name)
			} else {
				status = fmt.Sprintf(" [yellow]→ %s[-]", getPrimaryName(v.authors, primaryEmail))
				mainText = fmt.Sprintf("[yellow]%s[-]", author.Name)
			}
		}

		secondary := fmt.Sprintf("<%s> %d commits%s", author.Email, author.Commits, status)
		v.list.AddItem(mainText, secondary, 0, nil)
	}

	// Count pending merges
	mergeCount := 0
	for email, target := range v.merges {
		if email != target {
			mergeCount++
		}
	}

	// Count pending primary
	hasPrimary := false
	for email, target := range v.merges {
		if email == target {
			hasPrimary = true
			break
		}
	}

	// Dynamic help text based on state
	var helpText string
	if len(v.selected) >= 2 {
		helpText = fmt.Sprintf("[m] MERGE %d selected | [c] clear", len(v.selected))
	} else if mergeCount > 0 {
		helpText = fmt.Sprintf("[green][a] APPLY %d merge(s)[-] | [m] add more | [c] clear", mergeCount)
	} else if hasPrimary {
		helpText = "[m] add alias to PRIMARY | [a] apply | [c] clear"
	} else {
		helpText = "[m] mark as PRIMARY (first), then [m] on aliases"
	}

	v.info.SetText(fmt.Sprintf("[yellow]%d[-] authors | %s", len(v.authors), helpText))

	// Re-select current item
	if v.selectedIdx >= 0 && v.selectedIdx < len(v.authors) {
		v.list.SetCurrentItem(v.selectedIdx)
		v.showAuthorDetails(v.authors[v.selectedIdx])
	}
}

func getPrimaryName(authors []*stats.AuthorStats, email string) string {
	for _, a := range authors {
		if a.Email == email {
			return a.Name
		}
	}
	return email
}

func (v *AuthorsView) showAuthorDetails(author *stats.AuthorStats) {
	var content string

	content += fmt.Sprintf("[::b]%s[-:-:-]\n", author.Name)
	content += fmt.Sprintf("Email: [cyan]%s[-]\n\n", author.Email)

	content += "[yellow]━━━ Statistics ━━━[-]\n\n"
	content += fmt.Sprintf("  Commits:     [cyan]%d[-]\n", author.Commits)
	content += fmt.Sprintf("  Additions:   [green]+%d[-]\n", author.Additions)
	content += fmt.Sprintf("  Deletions:   [red]-%d[-]\n", author.Deletions)
	content += fmt.Sprintf("  Files:       [cyan]%d[-]\n", len(author.FilesTouched))

	if !author.FirstCommit.IsZero() {
		content += fmt.Sprintf("\n  First:       [gray]%s[-]\n", author.FirstCommit.Format("2006-01-02"))
		content += fmt.Sprintf("  Last:        [gray]%s[-]\n", author.LastCommit.Format("2006-01-02"))
	}

	// Show similar authors (potential merge candidates)
	content += "\n[yellow]━━━ Similar Authors ━━━[-]\n\n"
	similar := findSimilarAuthors(v.authors, author)
	if len(similar) > 0 {
		for _, s := range similar {
			content += fmt.Sprintf("  • %s <%s>\n", s.Name, s.Email)
		}
		content += "\n[gray]Press [m] to merge selected authors[-]\n"
	} else {
		content += "  [gray]No similar authors found[-]\n"
	}

	// Show merge status
	if email, ok := v.merges[author.Email]; ok {
		content += "\n[yellow]━━━ Merge Status ━━━[-]\n\n"
		if email == author.Email {
			content += "  [green]This is the PRIMARY identity[-]\n"
			content += "  Other authors will be merged into this one.\n"
		} else {
			content += fmt.Sprintf("  [yellow]Will be merged into: %s[-]\n", getPrimaryName(v.authors, email))
		}
	}

	v.detail.SetText(content)
}

func findSimilarAuthors(authors []*stats.AuthorStats, target *stats.AuthorStats) []*stats.AuthorStats {
	var similar []*stats.AuthorStats

	targetNameLower := toLowerCase(target.Name)
	targetEmailPrefix := getEmailPrefix(target.Email)

	for _, a := range authors {
		if a.Email == target.Email {
			continue
		}

		// Check name similarity
		nameLower := toLowerCase(a.Name)
		emailPrefix := getEmailPrefix(a.Email)

		// Same first name or similar email prefix
		if containsWord(nameLower, targetNameLower) ||
			containsWord(targetNameLower, nameLower) ||
			emailPrefix == targetEmailPrefix {
			similar = append(similar, a)
		}
	}

	// Sort by commits
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].Commits > similar[j].Commits
	})

	// Limit to top 5
	if len(similar) > 5 {
		similar = similar[:5]
	}

	return similar
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func getEmailPrefix(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}

func containsWord(s, word string) bool {
	if len(word) == 0 || len(s) == 0 {
		return false
	}
	// Check if first 3 chars match (handles "John" and "Johnny")
	if len(word) >= 3 && len(s) >= 3 {
		return s[:3] == word[:3]
	}
	return s == word
}

// SetMerges sets the current merge mappings
func (v *AuthorsView) SetMerges(merges map[string]string) {
	v.merges = make(map[string]string)
	for k, val := range merges {
		v.merges[k] = val
	}
}

// GetMerges returns the current merge mappings
func (v *AuthorsView) GetMerges() map[string]string {
	result := make(map[string]string)
	for k, val := range v.merges {
		result[k] = val
	}
	return result
}

// Root returns the root primitive
func (v *AuthorsView) Root() tview.Primitive {
	return v.root
}

// GetFocusable returns the focusable component
func (v *AuthorsView) GetFocusable() tview.Primitive {
	return v.list
}
