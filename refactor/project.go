package refactor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func findGopls() string {
	if path, err := exec.LookPath("gopls"); err == nil {
		return path
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "go", "bin", "gopls"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "gopls"
}

func funcAtLine(file string, line int) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return ""
	}

	var result string
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		start := fset.Position(fn.Pos()).Line
		end := fset.Position(fn.End()).Line
		if line >= start && line <= end {
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				result = formatExpr(fn.Recv.List[0].Type) + "." + fn.Name.Name
			} else {
				result = fn.Name.Name
			}
			return false
		}
		return true
	})
	return result
}

type ProjectInfo struct {
	Success   bool     `json:"success"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Module    string   `json:"module,omitempty"`
	GoVersion string   `json:"goVersion,omitempty"`
	Packages  int      `json:"packages"`
	GoFiles   int      `json:"goFiles"`
	TestFiles int      `json:"testFiles"`
	Dirs      []string `json:"dirs"`
}

func ProjectOverview(dir string) (*ProjectInfo, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	info := &ProjectInfo{
		Success: true,
		Name:    filepath.Base(absDir),
		Path:    absDir,
	}

	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				info.Module = strings.TrimPrefix(line, "module ")
			}
			if strings.HasPrefix(line, "go ") {
				info.GoVersion = strings.TrimPrefix(line, "go ")
			}
		}
	}

	pkgSet := make(map[string]bool)
	var dirs []string

	filepath.Walk(absDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := fi.Name()
		if fi.IsDir() {
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(absDir, path)
			if rel != "." {
				dirs = append(dirs, rel)
			}
		} else if strings.HasSuffix(path, ".go") {
			pkgDir := filepath.Dir(path)
			pkgSet[pkgDir] = true
			if strings.HasSuffix(path, "_test.go") {
				info.TestFiles++
			} else {
				info.GoFiles++
			}
		}
		return nil
	})

	info.Packages = len(pkgSet)
	info.Dirs = dirs

	return info, nil
}

type PackageInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	NumFiles int    `json:"numFiles"`
}

type PackagesResult struct {
	Success  bool          `json:"success"`
	Packages []PackageInfo `json:"packages"`
	Count    int           `json:"count"`
}

func ListPackages(dir string) (*PackagesResult, error) {
	var packages []PackageInfo

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	filepath.Walk(absDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return nil
		}
		base := fi.Name()
		if path != absDir && (strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata") {
			return filepath.SkipDir
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		var pkgName string
		var numFiles int
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			numFiles++
			if pkgName == "" {
				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, filepath.Join(path, e.Name()), nil, parser.PackageClauseOnly)
				if err == nil {
					pkgName = f.Name.Name
				}
			}
		}

		if numFiles > 0 {
			rel, _ := filepath.Rel(absDir, path)
			if rel == "" {
				rel = "."
			}
			packages = append(packages, PackageInfo{
				Name:     pkgName,
				Path:     rel,
				NumFiles: numFiles,
			})
		}

		return nil
	})

	return &PackagesResult{
		Success:  true,
		Packages: packages,
		Count:    len(packages),
	}, nil
}

type CheckResult struct {
	Success     bool     `json:"success"`
	BuildOK     bool     `json:"buildOK"`
	VetOK       bool     `json:"vetOK"`
	BuildErrors []string `json:"buildErrors,omitempty"`
	VetErrors   []string `json:"vetErrors,omitempty"`
}

func Check(dir string) (*CheckResult, error) {
	result := &CheckResult{Success: true, BuildOK: true, VetOK: true}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		result.BuildOK = false
		result.BuildErrors = strings.Split(strings.TrimSpace(string(output)), "\n")
	}

	cmd = exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		result.VetOK = false
		result.VetErrors = strings.Split(strings.TrimSpace(string(output)), "\n")
	}

	return result, nil
}

type TestResult struct {
	Success bool   `json:"success"`
	Passed  bool   `json:"passed"`
	Output  string `json:"output"`
}

func Test(pkg string) (*TestResult, error) {
	cmd := exec.Command("go", "test", "-v", pkg)
	output, err := cmd.CombinedOutput()

	result := &TestResult{
		Success: true,
		Passed:  err == nil,
		Output:  string(output),
	}

	return result, nil
}

type LocalVar struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Type string `json:"type,omitempty"`
	Line int    `json:"line"`
}

type FuncLocalsResult struct {
	Success bool       `json:"success"`
	Func    string     `json:"func"`
	File    string     `json:"file"`
	Params  []LocalVar `json:"params"`
	Results []LocalVar `json:"results"`
	Locals  []LocalVar `json:"locals"`
}

func FuncLocals(name string) (*FuncLocalsResult, error) {
	loc, err := locateFunc(name, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return nil, nil
	}

	// TODO: implement full AST walk for locals
	return &FuncLocalsResult{
		Success: true,
		Func:    name,
		File:    loc.File,
	}, nil
}

type GoplsLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Func   string `json:"func,omitempty"`
	Text   string `json:"text,omitempty"`
}

type DefinitionResult struct {
	Success  bool          `json:"success"`
	Symbol   string        `json:"symbol"`
	Location GoplsLocation `json:"location"`
}

func Definition(symbol string) (*DefinitionResult, error) {
	loc, err := locateSymbol(symbol, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return &DefinitionResult{Success: false}, nil
	}

	return &DefinitionResult{
		Success: true,
		Symbol:  symbol,
		Location: GoplsLocation{
			File: loc.File,
			Line: loc.Line,
		},
	}, nil
}

type ReferencesResult struct {
	Success    bool            `json:"success"`
	Symbol     string          `json:"symbol"`
	References []GoplsLocation `json:"references"`
	Count      int             `json:"count"`
}

func References(symbol string) (*ReferencesResult, error) {
	loc, err := locateSymbol(symbol, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return &ReferencesResult{Success: true, Symbol: symbol, Count: 0}, nil
	}

	col := loc.Column
	if col == 0 {
		col = 1
	}
	pos := fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, col)
	cmd := exec.Command(findGopls(), "references", pos)
	output, err := cmd.Output()
	if err != nil {
		return &ReferencesResult{Success: true, Symbol: symbol, Count: 0}, nil
	}

	var refs []GoplsLocation
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 2 {
			ref := GoplsLocation{
				File: parts[0],
				Line: atoi(parts[1]),
			}
			if len(parts) >= 3 {
				ref.Column = atoi(parts[2])
			}
			ref.Func = funcAtLine(ref.File, ref.Line)
			refs = append(refs, ref)
		}
	}

	return &ReferencesResult{
		Success:    true,
		Symbol:     symbol,
		References: refs,
		Count:      len(refs),
	}, nil
}

func Implementations(symbol string) (*ReferencesResult, error) {
	loc, err := locateSymbol(symbol, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return &ReferencesResult{Success: true, Symbol: symbol, Count: 0}, nil
	}

	col := loc.Column
	if col == 0 {
		col = 1
	}
	pos := fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, col)
	cmd := exec.Command(findGopls(), "implementation", pos)
	output, err := cmd.Output()
	if err != nil {
		return &ReferencesResult{Success: true, Symbol: symbol, Count: 0}, nil
	}

	var refs []GoplsLocation
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 2 {
			ref := GoplsLocation{
				File: parts[0],
				Line: atoi(parts[1]),
			}
			if len(parts) >= 3 {
				ref.Column = atoi(parts[2])
			}
			ref.Func = funcAtLine(ref.File, ref.Line)
			refs = append(refs, ref)
		}
	}

	return &ReferencesResult{
		Success:    true,
		Symbol:     symbol,
		References: refs,
		Count:      len(refs),
	}, nil
}

func Callers(funcName string) (*ReferencesResult, error) {
	return References(funcName)
}

type RenameResult struct {
	Error        string   `json:"error,omitempty"`
	Success      bool     `json:"success"`
	OldName      string   `json:"oldName"`
	NewName      string   `json:"newName"`
	FilesChanged []string `json:"filesChanged"`
}

func Rename(oldName, newName string) (*RenameResult, error) {
	loc, err := locateSymbol(oldName, ".")
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return nil, fmt.Errorf("symbol %s not found", oldName)
	}

	col := loc.Column
	if col == 0 {
		col = 1
	}
	pos := fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, col)
	cmd := exec.Command(findGopls(), "rename", "-l", "-w", pos, newName)
	output, _ := cmd.CombinedOutput()

	var files []string
	var error string
	var success = true
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "gopls: ") {
			error = strings.Split(line, "gopls: ")[1]
			success = false
			break
		}
		if strings.HasSuffix(line, ".go") {
			files = append(files, strings.TrimSpace(line))
		}
	}

	return &RenameResult{
		Error:        error,
		Success:      success,
		OldName:      oldName,
		NewName:      newName,
		FilesChanged: files,
	}, nil
}

func RenameLocal(funcName, oldVar, newVar string) (*RenameResult, error) {
	// TODO: implement via AST
	return &RenameResult{
		Success: false,
	}, nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

type ContextResult struct {
	Success  bool   `json:"success"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Scope    string `json:"scope"`
	Func     string `json:"func,omitempty"`
	Type     string `json:"type,omitempty"`
	Package  string `json:"package"`
	InBody   bool   `json:"inBody,omitempty"`
	LineText string `json:"lineText,omitempty"`
}

