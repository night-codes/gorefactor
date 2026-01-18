package refactor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type LinesResult struct {
	Success bool   `json:"success"`
	File    string `json:"file"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Lines   string `json:"lines"`
	Count   int    `json:"count"`
}

func ReadLines(file string, start, end int) (*LinesResult, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	total := len(lines)

	if start < 1 {
		start = 1
	}
	if end < 0 || end > total {
		end = total
	}
	if start > end {
		return nil, fmt.Errorf("start (%d) > end (%d)", start, end)
	}

	selected := lines[start-1 : end]

	return &LinesResult{
		Success: true,
		File:    file,
		Start:   start,
		End:     end,
		Lines:   strings.Join(selected, "\n"),
		Count:   len(selected),
	}, nil
}

func ReplaceLines(file string, start, end int, newContent string) (*ModifyResult, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	total := len(lines)

	if start < 1 {
		start = 1
	}
	if end < 0 || end > total {
		end = total
	}
	if start > end {
		return nil, fmt.Errorf("start (%d) > end (%d)", start, end)
	}

	newLines := strings.Split(newContent, "\n")

	var result []string
	result = append(result, lines[:start-1]...)
	result = append(result, newLines...)
	result = append(result, lines[end:]...)

	if err := os.WriteFile(file, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return nil, err
	}

	return &ModifyResult{
		Success: true,
		File:    file,
		Message: fmt.Sprintf("replaced lines %d-%d with %d lines", start, end, len(newLines)),
	}, nil
}

func DeleteLines(file string, start, end int) (*ModifyResult, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	total := len(lines)

	if start < 1 {
		start = 1
	}
	if end < 0 || end > total {
		end = total
	}
	if start > end {
		return nil, fmt.Errorf("start (%d) > end (%d)", start, end)
	}

	var result []string
	result = append(result, lines[:start-1]...)
	result = append(result, lines[end:]...)

	if err := os.WriteFile(file, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return nil, err
	}

	return &ModifyResult{
		Success: true,
		File:    file,
		Message: fmt.Sprintf("deleted lines %d-%d", start, end),
	}, nil
}

func InsertLines(file string, after int, newContent string) (*ModifyResult, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	total := len(lines)

	if after < 0 {
		after = 0
	}
	if after > total {
		after = total
	}

	newLines := strings.Split(newContent, "\n")

	var result []string
	result = append(result, lines[:after]...)
	result = append(result, newLines...)
	result = append(result, lines[after:]...)

	if err := os.WriteFile(file, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return nil, err
	}

	return &ModifyResult{
		Success: true,
		File:    file,
		Message: fmt.Sprintf("inserted %d lines after line %d", len(newLines), after),
	}, nil
}

func ParseLineRange(s string) (file string, start, end int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return "", 0, 0, fmt.Errorf("invalid format, expected file:N or file:N:M")
	}

	file = parts[0]
	start, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid start line: %s", parts[1])
	}

	if len(parts) >= 3 {
		end, err = strconv.Atoi(parts[2])
		if err != nil {
			return "", 0, 0, fmt.Errorf("invalid end line: %s", parts[2])
		}
	} else {
		end = start
	}

	return file, start, end, nil
}
