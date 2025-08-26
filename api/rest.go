package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func APIKeyAuth(validKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-API-Key")
			if key == "" || key != validKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing API key")
			}
			return next(c)
		}
	}
}

func StartWithApiKey(address, apiKey string) {
	e := echo.New()
	e.Use(APIKeyAuth(apiKey))
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!")
	})
	e.Logger.Fatal(e.Start(address))
}
