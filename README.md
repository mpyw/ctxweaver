# ctxweaver

[![Go Reference](https://pkg.go.dev/badge/github.com/mpyw/ctxweaver.svg)](https://pkg.go.dev/github.com/mpyw/ctxweaver)
[![Go Report Card](https://goreportcard.com/badge/github.com/mpyw/ctxweaver)](https://goreportcard.com/report/github.com/mpyw/ctxweaver)
[![Codecov](https://codecov.io/gh/mpyw/ctxweaver/graph/badge.svg)](https://codecov.io/gh/mpyw/ctxweaver)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> [!NOTE]
> This project was written by AI (Claude Code).

A Go code generator that weaves statements into functions receiving context-like parameters.

## Overview

`ctxweaver` automatically inserts or updates statements at the beginning of functions that receive `context.Context` or other context-carrying types. This is useful for:

- **APM instrumentation**: Automatically add tracing segments to all context-aware functions
- **Logging setup**: Insert structured logging initialization
- **Metrics collection**: Add timing or counting instrumentation
- **Custom middleware**: Any pattern that needs to run at function entry with context

## How It Works

```go
// Before: A function receiving context
func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
    // business logic...
}

// After: ctxweaver inserts your template at the top
func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
    defer apm.StartSegment(ctx, "(*myapp.Service).ProcessOrder").End()

    // business logic...
}
```

The inserted statement is fully customizable via Go templates.

## Installation & Usage

### Using [`go install`](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies)

```bash
go install github.com/mpyw/ctxweaver/cmd/ctxweaver@latest
ctxweaver -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' ./...
```

### Using [`go tool`](https://pkg.go.dev/cmd/go#hdr-Run_specified_go_tool) (Go 1.24+)

```bash
# Add to go.mod as a tool dependency
go get -tool github.com/mpyw/ctxweaver/cmd/ctxweaver@latest

# Run via go tool
go tool ctxweaver -template='...' ./...
```

### Using [`go run`](https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program)

```bash
go run github.com/mpyw/ctxweaver/cmd/ctxweaver@latest -template='...' ./...
```

> [!CAUTION]
> To prevent supply chain attacks, pin to a specific version tag instead of `@latest` in CI/CD pipelines (e.g., `@v0.1.0`).

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-template` | (required) | Go template for the statement to insert |
| `-template-file` | | Path to a file containing the template (alternative to `-template`) |
| `-import` | | Import to add (format: `path` or `alias=path`), can be repeated |
| `-context-carrier` | | Additional types to treat as context carriers (see below) |
| `-test` | `false` | Process test files (`*_test.go`) |
| `-dry-run` | `false` | Print changes without writing files |
| `-verbose` | `false` | Print processed files |

### Target Specification

ctxweaver accepts Go package patterns as arguments:

```bash
# Process all packages recursively
ctxweaver -template='...' ./...

# Process specific packages
ctxweaver -template='...' ./pkg/service ./pkg/repository

# Process by import path
ctxweaver -template='...' github.com/example/myapp/...
```

Alternatively, use file glob patterns with `-files`:

```bash
ctxweaver -template='...' -files='pkg/**/*.go,internal/**/*.go'
```

## Template System

### Available Variables

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Ctx}}` | `string` | Expression to access `context.Context` |
| `{{.CtxVar}}` | `string` | Name of the context parameter variable |
| `{{.FuncName}}` | `string` | Fully qualified function name |
| `{{.PackageName}}` | `string` | Package name |
| `{{.PackagePath}}` | `string` | Full import path of the package |
| `{{.FuncBaseName}}` | `string` | Function name without package/receiver |
| `{{.ReceiverType}}` | `string` | Receiver type name (empty if not a method) |
| `{{.ReceiverVar}}` | `string` | Receiver variable name (empty if not a method) |
| `{{.IsMethod}}` | `bool` | Whether this is a method |
| `{{.IsPointerReceiver}}` | `bool` | Whether the receiver is a pointer |

### Built-in Functions

| Function | Description |
|----------|-------------|
| `quote` | Wraps string in double quotes |
| `backtick` | Wraps string in backticks |

### Examples

**APM Tracing (New Relic style)**
```bash
ctxweaver \
  -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
  -import='github.com/example/myapp/internal/apm' \
  ./...
```

**OpenTelemetry Spans**
```bash
ctxweaver \
  -template='{{.CtxVar}}, span := tracer.Start({{.Ctx}}, {{.FuncName | quote}}); defer span.End()' \
  -import='go.opentelemetry.io/otel/trace' \
  ./...
```

**Structured Logging**
```bash
ctxweaver \
  -template='log := zerolog.Ctx({{.Ctx}}).With().Str("func", {{.FuncName | quote}}).Logger()' \
  -import='github.com/rs/zerolog' \
  ./...
```

**Multi-line Template (via file)**
```go
// template.txt
{{.CtxVar}}, span := otel.Tracer("myapp").Start({{.Ctx}}, {{.FuncName | quote}})
defer span.End()
```

```bash
ctxweaver -template-file=template.txt ./...
```

## Context Carriers

By default, ctxweaver recognizes `context.Context` as a context type. You can add additional types that carry context:

```bash
ctxweaver \
  -template='...' \
  -context-carrier='github.com/labstack/echo/v4.Context=.Request().Context()' \
  -context-carrier='github.com/urfave/cli/v2.Context=.Context' \
  -context-carrier='github.com/spf13/cobra.Command=.Context()' \
  ./...
```

**Format**: `package/path.TypeName=.accessor`

- The accessor is appended to the parameter variable to get `context.Context`
- For `context.Context` itself, the accessor is empty (just the variable name)
- Pointer types are automatically handled (`*cli.Context` matches `cli.Context` carrier)

### How Accessors Work

| Carrier Type | Accessor | `{{.Ctx}}` becomes |
|--------------|----------|-------------------|
| `context.Context` | (none) | `ctx` |
| `echo.Context` | `.Request().Context()` | `ctx.Request().Context()` |
| `*cli.Context` | `.Context` | `ctx.Context` |
| `*cobra.Command` | `.Context()` | `cmd.Context()` |

## Directives

### `//ctxweaver:skip`

Skip processing for a specific function or entire file:

```go
//ctxweaver:skip
func legacyHandler(ctx context.Context) {
    // This function will not be modified
}
```

File-level skip (place at the top of the file):

```go
//ctxweaver:skip

package legacy

// All functions in this file will be skipped
```

### Existing Statement Detection

ctxweaver detects if a matching statement already exists and:

1. **Skips** if the statement is up-to-date
2. **Updates** if the function name in the statement doesn't match (e.g., after rename)
3. **Inserts** if no matching statement exists

Detection is based on pattern matching the template structure, not exact string comparison.

## Performance

ctxweaver uses `golang.org/x/tools/go/packages` to load type information efficiently:

- **Single load**: All target packages are loaded in one pass
- **Parallel processing**: Files are processed concurrently
- **Incremental friendly**: Only modified files are written

For large codebases, consider using file patterns to limit scope:

```bash
# Process only specific directories
ctxweaver -template='...' -files='pkg/api/**/*.go'
```

## Import Management

ctxweaver automatically:

1. **Adds imports** specified via `-import` flag when the template uses them
2. **Preserves existing imports** and their aliases
3. **Maintains import grouping** (stdlib, external, local)

> [!NOTE]
> ctxweaver does not reorder or reformat existing imports. Use `goimports` or `gci` after ctxweaver if you need consistent import formatting.

## Comparison with Other Tools

| Feature | ctxweaver | Manual coding | go generate |
|---------|-----------|---------------|-------------|
| Automatic insertion | Yes | No | Partial |
| Keeps code in sync | Yes | No | No |
| Custom templates | Yes | N/A | Yes |
| Type-aware | Yes | N/A | No |
| Preserves comments | Yes | Yes | Varies |

## Development

```bash
# Run tests
go test ./...

# Build CLI
go build -o bin/ctxweaver ./cmd/ctxweaver

# Run on a project
./bin/ctxweaver -template='...' ./...
```

## License

MIT License
