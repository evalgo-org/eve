// Package common provides comprehensive Docker container and image management utilities.
// This package implements a high-level abstraction over the Docker API, offering
// functions for container lifecycle management, image operations, network management,
// and file transfer capabilities.
//
// The package serves as a foundation for containerized application deployment,
// development workflows, and infrastructure automation. It provides both simple
// convenience functions and advanced operations for complex Docker workflows.
//
// Core Functionality:
//   - Container lifecycle management (create, start, stop, remove)
//   - Image operations (pull, build, push, list)
//   - File transfer between host and containers
//   - Network management and container connectivity
//   - Volume operations and data persistence
//   - Registry authentication and private image access
//
// Docker API Integration:
//
//	Uses the official Docker Go SDK to communicate with Docker Engine via
//	the Docker socket. Supports both local Docker instances and remote
//	Docker hosts through configurable socket connections.
//
// Authentication Support:
//
//	Provides registry authentication for private Docker registries including
//	Docker Hub, Azure Container Registry, AWS ECR, and custom registries.
//
// Error Handling Philosophy:
//
//	The package uses a mixed approach to error handling - some functions
//	use panic for unrecoverable errors while others return errors for
//	graceful handling. This design supports both scripting scenarios
//	(where failures should halt execution) and application scenarios
//	(where errors should be handled gracefully).
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
	"github.com/google/uuid"
	homedir "github.com/mitchellh/go-homedir"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerView represents a simplified view of a Docker container for API responses.
// This structure provides essential container information in a format suitable
// for JSON serialization and client consumption.
//
// Fields:
//   - ID: Container's unique identifier (full Docker ID)
//   - Name: Human-readable container name
//   - Status: Current container status (running, stopped, etc.)
//   - Host: Hostname of the Docker host running the container
//
// Use Cases:
//   - REST API responses for container listing
//   - Dashboard and monitoring interfaces
//   - Container inventory and tracking systems
//   - Multi-host container management
type ContainerView struct {
	ID     string `json:"id"`     // Docker container ID
	Name   string `json:"name"`   // Container name
	Status string `json:"status"` // Container status
	Host   string `json:"host"`   // Docker host identifier
}

// CopyToVolumeOptions configures file copy operations from host to Docker volumes.
// This structure encapsulates all parameters needed for complex file transfer
// operations that involve temporary containers and volume mounting.
//
// The operation creates a temporary container with both source (host) and
// target (volume) mounted, then performs file synchronization using rsync.
//
// Fields:
//   - Ctx: Context for operation cancellation and timeout control
//   - Client: Docker API client for container operations
//   - Image: Base image for temporary container (should include rsync)
//   - Volume: Target Docker volume name
//   - LocalPath: Source path on the host filesystem
//   - VolumePath: Destination path within the volume
//
// Workflow:
//  1. Pull the specified base image
//  2. Create Docker volume if it doesn't exist
//  3. Create temporary container with host and volume mounts
//  4. Execute rsync to copy files from host to volume
//  5. Clean up temporary container (auto-remove enabled)
type CopyToVolumeOptions struct {
	Ctx        context.Context // Operation context
	Client     *client.Client  // Docker API client
	Image      string          // Base image for temporary container
	Volume     string          // Target volume name
	LocalPath  string          // Source path on host
	VolumePath string          // Destination path in volume
}

// RegistryAuth creates a base64-encoded authentication string for Docker registry operations.
// This function formats authentication credentials in the format expected by the
// Docker API for registry authentication during image pull and push operations.
//
// Authentication Flow:
//
//	The Docker API requires authentication credentials to be base64-encoded JSON
//	containing username and password. This function handles the encoding process
//	and returns the properly formatted authentication string.
//
// Supported Registries:
//   - Docker Hub (docker.io)
//   - Private Docker registries
//   - Cloud provider registries (Azure ACR, AWS ECR, Google GCR)
//   - Self-hosted registries (Harbor, Nexus, etc.)
//
// Parameters:
//   - username: Registry username or service account
//   - password: Registry password, token, or service account key
//
// Returns:
//   - string: Base64-encoded authentication string for Docker API
//
// Security Considerations:
//   - Credentials are temporarily held in memory during encoding
//   - The returned string contains sensitive authentication data
//   - Use secure credential storage (environment variables, secrets management)
//   - Avoid logging or persisting the authentication string
//
// Example Usage:
//
//	auth := RegistryAuth("myuser", "mypassword")
//	// Use auth string in ImagePull or ImagePush operations
//
// Error Handling:
//
//	Panics on JSON marshaling failure (should never occur with simple credentials)
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

