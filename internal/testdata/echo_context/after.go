package api

import (
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func GetUser(c echo.Context) error {
	defer newrelic.FromContext(c.Request().Context()).StartSegment("api.GetUser").End() //ctxweaver:generated

	id := c.Param("id")
	_ = id
	return nil
}

func CreateUser(c echo.Context) error {
	defer newrelic.FromContext(c.Request().Context()).StartSegment("api.CreateUser").End() //ctxweaver:generated

	// create user
	return nil
}
