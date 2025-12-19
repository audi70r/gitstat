package git

import "time"

// Commit represents a single parsed git commit
type Commit struct {
	Hash        string
	ShortHash   string
	Author      Author
	AuthorDate  time.Time
	Subject     string
	FileChanges []FileChange
	IsMerge     bool   // True if this is a merge commit
	PRNumber    int    // PR number if extracted from merge message
	MergeBranch string // Branch that was merged
}

// Author represents commit author info
type Author struct {
	Name  string
	Email string
}

// FileChange represents numstat output for a file
type FileChange struct {
	Additions int
	Deletions int
	FilePath  string
	IsBinary  bool
}

// ScanProgress reports parsing progress
type ScanProgress struct {
	CommitsParsed int
	TotalEstimate int
	CurrentHash   string
	Done          bool
}