// CtxCli creates a Docker client context and API client for Docker operations.
// This function establishes the connection to Docker Engine and configures
// the client with appropriate headers and API version settings.
//
// Docker Socket Configuration:
//
//	Supports various Docker socket configurations including:
//	- Local Unix socket: "unix:///var/run/docker.sock"
//	- Remote TCP socket: "tcp://remote-host:2376"
//	- Named pipes on Windows: "npipe:////./pipe/docker_engine"
//
// API Version:
//
//	Fixed to API version 1.49 for consistent behavior across Docker versions.
//	This version provides compatibility with recent Docker Engine releases
//	while maintaining stability for containerized applications.
//
// Headers Configuration:
//
//	Sets default Content-Type header for tar-based operations, which are
//	commonly used for file transfers and context uploads.
//
// Parameters:
//   - socket: Docker socket URI (local or remote)
//
// Returns:
//   - context.Context: Background context for Docker operations
//   - *client.Client: Configured Docker API client
//
// Error Handling:
//
//	Panics on client creation failure, ensuring immediate feedback for
//	connection issues or invalid socket configurations.
//
// Example Usage:
//
//	ctx, cli := CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
// Connection Patterns:
//   - Local development: Use default Unix socket
//   - Remote Docker: Use TCP with TLS configuration
//   - Docker Desktop: Use platform-specific defaults
//   - CI/CD environments: Use environment-provided socket
func CtxCli(socket string) (context.Context, *client.Client) {
	ctx := context.Background()
	defaultHeaders := map[string]string{"Content-Type": "application/tar"}
	cli, err := client.NewClient(socket, "v1.49", nil, defaultHeaders)
	// Alternative: cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return ctx, cli
}

// Containers retrieves a list of all running containers from Docker Engine.
// This function provides access to the raw Docker API container listing
// with all available container metadata and status information.
//
// Container Information:
//
//	Returns complete container summaries including:
//	- Container IDs and names
//	- Current status and state
//	- Image information and tags
//	- Network settings and port mappings
//	- Resource usage and limits
//	- Labels and metadata
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//
// Returns:
//   - []containertypes.Summary: Array of container summary objects
//
// Error Handling:
//
//	Panics on API communication failure, ensuring immediate notification
//	of Docker connectivity issues or permission problems.
//
// Use Cases:
//   - Container monitoring and status checking
//   - Resource utilization analysis
//   - Container discovery for orchestration
//   - Health checking and alerting systems
//
// Performance Notes:
//   - Efficient API call that retrieves all containers in single request
//   - Consider filtering for large container environments
//   - Results include only running containers by default
func Containers(ctx context.Context, cli *client.Client) []containertypes.Summary {
	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		panic(err)
	}
	return containers
}

// Containers_stop_all forcefully stops all running containers on the Docker host.
// This function performs a bulk stop operation with immediate termination,
// useful for cleanup operations, testing scenarios, and emergency shutdowns.
//
// Stopping Behavior:
//   - Uses zero timeout for immediate termination (no graceful shutdown)
//   - Processes all containers returned by Containers() function
//   - Stops containers sequentially with progress logging
//   - Does not remove containers after stopping
//
// Use Cases:
//   - Development environment cleanup
//   - Testing infrastructure reset
//   - Emergency shutdown procedures
//   - Bulk container management operations
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//
// Error Handling:
//
//	Panics on container stop failure, ensuring immediate notification
//	of operational issues that prevent container shutdown.
//
// Safety Considerations:
//   - Forceful stop may cause data loss in containers without proper shutdown
//   - Consider graceful shutdown periods for production environments
//   - Backup critical data before bulk stop operations
//   - Verify container dependencies before stopping
//
// Logging:
//
//	Provides progress feedback during stop operations, logging container
//	IDs and success status for monitoring and debugging purposes.
//
// Alternative Approaches:
//
//	For production environments, consider implementing graceful shutdown
//	with configurable timeout periods and dependency-aware stop ordering.
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

