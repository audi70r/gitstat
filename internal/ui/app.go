package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/config"
	"github.com/audi70r/gitstat/internal/git"
	"github.com/audi70r/gitstat/internal/stats"
	"github.com/audi70r/gitstat/internal/ui/views"
)

// App represents the main application
type App struct {
	tview      *tview.Application
	pages      *tview.Pages
	config     *config.Config
	repoStats  *stats.Repository
	aggregator *stats.Aggregator

	// UI components
	setupView    *views.SetupView
	progressView *views.ProgressView
	mainView     *MainView
}

// NewApp creates a new application instance
func NewApp() *App {
	app := &App{
		tview:  tview.NewApplication(),
		pages:  tview.NewPages(),
		config: config.Default(),
	}

	// Set current directory as default
	cwd, err := os.Getwd()
	if err == nil {
		app.config.RepoPath = cwd
	}

	// Default date range: last year
	app.config.Until = time.Now()
	app.config.Since = app.config.Until.AddDate(-1, 0, 0)

	app.setupViews()
	return app
}

func (a *App) setupViews() {
	// Setup view
	a.setupView = views.NewSetupView(a.config, a.onSetupComplete, a.tview)

	// Progress view
	a.progressView = views.NewProgressView()

	// Main view (will be populated after scan)
	a.mainView = NewMainView(a.tview, a.onRescan, a.onMergeAuthors)

	// Add pages
	a.pages.AddPage("setup", a.setupView.Root(), true, true)
	a.pages.AddPage("progress", a.progressView.Root(), true, false)
	a.pages.AddPage("main", a.mainView.Root(), true, false)

	a.tview.SetRoot(a.pages, true)
}

func (a *App) onMergeAuthors(merges map[string]string) {
	if a.repoStats == nil || len(merges) == 0 {
		return
	}

	// Show progress
	a.progressView.SetStatus("Applying author merges...")
	a.progressView.SetProgress(0, 100)
	a.pages.SwitchToPage("progress")

	go func() {
		// Apply merges to the repository stats
		a.repoStats.ApplyAuthorMerges(merges)

		// Refresh all views and switch back, keeping focus on Authors view
		a.tview.QueueUpdateDraw(func() {
			a.mainView.RefreshAllViews()
			a.pages.SwitchToPage("main")
			a.mainView.FocusAuthorsView()
		})
	}()
}

func (a *App) onSetupComplete() {
	// Get repos to scan
	repos := a.config.RepoPaths
	if len(repos) == 0 && a.config.RepoPath != "" {
		repos = []string{a.config.RepoPath}
	}

	if len(repos) == 0 {
		a.setupView.ShowError("No repositories selected")
		return
	}

	// Validate all repos
	for _, path := range repos {
		if !git.IsGitRepo(path) {
			a.setupView.ShowError(fmt.Sprintf("Not a git repository: %s", path))
			return
		}
	}

	// Switch to progress view and start scanning
	a.pages.SwitchToPage("progress")
	go a.scanRepositories(repos)
}

func (a *App) scanRepositories(repos []string) {
	ctx := context.Background()

	// Estimate total commits across all repos
	totalEstimate := 0
	for _, repoPath := range repos {
		parser := git.NewParser(repoPath)
		estimate, _ := parser.EstimateCommitCount(ctx, a.config.Since, a.config.Until)
		if estimate > 0 {
			totalEstimate += estimate
		}
	}
	a.progressView.SetTotal(totalEstimate)

	// Create aggregator with combined path info
	dateRange := stats.DateRange{
		Since: a.config.Since,
		Until: a.config.Until,
	}

	combinedPath := repos[0]
	if len(repos) > 1 {
		combinedPath = fmt.Sprintf("%d repositories", len(repos))
	}
	a.aggregator = stats.NewAggregator(combinedPath, dateRange, a.config.Timezone)

	// Scan each repository
	totalCommits := 0
	totalCodebaseSize := 0

	for i, repoPath := range repos {
		repoName := filepath.Base(repoPath)

		a.tview.QueueUpdateDraw(func() {
			a.progressView.SetStatus(fmt.Sprintf("Scanning %s (%d/%d)...", repoName, i+1, len(repos)))
		})

		parser := git.NewParser(repoPath)

		// Parse commits from this repo
		err := parser.Parse(ctx, a.config.Since, a.config.Until,
			func(progress git.ScanProgress) {
				a.tview.QueueUpdateDraw(func() {
					a.progressView.SetProgress(totalCommits+progress.CommitsParsed, totalEstimate)
					if progress.CurrentHash != "" {
						a.progressView.SetStatus(fmt.Sprintf("[%s] Processing %s...", repoName, progress.CurrentHash))
					}
				})
			},
			func(commit *git.Commit) {
				a.aggregator.ProcessCommit(commit)
			},
		)

		if err != nil {
			a.tview.QueueUpdateDraw(func() {
				a.progressView.SetStatus(fmt.Sprintf("Error in %s: %v", repoName, err))
			})
			// Continue with other repos
		}

		// Update total commits processed
		totalCommits = a.aggregator.GetResult().TotalCommits

		// Calculate codebase size for this repo
		a.tview.QueueUpdateDraw(func() {
			a.progressView.SetStatus(fmt.Sprintf("Calculating size for %s...", repoName))
		})
		size, _ := git.GetCodebaseSize(repoPath)
		totalCodebaseSize += size
	}

	// Finalize statistics
	a.repoStats = a.aggregator.Finalize()
	a.repoStats.CodebaseSize = totalCodebaseSize

	// Switch to main view
	a.tview.QueueUpdateDraw(func() {
		a.mainView.SetData(a.repoStats, a.config)
		a.pages.SwitchToPage("main")
		a.tview.SetFocus(a.mainView.GetFocusable())
	})
}

