package refactor

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Text    string `json:"text"`
	Context string `json:"context,omitempty"`
}

type GrepResult struct {
	Success bool        `json:"success"`
	Query   string      `json:"query"`
	Matches []GrepMatch `json:"matches"`
	Count   int         `json:"count"`
}

type GrepOptions struct {
	Regex      bool
	IgnoreCase bool
	Context    int
	FilePattern string
}

func Grep(pattern, dir string, opts *GrepOptions) (*GrepResult, error) {
	if opts == nil {
		opts = &GrepOptions{}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var matches []GrepMatch
	var re *regexp.Regexp

	if opts.Regex {
		flags := ""
		if opts.IgnoreCase {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + pattern)
		if err != nil {
			return nil, err
		}
	} else if opts.IgnoreCase {
		pattern = strings.ToLower(pattern)
	}

	filepath.Walk(absDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			if fi != nil && fi.IsDir() {
				base := fi.Name()
				if path != absDir && (strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" || base == "testdata") {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			if opts.FilePattern == "" {
				return nil
			}
			matched, _ := filepath.Match(opts.FilePattern, fi.Name())
			if !matched {
				return nil
			}
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		var lines []string

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			lines = append(lines, line)

			var found bool
			var col int

			if opts.Regex {
				loc := re.FindStringIndex(line)
				if loc != nil {
					found = true
					col = loc[0] + 1
				}
			} else {
				searchLine := line
				searchPattern := pattern
				if opts.IgnoreCase {
					searchLine = strings.ToLower(line)
				}
				idx := strings.Index(searchLine, searchPattern)
				if idx >= 0 {
					found = true
					col = idx + 1
				}
			}

			if found {
				relPath, _ := filepath.Rel(absDir, path)
				match := GrepMatch{
					File:   relPath,
					Line:   lineNum,
					Column: col,
					Text:   strings.TrimSpace(line),
				}

				if opts.Context > 0 && len(lines) > opts.Context {
					start := len(lines) - opts.Context - 1
					if start < 0 {
						start = 0
					}
					match.Context = strings.Join(lines[start:], "\n")
				}

				matches = append(matches, match)
			}
		}

		return nil
	})

	return &GrepResult{
		Success: true,
		Query:   pattern,
		Matches: matches,
		Count:   len(matches),
	}, nil
}
