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
  - do we need to remove serviceAccount token for apps deployed by users?
  - need to actually resolve the paths for loco.toml

Next Steps:

- loco-api will require cluster role, rolebinding, and service account for talking to kube-api-server and making cluster lvl changes to it. obs eventually, need to figure out a way to ensure loco-api is safe and cannot be abused
- need to actually implement the github oauth setup with JWT, so we are not exposing APIs
- implement something like getting logs or getting an app status endpoints once above is working.
- enhance logic of the deploy endpoint so it fails on re-deployment. that should actually be handled on the CLI side.
- fix the endpoints to be more restful. it should be METHOD: /api/v1/app. POST should do a deploy theoretically
- why do we keep running OOM
  figure out this bit and whether we should actually run it
  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
- rename loco-setup to loco-system
- fix the logging. should all be logging with context on the service
- cleanup the deployapp code, as well as the github callback code
- failing to build images is not being reported, and continues as Successfully
- cross platform builds seem to be problematic in that not all errors are being captured, and there are some interim docker issues
- loco cli should write to a debug file.

  ran into this issue:

k describe pods/loco-api-6f94566669-9wz4v -n loco-setup
Name: loco-api-6f94566669-9wz4v
Namespace: loco-setup
Priority: 0
Service Account: default
Node: worker-pool-t4zde/10.116.0.5
Start Time: Sun, 22 Jun 2025 12:40:48 -0400
Labels: app=loco-api
pod-template-hash=6f94566669
Annotations: <none>
Status: Pending
IP:
IPs: <none>
Controlled By: ReplicaSet/loco-api-6f94566669
Containers:
loco-api:
Container ID:
Image: ghcr.io/nikumar1206/loco:latest
Image ID:
Port: 8000/TCP
Host Port: 0/TCP
State: Waiting
Reason: ContainerCreating
Ready: False
Restart Count: 0
Limits:
cpu: 100m
memory: 100m
Requests:
cpu: 100m
memory: 100m
Environment Variables from:
env-config Secret Optional: false
Environment: <none>
Mounts:
/var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-dwsdg (ro)
Conditions:
Type Status
PodReadyToStartContainers False
Initialized True
Ready False
ContainersReady False
PodScheduled True
Volumes:
kube-api-access-dwsdg:
Type: Projected (a volume that contains injected data from multiple sources)
TokenExpirationSeconds: 3607
ConfigMapName: kube-root-ca.crt
Optional: false
DownwardAPI: true
QoS Class: Guaranteed
Node-Selectors: <none>
Tolerations: node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events: Type Reason Age From Message
---- ------ ---- ---- ------- Normal Scheduled 3m24s default-scheduler Successfully assigned loco-setup/loco-api-6f94566669-9wz4v to worker-pool-t4zde
Warning FailedMount 3m24s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_40_48.3551581184/token: no space left on device
Warning FailedMount 3m23s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_40_49.3931323834/namespace: no space left on device
Warning FailedMount 3m22s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_40_50.2625020399/token: no space left on device
Warning FailedMount 3m20s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_40_52.325776056/ca.crt: no space left on device
Warning FailedMount 3m16s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_40_56.3244351780/ca.crt: no space left on device
Warning FailedMount 3m8s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_41_04.753710740/token: no space left on device
Warning FailedMount 2m52s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_41_20.71064940/ca.crt: no space left on device
Warning FailedMount 2m20s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_41_52.3636505467/ca.crt: no space left on device
Warning FailedMount 76s kubelet MountVolume.SetUp failed for volume "kube-api-access-dwsdg" : write /var/lib/kubelet/pods/9325809b-7a46-4d73-916b-99e366b7aace/volumes/kubernetes.io~projected/kube-api-access-dwsdg/..2025_06_22_16_42_56.1446387085/ca.crt: no space left on device
