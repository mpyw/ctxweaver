# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ctxweaver** is a Go code generator that weaves statements into functions receiving [`context.Context`](https://pkg.go.dev/context#Context) or other context-carrying types. It automatically inserts or updates statements at the beginning of context-aware functions for APM instrumentation, logging setup, or custom middleware.

### Key Features

- **Template-based insertion**: Fully customizable via Go templates
- **Context carrier detection**: Recognizes `context.Context` and framework-specific types (Echo, Gin, Fiber, etc.)
- **Package filtering**: Filter packages by patterns and regex (only/omit)
- **Function filtering**: Filter functions by type, scope, and regex patterns
- **Statement detection**: Detects existing statements and updates them when function names change
- **Comment preservation**: Uses DST (Decorated Syntax Tree) to preserve comments and formatting

### Directives

- `//ctxweaver:skip` - Skip processing for a function or entire file

## Architecture

```
ctxweaver/
├── cmd/
│   └── ctxweaver/              # CLI entry point
│       └── main.go
├── pkg/
│   ├── config/                 # YAML configuration
│   │   └── config.go           # Config parsing, carrier definitions
│   ├── processor/              # Core processing logic
│   │   ├── processor.go        # Main processor struct
│   │   ├── process.go          # File processing pipeline
│   │   ├── inspect.go          # Function inspection, carrier matching
│   │   └── action.go           # Insert/Update/Remove actions
│   ├── template/               # Template handling
│   │   ├── template.go         # Template parsing and rendering
│   │   └── vars.go             # Template variable construction
│   └── carrier/                # Context carrier utilities
│       └── match.go            # Carrier type matching
├── internal/
│   ├── directive/              # Directive parsing
│   │   └── skip.go             # //ctxweaver:skip handling
│   ├── dstutil/                # DST utilities
│   │   ├── matcher.go          # Statement pattern matching
│   │   └── stmt.go             # Statement manipulation
│   ├── color.go                # TTY-aware color output utilities
│   └── helpers.go              # Shared utilities
├── testdata/                   # Integration test fixtures
├── docs/
│   └── ARCHITECTURE.md         # Technical specification
└── README.md
```

### Key Design Decisions

1. **DST over AST**: Uses [dave/dst](https://github.com/dave/dst) to preserve comments and formatting
2. **Single packages.Load**: All target packages loaded in one pass for performance
3. **YAML configuration**: Complex settings (templates, imports, carriers) in config file
4. **Embedded default carriers**: Built-in support for common frameworks via `//go:embed`
5. **First parameter only**: Only checks first function parameter for context (Go convention)
6. **No import ordering**: Uses external tools (`goimports`/`gci`) for import formatting

### File Processing Pipeline

```
1. Load config (YAML)
2. Parse template
3. Create carrier registry (defaults + custom)
4. Compile regex patterns (packages.regexps, functions.regexps)
5. packages.Load(patterns)
6. For each package:
   a. Check packages.regexps (only/omit) against package import path
   b. For each file:
      - Check file-level skip directive
      - Parse with fresh fset
      - Convert AST → DST
      - For each function:
        - Check function-level skip
        - Check function filter (types, scopes, regexps)
        - Check first parameter for carrier match
        - Render template with variables
        - Detect existing statement
        - Insert/Update/Skip
      - If modified:
        - Add imports via astutil
        - Format and write
7. Report results
```

## Development Commands

```bash
# Run tests
go test ./...

# Build CLI
go build -o bin/ctxweaver ./cmd/ctxweaver

# Run on a project
./bin/ctxweaver -config=ctxweaver.yaml ./...

# Dry run
./bin/ctxweaver -dry-run -verbose ./...
```

## Testing Strategy

- Integration tests in root `*_test.go` files
- Test fixtures in `testdata/` directory organized by scenario
- Each test directory contains:
  - `config.yaml` - Test configuration
  - `*.input.go` - Input source files
  - `*.golden.go` - Expected output files

## Code Style

- Follow standard Go conventions
- Use `golang.org/x/tools/go/packages` for type information
- Use `dave/dst` for AST manipulation with comment preservation
- Unexported types by default; only export what's needed

### Template Variables

| Variable | Description |
|----------|-------------|
| `Ctx` | Expression to access `context.Context` (e.g., `ctx`, `c.Request().Context()`) |
| `CtxVar` | Context parameter variable name |
| `FuncName` | Fully qualified function name |
| `PackageName` | Package name |
| `PackagePath` | Full import path |
| `FuncBaseName` | Function name without package/receiver |
| `ReceiverType` | Receiver type name (empty if not a method) |
| `ReceiverVar` | Receiver variable name |
| `IsMethod` | Whether this is a method |
| `IsPointerReceiver` | Whether receiver is a pointer |
| `IsGenericFunc` | Whether function has type parameters |
| `IsGenericReceiver` | Whether receiver type has type parameters |

## Related Projects

- [goroutinectx](https://github.com/mpyw/goroutinectx) - Goroutine context propagation linter
- [zerologlintctx](https://github.com/mpyw/zerologlintctx) - Zerolog context propagation linter
- [gormreuse](https://github.com/mpyw/gormreuse) - GORM instance reuse linter
