# Usage Examples

## APM Instrumentation

### New Relic Style

```bash
ctxweaver \
  -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
  -import='github.com/example/myapp/internal/apm' \
  ./...
```

**Before:**
```go
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    // validation
    if err := req.Validate(); err != nil {
        return nil, err
    }
    // ...
}
```

**After:**
```go
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    defer apm.StartSegment(ctx, "(*myapp.OrderService).CreateOrder").End()

    // validation
    if err := req.Validate(); err != nil {
        return nil, err
    }
    // ...
}
```

### OpenTelemetry

```bash
ctxweaver \
  -template='{{.CtxVar}}, span := otel.Tracer("myapp").Start({{.Ctx}}, {{.FuncName | quote}}); defer span.End()' \
  -import='go.opentelemetry.io/otel' \
  ./...
```

**Result:**
```go
func ProcessPayment(ctx context.Context, amount int) error {
    ctx, span := otel.Tracer("myapp").Start(ctx, "payment.ProcessPayment"); defer span.End()

    // ...
}
```

## Logging

### Zerolog Context Logger

```bash
ctxweaver \
  -template='log := zerolog.Ctx({{.Ctx}}).With().Str("func", {{.FuncName | quote}}).Logger()' \
  -import='github.com/rs/zerolog' \
  ./...
```

**Result:**
```go
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    log := zerolog.Ctx(ctx).With().Str("func", "(*db.UserRepository).FindByID").Logger()

    // ...
}
```

### Slog

```bash
ctxweaver \
  -template='slog.DebugContext({{.Ctx}}, "entering", "func", {{.FuncName | quote}})' \
  -import='log/slog' \
  ./...
```

## Framework-Specific

### Echo Handlers

```bash
ctxweaver \
  -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
  -import='github.com/example/myapp/internal/apm' \
  -context-carrier='github.com/labstack/echo/v4.Context=.Request().Context()' \
  ./pkg/api/...
```

**Before:**
```go
func (h *Handler) GetUser(c echo.Context) error {
    id := c.Param("id")
    // ...
}
```

**After:**
```go
func (h *Handler) GetUser(c echo.Context) error {
    defer apm.StartSegment(c.Request().Context(), "(*api.Handler).GetUser").End()

    id := c.Param("id")
    // ...
}
```

### CLI Applications (urfave/cli)

```bash
ctxweaver \
  -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
  -import='github.com/example/myapp/internal/apm' \
  -context-carrier='github.com/urfave/cli/v2.Context=.Context' \
  ./cmd/...
```

### Cobra Commands

```bash
ctxweaver \
  -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
  -import='github.com/example/myapp/internal/apm' \
  -context-carrier='github.com/spf13/cobra.Command=.Context()' \
  ./cmd/...
```

## Multi-line Templates

For complex insertions, use a template file:

**template.txt:**
```
{{.CtxVar}}, span := otel.Tracer("myapp").Start({{.Ctx}}, {{.FuncName | quote}})
defer span.End()
span.SetAttributes(attribute.String("package", {{.PackagePath | quote}}))
```

```bash
ctxweaver -template-file=template.txt -import='go.opentelemetry.io/otel' ./...
```

## Conditional Templates

Using Go template conditionals:

```bash
ctxweaver \
  -template='{{if .IsMethod}}defer methodTrace({{.Ctx}}, {{.ReceiverType | quote}}, {{.FuncBaseName | quote}}){{else}}defer funcTrace({{.Ctx}}, {{.FuncBaseName | quote}}){{end}}' \
  ./...
```

## Dry Run Mode

Preview changes without modifying files:

```bash
ctxweaver \
  -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
  -dry-run \
  ./...
```

## Selective Processing

### Specific Packages

```bash
# Only service layer
ctxweaver -template='...' ./pkg/service/...

# Multiple specific packages
ctxweaver -template='...' ./pkg/api ./pkg/service ./pkg/repository
```

### File Patterns

```bash
# Only handler files
ctxweaver -template='...' -files='pkg/api/**/handler*.go'

# Exclude test files (default behavior, but explicit)
ctxweaver -template='...' -test=false ./...

# Include test files
ctxweaver -template='...' -test=true ./...
```

## CI Integration

### GitHub Actions

```yaml
- name: Run ctxweaver
  run: |
    go run github.com/mpyw/ctxweaver/cmd/ctxweaver@v0.1.0 \
      -template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
      -import='github.com/example/myapp/internal/apm' \
      ./...

- name: Check for changes
  run: |
    git diff --exit-code || (echo "ctxweaver made changes; please commit them" && exit 1)
```

### Makefile

```makefile
.PHONY: weave
weave:
	go run github.com/mpyw/ctxweaver/cmd/ctxweaver@v0.1.0 \
		-template='defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()' \
		-import='github.com/example/myapp/internal/apm' \
		./...
	gci write .  # Optional: format imports after
```
