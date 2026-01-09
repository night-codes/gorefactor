package refactor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ReadTypeResult struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	EndLine int    `json:"endLine"`
	Code    string `json:"code"`
}

func ReadType(name, file string) (*ReadTypeResult, error) {
	if file == "" {
		loc, err := locateType(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("type %s not found", name)
		}
		file = loc.File
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}

			kind := "type"
			if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				kind = "interface"
			} else if _, ok := typeSpec.Type.(*ast.StructType); ok {
				kind = "struct"
			}

			return &ReadTypeResult{
				Success: true,
				Name:    typeSpec.Name.Name,
				Kind:    kind,
				File:    file,
				Line:    fset.Position(genDecl.Pos()).Line,
				EndLine: fset.Position(genDecl.End()).Line,
				Code:    formatNode(fset, genDecl),
			}, nil
		}
	}

	return nil, fmt.Errorf("type %s not found in %s", name, file)
}

type ReadFieldResult struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Parent  string `json:"parent"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Type    string `json:"type"`
	Tag     string `json:"tag,omitempty"`
	Code    string `json:"code"`
}

func ReadField(name, file string) (*ReadFieldResult, error) {
	var typeName, fieldName string
	if idx := strings.LastIndex(name, "."); idx > 0 {
		typeName = name[:idx]
		fieldName = name[idx+1:]
	} else {
		return nil, fmt.Errorf("field name must be in format Type.Field, got %s", name)
	}

	if file == "" {
		loc, err := locateSymbol(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("field %s not found", name)
		}
		file = loc.File
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}

			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				continue
			}

			for _, field := range st.Fields.List {
				for _, ident := range field.Names {
					if ident.Name == fieldName {
						fieldType := formatExpr(field.Type)
						code := fieldName + " " + fieldType
						if field.Tag != nil {
							code += " " + field.Tag.Value
						}

						result := &ReadFieldResult{
							Success: true,
							Name:    name,
							Parent:  typeName,
							File:    file,
							Line:    fset.Position(field.Pos()).Line,
							Type:    fieldType,
							Code:    code,
						}
						if field.Tag != nil {
							result.Tag = field.Tag.Value
						}
						return result, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("field %s not found in %s", name, file)
}

func ReplaceType(name, file string, newCode io.Reader) (*ModifyResult, error) {
	if file == "" {
		loc, err := locateType(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("type %s not found", name)
		}
		file = loc.File
	}

	fset := token.NewFileSet()
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var targetDecl *ast.GenDecl
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == name {
				targetDecl = genDecl
				break
			}
		}
	}

	if targetDecl == nil {
		return nil, fmt.Errorf("type %s not found in %s", name, file)
	}

	newCodeBytes, err := io.ReadAll(newCode)
	if err != nil {
		return nil, err
	}

	startPos := fset.Position(targetDecl.Pos()).Offset
	endPos := fset.Position(targetDecl.End()).Offset

	var result []byte
	result = append(result, src[:startPos]...)
	result = append(result, newCodeBytes...)
	result = append(result, src[endPos:]...)

	formatted, err := formatSource(result)
	if err != nil {
		formatted = result
	}

	if err := os.WriteFile(file, formatted, 0644); err != nil {
		return nil, err
	}

	return &ModifyResult{
		Success: true,
		File:    file,
		Message: fmt.Sprintf("replaced type %s", name),
	}, nil
}

func DeleteType(name, file string) (*ModifyResult, error) {
	if file == "" {
		loc, err := locateType(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("type %s not found", name)
		}
		file = loc.File
	}

	fset := token.NewFileSet()
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var targetDecl *ast.GenDecl
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == name {
				targetDecl = genDecl
				break
			}
		}
	}

	if targetDecl == nil {
		return nil, fmt.Errorf("type %s not found in %s", name, file)
	}

	startPos := fset.Position(targetDecl.Pos()).Offset
	endPos := fset.Position(targetDecl.End()).Offset

	for endPos < len(src) && (src[endPos] == '\n' || src[endPos] == '\r') {
		endPos++
	}

	var result []byte
	result = append(result, src[:startPos]...)
	result = append(result, src[endPos:]...)

	formatted, err := formatSource(result)
	if err != nil {
		formatted = result
	}

	if err := os.WriteFile(file, formatted, 0644); err != nil {
		return nil, err
	}

	return &ModifyResult{
		Success: true,
		File:    file,
		Message: fmt.Sprintf("deleted type %s", name),
	}, nil
}

func MoveType(name, dstFile, srcFile string) (*ModifyResult, error) {
	if srcFile == "" {
		loc, err := locateType(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("type %s not found", name)
		}
		srcFile = loc.File
	}

	readResult, err := ReadType(name, srcFile)
	if err != nil {
		return nil, err
	}

	if _, err := DeleteType(name, srcFile); err != nil {
		return nil, err
	}

	dstSrc, err := os.ReadFile(dstFile)
	if err != nil {
		return nil, err
	}

	var newDst []byte
	newDst = append(newDst, dstSrc...)
	newDst = append(newDst, '\n', '\n')
	newDst = append(newDst, []byte(readResult.Code)...)
	newDst = append(newDst, '\n')

	if err := os.WriteFile(dstFile, newDst, 0644); err != nil {
		return nil, err
	}

	exec.Command("goimports", "-w", dstFile).Run()
	exec.Command("goimports", "-w", srcFile).Run()

	return &ModifyResult{
		Success: true,
		File:    dstFile,
		Message: fmt.Sprintf("moved type %s from %s to %s", name, srcFile, dstFile),
	}, nil
}

type PackageAPIResult struct {
	Success  bool     `json:"success"`
	Package  string   `json:"package"`
	Path     string   `json:"path"`
	Symbols  []Symbol `json:"symbols"`
	NumFiles int      `json:"numFiles"`
}

func PackageAPI(pkgPath string) (*PackageAPIResult, error) {
	result, err := packageSymbols(pkgPath)
	if err != nil {
		return nil, err
	}

	var exported []Symbol
	for _, s := range result.Symbols {
		if s.Exported {
			exported = append(exported, s)
		}
	}

	entries, _ := os.ReadDir(pkgPath)
	numFiles := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			numFiles++
		}
	}

	return &PackageAPIResult{
		Success:  true,
		Package:  result.Package,
		Path:     pkgPath,
		Symbols:  exported,
		NumFiles: numFiles,
	}, nil
}
