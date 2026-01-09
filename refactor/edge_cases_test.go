package refactor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/night-codes/gorefactor/refactor"
)

func TestReplaceFuncNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	_, err := refactor.ReplaceFunc("NonExistentFunc", testFile, strings.NewReader("func Foo() {}"))
	if err == nil {
		t.Error("expected error for non-existent function")
	}
}

func TestDeleteFuncNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	_, err := refactor.DeleteFunc("NonExistentFunc", testFile)
	if err == nil {
		t.Error("expected error for non-existent function")
	}
}

func TestMoveFuncNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.go")
	dstFile := filepath.Join(tmpDir, "dest.go")
	copyTestFile(t, sampleFile, srcFile)
	os.WriteFile(dstFile, []byte("package test\n"), 0644)

	_, err := refactor.MoveFunc("NonExistentFunc", dstFile, srcFile)
	if err == nil {
		t.Error("expected error for non-existent function")
	}
}

func TestReplaceTypeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	_, err := refactor.ReplaceType("NonExistentType", testFile, strings.NewReader("type Foo struct{}"))
	if err == nil {
		t.Error("expected error for non-existent type")
	}
}

func TestReadTypeNotFound(t *testing.T) {
	_, err := refactor.ReadType("NonExistentType", sampleFile)
	if err == nil {
		t.Error("expected error for non-existent type")
	}
}

func TestFindVarAndConst(t *testing.T) {
	result, err := refactor.FindVar("GlobalConfig", testdataDir)
	if err != nil {
		t.Fatalf("FindVar error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("expected 1 var, got %d", result.Count)
	}
	if result.Count > 0 && result.Matches[0].Kind != "var" {
		t.Errorf("expected kind 'var', got %q", result.Matches[0].Kind)
	}

	result, err = refactor.FindConst("Version", testdataDir)
	if err != nil {
		t.Fatalf("FindConst error: %v", err)
	}
	if result.Count != 1 {
		t.Errorf("expected 1 const, got %d", result.Count)
	}
	if result.Count > 0 && result.Matches[0].Kind != "const" {
		t.Errorf("expected kind 'const', got %q", result.Matches[0].Kind)
	}
}

func TestSymbolsWithPackageName(t *testing.T) {
	// testdata package is in ../testdata relative to refactor/
	result, err := refactor.Symbols(testdataDir)
	if err != nil {
		t.Fatalf("Symbols error: %v", err)
	}
	if result.Package != "testdata" {
		t.Errorf("expected package 'testdata', got %q", result.Package)
	}
	if result.Count == 0 {
		t.Error("expected symbols, got 0")
	}
}

func TestSymbolsPathNormalization(t *testing.T) {
	// Use absolute path variations
	tests := []string{
		testdataDir,
		testdataDir + "/",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			result, err := refactor.Symbols(path)
			if err != nil {
				t.Fatalf("Symbols(%q) error: %v", path, err)
			}
			if result.Package != "testdata" {
				t.Errorf("Symbols(%q): expected package 'testdata', got %q", path, result.Package)
			}
		})
	}
}

func TestContextVariousScopes(t *testing.T) {
	tests := []struct {
		name      string
		line      int
		wantScope string
		wantFunc  string
	}{
		{"package line", 1, "package", ""},
		{"const", 3, "const", ""},
		{"var", 5, "var", ""},
		{"type", 8, "type", ""},
		{"func body", 35, "func_body", "ProcessOrder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := sampleFile + ":" + itoa(tt.line)
			result, err := refactor.Context(pos)
			if err != nil {
				t.Fatalf("Context error: %v", err)
			}
			if result.Scope != tt.wantScope {
				t.Errorf("got scope %q, want %q", result.Scope, tt.wantScope)
			}
			if tt.wantFunc != "" && result.Func != tt.wantFunc {
				t.Errorf("got func %q, want %q", result.Func, tt.wantFunc)
			}
		})
	}
}

func TestContextInvalidFormat(t *testing.T) {
	_, err := refactor.Context("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestModifyResultHasFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	result, err := refactor.DeleteFunc("helper", testFile)
	if err != nil {
		t.Fatalf("DeleteFunc error: %v", err)
	}
	if result.File == "" {
		t.Error("ModifyResult.File should not be empty")
	}
	if result.File != testFile {
		t.Errorf("expected file %q, got %q", testFile, result.File)
	}
}

func TestFindFuncReturnsLocation(t *testing.T) {
	result, err := refactor.FindFunc("ProcessOrder", testdataDir)
	if err != nil {
		t.Fatalf("FindFunc error: %v", err)
	}
	if result.Count == 0 {
		t.Fatal("expected at least 1 match")
	}

	match := result.Matches[0]
	if match.File == "" {
		t.Error("File should not be empty")
	}
	if match.Line == 0 {
		t.Error("Line should not be 0")
	}
	if match.Column == 0 {
		t.Error("Column should not be 0")
	}
	if match.Signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestReadFuncReturnsFullInfo(t *testing.T) {
	result, err := refactor.ReadFunc("ProcessOrder", sampleFile)
	if err != nil {
		t.Fatalf("ReadFunc error: %v", err)
	}

	if result.File == "" {
		t.Error("File should not be empty")
	}
	if result.Line == 0 {
		t.Error("Line should not be 0")
	}
	if result.EndLine == 0 {
		t.Error("EndLine should not be 0")
	}
	if result.EndLine <= result.Line {
		t.Errorf("EndLine (%d) should be greater than Line (%d)", result.EndLine, result.Line)
	}
	if result.Signature == "" {
		t.Error("Signature should not be empty")
	}
	if result.Code == "" {
		t.Error("Code should not be empty")
	}
}

func itoa(i int) string {
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
