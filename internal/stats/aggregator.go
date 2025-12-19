package stats

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/audi70r/gitstat/internal/git"
)

// Aggregator processes commits and builds statistics
type Aggregator struct {
	repo     *Repository
	timezone *time.Location
}

// NewAggregator creates a new statistics aggregator
func NewAggregator(repoPath string, dateRange DateRange, tz *time.Location) *Aggregator {
	if tz == nil {
		tz = time.Local
	}
	return &Aggregator{
		repo:     NewRepository(repoPath, dateRange),
		timezone: tz,
	}
}

// ProcessCommit adds a commit's data to the statistics
func (a *Aggregator) ProcessCommit(c *git.Commit) {
	a.repo.TotalCommits++

	// Process merge commits for PR stats
	if c.IsMerge {
		a.processMergeCommit(c)
	}

	// Author stats
	authorKey := c.Author.Email
	author, ok := a.repo.Authors[authorKey]
	if !ok {
		author = NewAuthorStats(c.Author.Name, c.Author.Email)
		a.repo.Authors[authorKey] = author
		a.repo.TotalAuthors++
	}

	author.Commits++
	if author.FirstCommit.IsZero() || c.AuthorDate.Before(author.FirstCommit) {
		author.FirstCommit = c.AuthorDate
	}
	if c.AuthorDate.After(author.LastCommit) {
		author.LastCommit = c.AuthorDate
	}

	// Daily activity
	dateKey := c.AuthorDate.In(a.timezone).Format("2006-01-02")
	a.repo.DailyActivity[dateKey]++

	// Hourly matrix (weekday x hour)
	localTime := c.AuthorDate.In(a.timezone)
	weekday := int(localTime.Weekday())
	// Convert Sunday=0 to Monday=0 format
	weekday = (weekday + 6) % 7
	hour := localTime.Hour()
	a.repo.HourlyMatrix[weekday][hour]++

	// Process file changes
	for _, fc := range c.FileChanges {
		if fc.IsBinary {
			continue
		}

		author.Additions += fc.Additions
		author.Deletions += fc.Deletions
		author.FilesTouched[fc.FilePath]++

		a.repo.TotalAdditions += fc.Additions
		a.repo.TotalDeletions += fc.Deletions

		// File stats
		fileStat, ok := a.repo.FileStats[fc.FilePath]
		if !ok {
			fileStat = NewFileStats(fc.FilePath)
			a.repo.FileStats[fc.FilePath] = fileStat
		}

		fileStat.Additions += fc.Additions
		fileStat.Deletions += fc.Deletions
		fileStat.TotalChanges += fc.Additions + fc.Deletions
		fileStat.TouchCount++
		fileStat.Authors[c.Author.Email]++

		// Directory stats
		dir := getTopDir(fc.FilePath)
		dirStat, ok := a.repo.DirStats[dir]
		if !ok {
			dirStat = NewDirStats(dir)
			a.repo.DirStats[dir] = dirStat
		}

		dirStat.TotalChanges += fc.Additions + fc.Deletions
		dirStat.TouchCount++

		dirAuthor, ok := dirStat.Authors[c.Author.Email]
		if !ok {
			dirAuthor = &DirAuthorStats{
				Name:  c.Author.Name,
				Email: c.Author.Email,
			}
			dirStat.Authors[c.Author.Email] = dirAuthor
		}
		dirAuthor.Commits++
		dirAuthor.Changes += fc.Additions + fc.Deletions
	}
}

// Finalize calculates derived statistics after all commits are processed
func (a *Aggregator) Finalize() *Repository {
	// Calculate directory ownership shares
	for _, dir := range a.repo.DirStats {
		if dir.TotalChanges > 0 {
			for _, author := range dir.Authors {
				author.Share = float64(author.Changes) / float64(dir.TotalChanges) * 100
			}
		}
	}

	return a.repo
}

// GetResult returns the current repository statistics
func (a *Aggregator) GetResult() *Repository {
	return a.repo
}

func getTopDir(path string) string {
	// Clean the path
	path = filepath.Clean(path)

	// Find the first directory component
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 1 {
		return parts[0]
	}

	// It's a root-level file
	return "."
}

