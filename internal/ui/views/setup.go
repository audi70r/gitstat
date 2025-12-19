package views

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/audi70r/gitstat/internal/config"
	"github.com/audi70r/gitstat/internal/git"
)

// SetupView handles directory and date range selection
type SetupView struct {
	root        *tview.Pages
	mainFlex    *tview.Flex
	repoList    *tview.List
	sinceInput  *tview.InputField
	untilInput  *tview.InputField
	errorText   *tview.TextView
	config      *config.Config
	onComplete  func()
	currentPath string
	app         *tview.Application
}

// NewSetupView creates a new setup view
func NewSetupView(cfg *config.Config, onComplete func(), app *tview.Application) *SetupView {
	s := &SetupView{
		config:     cfg,
		onComplete: onComplete,
		app:        app,
	}
	// Set initial path
	s.currentPath, _ = os.Getwd()
	s.setup()
	return s
}

func (s *SetupView) setup() {
	// Title
	title := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[::b]GitStat - Git Repository Analyzer[-:-:-]")
	title.SetBackgroundColor(tcell.ColorDarkBlue)

	// Instructions
	instructions := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]Select one or more repositories to analyze[-]")

	// Repository list
	s.repoList = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	s.repoList.SetBorder(true).SetTitle(" Selected Repositories (0) ")

	// Add current directory if it's a git repo
	if git.IsGitRepo(s.currentPath) {
		s.addRepo(s.currentPath)
	}

	// Date inputs in a form
	dateForm := tview.NewForm()
	dateForm.SetBorder(true).SetTitle(" Date Range ")

	s.sinceInput = tview.NewInputField().
		SetLabel("Since: ").
		SetText(s.config.Since.Format("2006-01-02")).
		SetFieldWidth(12)

	s.untilInput = tview.NewInputField().
		SetLabel("Until: ").
		SetText(s.config.Until.Format("2006-01-02")).
		SetFieldWidth(12)

	dateForm.AddFormItem(s.sinceInput)
	dateForm.AddFormItem(s.untilInput)

	// Buttons
	buttonForm := tview.NewForm()
	buttonForm.SetButtonsAlign(tview.AlignCenter)
	buttonForm.AddButton("Add Repository", s.showDirBrowser)
	buttonForm.AddButton("Scan All", s.validate)
	buttonForm.AddButton("Quit", func() { os.Exit(0) })

	// Error text
	s.errorText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Right panel with dates and buttons
	rightPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dateForm, 6, 0, false).
		AddItem(buttonForm, 5, 0, false).
		AddItem(s.errorText, 2, 0, false)

	// Main content
	content := tview.NewFlex().
		AddItem(s.repoList, 0, 2, true).
		AddItem(rightPanel, 40, 0, false)

	// Help text
	help := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]a[-] Add repo  [yellow]d[-] Remove  [yellow]s[-] Since  [yellow]u[-] Until  [yellow]Enter[-] Scan  [yellow]↑↓[-] Navigate")
	help.SetBackgroundColor(tcell.ColorDarkBlue)

	s.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(instructions, 1, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(help, 1, 0, false)

	// Use Pages as root to allow modal overlays
	s.root = tview.NewPages()
	s.root.AddPage("main", s.mainFlex, true, true)

	// Handle repo list input
	s.repoList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'a', 'A':
			s.showDirBrowser()
			return nil
		case 'd', 'D':
			s.removeSelectedRepo()
			return nil
		case 's':
			if s.app != nil {
				s.app.SetFocus(s.sinceInput)
			}
			return nil
		case 'u':
			if s.app != nil {
				s.app.SetFocus(s.untilInput)
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyEnter:
			s.validate()
			return nil
		}
		return event
	})

	// Handle escape from date inputs to return to repo list
	s.sinceInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
			if s.app != nil {
				s.app.SetFocus(s.repoList)
			}
			return nil
		}
		return event
	})
	s.untilInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
			if s.app != nil {
				s.app.SetFocus(s.repoList)
			}
			return nil
		}
		return event
	})
}

func (s *SetupView) addRepo(path string) {
	// Check if already added
	for i := 0; i < s.repoList.GetItemCount(); i++ {
		main, _ := s.repoList.GetItemText(i)
		if main == path {
			return
		}
	}

	// Add to list
	repoName := filepath.Base(path)
	s.repoList.AddItem(path, fmt.Sprintf("  %s", repoName), 0, nil)
	s.updateRepoCount()
}

func (s *SetupView) removeSelectedRepo() {
	idx := s.repoList.GetCurrentItem()
	if idx >= 0 && s.repoList.GetItemCount() > 0 {
		s.repoList.RemoveItem(idx)
		s.updateRepoCount()
	}
}

func (s *SetupView) updateRepoCount() {
	count := s.repoList.GetItemCount()
	s.repoList.SetTitle(fmt.Sprintf(" Selected Repositories (%d) ", count))
}

