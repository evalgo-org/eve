// Package web provides embedded web assets for EVE services
// This ensures consistent branding across all EVE microservices
package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed assets/*
var assetsFS embed.FS

// RegisterAssets registers the /assets/* route to serve embedded CSS files
// This makes the EVE corporate identity CSS available at /assets/eve-corporate.css
//
// Usage:
//
//	import "eve.evalgo.org/web"
//
//	e := echo.New()
//	web.RegisterAssets(e)
func RegisterAssets(e *echo.Echo) {
	// Get the subdirectory from embedded FS
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic(err)
	}

	// Serve at /assets/*
	e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))))
}

// GetCorporateCSS returns the EVE corporate identity CSS content
// Useful if you want to inline the CSS or serve it differently
func GetCorporateCSS() ([]byte, error) {
	return assetsFS.ReadFile("assets/eve-corporate.css")
}