// GetLeaderboard returns authors sorted by the given criteria
func (r *Repository) GetLeaderboard(sortBy string, ascending bool) []*AuthorStats {
	authors := make([]*AuthorStats, 0, len(r.Authors))
	for _, a := range r.Authors {
		authors = append(authors, a)
	}

	sort.Slice(authors, func(i, j int) bool {
		var cmp bool
		switch sortBy {
		case "name":
			cmp = authors[i].Name < authors[j].Name
		case "commits":
			cmp = authors[i].Commits < authors[j].Commits
		case "additions":
			cmp = authors[i].Additions < authors[j].Additions
		case "deletions":
			cmp = authors[i].Deletions < authors[j].Deletions
		case "net":
			cmp = (authors[i].Additions - authors[i].Deletions) <
				(authors[j].Additions - authors[j].Deletions)
		default:
			cmp = authors[i].Commits < authors[j].Commits
		}
		if ascending {
			return cmp
		}
		return !cmp
	})

	return authors
}

// GetTopFiles returns files sorted by the given criteria
func (r *Repository) GetTopFiles(sortBy string, ascending bool, limit int) []*FileStats {
	files := make([]*FileStats, 0, len(r.FileStats))
	for _, f := range r.FileStats {
		files = append(files, f)
	}

	sort.Slice(files, func(i, j int) bool {
		var cmp bool
		switch sortBy {
		case "path":
			cmp = files[i].Path < files[j].Path
		case "changes":
			cmp = files[i].TotalChanges < files[j].TotalChanges
		case "touches":
			cmp = files[i].TouchCount < files[j].TouchCount
		case "authors":
			cmp = len(files[i].Authors) < len(files[j].Authors)
		default:
			cmp = files[i].TotalChanges < files[j].TotalChanges
		}
		if ascending {
			return cmp
		}
		return !cmp
	})

	if limit > 0 && limit < len(files) {
		return files[:limit]
	}
	return files
}

// GetHotspots returns files with high churn and multiple authors
func (r *Repository) GetHotspots(limit int) []*HotspotFile {
	hotspots := make([]*HotspotFile, 0)

	// Find max values for normalization
	var maxChanges, maxTouches int
	for _, f := range r.FileStats {
		if f.TotalChanges > maxChanges {
			maxChanges = f.TotalChanges
		}
		if f.TouchCount > maxTouches {
			maxTouches = f.TouchCount
		}
	}

	if maxChanges == 0 {
		maxChanges = 1
	}
	if maxTouches == 0 {
		maxTouches = 1
	}

	for _, f := range r.FileStats {
		authorCount := len(f.Authors)
		if authorCount < 2 {
			continue // Skip single-author files
		}

		churnScore := float64(f.TotalChanges) / float64(maxChanges)
		touchScore := float64(f.TouchCount) / float64(maxTouches)
		authorScore := float64(authorCount) / float64(r.TotalAuthors)

		// Combined risk score: churn * frequency * author diversity
		riskScore := (churnScore*0.4 + touchScore*0.3 + authorScore*0.3) * 100

		hotspots = append(hotspots, &HotspotFile{
			Path:        f.Path,
			ChurnScore:  churnScore * 100,
			AuthorCount: authorCount,
			RiskScore:   riskScore,
			Changes:     f.TotalChanges,
			TouchCount:  f.TouchCount,
		})
	}

	// Sort by risk score descending
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].RiskScore > hotspots[j].RiskScore
	})

	if limit > 0 && limit < len(hotspots) {
		return hotspots[:limit]
	}
	return hotspots
}

// GetTimeline returns daily commit data with rolling average
func (r *Repository) GetTimeline(windowDays int) *TimelineData {
	if len(r.DailyActivity) == 0 {
		return &TimelineData{}
	}

	// Get sorted dates
	dates := make([]string, 0, len(r.DailyActivity))
	for d := range r.DailyActivity {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	// Find date range
	startDate, _ := time.Parse("2006-01-02", dates[0])
	endDate, _ := time.Parse("2006-01-02", dates[len(dates)-1])

	// Fill in all dates in range
	var labels []string
	var values []int
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		labels = append(labels, dateStr)
		values = append(values, r.DailyActivity[dateStr])
	}

	// Calculate rolling average
	rollingAvg := make([]float64, len(values))
	for i := range values {
		start := i - windowDays + 1
		if start < 0 {
			start = 0
		}
		sum := 0
		for j := start; j <= i; j++ {
			sum += values[j]
		}
		rollingAvg[i] = float64(sum) / float64(i-start+1)
	}

	return &TimelineData{
		Period:     "day",
		Labels:     labels,
		Values:     values,
		RollingAvg: rollingAvg,
	}
}

