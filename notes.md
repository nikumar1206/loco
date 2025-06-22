# 🚨 High Priority

### 🔧 Core Infrastructure

- ✅ Build out the **database structure**
  - 5-layer concept: **teams, users, apps, deployments, events**
- 🔐 Implement **OAuth** with GitHub/Google
  - We will _not_ manage passwords

### 📦 Container & Cluster Security

- 🛑 **Do not deploy image** if Docker Scout or vulnerabilities are detected
- 🔒 Set `privileged: true`, `runAsNonRoot: true` when running user images
- 🧾 Enforce `.dockerignore` or `.gitignore` during image build
- ✅ Enforce **TOML schema validation**

### ⚙️ Kubernetes Essentials

- ⚠️ Create a **ServiceAccount for `loco-api`**
- 🏷️ Add **labeling for Gateway and Namespace**

  - certain namespaces should be allowed to create gateway routes, not all.

- ✅ Implement **RBAC** for strict permission control
  - need to figure out what this looks like

### 📊 Monitoring & Logs

- 🧪 Start monitoring/logging with our own container: **`loco-api`**
- 🧵 Users should only access **logs for their own projects**
- 🔧 Don't forget: we need a way to **deliver logs back to users**

### 🚫 Abuse Prevention

- ⚙️ need to do this via a github collaborator-based system
- 📣 **Unhappy paths** must be clearly reflected to the user with next steps

---

# ⚠️ Medium Priority

### 🧠 Platform Logic

- 📖 Actually read the `loco.toml` to determine build/deploy behavior
- 🔐 Enable **secure env var passing**
- 🗂️ Extend to cover common deployments:
  - UI, cache (Redis), DB, blob

### 📈 Monitoring (Phase 2)

- Switch to `kube-prometheus-stack` (remove `eg-addons`)
- Configure `ServiceMonitor` manually for Envoy
- Set up with **non-emptyDir**
- Stick to **open standards**: Prometheus / Grafana / OpenTelemetry
- 🚫 Tracing: defer to future

### 📦 Docker Registry

- 🗃️ Cache Docker layers per project (Docker may handle this already)
- 🧹 Registry lifecycle policy (start with 6 months)
- 🏷️ Images must be **prefixed + random hash** to avoid collisions
- 🔐 Only allow **registry write** from our infra, not read
- 📦 Store only **last 2 images** per project
- 🐳 Support **Podman / OCI-based images**
- 🚧 Add max Docker image size (our cluster is limited)

---

# 🧁 Low Priority

### 🧪 CLI & Tooling

- 🧱 Create MVP **architecture diagram**
- 🧰 Potential CLI commands: `docs`, `variables`, `redeploy`, `login`, `account`
- 🍺 Make **brew package** for CLI + auto shell completions
- 🎯 Enable deployment to **path prefixes** (`/api`, `/`, etc.)
- 📦 Add **multi-port** container support
- 💡 Auto-fill **default `loco.toml`** values if omitted
- Centralize all the different timestamps we use.

### 🔐 Tokens & Auth

- 🔑 Deploy tokens should expire in **5 mins**
- 📦 Registry **read tokens**: max **10 mins**
- 📏 Use **Terraform** to create Kubernetes cluster/resources

### 🐞 GitHub Actions

- 🛠️ Fix “already exists” errors (gracefully handle or ignore)

### 🔢 Namespacing

- Generate namespace from **user-app hash** to avoid conflicts

- need to ensure nodes are ready before allowing loco deploy to occur?
- SSE based endpoint for deploying?
- need to have a validation for the necessary resources

- apiClient and locoConfig seem to be a shared object
- maybe switch to private docker hub vs gitlab, not sure if there will be any difference
- no need to re-generate a deploy token on backend again. just simply use the same one
- maybe we can eventually add artifact attestations. seems simple enough to add them to the image, but need to figure out kubernetes side of things

- kube configmap of secrets needs to be created separately
- need to create some sort of RBAC actually.
  - so if user provides env var, they should not be visible to the cluster owner like me.
  - they should be created as a configmap or secret, and not directly decodeable unless by user
