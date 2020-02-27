package handlers

import (
    "net/http"
    "github.com/labstack/echo/v4"
)

// Ping is simple keep-alive/ping handler
func Ping(c echo.Context) error {
    return c.String(http.StatusOK, "OK")

}

