package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())

	e.GET("/", getBadge)

	e.Start(":8081")
}

func getBadge(c echo.Context) error {
	r, err := http.Get(fmt.Sprintf("https://badgen.net/badge/views-counter/%d/green?icon=github", count(c.QueryParam("username"))))
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	c.Response().Header().Set("content-type", r.Header.Get("content-type"))
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.String(http.StatusOK, string(b))
}

func count(username string) int {
	return 234
}
