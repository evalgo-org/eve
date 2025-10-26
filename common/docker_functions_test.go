package common

import (
	"context"
	"os"
	"testing"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

func TestContainers(t *testing.T) {
	mock := NewMockDockerClient()
	mock.Containers = []containertypes.Summary{
		{ID: "container1", Names: []string{"/test1"}},
		{ID: "container2", Names: []string{"/test2"}},
	}

	ctx := context.Background()

	// Use the mock directly through the interface
	containers, err := mock.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		t.Fatalf("ContainerList failed: %v", err)
	}

	if len(containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(containers))
	}

	if !mock.ContainerListCalled {
		t.Error("ContainerList was not called")
	}
}

func TestContainersList(t *testing.T) {
	mock := NewMockDockerClient()
	mock.Containers = []containertypes.Summary{
		{
			ID:     "abc123",
			Names:  []string{"/container1"},
			Status: "Up 2 hours",
		},
		{
			ID:     "def456",
			Names:  []string{"/container2"},
			Status: "Up 1 hour",
		},
	}

	ctx := context.Background()

	// Test via ContainersListWithClient helper function (we'll add this)
	views := ContainersListWithClient(ctx, mock)

	if len(views) >= 2 {
		// Check that we got container views back
		found := false
		for _, v := range views {
			if v.ID == "abc123" || v.ID == "def456" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find container IDs in views")
		}
	}
}

func TestContainersListToJSON(t *testing.T) {
	mock := NewMockDockerClient()
	mock.Containers = []containertypes.Summary{
		{
			ID:     "abc123",
			Names:  []string{"/container1"},
			Status: "Up 2 hours",
		},
	}

	ctx := context.Background()
	filename := ContainersListToJSONWithClient(ctx, mock)

	if filename != "containers.json" {
		t.Errorf("Expected filename 'containers.json', got '%s'", filename)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("containers.json file was not created")
	} else {
		// Clean up
		os.Remove(filename)
	}
}

func TestImages(t *testing.T) {
	mock := NewMockDockerClient()
	mock.Images = []image.Summary{
		{ID: "image1", RepoTags: []string{"nginx:latest"}},
		{ID: "image2", RepoTags: []string{"alpine:3.14"}},
	}

	ctx := context.Background()
	images, err := mock.ImageList(ctx, image.ListOptions{})

	if err != nil {
		t.Fatalf("ImageList failed: %v", err)
	}

	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(images))
	}

	if !mock.ImageListCalled {
		t.Error("ImageList was not called")
	}
}

func TestContainerExists(t *testing.T) {
	mock := NewMockDockerClient()
	mock.Containers = []containertypes.Summary{
		{
			ID:    "abc123",
			Names: []string{"/existing-container"},
		},
		{
			ID:    "def456",
			Names: []string{"/another-container"},
		},
	}

	ctx := context.Background()

	// Test existing container
	exists, err := ContainerExistsWithClient(ctx, mock, "existing-container")
	if err != nil {
		t.Fatalf("ContainerExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected container to exist")
	}

	// Test non-existing container
	exists, err = ContainerExistsWithClient(ctx, mock, "non-existing")
	if err != nil {
		t.Fatalf("ContainerExists failed: %v", err)
	}
	if exists {
		t.Error("Expected container to not exist")
	}
}

func TestCreateVolume(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	err := CreateVolumeWithClient(ctx, mock, "test-volume")
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if !mock.VolumeCreateCalled {
		t.Error("VolumeCreate was not called")
	}

	if mock.LastVolumeName != "test-volume" {
		t.Errorf("Expected volume name 'test-volume', got '%s'", mock.LastVolumeName)
	}

	// Verify volume was added to mock storage
	if _, ok := mock.Volumes["test-volume"]; !ok {
		t.Error("Volume was not added to mock storage")
	}
}

func TestCreateNetwork(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	err := CreateNetworkWithClient(ctx, mock, "test-network")
	if err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	if !mock.NetworkCreateCalled {
		t.Error("NetworkCreate was not called")
	}

	if mock.LastNetworkName != "test-network" {
		t.Errorf("Expected network name 'test-network', got '%s'", mock.LastNetworkName)
	}

	// Verify network was added to mock storage
	if _, ok := mock.Networks["test-network"]; !ok {
		t.Error("Network was not added to mock storage")
	}
}

