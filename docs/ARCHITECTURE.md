# ctxweaver Architecture

This document describes the architecture and design decisions of ctxweaver.

## Goals

1. **Weave statements into context-aware functions** - Automatically insert/update code at function entry points
2. **User-defined templates** - Full flexibility in what gets inserted
3. **Performance at scale** - Handle large codebases efficiently
4. **Preserve code integrity** - Never lose comments or formatting
5. **Configuration-driven** - YAML config for complex settings

## Non-Goals

- Import ordering/formatting (use `goimports`/`gci` for that)
- Linting or error detection (see `goroutinectx` for that)
- Runtime code generation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        ctxweaver                            │
├─────────────────────────────────────────────────────────────┤
│  CLI Layer (cmd/ctxweaver)                                  │
│  - Flag parsing                                             │
│  - Config loading                                           │
├─────────────────────────────────────────────────────────────┤
│  Config Layer (config/)                                     │
│  - YAML config parsing                                      │
│  - Embedded default carriers                                │
│  - Carrier registry                                         │
├─────────────────────────────────────────────────────────────┤
│  Template Layer (template/)                                 │
│  - Go template parsing                                      │
│  - Variable substitution                                    │
│  - Built-in functions (quote, backtick)                     │
├─────────────────────────────────────────────────────────────┤
│  Processor Layer (processor/)                               │
│  - packages.Load for type info                              │
│  - DST-based transformation                                 │
│  - Statement detection/matching                             │
│  - Import management                                        │
└─────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. DST over AST

