module test

go 1.21

require github.com/labstack/echo/v4 v4.0.0

require github.com/newrelic/go-agent/v3/newrelic v0.0.0

replace github.com/labstack/echo/v4 => ../_stubs/github.com/labstack/echo/v4

replace github.com/newrelic/go-agent/v3/newrelic => ../_stubs/github.com/newrelic/go-agent/v3/newrelic
