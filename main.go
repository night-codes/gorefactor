package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/night-codes/gorefactor/refactor"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var result any
	var err error

	switch cmd {
	// === Project overview ===
	case "project":
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		result, err = refactor.ProjectOverview(dir)

	case "packages":
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		result, err = refactor.ListPackages(dir)

	case "symbols":
		if len(args) < 1 {
			fatal("usage: gorefactor symbols <file.go|package>")
		}
		result, err = refactor.Symbols(args[0])

	case "api":
		pkg := "."
		if len(args) > 0 {
			pkg = args[0]
		}
		result, err = refactor.PackageAPI(pkg)

	// === Find & Read (unified) ===
	case "find":
		if len(args) < 1 {
			fatal("usage: gorefactor find <name> [dir]")
		}
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		result, err = refactor.FindSymbol(args[0], dir)

	case "read":
		if len(args) < 1 {
			fatal("usage: gorefactor read <name> [file]")
		}
		file := ""
		if len(args) > 1 {
			file = args[1]
		}
		result, err = refactor.Read(args[0], file)


	case "grep":
		if len(args) < 1 {
			fatal("usage: gorefactor grep <pattern> [dir] [-i] [-r] [-f <filepattern>]")
		}
		dir := "."
		opts := &refactor.GrepOptions{}
		pattern := args[0]
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-i":
				opts.IgnoreCase = true
			case "-r":
				opts.Regex = true
			case "-f":
				if i+1 < len(args) {
					opts.FilePattern = args[i+1]
					i++
				}
			default:
				if !strings.HasPrefix(args[i], "-") {
					dir = args[i]
				}
			}
		}
		result, err = refactor.Grep(pattern, dir, opts)
	// === Modify code ===
	case "replace":
		if len(args) < 1 {
			fatal("usage: gorefactor replace <name> [file] < newcode")
		}
		file := ""
		if len(args) > 1 {
			file = args[1]
		}
		result, err = refactor.Replace(args[0], file, os.Stdin)

	case "delete":
		if len(args) < 1 {
			fatal("usage: gorefactor delete <name> [file]")
		}
		file := ""
		if len(args) > 1 {
			file = args[1]
		}
		result, err = refactor.Delete(args[0], file)

	case "add":
		if len(args) < 1 {
			fatal("usage: gorefactor add <file> < newcode")
		}
		result, err = refactor.AddFunc(args[0], os.Stdin)

	case "move":
		if len(args) < 2 {
			fatal("usage: gorefactor move <n> <target.go>")
		}
		result, err = refactor.Move(args[0], args[1])

	// === Lines ===
	case "lines":
		if len(args) < 1 {
			fatal("usage: gorefactor lines <file:N:M> or <file:N>")
		}
		file, start, end, e := refactor.ParseLineRange(args[0])
		if e != nil {
			fatal(e.Error())
		}
		result, err = refactor.ReadLines(file, start, end)

	case "replace-lines":
		if len(args) < 1 {
			fatal("usage: gorefactor replace-lines <file:N:M> < newcontent")
		}
		file, start, end, e := refactor.ParseLineRange(args[0])
		if e != nil {
			fatal(e.Error())
		}
		content, _ := io.ReadAll(os.Stdin)
		result, err = refactor.ReplaceLines(file, start, end, strings.TrimSuffix(string(content), "\n"))

	case "delete-lines":
		if len(args) < 1 {
			fatal("usage: gorefactor delete-lines <file:N:M>")
		}
		file, start, end, e := refactor.ParseLineRange(args[0])
		if e != nil {
			fatal(e.Error())
		}
		result, err = refactor.DeleteLines(file, start, end)

	case "insert-lines":
		if len(args) < 1 {
			fatal("usage: gorefactor insert-lines <file:N> < newcontent")
		}
		file, after, _, e := refactor.ParseLineRange(args[0])
		if e != nil {
			fatal(e.Error())
		}
		content, _ := io.ReadAll(os.Stdin)
		result, err = refactor.InsertLines(file, after, strings.TrimSuffix(string(content), "\n"))

	// === Navigation (gopls) ===
	case "definition":
		if len(args) < 1 {
			fatal("usage: gorefactor definition <symbol>")
		}
		result, err = refactor.Definition(args[0])

	case "references":
		if len(args) < 1 {
			fatal("usage: gorefactor references <symbol>")
		}
		result, err = refactor.References(args[0])

	case "implementations":
		if len(args) < 1 {
			fatal("usage: gorefactor implementations <interface>")
		}
		result, err = refactor.Implementations(args[0])

	case "callers":
		if len(args) < 1 {
			fatal("usage: gorefactor callers <func>")
		}
		result, err = refactor.Callers(args[0])

	case "context":
		if len(args) < 1 {
			fatal("usage: gorefactor context <file:line[:col]>")
		}
		result, err = refactor.Context(args[0])

	// === Refactoring ===
	case "rename":
		if len(args) < 2 {
			fatal("usage: gorefactor rename <old> <new>")
		}
		result, err = refactor.Rename(args[0], args[1])

	case "rename-package":
		if len(args) < 2 {
			fatal("usage: gorefactor rename-package <old> <new>")
		}
		result, err = refactor.RenamePackage(args[0], args[1])

	// === Validation ===
	case "format":
		target := "./..."
		if len(args) > 0 {
			target = args[0]
		}
		result, err = refactor.Format(target)

	case "check":
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		result, err = refactor.Check(dir)

	case "test":
		pkg := "./..."
		if len(args) > 0 {
			pkg = args[0]
		}
		result, err = refactor.Test(pkg)

	case "version":
		fmt.Println(version)
		return

	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		output := map[string]any{"success": false, "error": err.Error()}
		json.NewEncoder(os.Stdout).Encode(output)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func printUsage() {
	usage := `gorefactor - Go refactoring tool for LLM agents

PROJECT
  project [dir]           Project structure and stats
  packages [dir]          List all packages
  symbols <file|pkg>      List symbols in file/package
  api [pkg]               Public API of package

FIND & READ
  find <name> [dir]       Find symbol (func, type, var, const, field)
  read <name> [file]      Read code of function or type
  grep <pattern> [dir] Search text in project (-i ignore case, -r regex)

MODIFY (pipe new code via stdin: echo 'code' | gorefactor ...)
  replace <name> [file]    Replace symbol with new code
  delete <name> [file]     Delete symbol
  add <file>               Append code to file
  move <name> <dst>        Move symbol to another file in same package

LINES (raw line operations, file:N or file:N:M format)
  lines <file:N:M>          Read lines N to M (or single line N)
  replace-lines <file:N:M>  Replace lines N-M with stdin
  delete-lines <file:N:M>   Delete lines N-M
  insert-lines <file:N>     Insert stdin after line N

NAVIGATION (gopls)
  definition <symbol>     Where symbol is defined
  references <symbol>     All usages of symbol
  implementations <iface> Types implementing interface
  callers <func>          Functions calling this function
  context <file:line>     Scope/function at position

REFACTORING (gopls)
  rename <old> <new>           Rename symbol globally
  rename-package <old> <new>   Rename package and fix imports

VALIDATION
  format [target]         Format code (goimports/gofmt)
  check [dir]             Run go build + go vet
  test [pkg]              Run tests

EXAMPLES
  gorefactor find HandleRequest
  gorefactor find User.ID                    # struct field
  gorefactor read UserService.Create
  gorefactor references User.Name
  gorefactor rename OldName NewName
  gorefactor move formatNode types.go

  # Replace symbol (pipe new code via stdin):
  echo 'const Version = "2.0.0"' | gorefactor replace Version
  cat new_func.go | gorefactor replace MyFunc

  # Add code to file:
  echo 'func NewHelper() {}' | gorefactor add helpers.go

Output is JSON. File argument is optional - tool auto-finds in project.`

	fmt.Fprintln(os.Stderr, usage)
}

func fatal(msg string) {
	if !strings.HasPrefix(msg, "{") {
		msg = fmt.Sprintf(`{"success":false,"error":%q}`, msg)
	}
	fmt.Fprintln(os.Stdout, msg)
	os.Exit(1)
}
