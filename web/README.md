# EVE Web Assets Package

This package provides embedded web assets (CSS, etc.) for consistent branding across all EVE microservices.

## Corporate Identity

All EVE services should use the official corporate identity CSS to ensure consistent look and feel.

## Usage

### In Go Services

```go
package main

import (
    "eve.evalgo.org/web"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()

    // Register assets route - serves /assets/eve-corporate.css
    web.RegisterAssets(e)

    e.Start(":8080")
}
```

### In HTML Templates

```html
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="/assets/eve-corporate.css">
</head>
<body>
    <div class="container">
        <div class="card">
            <h1>My EVE Service</h1>
            <p class="lead">Using consistent branding!</p>
        </div>
    </div>
</body>
</html>
```

## Available CSS Classes

### Layout
- `.container` - Max-width container with padding
- `.grid` - Grid layout
- `.grid-2` - 2-column responsive grid
- `.grid-3` - 3-column responsive grid

### Components
- `.card` - White card with shadow
- `.btn`, `.btn-primary`, `.btn-secondary` - Buttons
- `.status-badge` - Status indicators (operational, degraded, outage)
- `.navbar` - Navigation bar
- `.footer` - Footer section

### Typography
- `.lead` - Larger introductory text
- `.text-center` - Center-aligned text
- `.text-muted` - Muted text color
- `.text-secondary` - Secondary text color

### Spacing Utilities
- `.mt-*`, `.mb-*` - Margins (sm, md, lg, xl)
- `.p-*` - Padding (sm, md, lg, xl)

## CSS Variables

All services have access to these CSS variables:

```css
--bg-primary: #f5f7fa
--bg-secondary: #ffffff
--text-primary: #2d3748
--text-secondary: #718096
--accent-primary: #4a5568
--status-operational: #10b981
--status-degraded: #f59e0b
--status-outage: #ef4444
--spacing-sm: 0.5rem
--spacing-md: 1rem
--spacing-lg: 2rem
--spacing-xl: 4rem
```

## Design System

Full design system documentation: [/docs/DESIGN_SYSTEM.md](../docs/DESIGN_SYSTEM.md)

## Examples

### Service with Status Badge

```html
<div class="container mt-xl">
    <div class="card">
        <h2>Service Status</h2>
        <span class="status-badge operational">Operational</span>
    </div>
</div>
```

### Navigation Bar

```html
<nav class="navbar">
    <div class="container">
        <a href="/" class="nav-brand">My Service</a>
        <ul class="nav-links">
            <li><a href="/" class="active">Home</a></li>
            <li><a href="/docs">Docs</a></li>
            <li><a href="/api">API</a></li>
        </ul>
    </div>
</nav>
```

### Grid Layout

```html
<div class="container">
    <div class="grid grid-3">
        <div class="card">Feature 1</div>
        <div class="card">Feature 2</div>
        <div class="card">Feature 3</div>
    </div>
</div>
```

## Updating Existing Services

1. Add import: `import "eve.evalgo.org/web"`
2. Register assets: `web.RegisterAssets(e)`
3. Update HTML to use `/assets/eve-corporate.css`
4. Replace custom CSS with EVE classes
5. Test responsive design on mobile

## File Structure

```
web/
├── assets.go           # Go package with embed directives
├── assets/
│   └── eve-corporate.css  # Corporate identity CSS
└── README.md          # This file
```

## Version

Current version: 1.0 (2025-01-08)

## License

Part of the EVE platform. See main repository for license details.
