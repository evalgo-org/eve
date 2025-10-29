package production

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"eve.evalgo.org/common"
)

func TestDefaultProductionConfig(t *testing.T) {
	config := DefaultProductionConfig()

	assert.Equal(t, "app-network", config.NetworkName)
	assert.True(t, config.CreateNetwork)
	assert.Empty(t, config.VolumeName)
	assert.False(t, config.CreateVolume)
}

func TestEnsureNetwork_CreatesNewNetwork(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
		Networks:         []network.Summary{},
	}

	err := EnsureNetwork(ctx, mock, "test-network")
	require.NoError(t, err)

	assert.True(t, mock.NetworkListCalled)
	assert.True(t, mock.NetworkCreateCalled)
	assert.Equal(t, "test-network", mock.LastNetworkName)
}

func TestEnsureNetwork_SkipsExistingNetwork(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
		Networks: []network.Summary{
			{Name: "test-network", ID: "existing-id"},
		},
	}

	err := EnsureNetwork(ctx, mock, "test-network")
	require.NoError(t, err)

	assert.True(t, mock.NetworkListCalled)
	assert.False(t, mock.NetworkCreateCalled, "Should not create network if it already exists")
}

func TestEnsureNetwork_HandlesListError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
	}
	mock.Err = assert.AnError

	err := EnsureNetwork(ctx, mock, "test-network")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list networks")
}

func TestEnsureVolume_CreatesNewVolume(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
		Volumes:          &volume.ListResponse{Volumes: []*volume.Volume{}},
	}

	err := EnsureVolume(ctx, mock, "test-volume")
	require.NoError(t, err)

	assert.True(t, mock.VolumeListCalled)
	assert.True(t, mock.VolumeCreateCalled)
	assert.Equal(t, "test-volume", mock.LastVolumeName)
}

func TestEnsureVolume_SkipsExistingVolume(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
		Volumes: &volume.ListResponse{
			Volumes: []*volume.Volume{
				{Name: "test-volume"},
			},
		},
	}

	err := EnsureVolume(ctx, mock, "test-volume")
	require.NoError(t, err)

	assert.True(t, mock.VolumeListCalled)
	assert.False(t, mock.VolumeCreateCalled, "Should not create volume if it already exists")
}

func TestEnsureVolume_HandlesListError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
	}
	mock.Err = assert.AnError

	err := EnsureVolume(ctx, mock, "test-volume")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list volumes")
}

func TestPrepareProductionEnvironment_CreatesNetworkAndVolume(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
		Networks:         []network.Summary{},
		Volumes:          &volume.ListResponse{Volumes: []*volume.Volume{}},
	}

	config := ProductionConfig{
		NetworkName:   "app-network",
		CreateNetwork: true,
		VolumeName:    "app-data",
		CreateVolume:  true,
	}

	err := PrepareProductionEnvironment(ctx, mock, config)
	require.NoError(t, err)

	assert.True(t, mock.NetworkListCalled)
	assert.True(t, mock.NetworkCreateCalled)
	assert.True(t, mock.VolumeListCalled)
	assert.True(t, mock.VolumeCreateCalled)
}

func TestPrepareProductionEnvironment_SkipsIfNotRequested(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
	}

	config := ProductionConfig{
		NetworkName:   "",
		CreateNetwork: false,
		VolumeName:    "",
		CreateVolume:  false,
	}

	err := PrepareProductionEnvironment(ctx, mock, config)
	require.NoError(t, err)

	assert.False(t, mock.NetworkListCalled)
	assert.False(t, mock.NetworkCreateCalled)
	assert.False(t, mock.VolumeListCalled)
	assert.False(t, mock.VolumeCreateCalled)
}

func TestPrepareProductionEnvironment_HandlesNetworkError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClientExtended{
		MockDockerClient: common.NewMockDockerClient(),
	}
	mock.Err = assert.AnError

	config := ProductionConfig{
		NetworkName:   "app-network",
		CreateNetwork: true,
	}

	err := PrepareProductionEnvironment(ctx, mock, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to prepare network")
}

// mockDockerClientExtended extends common.MockDockerClient with additional methods
type mockDockerClientExtended struct {
	*common.MockDockerClient
	Networks         []network.Summary
	Volumes          *volume.ListResponse
	NetworkListCalled bool
	VolumeListCalled  bool
	ContainerRemoveCalled bool
	VolumeRemoveCalled bool
}

func (m *mockDockerClientExtended) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	m.NetworkListCalled = true
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Networks, nil
}

func (m *mockDockerClientExtended) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	m.VolumeListCalled = true
	if m.Err != nil {
		return volume.ListResponse{}, m.Err
	}
	if m.Volumes == nil {
		return volume.ListResponse{Volumes: []*volume.Volume{}}, nil
	}
	return *m.Volumes, nil
}

func (m *mockDockerClientExtended) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	m.ContainerRemoveCalled = true
	m.LastContainerID = containerID
	return m.Err
}

func (m *mockDockerClientExtended) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	m.VolumeRemoveCalled = true
	m.LastVolumeName = volumeID
	return m.Err
}
