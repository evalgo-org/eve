// Package http provides documentation generation utilities for EVE services.
package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// ServiceDocConfig contains configuration for service documentation
type ServiceDocConfig struct {
	ServiceID    string
	ServiceName  string
	Description  string
	Version      string
	Port         int
	Capabilities []string
	Endpoints    []EndpointDoc
}

// EndpointDoc describes an API endpoint
type EndpointDoc struct {
	Method      string
	Path        string
	Description string
	RequestBody string
	Response    string
}

// DocumentationHandler creates a documentation page for a service
func DocumentationHandler(config ServiceDocConfig) echo.HandlerFunc {
	return func(c echo.Context) error {
		html := generateServiceDocHTML(config)
		return c.HTML(http.StatusOK, html)
	}
}

func generateServiceDocHTML(config ServiceDocConfig) string {
	// Build capabilities list
	capabilitiesHTML := ""
	for _, cap := range config.Capabilities {
		capabilitiesHTML += fmt.Sprintf(`<span class="badge">%s</span> `, cap)
	}

	// Build endpoints table
	endpointsHTML := ""
	if len(config.Endpoints) > 0 {
		endpointsHTML = `
		<h2>API Endpoints</h2>
		<table>
			<thead>
				<tr>
					<th>Method</th>
					<th>Path</th>
					<th>Description</th>
				</tr>
			</thead>
			<tbody>`

		for _, endpoint := range config.Endpoints {
			methodClass := strings.ToLower(endpoint.Method)
			endpointsHTML += fmt.Sprintf(`
				<tr>
					<td><span class="method method-%s">%s</span></td>
					<td><code>%s</code></td>
					<td>%s</td>
				</tr>`,
				methodClass, endpoint.Method, endpoint.Path, endpoint.Description)
		}

		endpointsHTML += `
			</tbody>
		</table>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s - API Documentation</title>
	<style>
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
			line-height: 1.6;
			color: #333;
			background: #f5f5f5;
		}
		.header {
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			color: white;
			padding: 2rem;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		.container {
			max-width: 1200px;
			margin: 0 auto;
			padding: 2rem;
		}
		.header .container {
			padding: 0 2rem;
		}
		h1 {
			font-size: 2rem;
			margin-bottom: 0.5rem;
		}
		h2 {
			color: #667eea;
			margin: 2rem 0 1rem 0;
			padding-bottom: 0.5rem;
			border-bottom: 2px solid #667eea;
		}
		.subtitle {
			opacity: 0.9;
			font-size: 1.1rem;
		}
		.info-grid {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 1rem;
			margin: 2rem 0;
		}
		.info-card {
			background: white;
			padding: 1.5rem;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
		}
		.info-label {
			font-size: 0.875rem;
			color: #666;
			text-transform: uppercase;
			letter-spacing: 0.5px;
			margin-bottom: 0.5rem;
		}
		.info-value {
			font-size: 1.25rem;
			font-weight: 600;
			color: #333;
		}
		.badge {
			display: inline-block;
			background: #667eea;
			color: white;
			padding: 0.25rem 0.75rem;
			border-radius: 12px;
			font-size: 0.875rem;
			margin: 0.25rem;
		}
		table {
			width: 100%%;
			background: white;
			border-radius: 8px;
			overflow: hidden;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			margin: 1rem 0;
		}
		thead {
			background: #667eea;
			color: white;
		}
		th, td {
			padding: 1rem;
			text-align: left;
		}
		tbody tr:nth-child(even) {
			background: #f8f9fa;
		}
		tbody tr:hover {
			background: #e9ecef;
		}
		code {
			background: #f4f4f4;
			padding: 0.2rem 0.4rem;
			border-radius: 3px;
			font-family: 'Monaco', 'Courier New', monospace;
			font-size: 0.9rem;
		}
		.method {
			display: inline-block;
			padding: 0.25rem 0.5rem;
			border-radius: 4px;
			font-weight: bold;
			font-size: 0.75rem;
		}
		.method-get { background: #28a745; color: white; }
		.method-post { background: #007bff; color: white; }
		.method-put { background: #ffc107; color: black; }
		.method-delete { background: #dc3545; color: white; }
		.method-patch { background: #17a2b8; color: white; }
		.footer {
			text-align: center;
			padding: 2rem;
			color: #666;
			font-size: 0.875rem;
		}
		.content {
			background: white;
			padding: 2rem;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			margin-bottom: 2rem;
		}
	</style>
</head>
<body>
	<div class="header">
		<div class="container">
			<h1>%s</h1>
			<p class="subtitle">%s</p>
		</div>
	</div>
	<div class="container">
		<div class="info-grid">
			<div class="info-card">
				<div class="info-label">Service ID</div>
				<div class="info-value">%s</div>
			</div>
			<div class="info-card">
				<div class="info-label">Version</div>
				<div class="info-value">%s</div>
			</div>
			<div class="info-card">
				<div class="info-label">Port</div>
				<div class="info-value">%d</div>
			</div>
		</div>

		<div class="content">
			<h2>Capabilities</h2>
			<div>
				%s
			</div>
		</div>

		<div class="content">
			%s
		</div>
	</div>
	<div class="footer">
		<p>Part of the EVE Ecosystem | <a href="http://localhost:8096">Registry Service</a></p>
	</div>
</body>
</html>`,
		config.ServiceName,
		config.ServiceName,
		config.Description,
		config.ServiceID,
		config.Version,
		config.Port,
		capabilitiesHTML,
		endpointsHTML,
	)

	return html
}