// ContainersList converts raw container data into a simplified view format.
// This function transforms Docker's detailed container summaries into a
// more consumable format suitable for APIs, dashboards, and client applications.
//
// Data Transformation:
//   - Extracts essential container information
//   - Adds host identification for multi-host environments
//   - Simplifies complex Docker metadata into basic fields
//   - Provides consistent JSON serialization format
//
// Host Information:
//
//	Automatically detects and includes the local hostname, enabling
//	container tracking across distributed Docker environments and
//	multi-host orchestration scenarios.
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//
// Returns:
//   - []ContainerView: Array of simplified container view objects
//
// Data Structure:
//
//	Each ContainerView includes:
//	- ID: Full Docker container identifier
//	- Name: Primary container name (first from names array)
//	- Status: Current container status string
//	- Host: Local hostname for multi-host tracking
//
// Use Cases:
//   - REST API responses for container management interfaces
//   - Dashboard data feeds for monitoring systems
//   - Container inventory for orchestration platforms
//   - Simplified container tracking across environments
//
// Implementation Notes:
//   - Uses the first name from container.Names array
//   - Hostname detection uses os.Hostname() for local identification
//   - Results are suitable for JSON marshaling and API responses
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

// ContainersListToJSON exports container information to a JSON file.
// This function combines container listing and JSON serialization,
// providing a convenient way to export container inventory for
// external processing, backup, or integration with other systems.
//
// Output Format:
//
//	Creates a JSON file containing an array of ContainerView objects
//	with standardized formatting suitable for consumption by external
//	tools, monitoring systems, or configuration management platforms.
//
// File Operations:
//   - Generates "containers.json" in the current working directory
//   - Uses standard JSON marshaling for consistent formatting
//   - Sets file permissions to 0644 (readable by owner and group)
//   - Overwrites existing files without confirmation
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//
// Returns:
//   - string: Filename of the created JSON file ("containers.json")
//
// Error Handling:
//   - JSON marshaling errors are logged via Logger.Error
//   - File writing errors are logged via Logger.Error
//   - Function continues execution despite errors (logs but doesn't panic)
//
// Use Cases:
//   - Container inventory export for compliance and auditing
//   - Integration with external monitoring and alerting systems
//   - Backup of container configuration for disaster recovery
//   - Data feed for business intelligence and reporting tools
//
// Integration Patterns:
//   - CI/CD pipeline integration for deployment tracking
//   - Monitoring system data ingestion
//   - Configuration management system synchronization
//   - Compliance reporting and audit trail generation
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

// Images retrieves a list of all Docker images available on the local Docker host.
// This function provides access to the complete image inventory including
// both tagged and untagged images with comprehensive metadata.
//
// Image Information:
//
//	Returns detailed image summaries containing:
//	- Image IDs and repository tags
//	- Image size and virtual size information
//	- Creation timestamps and metadata
//	- Parent image relationships
//	- Labels and configuration data
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//
// Returns:
//   - []image.Summary: Array of image summary objects from Docker API
//
// Error Handling:
//
//	Panics on API communication failure, providing immediate feedback
//	for Docker connectivity issues or permission problems.
//
// Use Cases:
//   - Image inventory and disk usage analysis
//   - Security scanning and vulnerability assessment
//   - Image lifecycle management and cleanup
//   - Build system integration and optimization
//
// Performance Considerations:
//   - Single API call retrieves all local images efficiently
//   - Large image inventories may require pagination in future versions
//   - Consider filtering options for specific use cases
func Images(ctx context.Context, cli *client.Client) []image.Summary {
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		panic(err)
	}
	return images
}

// ImagesList displays information about all Docker images on the local host.
// This function provides a simple way to inspect the local image inventory
// with logging output suitable for debugging and administrative tasks.
//
// Display Format:
//
//	Logs each image with:
//	- Image ID for unique identification
//	- Repository tags for version tracking
//	- Human-readable output via Logger.Info
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//
// Output Example:
//
//	sha256:abc123... [nginx:latest nginx:1.21]
//	sha256:def456... [alpine:3.14]
//	sha256:ghi789... [<none>:<none>]
//
// Use Cases:
//   - Quick image inventory inspection
//   - Debugging image availability issues
//   - Administrative image management tasks
//   - Development environment verification
func ImagesList(ctx context.Context, cli *client.Client) {
	images := Images(ctx, cli)
	for _, image := range images {
		Logger.Info(image.ID, image.RepoTags)
	}
}

