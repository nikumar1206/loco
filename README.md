# 🚂 Loco

> Deploy containerized apps right from your terminal.

Loco is a lightweight container orchestration platform that simplifies application deployment. Run `loco deploy` and Loco handles the rest - building, deploying, and scaling your applications on Kubernetes.

## ✨ Features

- **One click deployments** - Deploy with just `loco deploy`
- **Automatic Builds** - Dockerfile-based container builds
- **Auto-scaling** - CPU and memory-based horizontal scaling
- **HTTPS by default** - Automatic SSL certificate management, powered by Envoy Gateway
- **Simple Configuration** - Easy setup via `loco.toml`

## 🚀 Quick Start

1.  **Download the loco cli**

```bash
go install github.com/nikumar1206/loco@latest
```

2. **Run `loco init` to create a `loco.toml` file.**

   ```toml
   name = "myapp"
   port = 3000

   [replicas]
   min = 1
   max = 5
   ```

3. **Deploy your app:**
   ```bash
   loco deploy
   ```

Your app will be available at `https://myapp.loco.dev`

## 📦 Installation

### CLI Installation

```bash
# Install via Go
go install github.com/nikumar1206/loco@latest
```

Loco also generates completions for shells such as bash and zshrc.

```bash
loco completion zsh
```

## 📚 Documentation

To be added later.

## 🤝 Contributing

To be added later.

---

**Note:** This project is primarily educational, created so I can learn more about Kubernetes, networking, and security.

`
“Engines warming up…”

“Switching tracks…”

“Pushing to the mainline…”

“Pods aligned. Ready for departure.”
`
