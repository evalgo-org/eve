# EVE Documentation Site

This directory contains the static documentation site for eve.evalgo.org.

## Structure

```
docs/
├── index.html              # Landing page
├── services.html           # All 11 EVE services
├── architecture.html       # System architecture
├── getting-started.html    # Quick start guide
├── status.html            # Live status page
├── assets/
│   └── css/
│       ├── main.css       # Main stylesheet
│       └── status.css     # Status page styles
├── update-version.sh      # Version update script
└── README.md             # This file
```

## Version Management

The documentation displays the current EVE version in the navigation bar. The version is automatically synced with git tags.

### Automatic Version Updates

**Method 1: Manual Script**
```bash
# Update version to match latest git tag
./docs/update-version.sh
```

**Method 2: Pre-commit Hook**
The version is automatically updated whenever you create a new git tag:
```bash
git tag v0.0.32
# Version automatically updates in all HTML files
```

**Method 3: GitHub Actions (Recommended)**
On tag push, GitHub Actions automatically:
1. Updates version in all HTML files
2. Commits the changes
3. Deploys to eve.evalgo.org

### How It Works

1. Script reads the latest git tag: `git tag --sort=-v:refname | head -1`
2. Updates all `<span class="version">v0.0.X</span>` in HTML files
3. You commit and push the changes

## Deployment

The documentation is served from the `docs/` directory on the main branch.

**GitHub Pages Configuration:**
- Source: `main` branch, `/docs` folder
- Custom domain: `eve.evalgo.org`
- HTTPS: Enabled

**Manual Deployment:**
```bash
# 1. Update version
./docs/update-version.sh

# 2. Commit changes
git add docs/*.html
git commit -m "docs: Update version to $(git describe --tags)"

# 3. Push
git push
```

**Automated Deployment:**
Just push a tag and GitHub Actions handles everything:
```bash
git tag v0.0.32
git push origin v0.0.32
```

## Updating Content

### Adding a New Service

1. Edit `services.html`
2. Add service card to the grid
3. Follow the existing format

### Updating Styles

All EVE services and documentation use the corporate identity:
- Colors: Neutral grays, traffic light status colors
- Typography: System fonts, clear hierarchy
- Components: Cards, badges, buttons from `main.css`

Refer to `/docs/DESIGN_SYSTEM.md` for complete brand guidelines.

### Status Page Integration

The status page (`status.html`) connects to the statuspageservice API:
- WebSocket: Real-time updates every 5 seconds
- API: `http://localhost:8110/v1/api/system/status`
- Fallback: Shows cached data if service is down

## Local Development

Serve the docs locally:

```bash
# Simple HTTP server (Python)
python3 -m http.server 8000 --directory docs

# Or with Node.js
npx http-server docs -p 8000
```

Visit: http://localhost:8000

## Go Vanity Imports

The index.html includes meta tags for Go vanity imports:
```html
<meta name="go-import" content="eve.evalgo.org git https://github.com/evalgo-org/eve.git">
```

This allows Go developers to import EVE packages as:
```go
import "eve.evalgo.org/tracing"
import "eve.evalgo.org/statemanager"
import "eve.evalgo.org/web"
```

## Maintenance

- **Version updates:** Run `./docs/update-version.sh` after tagging
- **Broken links:** Check all internal links quarterly
- **Status API:** Verify statuspageservice connection
- **Screenshots:** Update if UI changes significantly

## Architecture

The documentation site is intentionally simple:
- ✅ Static HTML (no build step)
- ✅ Zero dependencies (no npm/webpack)
- ✅ Fast loading (< 100KB total)
- ✅ Works offline (except status page)
- ✅ Accessible (semantic HTML)

This aligns with EVE's philosophy: simplicity, reliability, maintainability.
