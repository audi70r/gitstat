package stats

import (
	"path/filepath"
	"time"
)

// DateRange represents the time range for analysis
type DateRange struct {
	Since time.Time
	Until time.Time
}

// Repository holds all computed statistics
type Repository struct {
	Path         string
	DateRange    DateRange
	TotalCommits int
	TotalAuthors int

	// Author statistics
	Authors map[string]*AuthorStats

	// File statistics
	FileStats map[string]*FileStats

	// Directory statistics
	DirStats map[string]*DirStats

	// Time-based data
	DailyActivity map[string]int // "2024-01-15" -> count
	HourlyMatrix  [7][24]int     // weekday x hour

	// Totals
	TotalAdditions int
	TotalDeletions int

	// Codebase info
	CodebaseSize int // Total lines in current codebase
}

// NewRepository creates a new Repository stats container
func NewRepository(path string, dateRange DateRange) *Repository {
	return &Repository{
		Path:          path,
		DateRange:     dateRange,
		Authors:       make(map[string]*AuthorStats),
		FileStats:     make(map[string]*FileStats),
		DirStats:      make(map[string]*DirStats),
		DailyActivity: make(map[string]int),
	}
}

// AuthorStats holds statistics for a single author
type AuthorStats struct {
	Name         string
	Email        string
	Commits      int
	Additions    int
	Deletions    int
	FilesTouched map[string]int // file -> touch count
	FirstCommit  time.Time
	LastCommit   time.Time
}

// NewAuthorStats creates a new AuthorStats
func NewAuthorStats(name, email string) *AuthorStats {
	return &AuthorStats{
		Name:         name,
		Email:        email,
		FilesTouched: make(map[string]int),
	}
}

// FileStats holds statistics for a single file
type FileStats struct {
	Path         string
	TotalChanges int            // additions + deletions
	TouchCount   int            // number of commits affecting this file
	Authors      map[string]int // author email -> commits
	Additions    int
	Deletions    int
}

// NewFileStats creates a new FileStats
func NewFileStats(path string) *FileStats {
	return &FileStats{
		Path:    path,
		Authors: make(map[string]int),
	}
}

// DirStats holds statistics for a directory
type DirStats struct {
	Path         string
	Authors      map[string]*DirAuthorStats
	TotalChanges int
	TouchCount   int
}

// NewDirStats creates a new DirStats
func NewDirStats(path string) *DirStats {
	return &DirStats{
		Path:    path,
		Authors: make(map[string]*DirAuthorStats),
	}
}

// DirAuthorStats holds per-author stats within a directory
type DirAuthorStats struct {
	Name    string
	Email   string
	Commits int
	Changes int
	Share   float64 // percentage of total changes
}

// TimelineData holds time-series commit data
type TimelineData struct {
	Period     string // "day" or "week"
	Labels     []string
	Values     []int
	RollingAvg []float64
}

// HeatmapData holds work hours heatmap data
type HeatmapData struct {
	Matrix   [7][24]int // weekday x hour
	MaxValue int
	Timezone *time.Location
}

// HotspotFile represents a file with risk signals
type HotspotFile struct {
	Path        string
	ChurnScore  float64 // normalized changes
	AuthorCount int
	RiskScore   float64 // combined score
	Changes     int
	TouchCount  int
}

// CodebaseStats holds overall codebase change statistics
type CodebaseStats struct {
	TotalAdditions    int
	TotalDeletions    int
	TotalChanges      int
	FilesAdded        int
	FilesModified     int
	FilesDeleted      int
	CodebaseSize      int     // Total lines in current codebase
	RefactoredPercent float64 // Percentage of codebase touched
}

// GetDirectory returns the top-level directory of a file path
func GetDirectory(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return "."
	}
	// Get the top-level directory
	parts := filepath.SplitList(dir)
	if len(parts) > 0 {
		return parts[0]
	}
	// Split by separator
	for i, c := range dir {
		if c == '/' || c == filepath.Separator {
			return dir[:i]
		}
	}
	return dir
}
