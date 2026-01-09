package refactor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
)

type ReadFuncResult struct {
	Success   bool   `json:"success"`
	Name      string `json:"name"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Receiver  string `json:"receiver,omitempty"`
	Signature string `json:"signature"`
	Code      string `json:"code"`
}

func ReadFunc(name, file string) (*ReadFuncResult, error) {
	if file == "" {
		loc, err := locateFunc(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("function %s not found", name)
		}
		file = loc.File
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if matchFunc(fn, name) {
			result := &ReadFuncResult{
				Success:   true,
				Name:      fn.Name.Name,
				File:      file,
				Line:      fset.Position(fn.Pos()).Line,
				EndLine:   fset.Position(fn.End()).Line,
				Signature: formatFuncSignature(fn),
				Code:      formatNode(fset, fn),
			}
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				result.Receiver = formatExpr(fn.Recv.List[0].Type)
				result.Name = result.Receiver + "." + fn.Name.Name
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("function %s not found in %s", name, file)
}

type ModifyResult struct {
	Success bool   `json:"success"`
	File    string `json:"file"`
	Message string `json:"message"`
}

func ReplaceFunc(name, file string, newCode io.Reader) (*ModifyResult, error) {
	if file == "" {
		loc, err := locateFunc(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("function %s not found", name)
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

	var funcDecl *ast.FuncDecl
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && matchFunc(fn, name) {
			funcDecl = fn
			break
		}
	}

	if funcDecl == nil {
		return nil, fmt.Errorf("function %s not found in %s", name, file)
	}

	newCodeBytes, err := io.ReadAll(newCode)
	if err != nil {
		return nil, err
	}

	startPos := fset.Position(funcDecl.Pos()).Offset
	endPos := fset.Position(funcDecl.End()).Offset

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
		Message: fmt.Sprintf("replaced function %s", name),
	}, nil
}

func DeleteFunc(name, file string) (*ModifyResult, error) {
	if file == "" {
		loc, err := locateFunc(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("function %s not found", name)
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

	var funcDecl *ast.FuncDecl
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && matchFunc(fn, name) {
			funcDecl = fn
			break
		}
	}

	if funcDecl == nil {
		return nil, fmt.Errorf("function %s not found in %s", name, file)
	}

	startPos := fset.Position(funcDecl.Pos()).Offset
	endPos := fset.Position(funcDecl.End()).Offset

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
		Message: fmt.Sprintf("deleted function %s", name),
	}, nil
}

func AddFunc(file string, newCode io.Reader) (*ModifyResult, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	newCodeBytes, err := io.ReadAll(newCode)
	if err != nil {
		return nil, err
	}

	var result []byte
	result = append(result, src...)
	result = append(result, '\n', '\n')
	result = append(result, newCodeBytes...)
	result = append(result, '\n')

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
		Message: "function added",
	}, nil
}

func MoveFunc(name, dstFile, srcFile string) (*ModifyResult, error) {
	if srcFile == "" {
		loc, err := locateFunc(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("function %s not found", name)
		}
		srcFile = loc.File
	}

	readResult, err := ReadFunc(name, srcFile)
	if err != nil {
		return nil, err
	}

	if _, err := DeleteFunc(name, srcFile); err != nil {
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
		Message: fmt.Sprintf("moved %s from %s to %s", name, srcFile, dstFile),
	}, nil
}

func matchFunc(fn *ast.FuncDecl, name string) bool {
	if fn.Name.Name == name {
		return true
	}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := formatExpr(fn.Recv.List[0].Type)
		fullName := recv + "." + fn.Name.Name
		if fullName == name {
			return true
		}
		if len(recv) > 0 && recv[0] == '*' && recv[1:]+"."+fn.Name.Name == name {
			return true
		}
	}
	return false
}

func locateVarConst(name, dir string) (*SymbolLocation, error) {
	matches, err := searchSymbols(name, dir, "")
	if err != nil {
		return nil, err
	}
	for _, m := range matches {
		if (m.Kind == "var" || m.Kind == "const") && m.Name == name {
			return &m, nil
		}
	}
	if len(matches) > 0 {
		for _, m := range matches {
			if m.Kind == "var" || m.Kind == "const" {
				return &m, nil
			}
		}
	}
	return nil, nil
}

type ReadVarConstResult struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	EndLine int    `json:"endLine"`
	Type    string `json:"type,omitempty"`
	Value   string `json:"value,omitempty"`
	Code    string `json:"code"`
}

func ReadVarConst(name, file string) (*ReadVarConstResult, error) {
	if file == "" {
		loc, err := locateVarConst(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("var/const %s not found", name)
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
		if !ok || (genDecl.Tok != token.VAR && genDecl.Tok != token.CONST) {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, ident := range valueSpec.Names {
				if ident.Name == name {
					kind := "var"
					if genDecl.Tok == token.CONST {
						kind = "const"
					}

					result := &ReadVarConstResult{
						Success: true,
						Name:    name,
						Kind:    kind,
						File:    file,
						Line:    fset.Position(genDecl.Pos()).Line,
						EndLine: fset.Position(genDecl.End()).Line,
						Code:    formatNode(fset, genDecl),
					}

					if valueSpec.Type != nil {
						result.Type = formatExpr(valueSpec.Type)
					}
					if len(valueSpec.Values) > i {
						result.Value = formatNode(fset, valueSpec.Values[i])
					}

					return result, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("var/const %s not found in %s", name, file)
}

func ReplaceVarConst(name, file string, newCode io.Reader) (*ModifyResult, error) {
	if file == "" {
		loc, err := locateVarConst(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("var/const %s not found", name)
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
		if !ok || (genDecl.Tok != token.VAR && genDecl.Tok != token.CONST) {
			continue
		}

		for _, spec := range genDecl.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				for _, ident := range valueSpec.Names {
					if ident.Name == name {
						targetDecl = genDecl
						break
					}
				}
			}
		}
	}

	if targetDecl == nil {
		return nil, fmt.Errorf("var/const %s not found in %s", name, file)
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
		Message: fmt.Sprintf("replaced var/const %s", name),
	}, nil
}

func DeleteVarConst(name, file string) (*ModifyResult, error) {
	if file == "" {
		loc, err := locateVarConst(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("var/const %s not found", name)
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
		if !ok || (genDecl.Tok != token.VAR && genDecl.Tok != token.CONST) {
			continue
		}

		for _, spec := range genDecl.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				for _, ident := range valueSpec.Names {
					if ident.Name == name {
						targetDecl = genDecl
						break
					}
				}
			}
		}
	}

	if targetDecl == nil {
		return nil, fmt.Errorf("var/const %s not found in %s", name, file)
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
		Message: fmt.Sprintf("deleted var/const %s", name),
	}, nil
}

func MoveVarConst(name, dstFile, srcFile string) (*ModifyResult, error) {
	if srcFile == "" {
		loc, err := locateVarConst(name, ".")
		if err != nil {
			return nil, err
		}
		if loc == nil {
			return nil, fmt.Errorf("var/const %s not found", name)
		}
		srcFile = loc.File
	}

	readResult, err := ReadVarConst(name, srcFile)
	if err != nil {
		return nil, err
	}

	if _, err := DeleteVarConst(name, srcFile); err != nil {
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
		Message: fmt.Sprintf("moved %s from %s to %s", name, srcFile, dstFile),
	}, nil
}
