package refactor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/night-codes/gorefactor/refactor"
)

func copyTestFile(t *testing.T, src, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read source: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("failed to write dest: %v", err)
	}
}

func TestReplaceFunc(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	newCode := `func ProcessOrder(id int) error {
	// new implementation
	return nil
}`

	result, err := refactor.ReplaceFunc("ProcessOrder", testFile, strings.NewReader(newCode))
	if err != nil {
		t.Fatalf("ReplaceFunc error: %v", err)
	}
	if !result.Success {
		t.Error("ReplaceFunc returned success=false")
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "// new implementation") {
		t.Error("new implementation not found in file")
	}
	if strings.Contains(string(content), "return nil\n}") && !strings.Contains(string(content), "// new implementation") {
		t.Error("old implementation still present")
	}
}

func TestDeleteFunc(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	result, err := refactor.DeleteFunc("helper", testFile)
	if err != nil {
		t.Fatalf("DeleteFunc error: %v", err)
	}
	if !result.Success {
		t.Error("DeleteFunc returned success=false")
	}

	content, _ := os.ReadFile(testFile)
	if strings.Contains(string(content), "func helper()") {
		t.Error("helper function still present after delete")
	}
	if !strings.Contains(string(content), "func ProcessOrder") {
		t.Error("ProcessOrder was accidentally deleted")
	}
}

func TestAddFunc(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	newFunc := `func NewFunction() string {
	return "hello"
}`

	result, err := refactor.AddFunc(testFile, strings.NewReader(newFunc))
	if err != nil {
		t.Fatalf("AddFunc error: %v", err)
	}
	if !result.Success {
		t.Error("AddFunc returned success=false")
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "func NewFunction()") {
		t.Error("new function not found in file")
	}
}

func TestMoveFunc(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.go")
	dstFile := filepath.Join(tmpDir, "dest.go")

	copyTestFile(t, sampleFile, srcFile)

	destContent := `package testdata

func ExistingFunc() {}
`
	os.WriteFile(dstFile, []byte(destContent), 0644)

	result, err := refactor.MoveFunc("helper", dstFile, srcFile)
	if err != nil {
		t.Fatalf("MoveFunc error: %v", err)
	}
	if !result.Success {
		t.Error("MoveFunc returned success=false")
	}

	srcContent, _ := os.ReadFile(srcFile)
	if strings.Contains(string(srcContent), "func helper()") {
		t.Error("helper still in source file")
	}

	dstContent, _ := os.ReadFile(dstFile)
	if !strings.Contains(string(dstContent), "func helper()") {
		t.Error("helper not found in dest file")
	}
}

func TestReplaceType(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	copyTestFile(t, sampleFile, testFile)

	newType := `type User struct {
	ID        int
	Name      string
	Age       int
	Email     string
	CreatedAt string
}`

	result, err := refactor.ReplaceType("User", testFile, strings.NewReader(newType))
	if err != nil {
		t.Fatalf("ReplaceType error: %v", err)
	}
	if !result.Success {
		t.Error("ReplaceType returned success=false")
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "Email") {
		t.Error("new field Email not found")
	}
	if !strings.Contains(string(content), "CreatedAt") {
		t.Error("new field CreatedAt not found")
	}
}