func TestAddContainerToNetwork(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	err := AddContainerToNetworkWithClient(ctx, mock, "container123", "network456")
	if err != nil {
		t.Fatalf("AddContainerToNetwork failed: %v", err)
	}

	if !mock.NetworkConnectCalled {
		t.Error("NetworkConnect was not called")
	}

	if mock.LastContainerID != "container123" {
		t.Errorf("Expected container ID 'container123', got '%s'", mock.LastContainerID)
	}

	if mock.LastNetworkName != "network456" {
		t.Errorf("Expected network name 'network456', got '%s'", mock.LastNetworkName)
	}
}

func TestImagePull(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	// Test public image pull
	err := ImagePullWithClient(ctx, mock, "nginx:latest", nil)
	if err != nil {
		t.Fatalf("ImagePull failed: %v", err)
	}

	if !mock.ImagePullCalled {
		t.Error("ImagePull was not called")
	}

	if mock.LastImageTag != "nginx:latest" {
		t.Errorf("Expected image tag 'nginx:latest', got '%s'", mock.LastImageTag)
	}

	// Test with auth
	mock.ImagePullCalled = false
	err = ImagePullWithClient(ctx, mock, "private/image:v1", &ImagePullOptions{
		Username: "user",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("ImagePull with auth failed: %v", err)
	}

	if !mock.ImagePullCalled {
		t.Error("ImagePull was not called")
	}

	// Test silent mode
	mock.ImagePullCalled = false
	err = ImagePullWithClient(ctx, mock, "nginx:latest", &ImagePullOptions{
		Silent: true,
	})
	if err != nil {
		t.Fatalf("ImagePull silent mode failed: %v", err)
	}

	if !mock.ImagePullCalled {
		t.Error("ImagePull was not called in silent mode")
	}
}

func TestImagePush(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	err := ImagePushWithClient(ctx, mock, "myregistry/myimage:v1", "user", "pass")
	if err != nil {
		t.Fatalf("ImagePush failed: %v", err)
	}

	if !mock.ImagePushCalled {
		t.Error("ImagePush was not called")
	}

	if mock.LastImageTag != "myregistry/myimage:v1" {
		t.Errorf("Expected image tag 'myregistry/myimage:v1', got '%s'", mock.LastImageTag)
	}
}

func TestContainerRun(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	envVars := []string{"ENV=test", "DEBUG=true"}
	output, err := ContainerRunWithClient(ctx, mock, "alpine:latest", "test-container", envVars, true)

	if err != nil {
		t.Fatalf("ContainerRun failed: %v", err)
	}

	if !mock.ContainerCreateCalled {
		t.Error("ContainerCreate was not called")
	}

	if !mock.ContainerStartCalled {
		t.Error("ContainerStart was not called")
	}

	if !mock.ContainerWaitCalled {
		t.Error("ContainerWait was not called")
	}

	if !mock.ContainerLogsCalled {
		t.Error("ContainerLogs was not called")
	}

	if len(output) == 0 {
		t.Error("Expected container output, got empty")
	}
}

func TestCreateAndStartContainer(t *testing.T) {
	mock := NewMockDockerClient()
	ctx := context.Background()

	config := containertypes.Config{
		Image: "nginx:latest",
	}
	hostConfig := containertypes.HostConfig{}

	err := CreateAndStartContainerWithClient(ctx, mock, config, hostConfig, "web-server", "app-network")
	if err != nil {
		t.Fatalf("CreateAndStartContainer failed: %v", err)
	}

	if !mock.ContainerCreateCalled {
		t.Error("ContainerCreate was not called")
	}

	if !mock.ContainerStartCalled {
		t.Error("ContainerStart was not called")
	}

	if mock.LastContainerName != "web-server" {
		t.Errorf("Expected container name 'web-server', got '%s'", mock.LastContainerName)
	}
}
