# 🚂 Loco — Minimal Container Orchestrator

Loco is a lightweight service that lets you deploy apps with a simple git push. Think of it as a minimal alternative to Heroku or Render, for containerized apps on Kubernetes.

This is mostly for educational purposes to learn more about container orchestration and deployment workflows.

---

## 🎯 MVP Features

- ✅ Git Push Deployment
- ✅ Dockerfile-based Build
- ✅ Pod Deployment to Kubernetes
- ✅ Simple Autoscaling (CPU/Memory)
- ✅ App Configuration via loco.toml
- ✅ Reverse Proxy with TLS & Load Balancing

---

## 📁 User Flow

1. Add a Git remote:

   ```bash
   git remote add loco git@yourhost:user/app.git
   ```

2. Push your code:

   ```bash
   git push loco main
   ```

Its also possible to have the loco CLI handle some of the loco onboarding required for git integration.

3. The Loco platform will:

   - Clone the repo
   - Build the Docker image
   - Apply settings from loco.toml
   - Deploy to K8s with autoscaling
   - Setup HTTPS at https://appname.loco.dev

---

## 🧱 Components

### 1. Build Server

- Accepts Git pushes (via Gitea or raw git-receive-pack)
- Writes repo to disk
- Triggers build pipeline

### 2. Builder

- Uses Buildkit to build the Docker images.
- Pushes image to a registry (local or remote) ~ (ECR for now maybe?)

### 3. Deployment Engine

Parses loco.toml like:

```toml
name = "myapp"
port = 3000
cpu = "200m"
memory = "512Mi"

[replicas]
min = 1
max = 5

[scalers]
cpu_target = 80
memory_target = 50

[logs]
structured = true
retention_period = "7d"

dockerfile_path = "."


```

This will create the following Kubernetes resources:

- Deployment
- HPA (Horizontal Pod Autoscaler)
- Service
- Ingress (via Traefik or NGINX)
  - Traefik is in Go, and surprisingly fast. Can handle load balancing, SSL termination, and routing quite well with minimal config.
  - Can 'upgrade' to envoy later for even more speed, at the cost of decent bit more configuration and work.
- RBAC?
  - might not need for Kubernetes for the MVP, unless we are letting ppl have direct access to underlying. def have application RBAC

### 4. CLI

CLI that can help with loco integrations

- Can generate loco.toml file, or validate it.
- Can generate suggestions to better improve your loco experience by suggesting improvements to loco.toml.
- Can link to a deployed project to get logs and app health. As well as metrics for CPU, memory, and health.
- These likely need to be built onto some sort of API that the CLI is just calling.
- So CLI likely needs to generate temp credentials and store under ~/.loco/credentials.

### Likely MSVCs or just APIs we will need

1. Deployment Engine

- will listen to git pushes via webhook, and build container images.
- Push container images onto a registry

2. User Front APIs

- handles users
  - adding new users to platform
  - removing them and their relevant projects
- handles projects
  - adding new projects, need to make sure names or subdomains atleast are unique. so thats gonna need a DB to validate
  - on new project, need to update the kubernetes configurations to add new ingress and underlying kubernetes deployment by using the commands.
  - stream deployment progress in near real-time
- handles project logs
  - stream logs in near real-time
  - eventually playback but prolly not
  - query logs super fast with filters (use managed like clickhouse?)
- handles project billing
  - users should only be billed for what they use.
  - we will have some default network ingress/egress costs.
  - can be later.

3. Resources handler
   - might need a way of watching our own resources and making sure they are healthy.

#### Things to Think About

- Simple, but super effective abusive stopper.
- Only whitelisted repos should be buildable, by certain users.
- Users should only be able to access logs for their projects. Nothing more, nothing less
- All unhappy paths are clearly reflected back to user, with obvious next steps on how to fix.
- Need to do things the kubernetes way with stuff like RBAC and whatnot.
- whatever we do, it needs to be extendible to the following deployments:
  - UI, cache (redis), database, blob
  - this covers like 90% of all possible deployments.

#### Notes

- dont deploy the image if docker scout or vulnerabilities are detected
- make sure all of the TOML makes sense and is 100% validated against.
- docker layers should be cached by project or something. maybe docker will automatically do this
- Docker building must follow .Dockerignore or .gitignore
- maybe use gitlabs free account for Docker image hosting
  - only hold maybe the last 2 images per project, for rollback purposes
  - switch to Harbor eventually to overcome these limits
- what do i need to do to support podman/oci-based images
- implement lifecycle policy on registry? maybe 6 mths to start
- think more about security. gotta make sure other users cannot pull down other images. Images must be prefixed or something man and have some sort of random hash to avoid collisions.
- we have read/write on registry. should just be write tbh.
- use terraform to create the necessary kubernetes cluster and related resources

#### Kube Commands

```bash

kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.4/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml

# Install RBAC for Traefik:
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.4/docs/content/reference/dynamic-configuration/kubernetes-crd-rbac.yml


# add tls as a secret
kubectl create secret tls loco-tls \
  --cert=deploy-app.com+1.pem \
  --key=deploy-app.com-key+1.pem \
  -n loco-setup


kubectl create configmap envoy-config \
  --from-file=envoy.yaml=./kube/envoy.yaml \
  --namespace=loco-setup


# envoy -gateway

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
k apply -f https://github.com/envoyproxy/gateway/releases/download/v1.4.1/envoy-gateway-crds.yaml



```
