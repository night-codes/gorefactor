package refactor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type SymbolLocation struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	Exported  bool   `json:"exported"`
	Signature string `json:"signature,omitempty"`
	Receiver  string `json:"receiver,omitempty"`
	Value     string `json:"value,omitempty"`
	Type      string `json:"type,omitempty"`
	Parent    string `json:"parent,omitempty"`
}

type FindResult struct {
	Success bool             `json:"success"`
	Query   string           `json:"query"`
	Matches []SymbolLocation `json:"matches"`
	Count   int              `json:"count"`
}

func FindSymbol(name, dir string) (*FindResult, error) {
	matches, err := searchSymbols(name, dir, "")
	if err != nil {
		return nil, err
	}
	return &FindResult{
		Success: true,
		Query:   name,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func FindFunc(name, dir string) (*FindResult, error) {
	matches, err := searchSymbols(name, dir, "func")
	if err != nil {
		return nil, err
	}
	return &FindResult{
		Success: true,
		Query:   name,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func FindType(name, dir string) (*FindResult, error) {
	matches, err := searchSymbols(name, dir, "type")
	if err != nil {
		return nil, err
	}
	return &FindResult{
		Success: true,
		Query:   name,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func FindVar(name, dir string) (*FindResult, error) {
	matches, err := searchSymbols(name, dir, "var")
	if err != nil {
		return nil, err
	}
	return &FindResult{
		Success: true,
		Query:   name,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func FindConst(name, dir string) (*FindResult, error) {
	matches, err := searchSymbols(name, dir, "const")
	if err != nil {
		return nil, err
	}
	return &FindResult{
		Success: true,
		Query:   name,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func FindField(name, dir string) (*FindResult, error) {
	matches, err := searchSymbols(name, dir, "field")
	if err != nil {
		return nil, err
	}
	return &FindResult{
		Success: true,
		Query:   name,
		Matches: matches,
		Count:   len(matches),
	}, nil
}

func searchSymbols(name, dir, kindFilter string) ([]SymbolLocation, error) {
	var matches []SymbolLocation

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			// Don't skip the root directory itself
			if path != absDir && (strings.HasPrefix(base, ".") || base == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if kindFilter != "" && kindFilter != "func" {
					continue
				}
				funcName := d.Name.Name
				var receiver string
				if d.Recv != nil && len(d.Recv.List) > 0 {
					receiver = formatExpr(d.Recv.List[0].Type)
					funcName = receiver + "." + funcName
				}
				if matchName(funcName, name) || matchName(d.Name.Name, name) {
					pos := fset.Position(d.Name.Pos())
					matches = append(matches, SymbolLocation{
						Name:      funcName,
						Kind:      "func",
						File:      path,
						Line:      pos.Line,
						Column:    pos.Column,
						EndLine:   fset.Position(d.End()).Line,
						Exported:  ast.IsExported(d.Name.Name),
						Signature: formatFuncSignature(d),
						Receiver:  receiver,
					})
				}

			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if kindFilter != "" && kindFilter != "type" && kindFilter != "field" {
							continue
						}
						typeName := s.Name.Name
						if kindFilter != "field" && matchName(typeName, name) {
							kind := "type"
							if _, ok := s.Type.(*ast.InterfaceType); ok {
								kind = "interface"
							} else if _, ok := s.Type.(*ast.StructType); ok {
								kind = "struct"
							}
							pos := fset.Position(s.Name.Pos())
							matches = append(matches, SymbolLocation{
								Name:     typeName,
								Kind:     kind,
								File:     path,
								Line:     pos.Line,
								Column:   pos.Column,
								EndLine:  fset.Position(s.End()).Line,
								Exported: ast.IsExported(typeName),
							})
						}
						// Search struct fields
						if st, ok := s.Type.(*ast.StructType); ok && st.Fields != nil {
							for _, field := range st.Fields.List {
								for _, fieldName := range field.Names {
									fullFieldName := typeName + "." + fieldName.Name
									if matchName(fullFieldName, name) || matchName(fieldName.Name, name) {
										pos := fset.Position(fieldName.Pos())
										matches = append(matches, SymbolLocation{
											Name:     fullFieldName,
											Kind:     "field",
											File:     path,
											Line:     pos.Line,
											Column:   pos.Column,
											EndLine:  fset.Position(field.End()).Line,
											Exported: ast.IsExported(fieldName.Name),
											Type:     formatExpr(field.Type),
											Parent:   typeName,
										})
									}
								}
							}
						}
					case *ast.ValueSpec:
						kind := "var"
						if d.Tok == token.CONST {
							kind = "const"
						}
						if kindFilter != "" && kindFilter != kind {
							continue
						}
						for i, ident := range s.Names {
							if matchName(ident.Name, name) {
								pos := fset.Position(ident.Pos())
								loc := SymbolLocation{
									Name:     ident.Name,
									Kind:     kind,
									File:     path,
									Line:     pos.Line,
									Column:   pos.Column,
									EndLine:  fset.Position(s.End()).Line,
									Exported: ast.IsExported(ident.Name),
								}
								if s.Type != nil {
									loc.Type = formatExpr(s.Type)
								}
								if len(s.Values) > i {
									loc.Value = formatNode(fset, s.Values[i])
								}
								matches = append(matches, loc)
							}
						}
					}
				}
			}
		}
		return nil
	})

	return matches, err
}

func matchName(fullName, query string) bool {
	if fullName == query {
		return true
	}
	if strings.EqualFold(fullName, query) {
		return true
	}
	if strings.Contains(strings.ToLower(fullName), strings.ToLower(query)) {
		return true
	}
	return false
}

func locateSymbol(name, dir string) (*SymbolLocation, error) {
	matches, err := searchSymbols(name, dir, "")
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	for _, m := range matches {
		if m.Name == name {
			return &m, nil
		}
	}
	return &matches[0], nil
}

func locateFunc(name, dir string) (*SymbolLocation, error) {
	matches, err := searchSymbols(name, dir, "func")
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	for _, m := range matches {
		if m.Name == name {
			return &m, nil
		}
	}
	return &matches[0], nil
}

func locateType(name, dir string) (*SymbolLocation, error) {
	matches, err := searchSymbols(name, dir, "type")
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	for _, m := range matches {
		if m.Name == name {
			return &m, nil
		}
	}
	return &matches[0], nil
}