func (a *App) onRescan() {
	a.pages.SwitchToPage("setup")
	a.tview.SetFocus(a.setupView.Root())
}

// Run starts the application
func (a *App) Run() error {
	return a.tview.Run()
}

// MainView is the main statistics display view
type MainView struct {
	root      *tview.Flex
	menuList  *tview.List
	viewPages *tview.Pages
	statusBar *tview.TextView
	header    *tview.TextView
	app       *tview.Application
	onRescan  func()
	onMerge   func(merges map[string]string)

	// Views
	leaderboardView *views.LeaderboardView
	codebaseView    *views.CodebaseView
	timelineView    *views.TimelineView
	heatmapView     *views.HeatmapView
	filesView       *views.FilesView
	hotspotsView    *views.HotspotsView
	ownershipView   *views.OwnershipView
	authorsView     *views.AuthorsView

	currentView string
	repoStats   *stats.Repository
	config      *config.Config
}

// NewMainView creates the main statistics view
func NewMainView(app *tview.Application, onRescan func(), onMerge func(map[string]string)) *MainView {
	m := &MainView{
		app:      app,
		onRescan: onRescan,
		onMerge:  onMerge,
	}

	m.setupLayout()
	return m
}

func (m *MainView) setupLayout() {
	// Create header
	m.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	m.header.SetBackgroundColor(tcell.ColorDarkBlue)

	// Create menu list
	m.menuList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)

	m.menuList.SetBorder(true).SetTitle(" Views ")

	menuItems := []struct {
		name     string
		shortcut rune
	}{
		{"Leaderboard", '1'},
		{"Codebase", '2'},
		{"Timeline", '3'},
		{"Work Hours", '4'},
		{"Top Files", '5'},
		{"Hotspots", '6'},
		{"Ownership", '7'},
		{"Authors", '8'},
	}

	for _, item := range menuItems {
		name := item.name
		m.menuList.AddItem(item.name, "", item.shortcut, func() {
			m.switchView(name)
		})
	}

	// Create view pages
	m.viewPages = tview.NewPages()
	m.viewPages.SetBorder(true)

	// Create individual views
	m.leaderboardView = views.NewLeaderboardView()
	m.codebaseView = views.NewCodebaseView()
	m.timelineView = views.NewTimelineView()
	m.heatmapView = views.NewHeatmapView()
	m.filesView = views.NewFilesView()
	m.hotspotsView = views.NewHotspotsView()
	m.ownershipView = views.NewOwnershipView()
	m.authorsView = views.NewAuthorsView(m.onMerge)

	// Add views to pages
	m.viewPages.AddPage("Leaderboard", m.leaderboardView.Root(), true, true)
	m.viewPages.AddPage("Codebase", m.codebaseView.Root(), true, false)
	m.viewPages.AddPage("Timeline", m.timelineView.Root(), true, false)
	m.viewPages.AddPage("Work Hours", m.heatmapView.Root(), true, false)
	m.viewPages.AddPage("Top Files", m.filesView.Root(), true, false)
	m.viewPages.AddPage("Hotspots", m.hotspotsView.Root(), true, false)
	m.viewPages.AddPage("Ownership", m.ownershipView.Root(), true, false)
	m.viewPages.AddPage("Authors", m.authorsView.Root(), true, false)

	m.currentView = "Leaderboard"
	m.viewPages.SetTitle(" Leaderboard ")

	// Create status bar
	m.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	m.statusBar.SetBackgroundColor(tcell.ColorDarkBlue)
	m.updateStatusBar()

	// Create content area (menu + views)
	contentFlex := tview.NewFlex().
		AddItem(m.menuList, 18, 0, true).
		AddItem(m.viewPages, 0, 1, false)

	// Create main layout
	m.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.header, 1, 0, false).
		AddItem(contentFlex, 0, 1, true).
		AddItem(m.statusBar, 1, 0, false)

	// Set up input handling
	m.root.SetInputCapture(m.handleInput)
}

