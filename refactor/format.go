package refactor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func formatFuncSignature(fn *ast.FuncDecl) string {
	var buf bytes.Buffer
	buf.WriteString("func ")
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		buf.WriteString("(")
		buf.WriteString(formatExpr(fn.Recv.List[0].Type))
		buf.WriteString(") ")
	}
	buf.WriteString(fn.Name.Name)
	buf.WriteString("(")

	var params []string
	if fn.Type.Params != nil {
		for _, p := range fn.Type.Params.List {
			ptype := formatExpr(p.Type)
			if len(p.Names) == 0 {
				params = append(params, ptype)
			} else {
				for _, n := range p.Names {
					params = append(params, n.Name+" "+ptype)
				}
			}
		}
	}
	buf.WriteString(strings.Join(params, ", "))
	buf.WriteString(")")

	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		buf.WriteString(" ")
		if len(fn.Type.Results.List) == 1 && len(fn.Type.Results.List[0].Names) == 0 {
			buf.WriteString(formatExpr(fn.Type.Results.List[0].Type))
		} else {
			buf.WriteString("(")
			var results []string
			for _, r := range fn.Type.Results.List {
				rtype := formatExpr(r.Type)
				if len(r.Names) == 0 {
					results = append(results, rtype)
				} else {
					for _, n := range r.Names {
						results = append(results, n.Name+" "+rtype)
					}
				}
			}
			buf.WriteString(strings.Join(results, ", "))
			buf.WriteString(")")
		}
	}

	return buf.String()
}

func formatSource(src []byte) ([]byte, error) {
	formatted, err := format.Source(src)
	if err == nil {
		return formatted, nil
	}
	// Fallback: try goimports or gofmt
	cmd := exec.Command("goimports")
	cmd.Stdin = bytes.NewReader(src)
	if out, e := cmd.Output(); e == nil {
		return out, nil
	}
	cmd = exec.Command("gofmt")
	cmd.Stdin = bytes.NewReader(src)
	if out, e := cmd.Output(); e == nil {
		return out, nil
	}
	return src, err
}

type FormatResult struct {
	Success      bool     `json:"success"`
	FilesChanged []string `json:"filesChanged"`
	Errors       []string `json:"errors,omitempty"`
}

func Format(target string) (*FormatResult, error) {
	result := &FormatResult{Success: true}

	var files []string
	if target == "./..." {
		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := info.Name()
				if strings.HasPrefix(base, ".") || base == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}
			return nil
		})
	} else {
		info, err := os.Stat(target)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			entries, _ := os.ReadDir(target)
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
					files = append(files, filepath.Join(target, e.Name()))
				}
			}
		} else {
			files = append(files, target)
		}
	}

	for _, file := range files {
		before, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		cmd := exec.Command("goimports", "-w", file)
		if output, err := cmd.CombinedOutput(); err != nil {
			cmd = exec.Command("gofmt", "-w", file)
			if output, err = cmd.CombinedOutput(); err != nil {
				result.Errors = append(result.Errors, strings.TrimSpace(string(output)))
				continue
			}
		}

		after, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if string(before) != string(after) {
			result.FilesChanged = append(result.FilesChanged, file)
		}
	}

	return result, nil
}

func itoa2(i int) string {
	return strconv.Itoa(i)
}

func atoi2(s string) (int, error) {
	return strconv.Atoi(s)
}

func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func formatExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + formatExpr(e.X)
	case *ast.SelectorExpr:
		return formatExpr(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + formatExpr(e.Elt)
	case *ast.MapType:
		return "map[" + formatExpr(e.Key) + "]" + formatExpr(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + formatExpr(e.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + formatExpr(e.Value)
	default:
		return "?"
	}
}


func formatNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, node)
	return buf.String()
}