func (s *SetupView) validate() {
	// Collect repos from list
	repos := make([]string, 0)
	for i := 0; i < s.repoList.GetItemCount(); i++ {
		path, _ := s.repoList.GetItemText(i)
		repos = append(repos, path)
	}

	if len(repos) == 0 {
		s.ShowError("Please add at least one repository")
		return
	}

	// Validate all repos
	for _, path := range repos {
		if !git.IsGitRepo(path) {
			s.ShowError(fmt.Sprintf("Not a git repo: %s", filepath.Base(path)))
			return
		}
	}

	// Parse dates
	since, err := time.Parse("2006-01-02", s.sinceInput.GetText())
	if err != nil {
		s.ShowError("Invalid 'Since' date. Use YYYY-MM-DD")
		return
	}
	s.config.Since = since

	until, err := time.Parse("2006-01-02", s.untilInput.GetText())
	if err != nil {
		s.ShowError("Invalid 'Until' date. Use YYYY-MM-DD")
		return
	}
	s.config.Until = until

	if until.Before(since) {
		s.ShowError("'Until' must be after 'Since'")
		return
	}

	// Update config
	s.config.RepoPaths = repos
	if len(repos) > 0 {
		s.config.RepoPath = repos[0]
	}

	s.errorText.SetText("")
	s.onComplete()
}

func (s *SetupView) showDirBrowser() {
	dirList := tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	dirList.SetBorder(true)

	// Help text for browser
	browserHelp := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]Enter[-] Open folder  [yellow]Space[-] Add repo  [yellow]Esc[-] Close")
	browserHelp.SetBackgroundColor(tcell.ColorDarkBlue)

	currentPath := s.currentPath

	// Track directory entries for easy lookup
	type dirEntry struct {
		name   string
		isRepo bool
	}
	var dirEntries []dirEntry

	var populateList func(path string)
	populateList = func(path string) {
		dirList.Clear()
		dirEntries = nil
		dirList.SetTitle(fmt.Sprintf(" %s ", path))
		currentPath = path

		// Parent directory
		dirList.AddItem("..", "Go up one directory", 0, nil)
		dirEntries = append(dirEntries, dirEntry{name: "..", isRepo: false})

		// List subdirectories
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if entry.IsDir() && !isHiddenDir(entry.Name()) {
				fullPath := filepath.Join(path, entry.Name())
				isRepo := git.IsGitRepo(fullPath)
				if isRepo {
					dirList.AddItem("[cyan]📦 "+entry.Name()+"[-]", "[Space] to add this repo", 0, nil)
				} else {
					dirList.AddItem("   "+entry.Name(), "Directory", 0, nil)
				}
				dirEntries = append(dirEntries, dirEntry{name: entry.Name(), isRepo: isRepo})
			}
		}
	}

	populateList(currentPath)

	// Browser layout
	browserBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(dirList, 0, 1, true).
		AddItem(browserHelp, 1, 0, false)
	browserBox.SetBorder(true).SetTitle(" Select Repository ")

	// Modal centered layout
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(browserBox, 24, 0, true).
			AddItem(nil, 0, 1, false), 80, 0, true).
		AddItem(nil, 0, 1, false)

	// Enter navigates into directories
	dirList.SetSelectedFunc(func(idx int, main, secondary string, shortcut rune) {
		if idx < 0 || idx >= len(dirEntries) {
			return
		}

		entry := dirEntries[idx]

		if entry.name == ".." {
			populateList(filepath.Dir(currentPath))
		} else {
			// Navigate into directory
			newPath := filepath.Join(currentPath, entry.name)
			populateList(newPath)
		}
	})

	// Helper to close modal and restore focus
	closeModal := func() {
		s.root.RemovePage("browser")
		s.root.SwitchToPage("main")
		if s.app != nil {
			s.app.SetFocus(s.repoList)
		}
	}

	// Space adds repos, Esc closes
	dirList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			closeModal()
			return nil
		case tcell.KeyRune:
			if event.Rune() == ' ' {
				// Add current selection if it's a repo
				idx := dirList.GetCurrentItem()
				if idx >= 0 && idx < len(dirEntries) {
					entry := dirEntries[idx]
					if entry.isRepo {
						newPath := filepath.Join(currentPath, entry.name)
						s.addRepo(newPath)
						s.currentPath = currentPath
						closeModal()
						return nil
					} else if entry.name == ".." {
						// If current dir is a repo, add it
						if git.IsGitRepo(currentPath) {
							s.addRepo(currentPath)
							s.currentPath = currentPath
							closeModal()
							return nil
						}
					}
				}
			}
		}
		return event
	})

	// Add browser as a page and switch to it
	s.root.AddPage("browser", modal, true, true)

	// Set focus on the directory list
	if s.app != nil {
		s.app.SetFocus(dirList)
	}
}

func isHiddenDir(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

// ShowError displays an error message
func (s *SetupView) ShowError(msg string) {
	s.errorText.SetText("[red]" + msg + "[-]")
}

// Root returns the root primitive
func (s *SetupView) Root() tview.Primitive {
	return s.root
}
