package common

import (
	"context"
	"io"
	"strings"

	"github.com/docker/docker/api/types/build"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// MockDockerClient is a mock implementation of Docker client for testing
type MockDockerClient struct {
	// Containers to return from ContainerList
	Containers []containertypes.Summary
	// Images to return from ImageList
	Images []image.Summary
	// Volumes to return from VolumeCreate/VolumeList
	Volumes map[string]*volume.Volume
	// VolumeListResponse to return from VolumeList
	VolumeListResponse *volume.ListResponse
	// Networks to return from NetworkCreate/NetworkList
	Networks map[string]string
	// NetworkListResponse to return from NetworkList
	NetworkListResponse []networktypes.Summary
	// Error to return from operations
	Err error
	// Track function calls
	ContainerListCalled   bool
	ContainerCreateCalled bool
	ContainerStartCalled  bool
	ContainerStopCalled   bool
	ContainerRemoveCalled bool
	ImageListCalled       bool
	ImagePullCalled       bool
	ImageBuildCalled      bool
	ImagePushCalled       bool
	VolumeCreateCalled    bool
	VolumeListCalled      bool
	VolumeRemoveCalled    bool
	NetworkCreateCalled   bool
	NetworkConnectCalled  bool
	NetworkListCalled     bool
	CopyToContainerCalled bool
	ContainerWaitCalled   bool
	ContainerLogsCalled   bool
	// Store last call parameters
	LastContainerID   string
	LastImageTag      string
	LastVolumeName    string
	LastNetworkName   string
	LastContainerName string
}

// NewMockDockerClient creates a new mock Docker client
func NewMockDockerClient() *MockDockerClient {
	return &MockDockerClient{
		Containers: []containertypes.Summary{},
		Images:     []image.Summary{},
		Volumes:    make(map[string]*volume.Volume),
		Networks:   make(map[string]string),
	}
}

// ContainerList mocks listing containers
func (m *MockDockerClient) ContainerList(ctx context.Context, options containertypes.ListOptions) ([]containertypes.Summary, error) {
	m.ContainerListCalled = true
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Containers, nil
}

// ContainerCreate mocks creating a container with v28.x signature (includes Platform parameter)
func (m *MockDockerClient) ContainerCreate(
	ctx context.Context,
	config *containertypes.Config,
	hostConfig *containertypes.HostConfig,
	networkingConfig *networktypes.NetworkingConfig,
	platform *ocispec.Platform,
	containerName string,
) (containertypes.CreateResponse, error) {
	m.ContainerCreateCalled = true
	m.LastContainerName = containerName
	if m.Err != nil {
		return containertypes.CreateResponse{}, m.Err
	}
	return containertypes.CreateResponse{ID: "mock-container-id-" + containerName}, nil
}

// ContainerStart mocks starting a container
func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options containertypes.StartOptions) error {
	m.ContainerStartCalled = true
	m.LastContainerID = containerID
	return m.Err
}

// ContainerStop mocks stopping a container
func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string, options containertypes.StopOptions) error {
	m.ContainerStopCalled = true
	m.LastContainerID = containerID
	return m.Err
}

// ContainerWait mocks waiting for a container
func (m *MockDockerClient) ContainerWait(ctx context.Context, containerID string, condition containertypes.WaitCondition) (<-chan containertypes.WaitResponse, <-chan error) {
	m.ContainerWaitCalled = true
	m.LastContainerID = containerID

	statusCh := make(chan containertypes.WaitResponse, 1)
	errCh := make(chan error, 1)

	if m.Err != nil {
		errCh <- m.Err
	} else {
		statusCh <- containertypes.WaitResponse{StatusCode: 0}
	}

	return statusCh, errCh
}

// ContainerLogs mocks getting container logs
func (m *MockDockerClient) ContainerLogs(ctx context.Context, containerID string, options containertypes.LogsOptions) (io.ReadCloser, error) {
	m.ContainerLogsCalled = true
	m.LastContainerID = containerID
	if m.Err != nil {
		return nil, m.Err
	}
	return io.NopCloser(strings.NewReader("mock container logs")), nil
}