// ImagePull downloads a Docker image from a registry with custom pull options.
// This function provides full control over the image pull process including
// authentication, platform selection, and progress monitoring.
//
// Pull Process:
//   - Initiates image download from specified registry
//   - Streams pull progress to stdout for real-time feedback
//   - Supports custom pull options for advanced scenarios
//   - Handles multi-architecture and platform-specific images
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - image: Image reference (repository:tag or repository@digest)
//   - po: Pull options including authentication and platform settings
//
// Pull Options Support:
//   - Registry authentication for private repositories
//   - Platform specification for multi-architecture images
//   - All-tags option for downloading all image tags
//   - Custom registry configuration
//
// Error Handling:
//   - Panics on pull initiation failure (network, authentication)
//   - Fatal logging on stream copy failure (I/O issues)
//
// Progress Monitoring:
//
//	Streams pull progress directly to stdout, providing real-time
//	feedback about download status, layer extraction, and completion.
//
// Example Usage:
//
//	opts := image.PullOptions{
//	    RegistryAuth: RegistryAuth("user", "pass"),
//	}
//	ImagePull(ctx, cli, "myregistry.com/myimage:latest", opts)
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

// ImagePullUpstream downloads a Docker image from public registries without authentication.
// This function provides a simplified interface for pulling publicly available
// images from Docker Hub and other public registries.
//
// Public Registry Support:
//   - Docker Hub (default registry for unqualified names)
//   - Public registries with anonymous access
//   - Official images and community contributions
//   - Multi-architecture image support
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - imageTag: Image reference (repository:tag format)
//
// Image Reference Formats:
//   - Short names: "nginx:latest" (resolves to docker.io/library/nginx:latest)
//   - Fully qualified: "docker.io/library/nginx:latest"
//   - Alternative registries: "quay.io/prometheus/prometheus:latest"
//
// Error Handling:
//   - Panics on pull initiation failure
//   - Fatal logging on progress stream failure
//
// Use Cases:
//   - Development environment setup
//   - CI/CD pipeline image preparation
//   - Base image updates and maintenance
//   - Quick testing and experimentation
//
// Performance Notes:
//   - Leverages Docker's layer caching for efficiency
//   - Progress feedback helps monitor large image downloads
//   - Network interruptions may require retry logic
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

// ImageAuthPull downloads a Docker image from a private registry with authentication.
// This function combines registry authentication with image pulling, providing
// a convenient interface for accessing private repositories.
//
// Authentication Process:
//   - Creates registry authentication from provided credentials
//   - Supports username/password and token-based authentication
//   - Compatible with Docker Hub private repositories and enterprise registries
//   - Handles authentication encoding automatically
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - imageTag: Image reference including registry and tag
//   - user: Registry username or service account
//   - pass: Registry password, token, or access key
//
// Registry Compatibility:
//   - Docker Hub private repositories
//   - Azure Container Registry (ACR)
//   - Amazon Elastic Container Registry (ECR)
//   - Google Container Registry (GCR)
//   - Self-hosted registries (Harbor, Nexus, etc.)
//
// Security Considerations:
//   - Credentials are passed as parameters (consider secure injection)
//   - Authentication data is temporarily held in memory
//   - Use environment variables or secret management for production
//   - Avoid hardcoding credentials in source code
//
// Error Handling:
//   - Panics on authentication or pull failure
//   - Fatal logging on progress stream issues
//
// Example Usage:
//
//	ImageAuthPull(ctx, cli, "myregistry.com/private/image:v1.0", "username", "password")
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

// ImageBuild constructs a Docker image from a Dockerfile and build context.
// This function provides programmatic access to Docker's image building
// capabilities with support for custom build contexts and configurations.
//
// Build Process:
//   - Creates tar archive from specified working directory
//   - Uploads build context to Docker Engine
//   - Executes Dockerfile instructions in isolated environment
//   - Streams build output for progress monitoring
//   - Tags resulting image with specified name
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - workDir: Directory containing Dockerfile and build context
//   - dockerFile: Dockerfile name (relative to workDir)
//   - tag: Tag name for the resulting image
//
// Build Context:
//   - All files in workDir are included in build context
//   - Large directories may slow build process
//   - Use .dockerignore to exclude unnecessary files
//   - Context is compressed and uploaded to Docker Engine
//
// Dockerfile Processing:
//   - Supports all standard Dockerfile instructions
//   - Build cache utilization for layer reuse
//   - Multi-stage builds for optimized images
//   - Build arguments and environment variables
//
// Error Handling:
//   - Fatal logging on build failure or I/O issues
//   - Build errors are streamed to stdout for debugging
//
// Output Streaming:
//
//	Build progress, including layer creation and instruction execution,
//	is streamed to stdout for real-time monitoring and debugging.
//
// Optimization Notes:
//   - Use .dockerignore to minimize build context size
//   - Order Dockerfile instructions for optimal cache utilization
//   - Consider multi-stage builds for production images
//
// Example Usage:
//
//	ImageBuild(ctx, cli, "/path/to/project", "Dockerfile", "myapp:latest")
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

