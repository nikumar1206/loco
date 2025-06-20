# 🚂 Loco

> Deploy containerized apps to Kubernetes with a simple git push

Loco is a lightweight container orchestration platform that simplifies application deployment. Push your code, and Loco handles the rest - building, deploying, and scaling your applications on Kubernetes.

## ✨ Features

- **Git Push Deployments** - Deploy with `git push loco main`
- **Automatic Builds** - Dockerfile-based container builds
- **Auto-scaling** - CPU and memory-based horizontal scaling
- **HTTPS by default** - Automatic SSL certificate management
- **Simple Configuration** - Easy setup via `loco.toml`

## 🚀 Quick Start

1. **Add Loco as a git remote:**

   ```bash
   git remote add loco git@your-loco-host:username/app.git
   ```

2. **Create a `loco.toml` configuration:**

   ```toml
   name = "myapp"
   port = 3000

   [replicas]
   min = 1
   max = 5
   ```

3. **Deploy your app:**
   ```bash
   git push loco main
   ```

Your app will be available at `https://myapp.loco.dev`

## 📦 Installation

### CLI Installation

```bash
# Install via Go
go install github.com/your-username/loco/cli@latest

# Or download binary from releases
curl -sSL https://github.com/your-username/loco/releases/latest/download/loco-linux-amd64 -o loco
chmod +x loco && sudo mv loco /usr/local/bin/
```

### Platform Setup

See [IMPLEMENTATION.md](./IMPLEMENTATION.md) for detailed setup instructions.

## 📚 Documentation

- [Implementation Details](./IMPLEMENTATION.md) - Architecture and technical details
- [Configuration Reference](./docs/configuration.md) - Complete loco.toml reference
- [Examples](./examples/) - Sample applications

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for details.

## 📧 Support

- **Issues:** [GitHub Issues](https://github.com/your-username/loco/issues)
- **Discussions:** [GitHub Discussions](https://github.com/your-username/loco/discussions)
- **Email:** loco-support@your-domain.com

## 📄 License

MIT License - see [LICENSE](./LICENSE) for details.

---

**Note:** This project is primarily educational, designed to explore container orchestration and deployment workflows.

`
“Engines warming up…”

“Switching tracks…”

“Pushing to the mainline…”

“Pods aligned. Ready for departure.”
`