func (m *MainView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab, tcell.KeyBacktab:
		m.toggleFocus()
		return nil
	case tcell.KeyEsc:
		if m.app.GetFocus() != m.menuList {
			m.app.SetFocus(m.menuList)
			return nil
		}
	}

	switch event.Rune() {
	case 'q', 'Q':
		m.app.Stop()
		return nil
	case 'R':
		if m.onRescan != nil {
			m.onRescan()
		}
		return nil
	case 's', 'S':
		m.cycleSortColumn()
		return nil
	case 'r':
		m.reverseSortOrder()
		return nil
	case '?':
		m.showHelp()
		return nil
	}

	return event
}

func (m *MainView) toggleFocus() {
	if m.app.GetFocus() == m.menuList {
		// Focus the active view's focusable component
		switch m.currentView {
		case "Leaderboard":
			m.app.SetFocus(m.leaderboardView.GetFocusable())
		case "Top Files":
			m.app.SetFocus(m.filesView.GetFocusable())
		case "Hotspots":
			m.app.SetFocus(m.hotspotsView.GetFocusable())
		case "Ownership":
			m.app.SetFocus(m.ownershipView.GetFocusable())
		case "Authors":
			m.app.SetFocus(m.authorsView.GetFocusable())
		}
	} else {
		m.app.SetFocus(m.menuList)
	}
}

func (m *MainView) cycleSortColumn() {
	switch m.currentView {
	case "Leaderboard":
		m.leaderboardView.CycleSortColumn()
		m.leaderboardView.Refresh(m.repoStats)
	case "Top Files":
		m.filesView.CycleSortColumn()
		m.filesView.Refresh(m.repoStats)
	case "Hotspots":
		m.hotspotsView.CycleSortColumn()
		m.hotspotsView.Refresh(m.repoStats)
	case "Ownership":
		m.ownershipView.CycleSortColumn()
		m.ownershipView.Refresh(m.repoStats)
	}
}

func (m *MainView) reverseSortOrder() {
	switch m.currentView {
	case "Leaderboard":
		m.leaderboardView.ReverseSortOrder()
		m.leaderboardView.Refresh(m.repoStats)
	case "Top Files":
		m.filesView.ReverseSortOrder()
		m.filesView.Refresh(m.repoStats)
	case "Hotspots":
		m.hotspotsView.ReverseSortOrder()
		m.hotspotsView.Refresh(m.repoStats)
	case "Ownership":
		m.ownershipView.ReverseSortOrder()
		m.ownershipView.Refresh(m.repoStats)
	}
}

func (m *MainView) showHelp() {
	// Could show a modal with help text
}

func (m *MainView) switchView(name string) {
	m.currentView = name
	m.viewPages.SwitchToPage(name)
	m.viewPages.SetTitle(" " + name + " ")
	m.updateStatusBar()
}

// updateStatusBar shows context-sensitive controls
func (m *MainView) updateStatusBar() {
	baseControls := "[yellow]Tab[-] Focus  [yellow]↑↓[-] Navigate  [yellow]R[-] Rescan  [yellow]q[-] Quit"

	var viewControls string
	switch m.currentView {
	case "Leaderboard", "Top Files", "Hotspots", "Ownership":
		viewControls = "[yellow]s[-] Sort  [yellow]r[-] Reverse  "
	case "Authors":
		viewControls = "[yellow]Space[-] Select  [yellow]m[-] Merge  [yellow]a[-] Apply  [yellow]c[-] Clear  "
	default:
		viewControls = ""
	}

	m.statusBar.SetText(viewControls + baseControls)
}

// SetData updates all views with repository statistics
func (m *MainView) SetData(repoStats *stats.Repository, cfg *config.Config) {
	m.repoStats = repoStats
	m.config = cfg

	// Update header
	repoName := filepath.Base(repoStats.Path)
	dateRange := fmt.Sprintf("%s to %s",
		cfg.Since.Format("2006-01-02"),
		cfg.Until.Format("2006-01-02"))
	m.header.SetText(fmt.Sprintf("[::b]GitStat[-:-:-] - %s (%s) - %d commits by %d authors",
		repoName, dateRange, repoStats.TotalCommits, repoStats.TotalAuthors))

	// Refresh all views
	m.leaderboardView.Refresh(repoStats)
	m.codebaseView.Refresh(repoStats)
	m.timelineView.Refresh(repoStats)
	m.heatmapView.Refresh(repoStats, cfg.Timezone)
	m.filesView.Refresh(repoStats)
	m.hotspotsView.Refresh(repoStats)
	m.ownershipView.Refresh(repoStats)
	m.authorsView.Refresh(repoStats)
}

// RefreshAllViews refreshes all views after merge operations
func (m *MainView) RefreshAllViews() {
	if m.repoStats == nil || m.config == nil {
		return
	}
	m.SetData(m.repoStats, m.config)
}

// Root returns the root primitive
func (m *MainView) Root() tview.Primitive {
	return m.root
}

// GetFocusable returns the focusable component
func (m *MainView) GetFocusable() tview.Primitive {
	return m.menuList
}

// FocusAuthorsView sets focus on the Authors view
func (m *MainView) FocusAuthorsView() {
	m.switchView("Authors")
	m.app.SetFocus(m.authorsView.GetFocusable())
}
