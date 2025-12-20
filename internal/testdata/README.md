# Test Design Philosophy

This document describes the testing strategy for ctxweaver.

## Test Categories

### 1. Transformation Tests (testdata-based)

**Location:** `internal/testdata/` + `pkg/processor/processor_insertion_test.go`

Tests that verify code transformation (before → after) should use testdata files:

```
internal/testdata/<case_name>/
├── before.go      # Input Go source
├── after.go       # Expected output
├── go.mod         # Module definition
└── config.yaml    # (optional) Test-specific configuration
```

**When to use testdata:**
- Testing that specific Go code patterns are transformed correctly
- Testing template rendering with various inputs
- Testing insert/update/remove transformations
- Testing edge cases in code generation

**Advantages:**
- Easy to review expected outputs
- Clear separation of test data from test logic
- Can be used as documentation/examples

### 2. Behavior Tests (inline)

**Location:** Various `*_test.go` files

Tests that verify runtime behavior, options, and error handling should remain inline:

**When to use inline tests:**
- Testing processor options (dry-run, verbose, remove mode behavior)
- Testing file filtering (test files, generated files, testdata directories)
- Testing error conditions (syntax errors, invalid config, missing files)
- Testing internal functions that require direct DST node manipulation
- Testing CLI flags and argument parsing

**Examples:**
- `process_test.go`: Processor option behavior
- `main_test.go`: CLI integration and flag handling
- `config_test.go`: Config validation and error cases
- `matcher_test.go`: DST node comparison (requires creating nodes directly)

## Guidelines

### Adding New Tests

1. **For new transformation cases:** Add a new directory under `internal/testdata/`
2. **For new behavior tests:** Add inline tests in the appropriate `*_test.go` file
3. **For new error cases:** Prefer inline tests that can verify error messages

### Test Naming

- Testdata directories: Use descriptive snake_case names (e.g., `skip_directive`, `method_pointer`)
- Test functions: Use descriptive names that explain the behavior being tested

### Configuration

Testdata cases can override the default configuration by providing `config.yaml`:

```yaml
template: |
  ctx, span := otel.Tracer("").Start({{.Ctx}}, {{.FuncName | quote}})
  defer span.End()
imports:
  - "go.opentelemetry.io/otel"
skip_remove: true  # Skip this case in remove mode tests
```

If `config.yaml` is not provided, the default NewRelic template is used.