**Decision**: Use [dave/dst](https://github.com/dave/dst) (Decorated Syntax Tree) instead of `go/ast`.

**Rationale**:
- AST does not preserve comment positioning relative to nodes
- Code generation with AST often loses or misplaces comments
- DST explicitly tracks decorations (comments, whitespace) per node
- Allows round-trip parsing → modification → printing without information loss

### 2. packages.Load Strategy

**Decision**: Load all target packages in a single `packages.Load()` call.

**Rationale**:
- `packages.Load()` is expensive (~100ms+ startup overhead)
- Calling it per-file or per-package is prohibitively slow
- Single load with type info provides accurate import resolution

**Implementation**:
```go
cfg := &packages.Config{
    Mode: packages.NeedName |
          packages.NeedFiles |
          packages.NeedSyntax |
          packages.NeedTypes |
          packages.NeedTypesInfo |
          packages.NeedImports,
}
pkgs, err := packages.Load(cfg, patterns...)
```

### 3. YAML Configuration

**Decision**: Use YAML config file instead of CLI flags for complex settings.

**Rationale**:
- Templates can be multi-line
- Multiple imports are common
- Custom carriers need structured data
- Easier to version control and share

**Structure**:
```yaml
template: |
  defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()
# Or load from file:
# template:
#   file: ./templates/trace.tmpl

imports:
  - github.com/example/myapp/internal/apm

packages:
  patterns:
    - ./...
  regexps:
    only: []      # Only process packages matching these (empty = all)
    omit:         # Skip packages matching these
      - /mock/
      - _test$

functions:
  types:          # Enum: "function" | "method" (default: both)
    - function
    - method
  scopes:         # Enum: "exported" | "unexported" (default: both)
    - exported
    - unexported
  regexps:
    only: []      # Regex patterns - only process matching (empty = all)
    omit: []      # Regex patterns - skip matching

# Simple form (array)
carriers:
  - package: github.com/custom/ctx
    type: Context
    accessor: .GetContext()

# Extended form (disable defaults)
# carriers:
#   custom:
#     - package: github.com/custom/ctx
#       type: Context
#   default: false

test: false
```

### 4. Embedded Default Carriers

**Decision**: Embed default carriers via `//go:embed` in YAML format.

**Rationale**:
- No runtime file dependencies
- Easy to maintain and update
- Clear separation from user config
- Users can override or extend

### 5. Carriers Union Type

**Decision**: Support both simple array and extended object forms for carriers.

**Rationale**:
- Simple form covers most use cases (add custom carriers while keeping defaults)
- Extended form allows disabling default carriers for full control
- Consistent with template union type pattern

**Implementation**:
```yaml
# Simple array (default carriers enabled)
carriers:
  - package: github.com/custom/ctx
    type: Context

# Extended object (control default carriers)
carriers:
  custom:
    - package: github.com/custom/ctx
      type: Context
  default: false  # Disable built-in carriers
```

### 6. Template Union Type

**Decision**: Support both inline strings and file references for templates.

**Rationale**:
- Simple templates work well inline in YAML
- Complex templates (multi-line, conditional logic) are easier to maintain in separate files
- File templates can be shared across projects
- YAML custom unmarshaling handles the union type transparently

**Implementation**:
```yaml
# Inline string
template: |
  defer trace({{.Ctx}})

# File reference
template:
  file: ./templates/trace.tmpl
```

### 7. First Parameter Only

**Decision**: Only check the first function parameter for context carriers.

**Rationale**:
- Go convention: context should be the first parameter
- Simplifies implementation
- Avoids ambiguity with multiple context-like parameters
- Reduces false positives

### 8. Statement Pattern Detection

**Decision**: Detect existing statements by structural pattern matching.

**Rationale**:
- Exact string matching is fragile
- Need to detect "same intent" even with different function names
- Allows updating when functions are renamed

**Current implementation**: Specific to `defer XXX.StartSegment(ctx, "name").End()` pattern.
Future work could generalize this.

### 9. No Built-in Import Ordering

**Decision**: Do not integrate `gci` or implement import ordering.

**Rationale**:
- Import ordering is a separate concern
- Many projects have their own preferences
- `goimports`/`gci` do this well
- Reduces complexity and dependencies

### 10. CLI Override Behavior

**Decision**: CLI arguments override (not merge) config file values.

**Behavior**:
| Source | Config Field | CLI | Behavior |
|--------|--------------|-----|----------|
| Package patterns | `packages.patterns` | positional args | **Override**: CLI args replace config entirely |
| Test mode | `test` | `-test` | **Override**: Only when flag is explicitly passed |

**Rationale**:
- Simple mental model: CLI takes precedence
- Explicit flag detection via `flag.Visit()` for boolean overrides
- No complex merge logic to reason about

## File Processing Pipeline

```
1. Load config (YAML)
2. Set defaults (types, scopes)
3. Parse template (inline or from file)
4. Create carrier registry (defaults + custom)
5. Compile regex patterns (packages.regexps, functions.regexps)
6. Run pre-hooks (if not --no-hooks)
7. packages.Load(patterns)
8. For each package:
   a. Check packages.regexps.only (skip if not matching)
   b. Check packages.regexps.omit (skip if matching)
   c. For each file:
      - Check file-level skip directive
      - Parse with fresh fset
      - Convert AST → DST
      - For each function:
        * Check function-level skip directive
        * Check functions.types filter (function/method)
        * Check functions.scopes filter (exported/unexported)
        * Check functions.regexps.only filter
        * Check functions.regexps.omit filter
        * Check first parameter for carrier match
        * Render template with variables
        * Detect existing statement
        * Insert/Update/Remove/Skip
      - If modified:
        * Convert DST → AST
        * Add imports via astutil
        * Format and write
9. Run post-hooks (if not --no-hooks)
10. Report results
```

## Template Variables

| Variable | Source | Example |
|----------|--------|---------|
| `Ctx` | carrier.BuildContextExpr(varName) | `ctx`, `c.Request().Context()` |
| `CtxVar` | param.Names[0].Name | `ctx`, `c` |
| `FuncName` | naming logic | `pkg.(*Service).Method` |
| `PackageName` | df.Name.Name | `service` |
| `PackagePath` | pkg.PkgPath | `github.com/example/myapp/pkg/service` |
| `FuncBaseName` | decl.Name.Name | `Method` |
| `ReceiverType` | receiver type analysis | `Service` |
| `ReceiverVar` | recv.Names[0].Name | `s` |
| `IsMethod` | decl.Recv != nil | `true` |
| `IsPointerReceiver` | receiver is *Type | `true` |
| `IsGenericFunc` | decl.Type.TypeParams != nil | `true` |
| `IsGenericReceiver` | receiver has type params | `true` |

## Filtering Mechanisms

### Package Filtering

Package filtering uses regex patterns matched against full import paths:

```
packages:
  regexps:
    only: [/handler/, /service/]  # Whitelist (empty = all)
    omit: [/mock/, _test$]        # Blacklist
```

**Filtering Logic**:
1. If `only` is non-empty: package must match at least one pattern
2. If `omit` matches: package is skipped regardless of `only`
3. Invalid regex patterns are logged as warnings and skipped

### Function Filtering

Function filtering combines type, scope, and regex criteria:

```
functions:
  types: [function, method]       # Enum: "function" | "method"
  scopes: [exported, unexported]  # Enum: "exported" | "unexported"
  regexps:
    only: [^Handle]               # Regex patterns (whitelist)
    omit: [Mock, Helper$]         # Regex patterns (blacklist)
```

**Type Filtering** (enum values):
- `"function"`: Top-level functions without receivers
- `"method"`: Functions with receivers (value or pointer)

**Scope Filtering** (enum values):
- `"exported"`: Functions starting with uppercase (e.g., `GetUser`)
- `"unexported"`: Functions starting with lowercase (e.g., `parseInput`)

**Filtering Order**:
1. Skip directive check
2. Type filter (function/method)
3. Scope filter (exported/unexported)
4. Regex `only` filter
5. Regex `omit` filter
6. Carrier match check

All filters must pass for a function to be processed.

## Error Handling

- **Config errors**: Fail fast (user configuration error)
- **Parse errors**: Report and skip file, continue with others
- **Template errors**: Fail fast (configuration error)
- **Write errors**: Report and continue (best effort)
- **Package load errors**: Report and continue
- **Invalid regex patterns**: Log warning and skip the pattern (continue processing)
- **Pre-hook failures**: Abort processing, no files modified
- **Post-hook failures**: Log error but files already modified

## Future Considerations

### Potential Features

1. **`--check` mode**: Exit non-zero if changes would be made (for CI)
2. **Generic pattern detection**: User-defined patterns for existing statement detection
3. **Multiple templates**: Different templates for different function patterns

### Not Planned

1. **IDE integration**: Use CLI + file watcher instead
2. **AST-only mode**: Defeats the purpose of preserving comments
3. **Import formatting**: Use external tools
