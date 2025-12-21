# GitHub Actions Workflows

## Build and Push Container Images

The `build.yml` workflow automatically builds and pushes container images to GitHub Container Registry (ghcr.io).

### Triggers

**Main Branch Commits:**
- Trigger: Push to `main` branch
- Images tagged as: `dev`
- Example:
  - `ghcr.io/cartermckinnon/watchclub:dev`
  - `ghcr.io/cartermckinnon/watchclub/ui:dev`

**Version Tags:**
- Trigger: Push a tag starting with `v` (e.g., `v1.0.0`)
- Images tagged as: version tag + `latest`
- Example for tag `v1.0.0`:
  - `ghcr.io/cartermckinnon/watchclub:v1.0.0`
  - `ghcr.io/cartermckinnon/watchclub:latest`
  - `ghcr.io/cartermckinnon/watchclub/ui:v1.0.0`
  - `ghcr.io/cartermckinnon/watchclub/ui:latest`

### Creating a Release

To create a new release and build container images:

```bash
# Create and push a version tag
git tag v1.0.0
git push origin v1.0.0
```

The workflow will:
1. Build backend and frontend images
2. Push with version tag (e.g., `v1.0.0`)
3. Also tag and push as `latest`

### Workflow Steps

1. **Checkout code** - Clone the repository
2. **Set up Docker Buildx** - Configure Docker for multi-platform builds
3. **Login to GHCR** - Authenticate with GitHub Container Registry using `GITHUB_TOKEN`
4. **Install Earthly** - Set up Earthly build tool
5. **Determine version** - Set version tag based on trigger (tag name or `dev`)
6. **Build and push backend** - Run `earthly --push +watchclub`
7. **Build and push frontend** - Run `earthly --push +ui`
8. **Tag as latest** - For version tags only, also tag and push as `latest`
9. **Summary** - Display published image names

### Required Permissions

The workflow requires:
- `contents: read` - To checkout code
- `packages: write` - To push to GitHub Container Registry

These are automatically provided by `GITHUB_TOKEN`.

### Local Testing

To test the build locally before pushing:

```bash
# Build backend (no push)
earthly +watchclub --VERSION=local-test

# Build frontend (no push)
earthly +ui --VERSION=local-test

# Build and push (requires authentication)
earthly --push +watchclub --VERSION=v1.0.0
earthly --push +ui --VERSION=v1.0.0
```

### Troubleshooting

**Images not appearing in GHCR:**
- Ensure the repository has package write permissions enabled
- Check that `GITHUB_TOKEN` has the necessary scopes

**Build failures:**
- Check the Actions tab for detailed logs
- Verify Earthfile syntax is correct
- Ensure all dependencies are available in the build environment

**Tag not triggering workflow:**
- Tags must start with `v` (e.g., `v1.0.0`, `v2.1.3-beta`)
- Ensure the tag was pushed to GitHub: `git push origin <tag>`
