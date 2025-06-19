package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/nikumar1206/loco/cli/internal/color"
	"github.com/nikumar1206/loco/cli/pkg/config"
)

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

func tarDirectory(srcDir string) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	err := filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name, _ = filepath.Rel(srcDir, file)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return io.NopCloser(buf), nil
}

// Format stream output for the TUI instead of printing directly
func formatColoredStream(s string) string {
	// Remove trailing newlines since logf will add them
	s = strings.TrimSuffix(s, "\n")

	switch {
	case len(s) > 4 && s[:4] == "Step":
		return color.Colorize(s, color.FgYellow)
	case len(s) > 12 && s[:12] == "Successfully":
		return color.Colorize(s, color.FgGreen)
	case len(s) > 6 && s[:6] == " ---> ":
		return color.Colorize(s, color.FgCyan)
	default:
		return s
	}
}

func printDockerOutput(r io.Reader, logf func(string)) error {
	scanner := bufio.NewScanner(r)
	seenStatuses := make(map[string]string) // track layer ID -> last status

	for scanner.Scan() {
		var msg Message
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			logf(fmt.Sprintf("Failed to parse Docker output: %v — line: %s", err, line))
			continue
		}

		switch {
		case msg.Status != "":
			// Skip duplicate status for the same ID
			if msg.ID != "" {
				if prev, ok := seenStatuses[msg.ID]; ok && prev == msg.Status {
					continue // skip redundant
				}
				seenStatuses[msg.ID] = msg.Status
			}

			statusMsg := color.ColorizeBold("» ", color.FgCyan) +
				color.Colorize(msg.Status, color.FgGreen)
			if msg.ID != "" {
				statusMsg += " " + color.Colorize("["+msg.ID+"]", color.FgYellow)
			}
			logf(statusMsg)

		case msg.Stream != "":
			formattedStream := formatColoredStream(msg.Stream)
			if formattedStream != "" {
				logf(formattedStream)
			}
		case msg.Aux.ID != "":
			logf(color.Colorize("Image ID: "+msg.Aux.ID, color.FgBrightBlue))
		default:
			if line != "" {
				logf(line)
			}
		}
	}
	return scanner.Err()
}

func (c *DockerClient) BuildImage(ctx context.Context, logf func(string)) error {
	fmt.Println("what is the imagename", c.ImageName)
	time.Sleep(5 * time.Second)
	buildContext, err := tarDirectory(c.cfg.ProjectPath)
	if err != nil {
		return err
	}
	defer buildContext.Close()

	options := build.ImageBuildOptions{
		Tags:       []string{c.ImageName},
		Dockerfile: c.cfg.DockerfilePath,
		Remove:     true, // remove intermediate containers
	}

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

	tag := fmt.Sprintf("%s-%s", c.cfg.Name, time.Now().Format("20060102-150405")+"-"+generateRand(4))

	if !strings.Contains(imageNameBase, ":") {
		imageNameBase += ":" + tag
	}
	c.ImageName = imageNameBase
	return imageNameBase
}

func generateRand(n int) string {
	token := make([]byte, n)
	rand.Read(token)
	return fmt.Sprintf("%x", token)
}
