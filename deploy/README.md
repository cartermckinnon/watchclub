# Deploying on Kubernetes

## Prerequisites

### Image Pull Secrets for Private GHCR Images

Since the container images are hosted in a private GitHub Container Registry, you need to create image pull secrets in each namespace.

**1. Create a GitHub Personal Access Token (PAT):**
- Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
- Click "Generate new token (classic)"
- Give it a descriptive name (e.g., "k8s-watchclub-pull")
- Select scope: `read:packages`
- Click "Generate token" and copy it

**2. Create the secret in Kubernetes:**

Create the secret in the appropriate namespace depending on the environment.

```bash
kubectl create secret docker-registry ghcr-credentials \
  --docker-server=ghcr.io \
  --docker-username=<YOUR_GITHUB_USERNAME> \
  --docker-password=<YOUR_PAT> \
  --docker-email=<YOUR_EMAIL>
```

### Email configuration

Create a config map in the appropriate namespace that contains the Resend config details:
```bash
kubectl create secret generic email-config \
    -n watchclub-dev \
    --from-literal=resend-api-key="API_KEY" \
    --from-literal=resend-from="noreply@watchclub-dev.squad.gay" \
    --from-literal=resend-from-name="WatchClub (Dev)"
```

### Deploy NGINX ingress controller

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install ingress-nginx ingress-nginx/ingress-nginx
```

---

## Deploying

### Development
```bash
kubectl apply -k deploy/overlays/dev
```

### Production
```bash
# Update image tags in production/kustomization.yaml first
# Then:
kubectl apply -k deploy/base
```

## Configuration

### Customizing Ingress

Update the host in the ingress patches:

**Development**: `deploy/overlays/development/kustomization.yaml`
```yaml
- host: watchclub-dev.example.com  # Change to your domain
```

**Production**: `deploy/overlays/production/kustomization.yaml`
```yaml
- host: watchclub.example.com      # Change to your domain
```

## Accessing the Application

After deployment, the application will be accessible via the Ingress:

**Development**: `http://watchclub-dev.example.com`
**Production**: `https://watchclub.example.com`

The Ingress routes:
- `/` → Frontend (static files)
- `/watchclub.WatchClubService/*` → Backend (gRPC-Web)

## Namespace Management

The deployments use separate namespaces:
- Development: `watchclub-dev`
- Production: `watchclub`

Create namespaces before deploying:

```bash
kubectl create namespace watchclub-dev
kubectl create namespace watchclub
```

## Updating Deployments

### After CI Builds

GitHub Actions automatically builds and pushes images:
- **Main branch commits** → `dev` tag
- **Version tags** → version tag + `latest`

**Update development deployment:**
```bash
# Development automatically uses :dev tag
# Just restart to pull latest dev image
kubectl rollout restart deployment -n watchclub-dev -l app=watchclub
```

**Update production with new version:**
```bash
# 1. Update image tags in deploy/overlays/production/kustomization.yaml
#    Change v1.0.0 to your new version (e.g., v1.1.0)

# 2. Apply the updated manifests
kubectl apply -k deploy/overlays/prod

# 3. Watch the rollout
kubectl rollout status deployment -n watchclub -l app=watchclub
```

**Quick production update with kubectl:**
```bash
# Update backend image
kubectl set image deployment/prod-backend watchclub=ghcr.io/cartermckinnon/watchclub:v1.1.0 -n watchclub

# Update frontend image
kubectl set image deployment/prod-frontend nginx=ghcr.io/cartermckinnon/watchclub/ui:v1.1.0 -n watchclub
```
