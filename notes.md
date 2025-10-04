## High Priority

- Flesh out the database structure (teams, users, apps, deployments, events)
- Implement RBAC for strict permission control
  - this is more of a general statement than an actionable item.
- Monitoring
  - implement e2e monitoring, starting with envoy-proxy, loco-api.
  - take what i learn and apply to user pod monitoring as well
  - Switch to `kube-prometheus-stack` (remove `eg-addons`)
  - Configure `ServiceMonitor` manually for Envoy
  - Set up monitoring with non-emptyDir, idk what this means
  - Stick to open standards: Prometheus / Grafana / OpenTelemetry
- Logging
  - implement e2e logging, starting with envoy-proxy, loco-api.
  - take what i learn and apply to user pod logging as well
  - logs should only be visible to the person owning the project.
- theoretically need 2 clusters for loco development; a dev cluster, and a prod cluster.

  - should users have some sort of environment feature?

- look into connectRPC for API server/client code generation. also supports streaming
- logs cmd should take an output flag so they can be serialized as JSON and users can use jq

  - should also have a simple yank command to grab the whole log as json
  - if we ever introduce streaming logs in real time, we should include a freeze

- Support and test different deployment types: UI, cache (Redis), DB, Blob
- Tracing
  - deferring for now, don't have an idea for this.

---

## Medium Priority

- Set registry lifecycle policy (start with 6 months)
- Require image prefixing with random hash
- Only allow registry writes from our infra, not reads
- Store only last 2 images per project
- Support Podman / OCI-based images
- Set max Docker image size (cluster limited)

---

## Low Priority

- Add potential CLI commands: `docs`, `env`, `account`

- Make brew package for CLI with auto shell completions
- Enable deployment to path prefixes (`/api`, `/`, etc.)
- Auto-fill default `loco.toml` values if omitted
- Centralize all timestamps
- Deploy tokens should expire in 5 mins
- Registry read tokens: max 10 mins
- Use Terraform to create Kubernetes clusters/resources
- Fix “already exists” errors gracefully in GitHub Actions
- Generate namespace from user-app hash to avoid conflicts
- Ensure nodes are ready before allowing loco deploy
- SSE-based endpoint for deploying
- Validate necessary resources before deploy
- apiClient and locoConfig seem to be a shared object
- Consider switching to private Docker Hub vs GitLab
- Reuse deploy token instead of regenerating on backend
- Potentially add artifact attestations to images
- Kubernetes configmap of secrets needs to be created separately
- Create RBAC to restrict secret visibility for env vars
- Resolve paths for loco.toml
- Wait for new pod readiness before marking deploy as successful
- Add example to README
- Provide example loco.toml path
- Improve loco.toml scalability and visibility
- Make loco CI-friendly via `loco deploy --non-interactive --token {GH-TOKEN}`
- Add `loco env` command to update just .env
- Respect/Allow specifying .dockerignore files / .gitignore files when building container images.
- Add multi-port container support
  - super low prio. i can see the usecase, but pref one entry point for simplicity
- Unhappy paths should offer clear, actionable steps

---

## Next Steps

- Add endpoints for getting logs and app status
- Enhance deploy endpoint to fail on redeploy (CLI side)
- Investigate OOM issues
- Evaluate whether to install metrics-server
- Improve logging with contextual information
- Cleanup deployApp and GitHub callback code
- Ensure image build failures are correctly reported
- Address cross-platform build errors and Docker issues
- Add debug log file support to loco CLI
- probably need a way to update the kube context for all deployments.
- how are we handling security patches?
- loco versions will need to not break previous apps deployed.
  - loco.toml should take in some sort of versioning ability
- Somehow already need to start cleaning the code up.
  - API code is horrendous tbh
- also gitlab fetch token is only valid at deployment. what if new node comes in and needs to pull down image, it cannot since gitlab token expires in like 5 mins.

\*\*
next thing i wanna work on is allowing for /prefix based routing instead of subdomain.

so same subdomain lets say koko can deploy:
koko.deploy-app.com / api
koko.deploy-app.com / ui

both should be deployable and routable to via just a single pyproject.toml
so does that mean one needs sub builds?
each build will need a separate docker file.
is this 2 separate pods ? or just one kubernetes pod machine with 2 containers?
