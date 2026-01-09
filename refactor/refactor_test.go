package refactor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/night-codes/gorefactor/refactor"
)

var testdataDir string
var sampleFile string

func init() {
	wd, _ := os.Getwd()
	testdataDir = filepath.Join(wd, "..", "testdata")
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		testdataDir = filepath.Join(wd, "testdata")
	}
	sampleFile = filepath.Join(testdataDir, "sample.go")
}

func TestFindFunc(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantCount int
		wantName  string
	}{
		{"exact match", "ProcessOrder", 1, "ProcessOrder"},
		{"method", "Create", 1, "*UserService.Create"},
		{"partial", "Delete", 1, "*UserService.Delete"},
		{"not found", "NonExistent", 0, ""},
		{"case insensitive", "processorder", 1, "ProcessOrder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := refactor.FindFunc(tt.query, testdataDir)
			if err != nil {
				t.Fatalf("FindFunc error: %v", err)
			}
			if result.Count != tt.wantCount {
				t.Errorf("got count %d, want %d", result.Count, tt.wantCount)
			}
			if tt.wantCount > 0 && result.Matches[0].Name != tt.wantName {
				t.Errorf("got name %q, want %q", result.Matches[0].Name, tt.wantName)
			}
		})
	}
}

func TestFindType(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantCount int
		wantKind  string
	}{
		{"struct User", "User", 1, "struct"},
		{"interface", "Reader", 1, "interface"},
		{"config", "Config", 1, "struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := refactor.FindType(tt.query, testdataDir)
			if err != nil {
				t.Fatalf("FindType error: %v", err)
			}
			if result.Count < 1 {
				t.Fatalf("got count %d, want at least 1", result.Count)
			}
			found := false
			for _, m := range result.Matches {
				if m.Kind == tt.wantKind {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected kind %q not found in results", tt.wantKind)
			}
		})
	}
}

func TestReadFunc(t *testing.T) {
	tests := []struct {
		name        string
		funcName    string
		wantSuccess bool
		wantInCode  string
	}{
		{"simple func", "ProcessOrder", true, "func ProcessOrder(id int) error"},
		{"method pointer receiver", "*UserService.Create", true, "func (s *UserService) Create"},
		{"method value receiver", "UserService.List", true, "func (s UserService) List"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := refactor.ReadFunc(tt.funcName, sampleFile)
			if tt.wantSuccess {
				if err != nil {
					t.Fatalf("ReadFunc error: %v", err)
				}
				if !strings.Contains(result.Code, tt.wantInCode) {
					t.Errorf("code doesn't contain %q:\n%s", tt.wantInCode, result.Code)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

func TestReadFuncNotFound(t *testing.T) {
	_, err := refactor.ReadFunc("NonExistent", sampleFile)
	if err == nil {
		t.Error("expected error for non-existent function")
	}
}

func TestReadType(t *testing.T) {
	result, err := refactor.ReadType("User", sampleFile)
	if err != nil {
		t.Fatalf("ReadType error: %v", err)
	}
	if result.Kind != "struct" {
		t.Errorf("got kind %q, want struct", result.Kind)
	}
	if !strings.Contains(result.Code, "ID") || !strings.Contains(result.Code, "int") {
		t.Errorf("code doesn't contain expected fields: %s", result.Code)
	}
}

func TestSymbols(t *testing.T) {
	result, err := refactor.Symbols(sampleFile)
	if err != nil {
		t.Fatalf("Symbols error: %v", err)
	}

	wantSymbols := map[string]string{
		"Version":      "const",
		"GlobalConfig": "var",
		"User":         "struct",
		"UserService":  "struct",
		"Reader":       "interface",
		"ProcessOrder": "func",
		"helper":       "func",
	}

	found := make(map[string]bool)
	for _, sym := range result.Symbols {
		if wantKind, ok := wantSymbols[sym.Name]; ok {
			if sym.Kind != wantKind {
				t.Errorf("symbol %s: got kind %q, want %q", sym.Name, sym.Kind, wantKind)
			}
			found[sym.Name] = true
		}
	}

	for name := range wantSymbols {
		if !found[name] {
			t.Errorf("symbol %s not found", name)
		}
	}
}

func TestPackageAPI(t *testing.T) {
	result, err := refactor.PackageAPI(testdataDir)
	if err != nil {
		t.Fatalf("PackageAPI error: %v", err)
	}

	for _, sym := range result.Symbols {
		if !sym.Exported {
			t.Errorf("non-exported symbol in API: %s", sym.Name)
		}
	}

	hasHelper := false
	for _, sym := range result.Symbols {
		if sym.Name == "helper" {
			hasHelper = true
		}
	}
	if hasHelper {
		t.Error("unexported 'helper' should not be in API")
	}
}
