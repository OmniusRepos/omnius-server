# Releasing Omnius Server

## How to Release

1. **Commit your changes** and push to `main`

2. **Create a GitHub release:**
   ```bash
   gh release create v<VERSION> --title "v<VERSION>" --notes "Release notes here"
   ```
   Example:
   ```bash
   gh release create v2.2.0 --title "v2.2.0" --notes "Add new feature X, fix bug Y"
   ```

3. **That's it.** Two workflows trigger automatically:

   - **`release.yml`** — Builds cross-platform binaries (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`) and attaches them to the release as `omnius-{os}-{arch}`
   - **`docker-publish.yml`** — Builds and pushes the Docker image to `ghcr.io/omniusrepos/omnius-server` with the version tag

## Version Injection

The version is baked into the binary at build time via:
```
-ldflags "-X main.Version=<VERSION>"
```

- Release binaries get the tag version (e.g. `2.1.0` from tag `v2.1.0`)
- Docker images get the tag version via the `VERSION` build arg
- Local dev builds default to `dev`

To build locally with a version:
```bash
go build -ldflags "-X main.Version=2.1.0" -o omnius-server .
```

## Auto-Update Flow

Running Omnius instances can update themselves from the Settings > Update tab:

1. Server calls `https://api.github.com/repos/OmniusRepos/omnius-server/releases/latest`
2. Compares `main.Version` against the latest release tag
3. If an update is available, downloads `omnius-{os}-{arch}` from the release assets
4. Replaces the running binary and restarts

**Docker users** see a message to redeploy the container instead (auto-update replaces the binary, which doesn't persist in Docker).

## Checklist

- [ ] All changes committed and pushed to `main`
- [ ] Version number follows semver (`MAJOR.MINOR.PATCH`)
- [ ] Release notes describe what changed
- [ ] After release, check [Actions](https://github.com/OmniusRepos/omnius-server/actions) to confirm builds succeed
- [ ] Verify binaries are attached to the release
