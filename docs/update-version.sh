#!/bin/bash
# Automatically update version number in EVE documentation from git tags
# Usage: ./update-version.sh

set -e

cd "$(dirname "$0")/.."

# Get the latest git tag
LATEST_TAG=$(git tag --sort=-v:refname | head -1)

if [ -z "$LATEST_TAG" ]; then
    echo "Error: No git tags found"
    exit 1
fi

echo "Latest EVE version: $LATEST_TAG"

# Update version in all HTML files
echo "Updating version in HTML files..."
for file in docs/*.html; do
    if [ -f "$file" ]; then
        # Use sed to replace version in the span tag
        sed -i "s|<span class=\"version\">v[0-9]\+\.[0-9]\+\.[0-9]\+</span>|<span class=\"version\">$LATEST_TAG</span>|g" "$file"
        echo "  ✓ Updated $(basename "$file")"
    fi
done

echo ""
echo "Version updated to $LATEST_TAG in all documentation files"
echo ""
echo "To deploy changes:"
echo "  git add docs/*.html"
echo "  git commit -m 'docs: Update version to $LATEST_TAG'"
echo "  git push"