// ImagePush uploads a Docker image to a registry with authentication.
// This function handles the complete image push process including
// authentication, upload progress monitoring, and completion confirmation.
//
// Push Process:
//   - Authenticates with target registry using provided credentials
//   - Uploads image layers and manifest to registry
//   - Streams upload progress for monitoring
//   - Confirms successful completion
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - tag: Image tag to push (must include registry if not Docker Hub)
//   - user: Registry username or service account
//   - pass: Registry password, token, or access key
//
// Registry Requirements:
//   - Image must be tagged with registry prefix for non-Docker Hub registries
//   - Example: "myregistry.com/namespace/image:tag"
//   - Registry must support Docker Registry API v2
//
// Authentication Support:
//   - Basic authentication (username/password)
//   - Token-based authentication for cloud registries
//   - Service account authentication for enterprise environments
//
// Upload Process:
//   - Only uploads layers not already present in registry (deduplication)
//   - Progress feedback shows upload status for each layer
//   - Manifest upload finalizes the image push
//
// Error Handling:
//   - Fatal logging on push initiation or authentication failure
//   - I/O errors during upload are reported via fatal logging
//
// Progress Monitoring:
//
//	Upload progress is streamed to stdout, showing layer upload
//	status and completion percentages for monitoring purposes.
//
// Example Usage:
//
//	ImagePush(ctx, cli, "myregistry.com/myapp:v1.0", "username", "password")
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

// parseEnvFile reads environment variables from a file and returns them as a slice.
// This utility function supports Docker's environment file format, enabling
// configuration management through external files.
//
// File Format:
//   - One environment variable per line
//   - Format: KEY=value or KEY="value with spaces"
//   - Comments start with # and are ignored
//   - Empty lines are ignored
//   - No variable substitution or expansion
//
// Parsing Rules:
//   - Lines are trimmed of leading/trailing whitespace
//   - Comment lines (starting with #) are skipped
//   - Empty lines are ignored
//   - Variable format follows Docker environment conventions
//
// Parameters:
//   - filePath: Path to environment file
//
// Returns:
//   - []string: Array of environment variable strings (KEY=value format)
//   - error: File reading or parsing errors
//
// Error Conditions:
//   - File not found or permission denied
//   - I/O errors during file reading
//   - Scanner errors (rare, usually I/O related)
//
// Example File Content:
//
//	# Database configuration
//	DB_HOST=localhost
//	DB_PORT=5432
//	DB_NAME=myapp
//
//	# Application settings
//	APP_ENV=production
//	DEBUG=false
//
// Use Cases:
//   - Container environment configuration
//   - Development/production environment separation
//   - Secret injection through external files
//   - Configuration management in deployment pipelines
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

// ContainerRun creates and runs a Docker container with specified configuration.
// This function provides a high-level interface for container execution with
// automatic lifecycle management and output capture.
//
// Container Lifecycle:
//  1. Creates container with specified image and environment
//  2. Starts container execution
//  3. Waits for container completion
//  4. Captures and returns container output
//  5. Optionally removes container after execution
//
// Configuration Options:
//   - Image: Docker image to run
//   - Environment variables: Runtime configuration
//   - Auto-removal: Cleanup after execution
//   - Output capture: Stdout/stderr collection
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - imageTag: Docker image to run
//   - cName: Container name for identification
//   - envVars: Environment variables (KEY=value format)
//   - remove: Whether to auto-remove container after execution
//
// Returns:
//   - []byte: Combined stdout/stderr output from container
//   - error: Container creation, execution, or output capture errors
//
// Execution Model:
//   - Synchronous execution (waits for container completion)
//   - Output is captured and returned as byte array
//   - Container state is monitored until completion
//
// Error Handling:
//
//	Returns errors for graceful handling in calling code:
//	- Container creation failures
//	- Execution errors or timeouts
//	- Output capture issues
//
// Use Cases:
//   - Batch processing and automation tasks
//   - Testing and validation workflows
//   - Data processing and transformation
//   - Build and deployment pipeline steps
//
// Example Usage:
//
//	envs := []string{"ENV=production", "DEBUG=false"}
//	output, err := ContainerRun(ctx, cli, "myapp:latest", "task-1", envs, true)
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

