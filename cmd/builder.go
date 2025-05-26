// responsible for building the Docker image
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/nikumar1206/loco/internal/color"
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
	// fmt.Println("what are we getting to print", s)
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
			fmt.Println(color.Colorize(line, color.FgRed)) // unparseable JSON
			continue
		}

		switch {
		case msg.Stream != "":
			printColoredStream(msg.Stream)
		case msg.Status != "":
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
			fmt.Println(color.Colorize("Image ID: "+msg.Aux.ID, color.FgBrightBlue))
		default:
			fmt.Println(line)
		}
	}
	return scanner.Err()
}

func buildDockerImage(ctx context.Context, c *client.Client) error {
	defer c.Close()

	imageName := "myapp:latest"
	contextDir := "." // directory containing Dockerfile and app

	buildContext, err := tarDirectory(contextDir)
	if err != nil {
		return err
	}
	defer buildContext.Close()

	options := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	response, err := c.ImageBuild(ctx, buildContext, options)
	if err != nil {
		fmt.Println("Build error:", err)
		return err
	}
	defer response.Body.Close()

	return printDockerOutput(response.Body)
}
