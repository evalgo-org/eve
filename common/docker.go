package common

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	// "github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
	homedir "github.com/mitchellh/go-homedir"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type ContainerView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Host   string `json:"host"`
}

type CopyToVolumeOptions struct {
	Ctx        context.Context
	Client     *client.Client
	Image      string
	Volume     string
	LocalPath  string
	VolumePath string
}

func RegistryAuth(username string, password string) string {
	authConfig := registry.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(encodedJSON)
}

func CtxCli(socket string) (context.Context, *client.Client) {
	ctx := context.Background()
	defaultHeaders := map[string]string{"Content-Type": "application/tar"}
	cli, err := client.NewClient(socket, "v1.49", nil, defaultHeaders)
	// cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return ctx, cli
}

func Containers(ctx context.Context, cli *client.Client) []containertypes.Summary {
	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		panic(err)
	}
	return containers
}

func Containers_stop_all(ctx context.Context, cli *client.Client) {
	containers := Containers(ctx, cli)
	for _, container := range containers {
		Logger.Info("Stopping container ", container.ID[:10], "... ")
		noWaitTimeout := 0 // to not wait for the container to exit gracefully
		if err := cli.ContainerStop(ctx, container.ID, containertypes.StopOptions{Timeout: &noWaitTimeout}); err != nil {
			panic(err)
		}
		Logger.Info("Success")
	}
}

func ContainersList(ctx context.Context, cli *client.Client) []ContainerView {
	containers := Containers(ctx, cli)
	nContainers := make([]ContainerView, len(containers))
	localHost, _ := os.Hostname()
	for _, container := range containers {
		nContainers = append(nContainers, ContainerView{
			ID:     container.ID,
			Name:   container.Names[0],
			Status: container.Status,
			Host:   localHost,
		})
		// Logger.Info(container.ID, container.Names[0])
	}
	return nContainers
}

func ContainersListToJSON(ctx context.Context, cli *client.Client) string {
	cList := ContainersList(ctx, cli)
	containersJson, err := json.Marshal(cList)
	if err != nil {
		Logger.Error(err)
	}
	err = ioutil.WriteFile("containers.json", containersJson, 0644)
	if err != nil {
		Logger.Error(err)
	}
	return "containers.json"
}

func Images(ctx context.Context, cli *client.Client) []image.Summary {
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		panic(err)
	}
	return images
}

func ImagesList(ctx context.Context, cli *client.Client) {
	images := Images(ctx, cli)
	for _, image := range images {
		Logger.Info(image.ID, image.RepoTags)
	}
}

func ImagePull(ctx context.Context, cli *client.Client, image string, po image.PullOptions) {
	reader, err := cli.ImagePull(ctx, image, po)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		Logger.Fatal(err)
	}
}

func ImagePullUpstream(ctx context.Context, cli *client.Client, imageTag string) {
	reader, err := cli.ImagePull(ctx, imageTag, image.PullOptions{})
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		Logger.Fatal(err)
	}
}

func ImageAuthPull(ctx context.Context, cli *client.Client, imageTag, user, pass string) {
	reader, err := cli.ImagePull(ctx, imageTag, image.PullOptions{RegistryAuth: RegistryAuth(user, pass)})
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		Logger.Fatal(err)
	}
}

func ImageBuild(ctx context.Context, cli *client.Client, workDir string, dockerFile string, tag string) {
	dockerBuildContext, err := os.Open("test.tar")
	defer dockerBuildContext.Close()
	opt := types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerFile,
	}
	filePath, _ := homedir.Expand(workDir)
	buildCtx, _ := archive.TarWithOptions(filePath, &archive.TarOptions{})
	x, err := cli.ImageBuild(context.Background(), buildCtx, opt)
	if err != nil {
		Logger.Fatal(err)
	}
	io.Copy(os.Stdout, x.Body)
	defer x.Body.Close()
}

func ImagePush(ctx context.Context, cli *client.Client, tag, user, pass string) {
	resp, err := cli.ImagePush(ctx, tag, image.PushOptions{
		RegistryAuth: RegistryAuth(user, pass),
	})
	if err != nil {
		Logger.Fatal(err)
	}
	defer resp.Close()
	_, err = io.Copy(os.Stdout, resp)
	if err != nil {
		Logger.Fatal(err)
	}
	Logger.Info("\nImage push complete.")
}