// ContainerRunFromEnv creates and starts a container with environment loaded from file.
// This function combines environment file parsing with container creation,
// providing a convenient way to run containers with externalized configuration.
//
// Configuration Workflow:
//  1. Loads environment variables from specified file
//  2. Creates container with loaded environment
//  3. Starts container execution
//  4. Returns immediately (asynchronous execution)
//
// Environment File Integration:
//   - Uses parseEnvFile() to load configuration
//   - Supports standard Docker environment file format
//   - Handles comments and empty lines gracefully
//   - Provides error feedback for file issues
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - environment: Base name for environment file (appends .env)
//   - imageTag: Docker image to run
//   - cName: Container name for identification
//   - command: Command to execute in container (can be empty)
//
// Returns:
//   - error: Environment loading, container creation, or start errors
//
// File Naming Convention:
//
//	Environment file is constructed by appending ".env" to the environment
//	parameter. For example, "production" becomes "production.env".
//
// Execution Model:
//   - Asynchronous execution (does not wait for completion)
//   - Container runs independently after start
//   - No output capture or monitoring
//
// Error Handling:
//
//	Returns errors for:
//	- Environment file reading/parsing issues
//	- Container creation failures
//	- Container start failures
//
// Use Cases:
//   - Environment-specific container deployment
//   - Configuration management with external files
//   - Development/staging/production environment separation
//   - Automated deployment with externalized configuration
//
// Example Usage:
//
//	cmd := []string{"/app/start.sh", "--config", "/etc/app.conf"}
//	err := ContainerRunFromEnv(ctx, cli, "production", "myapp:latest", "web-server", cmd)
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

// addFileToTar adds a single file to a tar archive with custom path mapping.
// This utility function supports the file copy operations by building
// tar archives with proper header information and path handling.
//
// Path Transformation:
//   - Calculates relative path from baseDir to filePath
//   - Prepends targetBasePath to create final archive path
//   - Preserves file permissions and metadata in tar header
//
// Parameters:
//   - tw: Tar writer instance for adding files
//   - filePath: Absolute path to source file
//   - baseDir: Base directory for relative path calculation
//   - targetBasePath: Prefix for paths within the archive
//
// Returns:
//   - error: File access, header creation, or writing errors
//
// Header Processing:
//   - Creates tar header from file info (permissions, size, etc.)
//   - Sets archive path by combining targetBasePath with relative path
//   - Preserves file metadata for accurate restoration
//
// Error Conditions:
//   - File not found or permission denied
//   - Path calculation failures
//   - Tar header creation issues
//   - I/O errors during file copying
//
// Usage Context:
//
//	Used internally by createTarArchive and container copy operations
//	to build properly structured tar archives for Docker API consumption.
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

// createTarArchive creates a tar archive from a source path with custom destination naming.
// This function builds tar archives suitable for Docker API operations,
// supporting both individual files and directory trees.
//
// Archive Creation:
//   - Handles both files and directories as source
//   - Recursively processes directory contents
//   - Creates proper tar headers with metadata
//   - Returns archive as in-memory buffer
//
// Path Handling:
//   - For directories: walks all files recursively
//   - For files: archives single file with custom naming
//   - Preserves relative directory structure in archive
//   - Uses destFileName as base path within archive
//
// Parameters:
//   - srcPath: Source file or directory path
//   - destFileName: Base path for files within the archive
//
// Returns:
//   - *bytes.Buffer: In-memory tar archive ready for use
//   - error: File access, archiving, or I/O errors
//
// Directory Processing:
//
//	When srcPath is a directory, the function walks all contained
//	files and adds them to the archive with proper relative paths.
//
// File Processing:
//
//	When srcPath is a single file, it's added to the archive with
//	the destFileName as its path within the archive.
//
// Error Handling:
//
//	Returns errors for:
//	- Source path access issues
//	- Directory walking failures
//	- Individual file processing errors
//	- Tar archive creation problems
//
// Use Cases:
//   - Container file copy operations
//   - Build context preparation
//   - Backup and archive creation
//   - Data transfer between host and containers
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

