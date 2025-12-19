package api

import "github.com/labstack/echo/v4"

func GetUser(c echo.Context) error {
	id := c.Param("id")
	_ = id
	return nil
}

func CreateUser(c echo.Context) error {
	// create user
	return nil
}