func Context(pos string) (*ContextResult, error) {
	parts := strings.Split(pos, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid position format, expected file:line or file:line:col")
	}

	file := parts[0]
	line := atoi(parts[1])
	col := 1
	if len(parts) >= 3 {
		col = atoi(parts[2])
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	result := &ContextResult{
		Success: true,
		File:    file,
		Line:    line,
		Column:  col,
		Package: f.Name.Name,
		Scope:   "package",
	}

	src, _ := os.ReadFile(file)
	if src != nil {
		lines := strings.Split(string(src), "\n")
		if line > 0 && line <= len(lines) {
			result.LineText = lines[line-1]
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		start := fset.Position(n.Pos()).Line
		end := fset.Position(n.End()).Line

		if line < start || line > end {
			return true
		}

		switch node := n.(type) {
		case *ast.FuncDecl:
			funcName := node.Name.Name
			if node.Recv != nil && len(node.Recv.List) > 0 {
				funcName = formatExpr(node.Recv.List[0].Type) + "." + funcName
			}
			result.Func = funcName
			result.Scope = "func"

			if node.Body != nil {
				bodyStart := fset.Position(node.Body.Pos()).Line
				bodyEnd := fset.Position(node.Body.End()).Line
				if line > bodyStart && line < bodyEnd {
					result.Scope = "func_body"
					result.InBody = true
				} else if line == bodyStart || line == bodyEnd {
					result.Scope = "func_body"
					result.InBody = true
				} else {
					result.Scope = "func_signature"
				}
			}

		case *ast.TypeSpec:
			if result.Type == "" {
				result.Type = node.Name.Name
				if result.Func == "" {
					result.Scope = "type"
				}
			}

		case *ast.GenDecl:
			if result.Func == "" && result.Type == "" {
				switch node.Tok {
				case token.VAR:
					result.Scope = "var"
				case token.CONST:
					result.Scope = "const"
				case token.IMPORT:
					result.Scope = "import"
				}
			}
		}

		return true
	})

	return result, nil
}

type RenamePackageResult struct {
	Success      bool     `json:"success"`
	OldName      string   `json:"oldName"`
	NewName      string   `json:"newName"`
	FilesChanged []string `json:"filesChanged"`
	ImportsFixed int      `json:"importsFixed"`
}

func RenamePackage(oldName, newName string) (*RenamePackageResult, error) {
	result := &RenamePackageResult{
		Success: true,
		OldName: oldName,
		NewName: newName,
	}

	absDir, _ := filepath.Abs(".")

	// Find package directory
	var pkgDir string
	filepath.Walk(absDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() || pkgDir != "" {
			return nil
		}
		base := fi.Name()
		if path != absDir && (strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata") {
			return filepath.SkipDir
		}

		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filepath.Join(path, e.Name()), nil, parser.PackageClauseOnly)
			if err == nil && f.Name.Name == oldName {
				pkgDir = path
				return filepath.SkipAll
			}
			break
		}
		return nil
	})

	if pkgDir == "" {
		return nil, fmt.Errorf("package %s not found", oldName)
	}

	// Get module path from go.mod
	var modulePath string
	if data, err := os.ReadFile(filepath.Join(absDir, "go.mod")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "module ") {
				modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
				break
			}
		}
	}

	// Calculate paths
	relPkgDir, _ := filepath.Rel(absDir, pkgDir)
	oldImportPath := modulePath
	if relPkgDir != "." {
		oldImportPath = modulePath + "/" + filepath.ToSlash(relPkgDir)
	}

	// Check if directory name matches package name (can rename dir)
	dirName := filepath.Base(pkgDir)
	canRenameDir := dirName == oldName

	var newPkgDir string
	var newImportPath string
	if canRenameDir {
		newPkgDir = filepath.Join(filepath.Dir(pkgDir), newName)
		newRelDir, _ := filepath.Rel(absDir, newPkgDir)
		newImportPath = modulePath
		if newRelDir != "." {
			newImportPath = modulePath + "/" + filepath.ToSlash(newRelDir)
		}
	} else {
		newPkgDir = pkgDir
		newImportPath = oldImportPath
	}

	// Step 1: Rename package declaration in all files of the package
	entries, _ := os.ReadDir(pkgDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		filePath := filepath.Join(pkgDir, e.Name())
		src, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		oldDecl := "package " + oldName
		newDecl := "package " + newName
		if strings.Contains(string(src), oldDecl) {
			newSrc := strings.Replace(string(src), oldDecl, newDecl, 1)
			if err := os.WriteFile(filePath, []byte(newSrc), 0644); err == nil {
				rel, _ := filepath.Rel(absDir, filePath)
				result.FilesChanged = append(result.FilesChanged, rel)
			}
		}
	}

	// Step 2: Rename directory if applicable
	if canRenameDir && pkgDir != newPkgDir {
		if err := os.Rename(pkgDir, newPkgDir); err != nil {
			return nil, fmt.Errorf("failed to rename directory: %w", err)
		}
		// Update FilesChanged paths
		for i, f := range result.FilesChanged {
			result.FilesChanged[i] = strings.Replace(f, oldName+"/", newName+"/", 1)
		}
	}

	// Step 3: Fix imports in all project files
	filepath.Walk(absDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			if fi != nil && fi.IsDir() {
				base := fi.Name()
				if strings.HasPrefix(base, ".") || base == "vendor" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(src)
		changed := false

		// Fix import path
		if oldImportPath != newImportPath && strings.Contains(content, `"`+oldImportPath+`"`) {
			content = strings.ReplaceAll(content, `"`+oldImportPath+`"`, `"`+newImportPath+`"`)
			changed = true
		}

		// Fix package usage: oldpkg.Something -> newpkg.Something
		if strings.Contains(content, oldName+".") {
			content = strings.ReplaceAll(content, oldName+".", newName+".")
			changed = true
		}

		if changed {
			if err := os.WriteFile(path, []byte(content), 0644); err == nil {
				rel, _ := filepath.Rel(absDir, path)
				alreadyListed := false
				for _, f := range result.FilesChanged {
					if f == rel {
						alreadyListed = true
						break
					}
				}
				if !alreadyListed {
					result.FilesChanged = append(result.FilesChanged, rel)
				}
				result.ImportsFixed++
			}
		}

		return nil
	})

	return result, nil
}
