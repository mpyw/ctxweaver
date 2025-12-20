# ctxweaver

[![Go Reference](https://pkg.go.dev/badge/github.com/mpyw/ctxweaver.svg)](https://pkg.go.dev/github.com/mpyw/ctxweaver)
[![Go Report Card](https://goreportcard.com/badge/github.com/mpyw/ctxweaver)](https://goreportcard.com/report/github.com/mpyw/ctxweaver)
[![Codecov](https://codecov.io/gh/mpyw/ctxweaver/graph/badge.svg)](https://codecov.io/gh/mpyw/ctxweaver)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> [!NOTE]
> This project was written by AI (Claude Code).

A Go code generator that weaves statements into functions receiving context-like parameters.

## Overview

`ctxweaver` automatically inserts or updates statements at the beginning of functions that receive [`context.Context`](https://pkg.go.dev/context#Context) or other context-carrying types. This is useful for:

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

### Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `template` | `string` | Go template for the statement to insert |
| `template_file` | `string` | Path to template file (alternative to inline `template`) |
| `imports` | `[]string` | Import paths to add when statement is inserted |
| `patterns` | `[]string` | Package patterns to process (overridden by CLI args) |
| `test` | `bool` | Whether to process test files (overridden by `-test` flag) |
| `carriers` | `[]object` | Custom context carrier definitions |
| `hooks.pre` | `[]string` | Shell commands to run before processing |
| `hooks.post` | `[]string` | Shell commands to run after processing |

> [!NOTE]
> - If both `template` and `template_file` are specified, `template` takes precedence.
> - CLI arguments and flags take precedence over config file values.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `ctxweaver.yaml` | Path to configuration file |
| `-dry-run` | `false` | Print changes without writing files |
| `-verbose` | `false` | Print processed files |
| `-silent` | `false` | Suppress all output except errors |
| `-test` | `false` | Process test files (`*_test.go`) |
| `-remove` | `false` | Remove generated statements instead of adding them |
| `-no-hooks` | `false` | Skip pre/post hooks defined in config |

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

# Remove previously inserted statements
ctxweaver -remove ./...

# Skip hooks (useful in CI)
ctxweaver -no-hooks ./...
```

## Template System

> [!TIP]
> For Go `text/template` syntax guide, see: https://docs.gomplate.ca/syntax/

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
| `{{.IsGenericFunc}}` | `bool` | Whether the function has type parameters |
| `{{.IsGenericReceiver}}` | `bool` | Whether the receiver type has type parameters |

### FuncName Format

`{{.FuncName}}` provides a fully qualified function name in the following format:

| Type | Format | Example |
|------|--------|---------|
| Function | `pkg.Func` | `service.CreateUser` |
| Method (pointer receiver) | `pkg.(*Type).Method` | `service.(*UserService).GetByID` |
| Method (value receiver) | `pkg.Type.Method` | `service.UserService.String` |
| Generic function | `pkg.Func[...]` | `service.Process[...]` |
| Generic method (pointer) | `pkg.(*Type[...]).Method` | `service.(*Container[...]).Get` |
| Generic method (value) | `pkg.Type[...].Method` | `service.Wrapper[...].Unwrap` |

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

If you need a different naming format, you can build it yourself using template variables. The following example replicates the default `{{.FuncName}}` behavior:

```yaml
template: |
  {{- $receiver := .ReceiverType -}}
  {{- if .IsGenericReceiver -}}
    {{- $receiver = printf "%s[...]" .ReceiverType -}}
  {{- end -}}
  {{- $name := "" -}}
  {{- if .IsMethod -}}
    {{- if .IsPointerReceiver -}}
      {{- $name = printf "%s.(*%s).%s" .PackageName $receiver .FuncBaseName -}}
    {{- else -}}
      {{- $name = printf "%s.%s.%s" .PackageName $receiver .FuncBaseName -}}
    {{- end -}}
  {{- else -}}
    {{- if .IsGenericFunc -}}
      {{- $name = printf "%s.%s[...]" .PackageName .FuncBaseName -}}
    {{- else -}}
      {{- $name = printf "%s.%s" .PackageName .FuncBaseName -}}
    {{- end -}}
  {{- end -}}
  defer newrelic.FromContext({{.Ctx}}).StartSegment({{$name | quote}}).End()
imports:
  - github.com/newrelic/go-agent/v3/newrelic
```

This gives you full control over the naming format. Available variables for building custom names:
- `{{.PackageName}}` - Package name (e.g., `service`)
- `{{.PackagePath}}` - Full import path (e.g., `github.com/example/myapp/pkg/service`)
- `{{.ReceiverType}}` - Receiver type name without generics (e.g., `UserService`, `Container`)
- `{{.FuncBaseName}}` - Function/method name (e.g., `GetByID`)
- `{{.IsMethod}}` - `true` if method, `false` if function
- `{{.IsPointerReceiver}}` - `true` if pointer receiver
- `{{.IsGenericFunc}}` - `true` if generic function (e.g., `func Foo[T any]()`)
- `{{.IsGenericReceiver}}` - `true` if generic receiver type (e.g., `func (c *Container[T]) Method()`)

## Built-in Context Carriers

ctxweaver recognizes the following types as context carriers (checks the **first parameter** only):

| Type | Accessor | Notes |
|------|----------|-------|
| [`context.Context`](https://pkg.go.dev/context#Context) | (none) | Standard library |
| [`echo.Context`](https://pkg.go.dev/github.com/labstack/echo/v4#Context) | `.Request().Context()` | Echo framework |
| [`*cli.Context`](https://pkg.go.dev/github.com/urfave/cli/v2#Context) | `.Context` | urfave/cli |
| [`*cobra.Command`](https://pkg.go.dev/github.com/spf13/cobra#Command) | `.Context()` | Cobra |
| [`*gin.Context`](https://pkg.go.dev/github.com/gin-gonic/gin#Context) | `.Request.Context()` | Gin |
| [`*fiber.Ctx`](https://pkg.go.dev/github.com/gofiber/fiber/v2#Ctx) | `.Context()` | Fiber |

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

## Hooks

ctxweaver supports pre and post hooks to run shell commands before and after processing.

```yaml
hooks:
  pre:
    - go mod tidy
  post:
    - gci write .
    - gofmt -w .
```

### Pre Hooks

Commands run sequentially before processing. If any command fails (non-zero exit), processing is aborted and no files are modified. Useful for:

- Running `go mod tidy` to ensure dependencies are up to date
- Validating preconditions

### Post Hooks

Commands run sequentially after processing. If any command fails, an error is reported but files have already been modified. Useful for:

- Formatting code with `gofmt`
- Organizing imports with `gci` or `goimports`
- Running linters with auto-fix

> [!TIP]
> ctxweaver adds imports but does not organize them. Since `goimports` only adds/removes imports without reordering, use tools like [`gci`](https://github.com/daixiang0/gci) or `golangci-lint run --fix` (with gci enabled) to enforce consistent import ordering.
>
> **Recommended post hooks:**
> ```yaml
> hooks:
>   post:
>     - gci write .
>     - gofmt -w .
> ```
>
> Or with golangci-lint:
> ```yaml
> hooks:
>   post:
>     - golangci-lint run --fix ./...
> ```

Use the `-no-hooks` flag to skip hooks (useful for CI or when running ctxweaver as part of a larger pipeline).

## Documentation

- [Architecture](./docs/ARCHITECTURE.md) - Technical specification and design decisions
- [CLAUDE.md](./CLAUDE.md) - AI assistant guidance for development

## Development

```bash
# Run tests
go test ./...

# Build CLI
go build -o bin/ctxweaver ./cmd/ctxweaver

# Run on a project
./bin/ctxweaver -config=ctxweaver.yaml ./...
```

## Why ctxweaver?

For Go instrumentation, there are two main approaches: **compile-time instrumentation** (like [Datadog Orchestrion](https://github.com/DataDog/orchestrion)) and **code generation** (like ctxweaver). Here's how they compare:

| Feature | ctxweaver | [Orchestrion](https://github.com/DataDog/orchestrion) |
|---------|-----------|-------------|
| Approach | Explicit code generation | Compile-time AST injection |
| Output visibility | Generated code in source files | Hidden in build process |
| Comment preservation | Yes ([DST](https://github.com/dave/dst)) | N/A (no source modification) |
| Vendor lock-in | None (template-based) | Datadog by default |
| Custom templates | Full control via Go templates | Limited (`//dd:span` directive) |
| Framework support | Built-in (Echo, Gin, Fiber, etc.) | Via integrations |
| Reversibility | `ctxweaver -remove` | Remove toolchain config |
| Git diff | Visible changes | No source changes |

### When to Choose ctxweaver

1. **You want visible, reviewable code**: Generated statements appear in your source files and git history. Code reviewers can see exactly what instrumentation is added.

2. **You need full template control**: Define exactly what gets inserted using Go templates. Not limited to predefined patterns.

3. **You want vendor independence**: Works with any APM (New Relic, OpenTelemetry, custom solutions). No SDK lock-in.

4. **You use context-carrying frameworks**: Built-in support for Echo, Gin, Fiber, Cobra, urfave/cli context types.

5. **You want idempotent updates**: Re-running ctxweaver updates existing statements (e.g., after function rename) without duplication.

### When to Choose Orchestrion

1. **You prefer zero source changes**: Instrumentation happens at compile time with no visible code modifications.

2. **You use Datadog**: Native integration with Datadog APM and ASM.

3. **You want automatic library instrumentation**: Orchestrion can instrument third-party library calls automatically.

> [!NOTE]
> Traditional AOP libraries ([gogap/aop](https://github.com/gogap/aop), [AspectGo](https://github.com/AkihiroSuda/aspectgo)) exist but are largely unmaintained. Go's culture favors explicit code over implicit magic, which is why ctxweaver generates visible source code rather than hiding instrumentation in the build process.

## Related Tools

- [goroutinectx](https://github.com/mpyw/goroutinectx) - Goroutine context propagation linter
- [zerologlintctx](https://github.com/mpyw/zerologlintctx) - Zerolog context propagation linter
- [gormreuse](https://github.com/mpyw/gormreuse) - GORM instance reuse linter

## License

MIT License