// GetHeatmap returns hourly commit distribution data
func (r *Repository) GetHeatmap(tz *time.Location) *HeatmapData {
	var maxValue int
	for day := 0; day < 7; day++ {
		for hour := 0; hour < 24; hour++ {
			if r.HourlyMatrix[day][hour] > maxValue {
				maxValue = r.HourlyMatrix[day][hour]
			}
		}
	}

	return &HeatmapData{
		Matrix:   r.HourlyMatrix,
		MaxValue: maxValue,
		Timezone: tz,
	}
}

// GetOwnership returns directories with author ownership data
func (r *Repository) GetOwnership(sortBy string, ascending bool) []*DirStats {
	dirs := make([]*DirStats, 0, len(r.DirStats))
	for _, d := range r.DirStats {
		dirs = append(dirs, d)
	}

	sort.Slice(dirs, func(i, j int) bool {
		var cmp bool
		switch sortBy {
		case "path":
			cmp = dirs[i].Path < dirs[j].Path
		case "changes":
			cmp = dirs[i].TotalChanges < dirs[j].TotalChanges
		case "touches":
			cmp = dirs[i].TouchCount < dirs[j].TouchCount
		case "authors":
			cmp = len(dirs[i].Authors) < len(dirs[j].Authors)
		default:
			cmp = dirs[i].TotalChanges < dirs[j].TotalChanges
		}
		if ascending {
			return cmp
		}
		return !cmp
	})

	return dirs
}

// GetCodebaseStats returns overall codebase statistics
func (r *Repository) GetCodebaseStats() *CodebaseStats {
	totalChanges := r.TotalAdditions + r.TotalDeletions

	// Calculate refactored percentage
	var refactoredPct float64
	if r.CodebaseSize > 0 {
		refactoredPct = float64(totalChanges) / float64(r.CodebaseSize) * 100
	}

	return &CodebaseStats{
		TotalAdditions:    r.TotalAdditions,
		TotalDeletions:    r.TotalDeletions,
		TotalChanges:      totalChanges,
		FilesModified:     len(r.FileStats),
		CodebaseSize:      r.CodebaseSize,
		RefactoredPercent: refactoredPct,
	}
}

// ApplyAuthorMerges merges authors based on the provided mapping
// merges maps email -> primary email
func (r *Repository) ApplyAuthorMerges(merges map[string]string) {
	if len(merges) == 0 {
		return
	}

	// Find all primary emails
	primaries := make(map[string]bool)
	for email, target := range merges {
		if email == target {
			primaries[email] = true
		}
	}

	// Merge author stats
	for aliasEmail, primaryEmail := range merges {
		if aliasEmail == primaryEmail {
			continue // Skip primaries
		}

		alias, aliasExists := r.Authors[aliasEmail]
		primary, primaryExists := r.Authors[primaryEmail]

		if !aliasExists || !primaryExists {
			continue
		}

		// Merge stats into primary
		primary.Commits += alias.Commits
		primary.Additions += alias.Additions
		primary.Deletions += alias.Deletions

		// Merge files touched
		for file, count := range alias.FilesTouched {
			primary.FilesTouched[file] += count
		}

		// Update date range
		if alias.FirstCommit.Before(primary.FirstCommit) {
			primary.FirstCommit = alias.FirstCommit
		}
		if alias.LastCommit.After(primary.LastCommit) {
			primary.LastCommit = alias.LastCommit
		}

		// Remove alias from authors map
		delete(r.Authors, aliasEmail)
		r.TotalAuthors--
	}

	// Update file stats authors
	for _, fileStat := range r.FileStats {
		for aliasEmail, primaryEmail := range merges {
			if aliasEmail == primaryEmail {
				continue
			}
			if count, exists := fileStat.Authors[aliasEmail]; exists {
				fileStat.Authors[primaryEmail] += count
				delete(fileStat.Authors, aliasEmail)
			}
		}
	}

	// Update directory stats authors
	for _, dirStat := range r.DirStats {
		for aliasEmail, primaryEmail := range merges {
			if aliasEmail == primaryEmail {
				continue
			}

			alias, aliasExists := dirStat.Authors[aliasEmail]
			primary, primaryExists := dirStat.Authors[primaryEmail]

			if !aliasExists {
				continue
			}

			if !primaryExists {
				// Rename alias to primary
				dirStat.Authors[primaryEmail] = alias
				alias.Email = primaryEmail
			} else {
				// Merge into primary
				primary.Commits += alias.Commits
				primary.Changes += alias.Changes
			}

			delete(dirStat.Authors, aliasEmail)
		}

		// Recalculate shares
		if dirStat.TotalChanges > 0 {
			for _, author := range dirStat.Authors {
				author.Share = float64(author.Changes) / float64(dirStat.TotalChanges) * 100
			}
		}
	}
}

