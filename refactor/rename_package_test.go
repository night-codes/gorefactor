package refactor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/night-codes/gorefactor/refactor"
)

func TestRenamePackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	gomod := `module example.com/test

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomod), 0644)

	// Create package directory
	pkgDir := filepath.Join(tmpDir, "oldpkg")
	os.MkdirAll(pkgDir, 0755)

	// Create package file
	pkgFile := `package oldpkg

func Helper() string {
	return "hello"
}
`
	os.WriteFile(filepath.Join(pkgDir, "utils.go"), []byte(pkgFile), 0644)

	// Create main.go that imports the package
	mainFile := `package main

import "example.com/test/oldpkg"

func main() {
	println(oldpkg.Helper())
}
`
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainFile), 0644)

	// Change to tmpDir and run rename
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	result, err := refactor.RenamePackage("oldpkg", "newpkg")
	if err != nil {
		t.Fatalf("RenamePackage error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}

	if len(result.FilesChanged) == 0 {
		t.Error("FilesChanged should not be empty")
	}

	// Check directory was renamed
	if _, err := os.Stat(filepath.Join(tmpDir, "newpkg")); os.IsNotExist(err) {
		t.Error("newpkg directory should exist")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "oldpkg")); !os.IsNotExist(err) {
		t.Error("oldpkg directory should not exist")
	}

	// Check package declaration was changed
	content, _ := os.ReadFile(filepath.Join(tmpDir, "newpkg", "utils.go"))
	if !strings.Contains(string(content), "package newpkg") {
		t.Error("package declaration should be changed to newpkg")
	}

	// Check import was updated
	mainContent, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if !strings.Contains(string(mainContent), `"example.com/test/newpkg"`) {
		t.Error("import should be updated to newpkg")
	}
	if !strings.Contains(string(mainContent), "newpkg.Helper()") {
		t.Error("usage should be updated to newpkg.Helper()")
	}
}

func TestRenamePackageNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, err := refactor.RenamePackage("nonexistent", "newname")
	if err == nil {
		t.Error("expected error for non-existent package")
	}
}

func TestRenamePackageSameName(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)

	pkgDir := filepath.Join(tmpDir, "mypkg")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "code.go"), []byte("package mypkg\n"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	result, err := refactor.RenamePackage("mypkg", "mypkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed but not change anything meaningful
	if !result.Success {
		t.Error("expected success even for same name")
	}
}
