package refactor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Exported  bool   `json:"exported"`
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Signature string `json:"signature,omitempty"`
	Receiver  string `json:"receiver,omitempty"`
}

type SymbolsResult struct {
	Success bool     `json:"success"`
	Path    string   `json:"path"`
	Package string   `json:"package,omitempty"`
	Symbols []Symbol `json:"symbols"`
	Count   int      `json:"count"`
}

func Symbols(path string) (*SymbolsResult, error) {
	// Normalize path: remove trailing slash and leading ./
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "./")
	if path == "" {
		path = "."
	}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return packageSymbols(path)
		}
		return fileSymbols(path)
	}

	// Path doesn't exist, try to find package by name
	pkgPath, found := findPackageByName(path, ".")
	if found {
		return packageSymbols(pkgPath)
	}

	return nil, err
}

func findPackageByName(name, dir string) (string, bool) {
	absDir, _ := filepath.Abs(dir)
	var result string
	found := false

	filepath.Walk(absDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() || found {
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
			if err == nil && f.Name.Name == name {
				result = path
				found = true
				return filepath.SkipAll
			}
			break
		}
		return nil
	})

	return result, found
}

func fileSymbols(filename string) (*SymbolsResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := Symbol{
				Name:     d.Name.Name,
				Kind:     "func",
				Exported: ast.IsExported(d.Name.Name),
				Line:     fset.Position(d.Pos()).Line,
				EndLine:  fset.Position(d.End()).Line,
			}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				sym.Kind = "method"
				sym.Receiver = formatExpr(d.Recv.List[0].Type)
				sym.Name = sym.Receiver + "." + d.Name.Name
			}
			sym.Signature = formatFuncSignature(d)
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					kind := "type"
					if _, ok := s.Type.(*ast.InterfaceType); ok {
						kind = "interface"
					} else if _, ok := s.Type.(*ast.StructType); ok {
						kind = "struct"
					}
					symbols = append(symbols, Symbol{
						Name:     s.Name.Name,
						Kind:     kind,
						Exported: ast.IsExported(s.Name.Name),
						Line:     fset.Position(s.Pos()).Line,
						EndLine:  fset.Position(s.End()).Line,
					})

				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						symbols = append(symbols, Symbol{
							Name:     name.Name,
							Kind:     kind,
							Exported: ast.IsExported(name.Name),
							Line:     fset.Position(s.Pos()).Line,
							EndLine:  fset.Position(s.End()).Line,
						})
					}
				}
			}
		}
	}

	return &SymbolsResult{
		Success: true,
		Path:    filename,
		Package: file.Name.Name,
		Symbols: symbols,
		Count:   len(symbols),
	}, nil
}

func packageSymbols(pkgPath string) (*SymbolsResult, error) {
	fset := token.NewFileSet()
	var symbols []Symbol
	var pkgName string

	entries, err := os.ReadDir(pkgPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filename := filepath.Join(pkgPath, entry.Name())
		file, err := parser.ParseFile(fset, filename, nil, 0)
		if err != nil {
			continue
		}

		if pkgName == "" {
			pkgName = file.Name.Name
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				sym := Symbol{
					Name:     d.Name.Name,
					Kind:     "func",
					Exported: ast.IsExported(d.Name.Name),
					Line:     fset.Position(d.Pos()).Line,
					EndLine:  fset.Position(d.End()).Line,
				}
				if d.Recv != nil && len(d.Recv.List) > 0 {
					sym.Kind = "method"
					sym.Receiver = formatExpr(d.Recv.List[0].Type)
					sym.Name = sym.Receiver + "." + d.Name.Name
				}
				sym.Signature = formatFuncSignature(d)
				symbols = append(symbols, sym)

			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						kind := "type"
						if _, ok := s.Type.(*ast.InterfaceType); ok {
							kind = "interface"
						} else if _, ok := s.Type.(*ast.StructType); ok {
							kind = "struct"
						}
						symbols = append(symbols, Symbol{
							Name:     s.Name.Name,
							Kind:     kind,
							Exported: ast.IsExported(s.Name.Name),
							Line:     fset.Position(s.Pos()).Line,
							EndLine:  fset.Position(s.End()).Line,
						})

					case *ast.ValueSpec:
						kind := "var"
						if d.Tok == token.CONST {
							kind = "const"
						}
						for _, name := range s.Names {
							symbols = append(symbols, Symbol{
								Name:     name.Name,
								Kind:     kind,
								Exported: ast.IsExported(name.Name),
								Line:     fset.Position(s.Pos()).Line,
								EndLine:  fset.Position(s.End()).Line,
							})
						}
					}
				}
			}
		}
	}

	return &SymbolsResult{
		Success: true,
		Path:    pkgPath,
		Package: pkgName,
		Symbols: symbols,
		Count:   len(symbols),
	}, nil
}
