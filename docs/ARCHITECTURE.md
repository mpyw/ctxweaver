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
imports:
  - github.com/example/myapp/internal/apm
carriers:
  - package: github.com/custom/ctx
    type: Context
    accessor: .GetContext()
patterns:
  - ./...
test: false
```

### 4. Embedded Default Carriers

**Decision**: Embed default carriers via `//go:embed` in YAML format.

**Rationale**:
- No runtime file dependencies
- Easy to maintain and update
- Clear separation from user config
- Users can override or extend

### 5. First Parameter Only

**Decision**: Only check the first function parameter for context carriers.

**Rationale**:
- Go convention: context should be the first parameter
- Simplifies implementation
- Avoids ambiguity with multiple context-like parameters
- Reduces false positives

### 6. Statement Pattern Detection

**Decision**: Detect existing statements by structural pattern matching.

**Rationale**:
- Exact string matching is fragile
- Need to detect "same intent" even with different function names
- Allows updating when functions are renamed

**Current implementation**: Specific to `defer XXX.StartSegment(ctx, "name").End()` pattern.
Future work could generalize this.

### 7. No Built-in Import Ordering

**Decision**: Do not integrate `gci` or implement import ordering.

**Rationale**:
- Import ordering is a separate concern
- Many projects have their own preferences
- `goimports`/`gci` do this well
- Reduces complexity and dependencies

## File Processing Pipeline

```
1. Load config (YAML)
2. Parse template
3. Create carrier registry (defaults + custom)
4. packages.Load(patterns)
5. For each package:
   For each file:
     a. Check file-level skip directive
     b. Parse with fresh fset
     c. Convert AST → DST
     d. For each function:
        - Check function-level skip
        - Check first parameter for carrier match
        - Render template with variables
        - Detect existing statement
        - Insert/Update/Skip
     e. If modified:
        - Convert DST → AST
        - Add imports via astutil
        - Format and write
6. Report results
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

## Error Handling

- **Config errors**: Fail fast (user configuration error)
- **Parse errors**: Report and skip file, continue with others
- **Template errors**: Fail fast (configuration error)
- **Write errors**: Report and continue (best effort)
- **Package load errors**: Report and continue

## Future Considerations

### Potential Features

1. **`--check` mode**: Exit non-zero if changes would be made (for CI)
2. **Generic pattern detection**: User-defined patterns for existing statement detection
3. **Multiple templates**: Different templates for different function patterns

### Not Planned

1. **IDE integration**: Use CLI + file watcher instead
2. **AST-only mode**: Defeats the purpose of preserving comments
3. **Import formatting**: Use external tools
