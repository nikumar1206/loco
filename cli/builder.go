// responsible for building the Docker image
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/nikumar1206/loco/cli/internal/color"
)

type dockerMessage struct {
	Stream string `json:"stream"`
	Status string `json:"status"`
	ID     string `json:"id"`
	Aux    struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

func createDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		// This error is returned, so the caller (e.g., main.go) should handle printing it.
		// No locoErr here needed unless we also want to log it before returning.
		return nil, err
	}
	return cli, nil
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

		// Fix header name so it's relative
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

func printColoredStream(s string) {
	// This function prints parts of Docker's output with specific coloring.
	// It does not use locoOut/locoErr to avoid double-prefixing.
	switch {
	case len(s) > 1 && s[:4] == "Step":
		fmt.Print(color.Colorize(s, color.FgYellow))
	case len(s) > 1 && s[:12] == "Successfully":
		fmt.Print(color.Colorize(s, color.FgGreen))
	case len(s) > 1 && s[:6] == " ---> ":
		fmt.Print(color.Colorize(s, color.FgCyan))
	default:
		fmt.Print(s)
	}
}

func printDockerOutput(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var msg dockerMessage
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Changed to locoErr for unparseable JSON
			locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Error parsing Docker output line: %s", line))
			continue
		}

		switch {
		case msg.Stream != "":
			printColoredStream(msg.Stream) // Relies on its own coloring, no prefix
		case msg.Status != "":
			// This prints Docker's status messages with specific formatting.
			// Avoid double-prefixing.
			fmt.Println(
				color.ColorizeBold("» ", color.FgCyan) +
					color.Colorize(msg.Status, color.FgGreen) +
					func() string {
						if msg.ID != "" {
							return " " + color.Colorize("["+msg.ID+"]", color.FgYellow)
						}
						return ""
					}())
		case msg.Aux.ID != "":
			// This prints the image ID with specific formatting. Avoid double-prefixing.
			fmt.Println(color.Colorize("Image ID: "+msg.Aux.ID, color.FgBrightBlue))
		default:
			// Changed to locoOut for general Docker messages not caught by other cases
			locoOut(LOCO__OK_PREFIX, line)
		}
	}
	return scanner.Err()
}

func buildDockerImage(ctx context.Context, c *client.Client, imageName string) error {
	// Note: No c.Close() here, it should be managed by the caller (main.go)
	// as the client might be reused or closed in a defer there.

	contextDir := "." // directory containing Dockerfile and app

	// Potentially add locoOut here for "Starting to tar directory..."
	buildContext, err := tarDirectory(contextDir)
	if err != nil {
		// Errors from tarDirectory are returned and should be handled by the caller.
		// locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Error creating tar directory: %v", err))
		return err
	}
	defer buildContext.Close()

	options := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true, // Remove intermediate containers
	}

	// Potentially add locoOut here for "Starting image build..."
	response, err := c.ImageBuild(ctx, buildContext, options)
	if err != nil {
		// Changed to locoErr
		locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Build error: %v", err))
		return err
	}
	defer response.Body.Close()

	// Potentially add locoOut here for "Processing build output..."
	return printDockerOutput(response.Body)
}

// func dockerLogin(c *client.Client, username, password, serverAddress string) (string, error) {
// 	authConfig := registry.AuthConfig{
// 		Username:      username,
// 		Password:      password,
// 		ServerAddress: serverAddress,
// 	}

// 	resp, err := c.RegistryLogin(context.Background(), authConfig)
// 	if err != nil {
// 		// Original: fmt.Println("Login error:", err)
// 		// Would become: locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Login error: %v", err))
// 		return "", err
// 	}
// 	// Original: fmt.Println("")
// 	// Would become: locoOut(LOCO__OK_PREFIX, "") or just be removed if only for spacing
// 	base64 := base64.StdEncoding.EncodeToString([]byte(resp.IdentityToken))
// 	return base64, nil
// }

// func tagImage(c *client.Client, imageName, tag string) error {
// 	// Tag the image with the specified tag
// 	if err := c.ImageTag(context.Background(), imageName, tag); err != nil {
// 		return fmt.Errorf("error when tagging image %s with tag %s. err: %v", imageName, tag, err)
// 	}
// 	// Original: fmt.Printf("Image %s tagged with %s\n", imageName, tag)
// 	// Would become: locoOut(LOCO__OK_PREFIX, fmt.Sprintf("Image %s tagged with %s", imageName, tag))
// 	return nil
// }

func dockerPush(c *client.Client, username, password, serverAddress, imageName string) error {
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: serverAddress,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		// This error is returned, handled by caller.
		return fmt.Errorf("error when encoding authConfig. err: %v", err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	pushOptions := image.PushOptions{
		RegistryAuth: authStr,
	}

	// Potentially add locoOut for "Pushing image..."
	rc, err := c.ImagePush(context.Background(), imageName, pushOptions)
	if err != nil {
		// This error is returned, handled by caller.
		// locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Error when pushing image: %v", err))
		return fmt.Errorf("error when pushing image. err: %v", err)
	}
	defer rc.Close() // Ensure the response body is closed for push

	// Potentially add locoOut for "Processing push output..."
	return printDockerOutput(rc)
}