// CopyToContainer transfers files from host to a running Docker container.
// This function provides a high-level interface for copying files into
// containers using Docker's tar-based copy API.
//
// Copy Process:
//  1. Creates tar archive from host file/directory
//  2. Transfers archive to container via Docker API
//  3. Extracts archive contents at specified container path
//  4. Preserves file permissions and ownership
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - containerID: Target container ID or name
//   - hostFilePath: Source path on host filesystem
//   - containerDestPath: Destination directory in container
//
// Returns:
//   - error: Archive creation, transfer, or extraction errors
//
// Path Handling:
//   - hostFilePath can be file or directory
//   - containerDestPath should be destination directory
//   - Preserves relative directory structure for directory copies
//   - Files maintain original names within destination
//
// Copy Options:
//   - AllowOverwriteDirWithFile: false (prevents file/dir conflicts)
//   - CopyUIDGID: true (preserves ownership information)
//
// Container Requirements:
//   - Container must be created (running or stopped)
//   - Destination path must be valid container path
//   - Container filesystem must be writable
//
// Error Conditions:
//   - Source path not found or inaccessible
//   - Container not found or inaccessible
//   - Insufficient permissions in container
//   - Archive creation or transfer failures
//
// Use Cases:
//   - Configuration file deployment
//   - Application code updates
//   - Data import and processing
//   - Build artifact transfer
//
// Example Usage:
//
//	err := CopyToContainer(ctx, cli, "web-server", "/local/config.json", "/app/config/")
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

// CopyRenameToContainer transfers files from host to container with name change.
// This function extends CopyToContainer by allowing the copied file to have
// a different name in the container than on the host filesystem.
//
// Rename Functionality:
//   - Copies file from hostFilePath on host
//   - Places file at containerFilePath within container
//   - Allows complete path and name transformation
//   - Maintains file content and permissions
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - containerID: Target container ID or name
//   - hostFilePath: Source file path on host
//   - containerFilePath: Desired file path/name in container
//   - containerDestPath: Destination directory in container
//
// Archive Process:
//
//	Creates tar archive with custom internal naming, allowing the
//	file to appear with a different name within the container.
//
// Error Handling:
//   - Panics on archive creation failure (immediate feedback)
//   - Fatal logging on copy operation failure
//   - No graceful error return (designed for scripting scenarios)
//
// Use Cases:
//   - Configuration template deployment with environment-specific names
//   - Binary deployment with version-specific naming
//   - Data file processing with standardized naming conventions
//   - Secret injection with specific file names
//
// Example Usage:
//
//	CopyRenameToContainer(ctx, cli, "app-container",
//	                     "/host/config.prod.json", "config.json", "/app/config/")
//
// Naming Strategy:
//
//	The containerFilePath parameter specifies the exact path and name
//	the file should have within the container, regardless of its
//	original name on the host.
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

// AddContainerToNetwork connects an existing container to a Docker network.
// This function provides programmatic network management, enabling containers
// to join custom networks for service discovery and communication.
//
// Network Connection:
//   - Connects container to specified network
//   - Uses default endpoint settings
//   - Preserves existing network connections
//   - Enables network-based service discovery
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - containerID: Container ID or name to connect
//   - networkName: Target network name or ID
//
// Returns:
//   - error: Network connection or configuration errors
//
// Network Requirements:
//   - Network must exist before connection attempt
//   - Container must be created (can be running or stopped)
//   - Network driver must support container attachment
//
// Endpoint Configuration:
//
//	Uses default endpoint settings which provide:
//	- Automatic IP allocation within network subnet
//	- Default network aliases and hostname resolution
//	- Standard network interface configuration
//
// Use Cases:
//   - Microservice architecture connectivity
//   - Multi-container application networking
//   - Service mesh and load balancer integration
//   - Development environment networking
//
// Error Conditions:
//   - Network not found
//   - Container not found
//   - Container already connected to network
//   - Network driver limitations
//
// Example Usage:
//
//	err := AddContainerToNetwork(ctx, cli, "web-server", "app-network")
func AddContainerToNetwork(ctx context.Context, cli *client.Client, containerID, networkName string) error {
	return cli.NetworkConnect(ctx, networkName, containerID, &networktypes.EndpointSettings{})
}