func parseEnvFile(filePath string) ([]string, error) {
	var envs []string
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		envs = append(envs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return envs, nil
}

func ContainerRun(ctx context.Context, cli *client.Client, imageTag, cName string, envVars []string, remove bool) ([]byte, error) {
	resp, err := cli.ContainerCreate(
		ctx,
		&containertypes.Config{
			Image:        imageTag,
			Env:          envVars,
			AttachStdout: true,
			AttachStderr: true},
		&containertypes.HostConfig{AutoRemove: remove},
		&networktypes.NetworkingConfig{},
		&ocispec.Platform{},
		cName)
	if err != nil {
		Logger.Info("error ", err)
		return nil, err
	}
	Logger.Info(resp.ID)
	err = cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{})
	if err != nil {
		Logger.Info("error ", err)
		return nil, err
	}
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, containertypes.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case <-statusCh:
	}
	out, err := cli.ContainerLogs(ctx, resp.ID, containertypes.LogsOptions{ShowStdout: true})
	if err != nil {
		return nil, err
	}
	// stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	output, err := io.ReadAll(out)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func ContainerRunFromEnv(ctx context.Context, cli *client.Client, environment, imageTag, cName string, command []string) error {
	Logger.Info(environment)
	// Load env vars from file
	envVars, err := parseEnvFile(environment + ".env")
	if err != nil {
		return err
	}
	// Create container with env vars
	resp, err := cli.ContainerCreate(ctx, &containertypes.Config{
		Image: imageTag,
		Cmd:   command,
		Env:   envVars,
	}, nil, nil, nil, cName)
	if err != nil {
		return err
	}
	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return err
	}
	Logger.Info("Container started with environment from file.")
	return nil
}

func addFileToTar(tw *tar.Writer, filePath, baseDir, targetBasePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		return err
	}

	// Prepend custom target base path
	header.Name = filepath.Join(targetBasePath, relPath)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

func createTarArchive(srcPath string, destFileName string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		err = filepath.Walk(srcPath, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			return addFileToTar(tw, path, srcPath, destFileName)
		})
	} else {
		err = addFileToTar(tw, srcPath, filepath.Dir(srcPath), destFileName)
	}

	if err != nil {
		return nil, err
	}
	return buf, nil
}

func CopyToContainer(ctx context.Context, cli *client.Client, containerID, hostFilePath, containerDestPath string) error {
	tarBuffer, err := createTarArchive(hostFilePath, hostFilePath)
	if err != nil {
		return err
	}
	return cli.CopyToContainer(ctx, containerID, containerDestPath, tarBuffer, containertypes.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                true,
	})
}

func CopyRenameToContainer(ctx context.Context, cli *client.Client, containerID, hostFilePath, containerFilePath, containerDestPath string) {
	tarBuffer, err := createTarArchive(hostFilePath, containerFilePath)
	if err != nil {
		panic(err)
	}
	err = cli.CopyToContainer(ctx, containerID, containerDestPath, tarBuffer, containertypes.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                true,
	})
	if err != nil {
		Logger.Fatal(err)
	}
}

func AddContainerToNetwork(ctx context.Context, cli *client.Client, containerID, networkName string) error {
	return cli.NetworkConnect(ctx, networkName, containerID, &networktypes.EndpointSettings{})
}

func ContainerExists(ctx context.Context, cli *client.Client, name string) bool {
	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{
		All: true,
	})
	if err != nil {
		Logger.Fatal("Error listing containers:", err)
	}
	for _, container := range containers {
		for _, n := range container.Names {
			if n == "/"+name {
				return true
			}
		}
	}
	return false
}

func CreateAndStartContainer(ctx context.Context, cli *client.Client, config containertypes.Config, hostConfig containertypes.HostConfig, name, networkName string) error {
	resp, err := cli.ContainerCreate(ctx, &config, &hostConfig, &networktypes.NetworkingConfig{
		EndpointsConfig: map[string]*networktypes.EndpointSettings{networkName: {}},
	}, nil, name)
	if err != nil {
		return err
	}
	if err = cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return err
	}
	return nil
}

func CopyToVolume(copyInfo CopyToVolumeOptions) error {
	// create container with copyInfo.Volume mounted from copyInfo.Image
	// pull image
	ImagePull(copyInfo.Ctx, copyInfo.Client, copyInfo.Image, image.PullOptions{})
	// create volume if it does not exist
	copyInfo.Client.VolumeCreate(copyInfo.Ctx, volume.CreateOptions{
		Name: copyInfo.Volume,
	})
	// create container
	resp, err := copyInfo.Client.ContainerCreate(copyInfo.Ctx, &containertypes.Config{
		Image: copyInfo.Image,
		Cmd:   []string{"bash", "-c", "apt update -y && apt install -y rsync && rsync -Pavz /src/ /tgt/"},
	}, &containertypes.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: copyInfo.LocalPath, Target: "/src"},
			{Type: mount.TypeVolume, Source: copyInfo.Volume, Target: "/tgt"},
		},
	}, &network.NetworkingConfig{}, nil, "tmp--"+uuid.New().String())
	if err != nil {
		return err
	}
	// start container
	if err := copyInfo.Client.ContainerStart(copyInfo.Ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return err
	}
	err = CopyToContainer(copyInfo.Ctx, copyInfo.Client, resp.ID, copyInfo.LocalPath, copyInfo.VolumePath)
	if err != nil {
		return err
	}
	statusCh, errCh := copyInfo.Client.ContainerWait(copyInfo.Ctx, resp.ID, containertypes.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
		return nil
	}
	return nil
}
