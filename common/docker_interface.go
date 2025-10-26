package common

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/build"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClient defines the interface for Docker operations.
// This interface abstracts the Docker SDK client to enable dependency injection
// and testing with mock implementations.
type DockerClient interface {
	// Container operations
	ContainerList(ctx context.Context, options containertypes.ListOptions) ([]containertypes.Summary, error)
	ContainerCreate(
		ctx context.Context,
		config *containertypes.Config,
		hostConfig *containertypes.HostConfig,
		networkingConfig *networktypes.NetworkingConfig,
		platform *ocispec.Platform,
		containerName string,
	) (containertypes.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options containertypes.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options containertypes.StopOptions) error
	ContainerWait(ctx context.Context, containerID string, condition containertypes.WaitCondition) (<-chan containertypes.WaitResponse, <-chan error)
	ContainerLogs(ctx context.Context, containerID string, options containertypes.LogsOptions) (io.ReadCloser, error)
	CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options containertypes.CopyToContainerOptions) error

	// Image operations
	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ImageBuild(ctx context.Context, buildContext io.Reader, options build.ImageBuildOptions) (build.ImageBuildResponse, error)
	ImagePush(ctx context.Context, image string, options image.PushOptions) (io.ReadCloser, error)

	// Volume operations
	VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error)

	// Network operations
	NetworkCreate(ctx context.Context, name string, options networktypes.CreateOptions) (networktypes.CreateResponse, error)
	NetworkConnect(ctx context.Context, networkID, containerID string, config *networktypes.EndpointSettings) error

	// Client lifecycle
	Close() error
}
