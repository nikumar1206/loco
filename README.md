# 🚂 Loco

> Deploy containerized apps right from your terminal.

Loco is a container orchestration platform that simplifies application deployment. Run `loco deploy` and Loco handles the rest - building, deploying, and scaling your applications on Kubernetes.

## Features

- **One click deployments** - Deploy with just `loco deploy`
- **Automatic Builds** - Dockerfile-based container builds.
- **Auto-scaling** - Sensible CPU based horizontal scaling that can be easily configured.
- **HTTPS by default** - Automatic SSL certificate management, powered by Let's Encrypt and Certificate Manager.
- **Fast Request Proxy** - Simple, sensible, and scalable infra. Network requests to Loco Applications go through an NLB (on Digital Ocean), followed by an ALB (Envoy Gateway via the Kubernetes Gateway API).
- **Simple Configuration** - Easy setup via `loco.toml`. A sample spec can be generated via `loco init`

## Architecture Diagram

![Architecture Diagram](./arch-light.png)

## Quick Start

1.  **Download the loco cli**

```bash
go install github.com/nikumar1206/loco@latest
```

2. **Run `loco init` to create a `loco.toml` file.**

3. **Deploy your app via `loco deploy`**

Your app will be available at `https://myapp.deploy-app.com`

See all loco cli commands via `loco help`.
Loco also generates completions for shells such as bash and zshrc.

```bash
loco completion zsh
```

## Abuse Prevention

To avoid abuse, Loco uses an invitation system. The repo collaborators is re-purposed as an invitation list and determines who can deploy with Loco.
You must first reach out to me, nikumar1206, if you would like to deploy on this platform.

## Documentation

To be added later.

## Contributing

To be added later.

---

**Note:** This project is primarily educational, created so I can learn more about Kubernetes, networking, and security.

`
“Engines warming up…”

“Switching tracks…”

“Pushing to the mainline…”

“Pods aligned. Ready for departure.”
