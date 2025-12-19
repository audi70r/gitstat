package git

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Match "Merge pull request #123 from user/branch"
	prNumberRegex = regexp.MustCompile(`[Mm]erge pull request #(\d+)`)
	// Match "Merge branch 'feature'" or "Merge branch 'feature' into 'main'"
	mergeBranchRegex = regexp.MustCompile(`[Mm]erge (?:pull request #\d+ from |branch '?)([^'"\s]+)`)
)

const (
	commitStart = "COMMIT_START"
	commitEnd   = "COMMIT_END"
)

// Parser handles git log parsing
type Parser struct {
	RepoPath string
}

// NewParser creates a new git parser for the given repository path
func NewParser(repoPath string) *Parser {
	return &Parser{RepoPath: repoPath}
}

// EstimateCommitCount returns an estimate of commits in the date range
func (p *Parser) EstimateCommitCount(ctx context.Context, since, until time.Time) (int, error) {
	args := []string{"rev-list", "--count", "HEAD"}

	if !since.IsZero() {
		args = append(args, "--since="+since.Format(time.RFC3339))
	}
	if !until.IsZero() {
		args = append(args, "--until="+until.Format(time.RFC3339))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = p.RepoPath

	output, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return -1, err
	}

	return count, nil
}

// Parse executes git log and streams commits via callback
func (p *Parser) Parse(ctx context.Context, since, until time.Time,
	onProgress func(ScanProgress), onCommit func(*Commit)) error {

	// %P = parent hashes (space-separated), used to detect merge commits
	format := "COMMIT_START%n%H%n%h%n%an%n%ae%n%aI%n%P%n%s%nCOMMIT_END"

	args := []string{
		"log",
		"--format=" + format,
		"--numstat",
	}

	if !since.IsZero() {
		args = append(args, "--since="+since.Format(time.RFC3339))
	}
	if !until.IsZero() {
		args = append(args, "--until="+until.Format(time.RFC3339))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = p.RepoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	var current *Commit
	lineNum := 0
	commitCount := 0
	inNumstat := false
	seenNumstatContent := false // Track if we've seen any numstat content

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case line == commitStart:
			// If we have a pending commit, emit it
			if current != nil {
				onCommit(current)
				commitCount++
				if onProgress != nil {
					onProgress(ScanProgress{
						CommitsParsed: commitCount,
						CurrentHash:   current.ShortHash,
					})
				}
			}
			current = &Commit{}
			lineNum = 0
			inNumstat = false
			seenNumstatContent = false

		case line == commitEnd:
			inNumstat = true
			seenNumstatContent = false

		case current != nil && !inNumstat:
			parseCommitLine(current, lineNum, line)
			lineNum++

		case current != nil && inNumstat:
			if line == "" {
				// Skip empty lines before numstat content
				// Only finalize if we've seen content and this is the separator
				if seenNumstatContent {
					// This empty line ends the numstat section
					// But don't emit yet - wait for COMMIT_START
				}
			} else {
				fc := parseNumstat(line)
				if fc != nil {
					current.FileChanges = append(current.FileChanges, *fc)
					seenNumstatContent = true
				}
			}
		}
	}

	// Handle last commit
	if current != nil {
		onCommit(current)
		commitCount++
		if onProgress != nil {
			onProgress(ScanProgress{
				CommitsParsed: commitCount,
				CurrentHash:   current.ShortHash,
				Done:          true,
			})
		}
	}

	if onProgress != nil {
		onProgress(ScanProgress{
			CommitsParsed: commitCount,
			Done:          true,
		})
	}

	return cmd.Wait()
}

func parseCommitLine(c *Commit, lineNum int, line string) {
	switch lineNum {
	case 0:
		c.Hash = line
	case 1:
		c.ShortHash = line
	case 2:
		c.Author.Name = line
	case 3:
		c.Author.Email = line
	case 4:
		c.AuthorDate, _ = time.Parse(time.RFC3339, line)
	case 5:
		// Parent hashes - merge commits have 2+ parents
		parents := strings.Fields(line)
		c.IsMerge = len(parents) >= 2
	case 6:
		c.Subject = line
		// Extract PR number and branch from merge commit message
		if c.IsMerge {
			if matches := prNumberRegex.FindStringSubmatch(line); len(matches) >= 2 {
				c.PRNumber, _ = strconv.Atoi(matches[1])
			}
			if matches := mergeBranchRegex.FindStringSubmatch(line); len(matches) >= 2 {
				c.MergeBranch = matches[1]
			}
		}
	}
}

func parseNumstat(line string) *FileChange {
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return nil
	}

	fc := &FileChange{FilePath: parts[2]}

	if parts[0] == "-" {
		fc.IsBinary = true
	} else {
		fc.Additions, _ = strconv.Atoi(parts[0])
		fc.Deletions, _ = strconv.Atoi(parts[1])
	}

	return fc
}

// IsGitRepo checks if the path is a valid git repository
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	return cmd.Run() == nil
}

// GetCodebaseSize returns total lines of code in the repository
func GetCodebaseSize(repoPath string) (int, error) {
	// Get list of tracked files and count lines
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 0 || (len(files) == 1 && files[0] == "") {
		return 0, nil
	}

	totalLines := 0
	for _, file := range files {
		if file == "" {
			continue
		}
		// Count lines in file
		wcCmd := exec.Command("wc", "-l", file)
		wcCmd.Dir = repoPath
		wcOutput, err := wcCmd.Output()
		if err != nil {
			continue // Skip files that can't be read
		}
		parts := strings.Fields(string(wcOutput))
		if len(parts) > 0 {
			lines, _ := strconv.Atoi(parts[0])
			totalLines += lines
		}
	}

	return totalLines, nil
}
