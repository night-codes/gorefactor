package refactor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/night-codes/gorefactor/refactor"
)

func TestProjectOverview(t *testing.T) {
	// Test on the gorefactor project itself
	wd, _ := os.Getwd()
	projectDir := filepath.Join(wd, "..")

	result, err := refactor.ProjectOverview(projectDir)
	if err != nil {
		t.Fatalf("ProjectOverview error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Module != "github.com/night-codes/gorefactor" {
		t.Errorf("expected module github.com/night-codes/gorefactor, got %s", result.Module)
	}
	if result.GoFiles < 5 {
		t.Errorf("expected at least 5 go files, got %d", result.GoFiles)
	}
}

func TestListPackages(t *testing.T) {
	wd, _ := os.Getwd()
	projectDir := filepath.Join(wd, "..")

	result, err := refactor.ListPackages(projectDir)
	if err != nil {
		t.Fatalf("ListPackages error: %v", err)
	}
	if result.Count < 1 {
		t.Errorf("expected at least 1 package, got %d", result.Count)
	}
}

func TestCheck(t *testing.T) {
	wd, _ := os.Getwd()
	projectDir := filepath.Join(wd, "..")

	result, err := refactor.Check(projectDir)
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !result.BuildOK {
		t.Errorf("gorefactor should build OK, got errors: %v", result.BuildErrors)
	}
}
