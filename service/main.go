package main

import (
	"log"
	"os/exec"

	"github.com/gofiber/fiber/v2"
)

type DeployRequest struct {
	Repo   string `json:"repo"`    // e.g. "someuser/myapp"
	GitURL string `json:"git_url"` // e.g. "https://github.com/someuser/myapp.git"`
	Ref    string `json:"ref"`     // optional: "refs/heads/main"
}

func main() {
	app := fiber.New()

	app.Post("/api/deploy", func(c *fiber.Ctx) error {
		var payload DeployRequest
		if err := c.BodyParser(&payload); err != nil {
			log.Println("invalid payload:", err)
			return c.Status(fiber.StatusBadRequest).SendString("invalid payload")
		}

		if payload.Ref != "" && payload.Ref != "refs/heads/main" {
			log.Println("not main branch, skipping")
			return c.SendStatus(fiber.StatusOK)
		}

		log.Printf("Triggering deploy for %s (%s)", payload.Repo, payload.GitURL)

		// Kick off deploy in background
		go runPipeline(payload.GitURL, payload.Repo)

		return c.SendStatus(fiber.StatusAccepted)
	})

	log.Fatal(app.Listen(":8080"))
}

func runPipeline(gitURL, repo string) {
	log.Printf("[BUILD] Cloning %s", gitURL)

	// Clone into a temp dir
	tmpDir := "/tmp/" + repo
	cmd := exec.Command("git", "clone", gitURL, tmpDir)
	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] git clone failed: %v", err)
		return
	}

	// TODO: parse loco.toml, build Docker image, deploy to K8s
	log.Printf("[DONE] Cloned repo to %s", tmpDir)
}
