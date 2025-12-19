# Design Document

## Goals

1. **Weave statements into context-aware functions** - Automatically insert/update code at function entry points
2. **User-defined templates** - Full flexibility in what gets inserted
3. **Performance at scale** - Handle large codebases efficiently
4. **Preserve code integrity** - Never lose comments or formatting

## Non-Goals

- Import ordering/formatting (use `goimports`/`gci` for that)
- Linting or error detection (see `goroutinectx` for that)
- Runtime code generation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        ctxweaver                            │
├─────────────────────────────────────────────────────────────┤
│  CLI Layer                                                  │
│  - Flag parsing                                             │
│  - Template loading                                         │
│  - Configuration                                            │
├─────────────────────────────────────────────────────────────┤
│  Core Layer                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Loader    │  │  Processor  │  │      Writer         │  │
│  │             │  │             │  │                     │  │
│  │ packages.   │→ │ DST-based   │→ │ Format & write      │  │
│  │ Load()      │  │ transform   │  │ modified files      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  Support Layer                                              │
│  - Context carrier resolution                               │
│  - Template rendering                                       │
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

**Trade-off**:
- DST is slower than raw AST
- Additional dependency
- Less tooling ecosystem support

### 2. packages.Load Strategy

**Decision**: Load all target packages in a single `packages.Load()` call.

**Rationale**:
- `packages.Load()` is expensive (~100ms+ startup overhead)
- Calling it per-file or per-package is prohibitively slow
- Single load with `NeedTypes | NeedSyntax | NeedImports` provides everything needed
- Memory usage is acceptable for typical codebases

**Implementation**:
```go
cfg := &packages.Config{
    Mode: packages.NeedName |
          packages.NeedFiles |
          packages.NeedSyntax |
          packages.NeedTypes |
          packages.NeedImports |
          packages.NeedDeps,
    // ...
}
pkgs, err := packages.Load(cfg, patterns...)
```

**Alternative considered**:
- Parse AST without type info → faster but can't accurately resolve imports
- Lazy loading per package → simpler but O(n) Load calls

### 3. Template-Based Generation

**Decision**: Use Go's `text/template` for statement generation.

**Rationale**:
- Familiar to Go developers
- Powerful enough for complex expressions
- Easy to add custom functions
- Can be loaded from files for complex templates

**Example**:
```go
tmpl := `defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()`
```

**Variables provided**:
- Derived from function signature and package info
- All values are strings (safe for template interpolation)
- Boolean flags for conditional logic

### 4. Statement Detection

**Decision**: Pattern-match existing statements structurally, not by string comparison.

**Rationale**:
- Exact string matching is fragile (whitespace, formatting)
- Need to detect "same intent" even with different function names
- Allows updating statements when functions are renamed

**Implementation approach**:
1. Parse the template to understand its structure
2. Walk existing function body looking for matching structure
3. Compare key parts (function calls, identifiers) while ignoring dynamic parts

### 5. Context Carrier System

**Decision**: Allow user-defined types as context carriers with custom accessors.

**Format**: `package/path.Type=.accessor`

**Rationale**:
- Many frameworks wrap `context.Context` (Echo, CLI, etc.)
- Need extensibility for custom wrapper types
- Accessor syntax is intuitive and matches Go code

**Examples**:
```
context.Context              → ctx           (accessor: "")
echo.Context                 → ctx.Request().Context()  (accessor: ".Request().Context()")
*cli.Context                 → ctx.Context   (accessor: ".Context")
```

### 6. No Built-in Import Ordering

**Decision**: Do not integrate `gci` or implement import ordering.

**Rationale**:
- Import ordering is a separate concern
- Many projects have their own preferences
- `goimports`/`gci` do this well
- Reduces complexity and dependencies
- Users can chain tools: `ctxweaver ./... && gci write .`

**What we DO**:
- Add required imports cleanly
- Preserve existing import structure
- Never remove or reorder imports

## File Processing Pipeline

```
1. Parse patterns → package paths or file globs
2. Load packages (single packages.Load call)
3. For each file:
   a. Check skip directive
   b. Convert AST → DST
   c. For each function:
      - Check skip directive
      - Match context parameter
      - Render template with variables
      - Detect existing statement
      - Insert/Update/Skip as needed
   d. If modified:
      - Add required imports
      - Convert DST → AST
      - Format and write

4. Report summary
```

## Error Handling

- **Parse errors**: Report and skip file, continue with others
- **Template errors**: Fail fast (configuration error)
- **Write errors**: Report and continue (best effort)
- **Type resolution failures**: Gracefully degrade to name-based matching

## Future Considerations

### Potential Features

1. **`--check` mode**: Exit non-zero if changes would be made (for CI)
2. **Position control**: Insert at end of function instead of beginning
3. **Multiple templates**: Different templates for different function patterns
4. **Exclude patterns**: Skip specific packages or functions by pattern

### Not Planned

1. **IDE integration**: Too complex, use CLI + file watcher instead
2. **Incremental caching**: Complexity not justified for typical use
3. **AST-only mode**: Defeats the purpose of preserving comments
