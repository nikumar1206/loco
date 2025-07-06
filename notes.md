## High Priority

- Build out the database structure (teams, users, apps, deployments, events)
- Do not deploy image if Docker Scout or vulnerabilities are detected
- Implement RBAC for strict permission control
  - this is more of a general statement than an actionable item.
- Start monitoring/log exporting? with `loco-api`.
- Loco admins should have no access to user pod logs
- Provide a way to deliver logs back to users
- Unhappy paths should offer clear, actionable steps
- theoretically need a development cluster, followed by a production cluster.

---

## Medium Priority

- Support common deployments: UI, cache (Redis), DB, blob
- Switch to `kube-prometheus-stack` (remove `eg-addons`)
- Configure `ServiceMonitor` manually for Envoy
- Set up monitoring with non-emptyDir
- Stick to open standards: Prometheus / Grafana / OpenTelemetry
- Defer tracing to future
- Cache Docker layers per project
- Set registry lifecycle policy (start with 6 months)
- Require image prefixing with random hash
- Only allow registry writes from our infra, not reads
- Store only last 2 images per project
- Support Podman / OCI-based images
- Set max Docker image size (cluster limited)

---

## Low Priority

- Add potential CLI commands: `docs`, `variables`, `login`, `account`
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

---

## Next Steps

- Add endpoints for getting logs and app status
- Enhance deploy endpoint to fail on redeploy (CLI side)
- Fix endpoints to follow RESTful conventions (`POST /api/v1/app` should deploy)
- Investigate OOM issues
- Evaluate whether to install metrics-server
- Rename `loco-setup` to `loco-system`
- Improve logging with contextual information
- Cleanup deployApp and GitHub callback code
- Ensure image build failures are correctly reported
- Address cross-platform build errors and Docker issues
- Add debug log file support to loco CLI
- probably need a way to update the kube context for all deployments.
- how are we handling security patches?
- loco versions will need to not break previous apps deployed.
  - loco.toml should take in some sort of versioning ability
