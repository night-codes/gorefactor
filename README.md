# gorefactor

AST-based Go refactoring tool for LLM agents.

## Install

```bash
go install github.com/night-codes/gorefactor@latest
```

## Commands

### Project Overview

```bash
gorefactor project              # Structure and stats
gorefactor packages             # List all packages
gorefactor symbols ./pkg        # All symbols in package
gorefactor api ./pkg            # Public API only
```

### Find (project-wide search)

```bash
gorefactor find <name> [dir]      # Find symbol (func, type, var, const, field)
```

### Read Code

```bash
gorefactor read <name> [file]     # Read code of function or type
```

### Modify Code

```bash
# Replace function
echo 'func ProcessOrder(ctx context.Context, id int) error {
    return nil
}' | gorefactor replace ProcessOrder

# Delete function
gorefactor delete OldHandler

# Add function to specific file
echo 'func NewHelper() {}' | gorefactor add helpers.go

# Move function between files
gorefactor move ProcessOrder newfile.go
```

### Navigation (via gopls)

```bash
gorefactor definition UserService    # Where defined
gorefactor references ProcessOrder   # All usages
gorefactor implementations Reader    # Types implementing interface
gorefactor callers SaveUser          # Who calls this
```

### Refactoring (via gopls)

```bash
gorefactor rename OldName NewName    # Rename globally
```

### Validation

```bash
gorefactor check     # go build + go vet
gorefactor test      # Run tests
```

## Output

All commands return JSON:

```json
{
	"success": true,
	"name": "ProcessOrder",
	"file": "/path/to/service.go",
	"line": 42,
	"code": "func ProcessOrder(...) { ... }"
}
```

## Why This Tool?

For LLM agents working with Go code:

| Problem                                 | Solution                               |
| --------------------------------------- | -------------------------------------- |
| Read 500-line file to find one function | `read Name` returns just that function |
| Text matching fails on whitespace       | AST-based, works by symbol name        |
| Don't know which file has the code      | Auto-search across project             |
| Replace function, match exact text      | `replace Name` — no text matching      |

## Requirements

-   Go 1.21+
-   `gopls` (for navigation/rename commands)

## License

MIT
