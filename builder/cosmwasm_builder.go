package builder

import (
	"context"
	"fmt"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

type CosmWasmBuilder struct {
	BuilderImage string
}

func (b CosmWasmBuilder) BuildWasm(repoDir, projectName, crateName string, allowArm bool) ([]byte, error) {
	cli, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}
	defer func() {
		if err = cli.Close(); err != nil {
			panic(err)
		}
	}()

	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, b.BuilderImage, dockertypes.ImagePullOptions{})
	if err == nil {
		defer func(reader io.ReadCloser) {
			err = reader.Close()
			if err != nil {
				panic(err)
			}
		}(reader)
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read pull image output")
		}
	} else {
		return nil, errors.Wrapf(err, "failed to pull image %s", b.BuilderImage)
	}

	imageInfo, _, err := cli.ImageInspectWithRaw(ctx, b.BuilderImage)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to inspect image %s", b.BuilderImage)
	}
	var archSuffix string
	if imageInfo.Architecture == "arm64" {
		if allowArm {
			archSuffix = "-aarch64"
		} else {
			return nil, errors.Errorf(
				`ARM builds are not allowed. 
You may either use x86_64 rust-optimizer image or use --allow-arm flag to bypass this requirement`,
			)
		}
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: repoDir,
			Target: "/code",
		},
		{
			Type:   mount.TypeVolume,
			Source: "registry_cache",
			Target: "/usr/local/cargo/registry",
		},
		{
			Type:   mount.TypeVolume,
			Source: fmt.Sprintf("%s_cache", projectName),
			Target: "/code/target",
		},
	}
	_, err = cli.ContainerCreate(ctx, &container.Config{
		Image: b.BuilderImage,
	}, &container.HostConfig{Mounts: mounts}, nil, nil, containerName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create builder container")
	}

	if err = cli.ContainerStart(ctx, containerName, dockertypes.ContainerStartOptions{}); err != nil {
		return nil, errors.Wrapf(err, "failed to start builder container")
	}
	defer func() {
		_ = cli.ContainerStop(ctx, containerName, nil)
		if err = cli.ContainerRemove(ctx, containerName, dockertypes.ContainerRemoveOptions{Force: true}); err != nil {
			panic(err)
		}
	}()

	done := make(chan struct{})
	go func() {
		for {
			out, err := cli.ContainerLogs(ctx, containerName, dockertypes.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
			if err != nil {
				panic(err)
			}
			if _, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
				panic(err)
			}
			select {
			case <-done:
				return
			default:
				continue
			}
		}
	}()

	statusCh, errCh := cli.ContainerWait(ctx, containerName, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			return nil, errors.Wrapf(err, "failed to wait for builder container")
		}
	case <-statusCh:
		done <- struct{}{}
		log.Info("Container exited")
	}

	wasmName := strings.Replace(crateName, "-", "_", -1) + archSuffix

	return readWasmFile(fmt.Sprintf("%s/artifacts/%s.wasm", repoDir, wasmName)), nil
}
