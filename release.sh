#!/bin/bash
set -e

# Release script for Omnius Server
# Usage: ./release.sh [version]  - specify version like 2.2.0
#        ./release.sh            - auto-increment patch (e.g. 2.1.0 → 2.1.1)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Validate version format (x.x.x)
validate_version() {
    [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

# Get current version from latest git tag
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
echo "Current version: $CURRENT_VERSION"

# Auto-increment patch
increment_version() {
    IFS='.' read -r major minor patch <<< "$CURRENT_VERSION"
    patch=$((patch + 1))
    echo "$major.$minor.$patch"
}

# Set new version
if [ -n "$1" ]; then
    NEW_VERSION="${1#v}"  # Strip leading v if present
    if ! validate_version "$NEW_VERSION"; then
        echo "Error: Invalid version format '$1'. Must be x.x.x (e.g., 2.2.0)"
        NEXT=$(increment_version)
        read -p "Release $NEXT instead? [y/N] " -n 1 -r
        echo
        [[ $REPLY =~ ^[Yy]$ ]] && NEW_VERSION="$NEXT" || exit 1
    fi
else
    NEW_VERSION=$(increment_version)
fi

echo "New version: $NEW_VERSION"
echo ""

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
    echo "Warning: You have uncommitted changes."
    read -p "Commit them before releasing? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git add -A
        git commit -m "Pre-release changes for v$NEW_VERSION"
    else
        echo "Aborted. Commit or stash your changes first."
        exit 1
    fi
fi

# Build frontend
echo "Building frontend..."
cd frontend && npm run build && cd ..

# Check if frontend build changed anything
if [ -n "$(git status --porcelain static/)" ]; then
    git add static/admin/
    git commit -m "Build frontend for v$NEW_VERSION"
fi

# Push
git push origin main

# Create GitHub release (triggers release.yml workflow to build binaries)
echo "Creating GitHub release v$NEW_VERSION..."
read -p "Enter release notes (or press Enter for default): " NOTES
if [ -z "$NOTES" ]; then
    NOTES="Release v$NEW_VERSION"
fi

gh release create "v$NEW_VERSION" \
    --title "v$NEW_VERSION" \
    --notes "$NOTES"

echo ""
echo "Released v$NEW_VERSION"
echo "GitHub Actions is now building binaries and Docker image."
echo "Check: https://github.com/OmniusRepos/omnius-server/actions"
