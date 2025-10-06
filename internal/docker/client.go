package docker

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/moby/go-archive"
	"github.com/nikumar1206/loco/internal/config"
)

var MINIMUM_DOCKER_ENGINE_VERSION = "28.0.0"

type DockerClient struct {
	dockerClient *client.Client
	cfg          config.Config
	registryUrl  string
	ImageName    string
}

func NewDockerClient(cfg config.Config) (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	v, err := cli.ServerVersion(context.Background())
	if err != nil {
		return nil, err
	}

	if v.Version < MINIMUM_DOCKER_ENGINE_VERSION {
		return nil, fmt.Errorf("loco requires minimum Docker engine version of %s. Please update your Docker version", MINIMUM_DOCKER_ENGINE_VERSION)
	}

	return &DockerClient{
		dockerClient: cli,
		cfg:          cfg,
		registryUrl:  "registry.gitlab.com",
	}, nil
}

func (c *DockerClient) Close() error {
	if c.dockerClient != nil {
		return c.dockerClient.Close()
	}
	return nil
}

type Message struct {
	Stream string `json:"stream"`
	Status string `json:"status"`
	ID     string `json:"id"`
	Aux    struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

func printDockerOutput(r io.Reader, logf func(string)) error {
	scanner := bufio.NewScanner(r)
	seenStatuses := make(map[string]string)

	for scanner.Scan() {
		var msg Message
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // skip unparseable lines
		}
		switch {
		case msg.Status != "":
			// Only log new, meaningful status changes (skip "Waiting", "Downloading", etc.)
			if msg.ID != "" {
				if prev, ok := seenStatuses[msg.ID]; ok && prev == msg.Status {
					continue
				}
				seenStatuses[msg.ID] = msg.Status
			}
			// only log certain messages, to reduce noise
			if strings.Contains(msg.Status, "Built") ||
				strings.Contains(msg.Status, "Pushed") ||
				strings.Contains(msg.Status, "Successfully") ||
				strings.Contains(msg.Status, "latest") {
				logf(msg.Status)
			}
		case msg.Stream != "":
			if strings.HasPrefix(msg.Stream, "Step") ||
				strings.HasPrefix(msg.Stream, "Successfully") {
				logf(strings.TrimSpace(msg.Stream))
			}
		case msg.Aux.ID != "":
			logf("Image ID: " + msg.Aux.ID)
		}
	}
	return scanner.Err()
}

func (c *DockerClient) BuildImage(ctx context.Context, logf func(string)) error {
	buildContext, err := archive.TarWithOptions(c.cfg.ProjectPath, &archive.TarOptions{})
	if err != nil {
		return err
	}
	defer buildContext.Close()

	slog.Debug("built docker context", slog.String("project", c.cfg.ProjectPath))
	relDockerfilePath, err := filepath.Rel(c.cfg.ProjectPath, c.cfg.LocoConfig.Build.DockerfilePath)
	if err != nil {
		return err
	}

	slog.Debug("dockerfile path", slog.String("path", relDockerfilePath))
	options := build.ImageBuildOptions{
		Tags:       []string{c.ImageName},
		Dockerfile: relDockerfilePath,
		Remove:     true, // remove intermediate containers
		Platform:   "linux/amd64",
		Version:    build.BuilderBuildKit,
	}
	// todo: should we have memory limits or similar for the build process?

	response, err := c.dockerClient.ImageBuild(ctx, buildContext, options)
	if err != nil {
		return fmt.Errorf("build error: %v", err)
	}
	defer response.Body.Close()

	return printDockerOutput(response.Body, logf)
}

func (c *DockerClient) PushImage(ctx context.Context, logf func(string), username, password string) error {
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: c.registryUrl,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return fmt.Errorf("error when encoding authConfig: %v", err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	pushOptions := image.PushOptions{
		RegistryAuth: authStr,
	}

	rc, err := c.dockerClient.ImagePush(ctx, c.ImageName, pushOptions)
	if err != nil {
		return fmt.Errorf("error when pushing image: %v", err)
	}
	defer rc.Close()

	return printDockerOutput(rc, logf)
}

func (c *DockerClient) GenerateImageTag(imageBase string) string {
	imageNameBase := imageBase

	tag := fmt.Sprintf("%s-%s", c.cfg.LocoConfig.Metadata.Name, time.Now().Format("20060102-150405")+"-"+GenerateRand(4))

	if !strings.Contains(imageNameBase, ":") {
		imageNameBase += ":" + tag
	}
	c.ImageName = imageNameBase
	return imageNameBase
}

func GenerateRand(n int) string {
	token := make([]byte, n)
	rand.Read(token)
	return fmt.Sprintf("%x", token)
}
