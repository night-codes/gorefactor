package refactor

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strings"
)

type ReadResult struct {
	Success   bool   `json:"success"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Receiver  string `json:"receiver,omitempty"`
	Signature string `json:"signature,omitempty"`
	Code      string `json:"code"`
	Value     string `json:"value,omitempty"`
	Type      string `json:"type,omitempty"`
}

type ReadResults struct {
	Success bool         `json:"success"`
	Results []ReadResult `json:"results"`
	Count   int          `json:"count"`
}

func getPackageName(file string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	return f.Name.Name, nil
}

func Read(name, file string) (*ReadResults, error) {
	matches, err := searchSymbols(name, ".", "")
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("symbol %s not found", name)
	}

	// Filter by file if specified
	if file != "" {
		absFile, _ := filepath.Abs(file)
		var filtered []SymbolLocation
		for _, m := range matches {
			absMatch, _ := filepath.Abs(m.File)
			if absMatch == absFile {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("symbol %s not found in %s", name, file)
		}
		matches = filtered
	}

	var results []ReadResult
	for _, loc := range matches {
		targetFile := loc.File
		var res *ReadResult

		parts := strings.Split(loc.Name, ".")

		if loc.Name != name && parts[len(parts)-1] != name {
			continue
		}

		switch loc.Kind {
		case "func":
			r, err := ReadFunc(loc.Name, targetFile)
			if err != nil {
				continue
			}
			res = &ReadResult{
				Success:   true,
				Name:      r.Name,
				Kind:      "func",
				File:      r.File,
				Line:      r.Line,
				EndLine:   r.EndLine,
				Receiver:  r.Receiver,
				Signature: r.Signature,
				Code:      r.Code,
			}

		case "struct", "interface", "type":
			r, err := ReadType(loc.Name, targetFile)
			if err != nil {
				continue
			}
			res = &ReadResult{
				Success: true,
				Name:    r.Name,
				Kind:    r.Kind,
				File:    r.File,
				Line:    r.Line,
				EndLine: r.EndLine,
				Code:    r.Code,
			}

		case "var", "const":
			r, err := ReadVarConst(loc.Name, targetFile)
			if err != nil {
				continue
			}
			res = &ReadResult{
				Success: true,
				Name:    r.Name,
				Kind:    r.Kind,
				File:    r.File,
				Line:    r.Line,
				EndLine: r.EndLine,
				Code:    r.Code,
				Value:   r.Value,
				Type:    r.Type,
			}

		case "field":
			r, err := ReadField(loc.Name, targetFile)
			if err != nil {
				continue
			}
			res = &ReadResult{
				Success: true,
				Name:    r.Name,
				Kind:    "field",
				File:    r.File,
				Line:    r.Line,
				EndLine: r.Line,
				Code:    r.Code,
				Type:    r.Type,
			}
		}

		if res != nil {
			results = append(results, *res)
		}
	}

	return &ReadResults{
		Success: true,
		Results: results,
		Count:   len(results),
	}, nil
}

func Replace(name, file string, newCode io.Reader) (*ModifyResult, error) {
	loc, err := locateSymbol(name, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return nil, fmt.Errorf("symbol %s not found", name)
	}

	if file == "" {
		file = loc.File
	}

	switch loc.Kind {
	case "func":
		return ReplaceFunc(name, file, newCode)
	case "struct", "interface", "type":
		return ReplaceType(name, file, newCode)
	case "var", "const":
		return ReplaceVarConst(name, file, newCode)
	default:
		return nil, fmt.Errorf("cannot replace symbol of kind %s", loc.Kind)
	}
}

func Move(name, dstFile string) (*ModifyResult, error) {
	// Determine package from destination file
	dstPkg, err := getPackageName(dstFile)
	if err != nil {
		return nil, fmt.Errorf("cannot determine package of %s: %v", dstFile, err)
	}

	dstDir := filepath.Dir(dstFile)
	absDstDir, _ := filepath.Abs(dstDir)

	// Search symbol only in the same package
	matches, err := searchSymbols(name, absDstDir, "")
	if err != nil {
		return nil, err
	}

	// Filter to exact match in this package
	var loc *SymbolLocation
	for _, m := range matches {
		pkg, _ := getPackageName(m.File)
		if pkg == dstPkg && m.Name == name {
			loc = &m
			break
		}
	}
	if loc == nil {
		for _, m := range matches {
			pkg, _ := getPackageName(m.File)
			if pkg == dstPkg {
				loc = &m
				break
			}
		}
	}
	if loc == nil {
		return nil, fmt.Errorf("symbol %s not found in package %s", name, dstPkg)
	}

	srcFile := loc.File

	// Don't move to same file
	absSrc, _ := filepath.Abs(srcFile)
	absDst, _ := filepath.Abs(dstFile)
	if absSrc == absDst {
		return nil, fmt.Errorf("symbol %s is already in %s", name, dstFile)
	}

	switch loc.Kind {
	case "func":
		return MoveFunc(name, dstFile, srcFile)
	case "struct", "interface", "type":
		return MoveType(name, dstFile, srcFile)
	case "var", "const":
		return MoveVarConst(name, dstFile, srcFile)
	default:
		return nil, fmt.Errorf("cannot move symbol of kind %s", loc.Kind)
	}
}

func Delete(name, file string) (*ModifyResult, error) {
	loc, err := locateSymbol(name, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return nil, fmt.Errorf("symbol %s not found", name)
	}

	if file == "" {
		file = loc.File
	}

	switch loc.Kind {
	case "func":
		return DeleteFunc(name, file)
	case "struct", "interface", "type":
		return DeleteType(name, file)
	case "var", "const":
		return DeleteVarConst(name, file)
	default:
		return nil, fmt.Errorf("cannot delete symbol of kind %s", loc.Kind)
	}
}
