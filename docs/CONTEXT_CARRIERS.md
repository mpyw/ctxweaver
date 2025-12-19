# Context Carriers

## Overview

A "context carrier" is any type that carries or provides access to a `context.Context`. While `context.Context` itself is the most common, many frameworks use wrapper types that contain context.

ctxweaver needs to know how to extract the actual `context.Context` from these wrapper types so it can generate the correct template variables.

## Built-in Carriers

| Type | Accessor | Notes |
|------|----------|-------|
| `context.Context` | (none) | Standard library context |

## Common Framework Carriers

These are commonly used but must be explicitly enabled via `-context-carrier`:

| Type | Accessor | Flag Value |
|------|----------|------------|
| `echo.Context` | `.Request().Context()` | `github.com/labstack/echo/v4.Context=.Request().Context()` |
| `*cli.Context` | `.Context` | `github.com/urfave/cli/v2.Context=.Context` |
| `*cobra.Command` | `.Context()` | `github.com/spf13/cobra.Command=.Context()` |
| `*gin.Context` | `.Request.Context()` | `github.com/gin-gonic/gin.Context=.Request.Context()` |
| `*fiber.Ctx` | `.Context()` | `github.com/gofiber/fiber/v2.Ctx=.Context()` |

## Defining Custom Carriers

### Flag Format

```
-context-carrier='<package-path>.<TypeName>=<accessor>'
```

- **package-path**: Full import path (e.g., `github.com/example/mylib`)
- **TypeName**: Type name without pointer (e.g., `MyContext`, not `*MyContext`)
- **accessor**: Go expression suffix to get `context.Context`

### Accessor Expressions

The accessor is appended to the parameter variable name to form the `{{.Ctx}}` template variable:

| If parameter is | And accessor is | `{{.Ctx}}` becomes |
|-----------------|-----------------|-------------------|
| `ctx` | (empty) | `ctx` |
| `c` | `.Context` | `c.Context` |
| `cmd` | `.Context()` | `cmd.Context()` |
| `e` | `.Request().Context()` | `e.Request().Context()` |

### Pointer Handling

The type in the flag should always be without the pointer. ctxweaver automatically matches both pointer and non-pointer versions:

```bash
# This matches both `cli.Context` and `*cli.Context`
-context-carrier='github.com/urfave/cli/v2.Context=.Context'
```

## Examples

### Single Carrier

```bash
ctxweaver \
  -template='defer trace({{.Ctx}}, {{.FuncName | quote}})' \
  -context-carrier='github.com/labstack/echo/v4.Context=.Request().Context()' \
  ./...
```

### Multiple Carriers

```bash
ctxweaver \
  -template='defer trace({{.Ctx}}, {{.FuncName | quote}})' \
  -context-carrier='github.com/labstack/echo/v4.Context=.Request().Context()' \
  -context-carrier='github.com/urfave/cli/v2.Context=.Context' \
  -context-carrier='github.com/spf13/cobra.Command=.Context()' \
  ./...
```

### Custom Wrapper Types

If your project has a custom context wrapper:

```go
// pkg/web/context.go
package web

type Context struct {
    ctx context.Context
    // other fields...
}

func (c *Context) Context() context.Context {
    return c.ctx
}
```

```bash
ctxweaver \
  -template='...' \
  -context-carrier='github.com/yourproject/pkg/web.Context=.Context()' \
  ./...
```

## Implementation Details

### Type Resolution

ctxweaver uses `go/types` to accurately resolve import paths:

1. Parse the function parameter type
2. If it's a selector expression (`pkg.Type`), look up the import path
3. Match against registered carriers
4. Generate accessor expression

### Import Alias Handling

ctxweaver correctly handles import aliases:

```go
import (
    ec "github.com/labstack/echo/v4"
)

func handler(c ec.Context) error {  // Correctly identified as echo.Context
    // ...
}
```

### Interface Types

Carriers are matched by the declared type, not the underlying implementation:

```go
// This works because the parameter type is echo.Context
func handler(c echo.Context) error { ... }

// This does NOT work - parameter type is just an interface
func handler(c interface{ Request() *http.Request }) error { ... }
```

## Troubleshooting

### Carrier Not Matched

If ctxweaver isn't detecting your context carrier:

1. Check the import path is exact (including version suffixes like `/v4`)
2. Verify the type name matches (without pointer)
3. Use `-verbose` to see what types are being detected

### Wrong Accessor

If `{{.Ctx}}` generates incorrect code:

1. The accessor is appended as-is to the variable name
2. Include the `.` prefix for field/method access
3. Include `()` suffix for method calls

### Version Mismatches

For versioned modules, be specific:

```bash
# Wrong - won't match github.com/labstack/echo/v4
-context-carrier='github.com/labstack/echo.Context=...'

# Correct
-context-carrier='github.com/labstack/echo/v4.Context=...'
```
