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
    defer newrelic.FromContext(ctx).StartSegment("myapp.(*Service).ProcessOrder").End()

    // business logic...
}
```

The inserted statement is fully customizable via Go templates.

## Installation & Usage

### Using [`go install`](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies)

```bash
go install github.com/mpyw/ctxweaver/cmd/ctxweaver@latest
ctxweaver ./...
```

### Using [`go tool`](https://pkg.go.dev/cmd/go#hdr-Run_specified_go_tool) (Go 1.24+)

```bash
# Add to go.mod as a tool dependency
go get -tool github.com/mpyw/ctxweaver/cmd/ctxweaver@latest

# Run via go tool
go tool ctxweaver ./...
```

### Using [`go run`](https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program)

```bash
go run github.com/mpyw/ctxweaver/cmd/ctxweaver@latest ./...
```

> [!CAUTION]
> To prevent supply chain attacks, pin to a specific version tag instead of `@latest` in CI/CD pipelines (e.g., `@v0.1.0`).

## Configuration

ctxweaver uses a YAML configuration file. Create `ctxweaver.yaml` in your project root:

```yaml
# ctxweaver.yaml
template: |
  defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()

imports:
  - github.com/newrelic/go-agent/v3/newrelic

patterns:
  - ./...

test: false
```

See [`ctxweaver.example.yaml`](./ctxweaver.example.yaml) for a complete example with all options.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `ctxweaver.yaml` | Path to configuration file |
| `-dry-run` | `false` | Print changes without writing files |
| `-verbose` | `false` | Print processed files |
| `-test` | `false` | Process test files (`*_test.go`) |

### Examples

```bash
# Use default config file (ctxweaver.yaml)
ctxweaver ./...

# Use custom config file
ctxweaver -config=.ctxweaver.yaml ./...

# Dry run - preview changes
ctxweaver -dry-run -verbose ./...

# Include test files
ctxweaver -test ./...
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

### FuncName Format

`{{.FuncName}}` provides a fully qualified function name in the following format:

| Type | Format | Example |
|------|--------|---------|
| Function | `pkg.Func` | `service.CreateUser` |
| Method (pointer receiver) | `pkg.(*Type).Method` | `service.(*UserService).GetByID` |
| Method (value receiver) | `pkg.Type.Method` | `service.UserService.String` |

### Built-in Functions

| Function | Description |
|----------|-------------|
| `quote` | Wraps string in double quotes |
| `backtick` | Wraps string in backticks |

### Basic Example

**New Relic**
```yaml
template: |
  defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()
imports:
  - github.com/newrelic/go-agent/v3/newrelic
```

**OpenTelemetry**
```yaml
template: |
  {{.CtxVar}}, span := otel.Tracer({{.PackageName | quote}}).Start({{.Ctx}}, {{.FuncName | quote}}); defer span.End()
imports:
  - go.opentelemetry.io/otel
```

### Custom Function Name Format

If you need a different naming format, you can build it yourself using template variables:

```yaml
template: |
  {{- $name := "" -}}
  {{- if .IsMethod -}}
    {{- if .IsPointerReceiver -}}
      {{- $name = printf "%s.(*%s).%s" .PackageName .ReceiverType .FuncBaseName -}}
    {{- else -}}
      {{- $name = printf "%s.%s.%s" .PackageName .ReceiverType .FuncBaseName -}}
    {{- end -}}
  {{- else -}}
    {{- $name = printf "%s.%s" .PackageName .FuncBaseName -}}
  {{- end -}}
  defer newrelic.FromContext({{.Ctx}}).StartSegment({{$name | quote}}).End()
imports:
  - github.com/newrelic/go-agent/v3/newrelic
```

This gives you full control over the naming format. Available variables for building custom names:
- `{{.PackageName}}` - Package name (e.g., `service`)
- `{{.PackagePath}}` - Full import path (e.g., `github.com/example/myapp/pkg/service`)
- `{{.ReceiverType}}` - Receiver type (e.g., `UserService`)
- `{{.FuncBaseName}}` - Function/method name (e.g., `GetByID`)
- `{{.IsMethod}}` - `true` if method, `false` if function
- `{{.IsPointerReceiver}}` - `true` if pointer receiver

## Built-in Context Carriers

ctxweaver recognizes the following types as context carriers (checks the **first parameter** only):

| Type | Accessor | Notes |
|------|----------|-------|
| `context.Context` | (none) | Standard library |
| `echo.Context` | `.Request().Context()` | Echo framework |
| `*cli.Context` | `.Context` | urfave/cli |
| `*cobra.Command` | `.Context()` | Cobra |
| `*gin.Context` | `.Request.Context()` | Gin |
| `*fiber.Ctx` | `.Context()` | Fiber |

### Custom Carriers

Add custom carriers in your config file:

```yaml
carriers:
  - package: github.com/example/myapp/pkg/web
    type: Context
    accessor: .Ctx()
```

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

## Existing Statement Detection

ctxweaver detects if a matching statement already exists and:

1. **Skips** if the statement is up-to-date
2. **Updates** if the function name in the statement doesn't match (e.g., after rename)
3. **Inserts** if no matching statement exists

Currently, detection is specific to the `defer XXX.StartSegment(ctx, "name").End()` pattern.

## Performance

ctxweaver uses `golang.org/x/tools/go/packages` to load type information efficiently:

- **Single load**: All target packages are loaded in one pass
- **Accurate type resolution**: Import paths are resolved correctly via type information
- **Comment preservation**: Uses DST (Decorated Syntax Tree) to preserve comments

## Import Management

ctxweaver automatically adds imports specified in the config file when statements are inserted.

> [!NOTE]
> ctxweaver does not reorder or reformat existing imports. Use `goimports` or `gci` after ctxweaver if you need consistent import formatting.

## Development

```bash
# Run tests
go test ./...

# Build CLI
go build -o bin/ctxweaver ./cmd/ctxweaver

# Run on a project
./bin/ctxweaver -config=ctxweaver.yaml ./...
```

## License

MIT License