// ContainerExists checks if a container with the specified name exists.
// This function queries the Docker API to determine container existence,
// useful for conditional container operations and duplicate prevention.
//
// Search Process:
//   - Retrieves all containers (running and stopped)
//   - Searches through container names for exact match
//   - Handles Docker's name formatting (leading slash)
//   - Returns boolean result for existence check
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - name: Container name to search for
//
// Returns:
//   - bool: true if container exists, false otherwise
//
// Name Matching:
//
//	Docker prefixes container names with "/" in the API response.
//	This function handles the prefix automatically for accurate matching.
//
// Container States:
//
//	Searches both running and stopped containers by using the All: true
//	option in ListOptions, providing comprehensive existence checking.
//
// Error Handling:
//
//	Fatal logging on API communication failure, ensuring immediate
//	notification of Docker connectivity issues.
//
// Use Cases:
//   - Preventing duplicate container creation
//   - Conditional container operations
//   - Container lifecycle management
//   - Deployment script validation
//
// Example Usage:
//
//	if ContainerExists(ctx, cli, "my-app") {
//	    Logger.Info("Container already exists")
//	} else {
//	    // Create new container
//	}
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

// CreateAndStartContainer creates a new container with network configuration and starts it.
// This function combines container creation with network attachment and startup,
// providing a convenient interface for deploying networked containers.
//
// Deployment Process:
//  1. Creates container with specified configuration
//  2. Connects container to specified network during creation
//  3. Starts container execution immediately
//  4. Returns after successful startup
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout
//   - cli: Docker API client instance
//   - config: Container configuration (image, environment, etc.)
//   - hostConfig: Host-specific settings (mounts, ports, etc.)
//   - name: Container name for identification
//   - networkName: Network to connect container to
//
// Returns:
//   - error: Container creation, network connection, or startup errors
//
// Network Integration:
//
//	The container is connected to the specified network during creation,
//	enabling immediate network communication and service discovery.
//
// Configuration Support:
//   - Accepts full containertypes.Config for maximum flexibility
//   - Supports all HostConfig options (volumes, ports, resources)
//   - Handles network configuration automatically
//
// Error Handling:
//
//	Returns errors for graceful handling:
//	- Container creation failures
//	- Network connection issues
//	- Container startup problems
//
// Use Cases:
//   - Microservice deployment with service discovery
//   - Multi-container application orchestration
//   - Development environment setup
//   - CI/CD pipeline container deployment
//
// Example Usage:
//
//	config := containertypes.Config{
//	    Image: "nginx:latest",
//	    ExposedPorts: nat.PortSet{"80/tcp": struct{}{}},
//	}
//	hostConfig := containertypes.HostConfig{
//	    PortBindings: nat.PortMap{"80/tcp": []nat.PortBinding{{HostPort: "8080"}}},
//	}
//	err := CreateAndStartContainer(ctx, cli, config, hostConfig, "web-server", "app-network")
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

// CopyToVolume copies files from host to a Docker volume using a temporary container.
// This function provides a robust method for populating Docker volumes with
// host data using rsync for reliable file synchronization.
//
// Copy Process:
//  1. Pulls specified base image (with rsync capability)
//  2. Creates target volume if it doesn't exist
//  3. Creates temporary container with host and volume mounts
//  4. Executes rsync to synchronize files from host to volume
//  5. Waits for operation completion and cleanup
//
// Parameters:
//   - copyInfo: CopyToVolumeOptions struct containing all operation parameters
//
// Returns:
//   - error: Image pull, volume creation, container operations, or sync errors
//
// Volume Operations:
//   - Automatically creates volume if it doesn't exist
//   - Preserves existing volume data (rsync merge behavior)
//   - Maintains file permissions and ownership
//   - Supports incremental updates
//
// Container Strategy:
//
//	Uses temporary container with unique UUID name for isolation:
//	- Mounts host path as read-only source (/src)
//	- Mounts target volume as destination (/tgt)
//	- Executes package updates and rsync installation
//	- Performs file synchronization with progress output
//	- Auto-removes container after completion
//
// Rsync Features:
//   - Preserves permissions and ownership (-Pavz flags)
//   - Provides progress feedback during transfer
//   - Handles large file sets efficiently
//   - Supports incremental synchronization
//
// Error Conditions:
//   - Base image pull failures
//   - Volume creation issues
//   - Container creation or execution problems
//   - Rsync execution failures
//   - Mount path accessibility issues
//
// Use Cases:
//   - Database initialization with seed data
//   - Static asset deployment to volumes
//   - Configuration file distribution
//   - Backup restoration to volumes
//   - Development environment data setup
//
// Example Usage:
//
//	opts := CopyToVolumeOptions{
//	    Ctx:        ctx,
//	    Client:     cli,
//	    Image:      "ubuntu:latest",
//	    Volume:     "app-data",
//	    LocalPath:  "/host/data",
//	    VolumePath: "/data",
//	}
//	err := CopyToVolume(opts)
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