// ImageList mocks listing images
func (m *MockDockerClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	m.ImageListCalled = true
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Images, nil
}

// ImagePull mocks pulling an image
func (m *MockDockerClient) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	m.ImagePullCalled = true
	m.LastImageTag = refStr
	if m.Err != nil {
		return nil, m.Err
	}
	return io.NopCloser(strings.NewReader(`{"status":"Pull complete"}`)), nil
}

// ImageBuild mocks building an image
func (m *MockDockerClient) ImageBuild(ctx context.Context, buildContext io.Reader, options build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	m.ImageBuildCalled = true
	if len(options.Tags) > 0 {
		m.LastImageTag = options.Tags[0]
	}
	if m.Err != nil {
		return build.ImageBuildResponse{}, m.Err
	}
	return build.ImageBuildResponse{
		Body: io.NopCloser(strings.NewReader(`{"stream":"Successfully built mock-image"}`)),
	}, nil
}

// ImagePush mocks pushing an image
func (m *MockDockerClient) ImagePush(ctx context.Context, image string, options image.PushOptions) (io.ReadCloser, error) {
	m.ImagePushCalled = true
	m.LastImageTag = image
	if m.Err != nil {
		return nil, m.Err
	}
	return io.NopCloser(strings.NewReader(`{"status":"Push complete"}`)), nil
}

// VolumeCreate mocks creating a volume
func (m *MockDockerClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	m.VolumeCreateCalled = true
	m.LastVolumeName = options.Name
	if m.Err != nil {
		return volume.Volume{}, m.Err
	}

	vol := volume.Volume{
		Name:       options.Name,
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/" + options.Name + "/_data",
	}
	m.Volumes[options.Name] = &vol
	return vol, nil
}

// NetworkCreate mocks creating a network
func (m *MockDockerClient) NetworkCreate(ctx context.Context, name string, options networktypes.CreateOptions) (networktypes.CreateResponse, error) {
	m.NetworkCreateCalled = true
	m.LastNetworkName = name
	if m.Err != nil {
		return networktypes.CreateResponse{}, m.Err
	}

	networkID := "mock-network-id-" + name
	m.Networks[name] = networkID
	return networktypes.CreateResponse{ID: networkID}, nil
}

// NetworkConnect mocks connecting a container to a network
func (m *MockDockerClient) NetworkConnect(ctx context.Context, networkID, containerID string, config *networktypes.EndpointSettings) error {
	m.NetworkConnectCalled = true
	m.LastNetworkName = networkID
	m.LastContainerID = containerID
	return m.Err
}

// CopyToContainer mocks copying files to a container
func (m *MockDockerClient) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options containertypes.CopyToContainerOptions) error {
	m.CopyToContainerCalled = true
	m.LastContainerID = containerID
	return m.Err
}

// Close mocks closing the client
func (m *MockDockerClient) Close() error {
	return nil
}

// VolumeList mocks listing volumes
func (m *MockDockerClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	m.VolumeListCalled = true
	if m.Err != nil {
		return volume.ListResponse{}, m.Err
	}
	if m.VolumeListResponse != nil {
		return *m.VolumeListResponse, nil
	}
	// Return volumes from map if VolumeListResponse is not set
	vols := make([]*volume.Volume, 0, len(m.Volumes))
	for _, v := range m.Volumes {
		vols = append(vols, v)
	}
	return volume.ListResponse{Volumes: vols}, nil
}

// VolumeRemove mocks removing a volume
func (m *MockDockerClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	m.VolumeRemoveCalled = true
	m.LastVolumeName = volumeID
	if m.Err != nil {
		return m.Err
	}
	delete(m.Volumes, volumeID)
	return nil
}

// NetworkList mocks listing networks
func (m *MockDockerClient) NetworkList(ctx context.Context, options networktypes.ListOptions) ([]networktypes.Summary, error) {
	m.NetworkListCalled = true
	if m.Err != nil {
		return nil, m.Err
	}
	return m.NetworkListResponse, nil
}

// ContainerRemove mocks removing a container
func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options containertypes.RemoveOptions) error {
	m.ContainerRemoveCalled = true
	m.LastContainerID = containerID
	return m.Err
}