// processMergeCommit processes a merge commit for PR statistics
func (a *Aggregator) processMergeCommit(c *git.Commit) {
	prStats := a.repo.PRStats
	prStats.TotalMerges++

	// Track daily merges
	dateKey := c.AuthorDate.In(a.timezone).Format("2006-01-02")
	prStats.DailyMerges[dateKey]++

	// Calculate totals for this merge
	additions := 0
	deletions := 0
	for _, fc := range c.FileChanges {
		if !fc.IsBinary {
			additions += fc.Additions
			deletions += fc.Deletions
		}
	}

	// Track author stats
	authorKey := c.Author.Email
	authorStats, ok := prStats.MergesByAuthor[authorKey]
	if !ok {
		authorStats = &PRAuthorStats{
			Name:      c.Author.Name,
			Email:     c.Author.Email,
			PRNumbers: make([]int, 0),
		}
		prStats.MergesByAuthor[authorKey] = authorStats
	}
	authorStats.MergeCount++
	authorStats.TotalChanges += additions + deletions
	if c.PRNumber > 0 {
		authorStats.PRNumbers = append(authorStats.PRNumbers, c.PRNumber)
	}

	// Track PR info
	if c.PRNumber > 0 {
		prStats.TotalPRs++
	}

	prInfo := &PRInfo{
		PRNumber:      c.PRNumber,
		MergedBy:      c.Author.Name,
		MergedByEmail: c.Author.Email,
		MergedAt:      c.AuthorDate,
		Branch:        c.MergeBranch,
		Subject:       c.Subject,
		Additions:     additions,
		Deletions:     deletions,
		FilesCount:    len(c.FileChanges),
	}
	prStats.PRList = append(prStats.PRList, prInfo)
}

// GetPRLeaderboard returns authors sorted by merge count
func (r *Repository) GetPRLeaderboard(sortBy string, ascending bool) []*PRAuthorStats {
	authors := make([]*PRAuthorStats, 0, len(r.PRStats.MergesByAuthor))
	for _, a := range r.PRStats.MergesByAuthor {
		authors = append(authors, a)
	}

	sort.Slice(authors, func(i, j int) bool {
		var cmp bool
		switch sortBy {
		case "name":
			cmp = authors[i].Name < authors[j].Name
		case "merges":
			cmp = authors[i].MergeCount < authors[j].MergeCount
		case "changes":
			cmp = authors[i].TotalChanges < authors[j].TotalChanges
		default:
			cmp = authors[i].MergeCount < authors[j].MergeCount
		}
		if ascending {
			return cmp
		}
		return !cmp
	})

	return authors
}

// GetPRList returns PRs sorted by date or size
func (r *Repository) GetPRList(sortBy string, ascending bool, limit int) []*PRInfo {
	prs := make([]*PRInfo, len(r.PRStats.PRList))
	copy(prs, r.PRStats.PRList)

	sort.Slice(prs, func(i, j int) bool {
		var cmp bool
		switch sortBy {
		case "date":
			cmp = prs[i].MergedAt.Before(prs[j].MergedAt)
		case "size":
			cmp = (prs[i].Additions + prs[i].Deletions) < (prs[j].Additions + prs[j].Deletions)
		case "files":
			cmp = prs[i].FilesCount < prs[j].FilesCount
		default:
			cmp = prs[i].MergedAt.Before(prs[j].MergedAt)
		}
		if ascending {
			return cmp
		}
		return !cmp
	})

	if limit > 0 && limit < len(prs) {
		return prs[:limit]
	}
	return prs
}
