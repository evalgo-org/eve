package production

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"eve.evalgo.org/common"
)

// LakeFSProductionConfig holds configuration for production LakeFS deployment.
type LakeFSProductionConfig struct {
	// ContainerName is the name for the LakeFS container
	ContainerName string
	// Image is the Docker image to use (default: "treeverse/lakefs:1.70")
	Image string
	// Port is the host port to expose LakeFS HTTP API and UI (default: 8000)
	Port string
	// AccessKeyID is the initial admin access key ID
	AccessKeyID string
	// SecretAccessKey is the initial admin secret access key
	SecretAccessKey string
	// DataVolume is the volume name for LakeFS data persistence
	DataVolume string
	// Production holds common production configuration
	Production ProductionConfig
}

// DefaultLakeFSProductionConfig returns the default LakeFS production configuration.
func DefaultLakeFSProductionConfig() LakeFSProductionConfig {
	return LakeFSProductionConfig{
		ContainerName:   "lakefs",
		Image:           "treeverse/lakefs:1.70",
		Port:            "8000",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		DataVolume:      "lakefs-data",
		Production: ProductionConfig{
			NetworkName:   "app-network",
			CreateNetwork: true,
			VolumeName:    "lakefs-data",
			CreateVolume:  true,
		},
	}
}

// DeployLakeFS deploys a production-ready LakeFS container.
//
// LakeFS is an open-source data lake versioning system that provides Git-like operations
// (branch, commit, merge, revert) for data stored in object storage. This function deploys
// a LakeFS container suitable for production use with persistent data storage.
//
// Container Features:
//   - Named container for consistent identification
//   - Persistent volume for metadata and block storage
//   - Custom network connectivity
//   - Fixed port mapping for stable access
//   - Restart policy for high availability
//   - Health checks for monitoring
//   - S3-compatible API for data operations
//   - Web UI for repository management
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - config: LakeFS production configuration
//
// Returns:
//   - string: Container ID of the deployed LakeFS container
//   - error: Deployment errors
//
// Example Usage:
//
//	ctx, cli := common.CtxCli("unix:///var/run/docker.sock")
//	defer cli.Close()
//
//	config := DefaultLakeFSProductionConfig()
//	config.AccessKeyID = "my-access-key"
//	config.SecretAccessKey = "my-secret-key"  // Use secure credentials!
//
//	containerID, err := DeployLakeFS(ctx, cli, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("LakeFS deployed with ID: %s", containerID)
//	log.Printf("LakeFS UI: http://localhost:%s", config.Port)
//
// Connection URLs:
//
//	HTTP UI:
//	http://localhost:{port}
//	http://{container_name}:{port} (from other containers)
//
//	API endpoint:
//	http://localhost:{port}/api/v1
//
//	Health endpoint:
//	http://localhost:{port}/api/v1/healthcheck
//
//	S3 Gateway endpoint:
//	http://localhost:{port}/s3
//
//	Login:
//	http://localhost:{port}
//	Access Key ID: {config.AccessKeyID}
//	Secret Access Key: {config.SecretAccessKey}
//
// Data Persistence:
//
//	LakeFS data is stored in a Docker volume ({config.DataVolume}).
//	This ensures metadata, commits, branches, and block storage persist across container restarts.
//
//	Volume mount points:
//	- /data - Block storage, metadata database, and commit history
//
// Network Configuration:
//
//	The container joins the specified Docker network, enabling
//	communication with other containers on the same network.
//
//	Applications can access LakeFS using:
//	http://{container_name}:8000
//
//	S3-compatible applications can use:
//	http://{container_name}:8000/s3
//
// Security:
//
//	IMPORTANT: Change the default access credentials for production!
//	Default credentials are example values and must be customized.
//
//	For production use:
//	- Set strong access key ID and secret access key
//	- Configure authentication (OpenID Connect, LDAP, SAML)
//	- Enable HTTPS with TLS certificates
//	- Use reverse proxy (nginx, traefik)
//	- Configure role-based access control (RBAC)
//	- Enable audit logging
//	- Use external database for metadata (PostgreSQL)
//	- Configure external block storage (S3, Azure, GCS)
//	- Enable encryption at rest and in transit
//
// Restart Policy:
//
//	The container is configured with restart policy "unless-stopped",
//	ensuring it automatically restarts if it crashes.
//
// Health Monitoring:
//
//	Health check runs HTTP GET http://localhost:8000 every 30 seconds:
//	- Timeout: 10 seconds
//	- Retries: 3 consecutive failures before unhealthy
//
// Performance Tuning:
//
//	For production workloads, consider:
//	- Adequate disk space for block storage
//	- SSD storage for better I/O performance
//	- External PostgreSQL for metadata (better than embedded DB)
//	- External object storage (S3, GCS, Azure) for scalability
//	- Resource limits (CPU, memory) based on workload
//	- Network bandwidth for data transfers
//	- Optimize block size for your data patterns
//	- Configure cache settings for frequently accessed data
//
// LakeFS Features:
//
//	Data versioning and management for data lakes:
//	- Git-like operations (branch, commit, merge, revert)
//	- Atomic commits across multiple objects
//	- Zero-copy branching (instant, cost-free branches)
//	- S3-compatible API for data operations
//	- Web UI for repository management
//	- CLI tool for automation
//	- Hooks for CI/CD integration (pre-commit, pre-merge, post-commit, post-merge)
//	- Data lineage and provenance tracking
//	- Isolated development and testing environments
//	- Rollback and time travel capabilities
//	- Experimentation without data duplication
//	- Reproducible data pipelines
//	- Data quality gates and validation
//	- Multi-format support (Parquet, ORC, Avro, CSV, JSON)
//	- Integration with Spark, Hive, Presto, Trino
//
// Architecture:
//
//	LakeFS consists of several components:
//	1. API Server - REST API for all operations
//	2. Block Adapter - Interface to underlying storage
//	3. Metadata Database - Stores commits, branches, refs
//	4. S3 Gateway - S3-compatible endpoint for data access
//	5. Web UI - Browser-based interface
//
//	Storage Backends:
//	- Local filesystem (development and testing)
//	- Amazon S3 (production)
//	- Azure Blob Storage (production)
//	- Google Cloud Storage (production)
//	- MinIO (self-hosted S3-compatible)
//
//	Metadata Backends:
//	- Embedded database (development and testing)
//	- PostgreSQL (production, recommended)
//
// Data Model:
//
//	LakeFS organizes data using Git-like concepts:
//	- Repository: Top-level container for data
//	- Branch: Isolated copy of data for development
//	- Commit: Immutable snapshot of data at a point in time
//	- Tag: Named reference to a specific commit
//	- Object: Individual data file in the repository
//	- Reference: Pointer to a commit (branch head or tag)
//
// Operations:
//
//	Branch Operations:
//	- Create branch: Instant, zero-copy isolation
//	- List branches: View all active branches
//	- Delete branch: Remove unused branches
//	- Get branch: Retrieve branch metadata
//
//	Commit Operations:
//	- Commit changes: Create immutable snapshot
//	- List commits: View commit history
//	- Get commit: Retrieve commit details
//	- Diff commits: Compare two commits
//
//	Merge Operations:
//	- Merge branches: Integrate changes from one branch to another
//	- Revert commit: Undo specific commits
//	- Cherry-pick: Apply specific commits to another branch
//
//	Object Operations:
//	- Upload object: Add or update data files
//	- Get object: Retrieve data files
//	- List objects: Browse repository contents
//	- Delete object: Remove data files
//	- Get object metadata: Retrieve file information
//
// Git-Like Workflow:
//
//	Typical development workflow:
//	1. Create feature branch from main
//	2. Upload and modify data on feature branch
//	3. Run data quality checks and tests
//	4. Commit changes to feature branch
//	5. Review changes using diff
//	6. Merge feature branch to main
//	7. Tag release for reproducibility
//
// S3 API Compatibility:
//
//	LakeFS provides S3-compatible API for seamless integration:
//	- Bucket operations (list, create, delete)
//	- Object operations (get, put, delete, list)
//	- Multipart uploads for large files
//	- Presigned URLs for temporary access
//	- Server-side copy for efficient data movement
//
//	Configure AWS SDK or S3 clients:
//	Endpoint: http://localhost:8000/s3
//	Access Key ID: {config.AccessKeyID}
//	Secret Access Key: {config.SecretAccessKey}
//	Region: us-east-1 (any value works)
//	Path Style: true (required)
//
// Integration with Data Tools:
//
//	LakeFS integrates with popular data tools:
//
//	Apache Spark:
//	spark.read.parquet("s3a://repo/branch/path/to/data.parquet")
//
//	Presto/Trino:
//	SELECT * FROM hive.schema.table WHERE path LIKE 's3://repo/branch/%'
//
//	Hive:
//	CREATE EXTERNAL TABLE data (...)
//	LOCATION 's3://repo/branch/path/to/data'
//
//	AWS CLI:
//	aws s3 ls s3://repo/branch/path/ --endpoint-url http://localhost:8000/s3
//
//	Python (boto3):
//	s3 = boto3.client('s3', endpoint_url='http://localhost:8000/s3')
//	s3.get_object(Bucket='repo/branch', Key='path/to/file')
//
// Hooks and Automation:
//
//	LakeFS supports hooks for CI/CD integration:
//	- pre-commit: Validate data before commit
//	- pre-merge: Run tests before merging
//	- post-commit: Trigger downstream workflows
//	- post-merge: Deploy to production
//
//	Hook types:
//	- Airflow DAGs: Trigger data pipelines
//	- dbt runs: Run data transformations
//	- Great Expectations: Validate data quality
//	- Custom scripts: Any executable or webhook
//
// Use Cases:
//
//	Development and Testing:
//	- Create isolated branches for development
//	- Test data transformations without affecting production
//	- Roll back failed experiments instantly
//	- Share reproducible datasets with team
//
//	Data Quality and Validation:
//	- Validate data quality before merging to main
//	- Implement data quality gates with hooks
//	- Enforce schema and format requirements
//	- Track data lineage and provenance
//
//	ML and AI Workflows:
//	- Version training datasets and models
//	- Reproduce experiments with exact data versions
//	- Compare model performance across data versions
//	- Isolate feature engineering experiments
//
//	ETL and Data Pipelines:
//	- Atomic commits for multi-file ETL outputs
//	- Roll back failed pipeline runs
//	- Test pipeline changes on branches
//	- Promote data through environments (dev, staging, prod)
//
//	Data Collaboration:
//	- Multiple teams working on shared data lake
//	- Isolated workspaces per team or project
//	- Controlled merging with review process
//	- Audit trail of all data changes
//
// API Endpoints:
//
//	Key HTTP API endpoints:
//	- GET  /api/v1/healthcheck - Health check
//	- GET  /api/v1/repositories - List repositories
//	- POST /api/v1/repositories - Create repository
//	- GET  /api/v1/repositories/{repo}/branches - List branches
//	- POST /api/v1/repositories/{repo}/branches - Create branch
//	- GET  /api/v1/repositories/{repo}/refs/{ref}/objects - List objects
//	- PUT  /api/v1/repositories/{repo}/branches/{branch}/objects - Upload object
//	- GET  /api/v1/repositories/{repo}/refs/{ref}/objects - Get object
//	- POST /api/v1/repositories/{repo}/branches/{branch}/commits - Commit changes
//	- GET  /api/v1/repositories/{repo}/refs/{ref}/commits - List commits
//	- POST /api/v1/repositories/{repo}/refs/{dest}/merge/{source} - Merge branches
//	- GET  /api/v1/repositories/{repo}/refs/{ref}/diff/{other} - Diff refs
//	- POST /api/v1/repositories/{repo}/refs/{ref}/revert - Revert commit
//	- GET  /api/v1/user - Get current user
//	- GET  /api/v1/config - Get LakeFS configuration
//
// CLI Usage:
//
//	LakeFS provides a CLI tool (lakectl) for automation:
//
//	Configure CLI:
//	lakectl config
//
//	Repository operations:
//	lakectl repo list
//	lakectl repo create lakefs://myrepo s3://my-bucket/
//
//	Branch operations:
//	lakectl branch list lakefs://myrepo
//	lakectl branch create lakefs://myrepo/dev
//	lakectl branch delete lakefs://myrepo/old-branch
//
//	Commit operations:
//	lakectl commit lakefs://myrepo/dev -m "Update dataset"
//	lakectl log lakefs://myrepo/main
//
//	Merge operations:
//	lakectl merge lakefs://myrepo/dev lakefs://myrepo/main
//
//	Object operations:
//	lakectl fs ls lakefs://myrepo/main/path/
//	lakectl fs upload lakefs://myrepo/dev/file.txt ./local-file.txt
//	lakectl fs download lakefs://myrepo/main/file.txt ./local-file.txt
//
//	Diff operations:
//	lakectl diff lakefs://myrepo/main lakefs://myrepo/dev
//
// Monitoring:
//
//	Monitor these metrics and endpoints:
//	- Health check endpoint: /api/v1/healthcheck
//	- Repository count and size
//	- Branch count per repository
//	- Commit rate and frequency
//	- Object count and total size
//	- API response times
//	- Error rates and types
//	- Block storage usage
//	- Metadata database size
//	- Active connections
//
// Backup Strategy:
//
//	Important: Implement regular backups!
//	- Backup metadata database (PostgreSQL dump)
//	- Backup block storage (depends on backend)
//	- For local storage: backup /data volume
//	- For S3 backend: use S3 versioning and replication
//	- Export repository metadata for disaster recovery
//	- Document restore procedures
//	- Test restore regularly
//	- Keep multiple backup copies
//	- Store backups in different locations
//
// Migration from Non-Versioned Data Lakes:
//
//	Steps to migrate existing data to LakeFS:
//	1. Deploy LakeFS with appropriate storage backend
//	2. Create repository pointing to existing data location
//	3. Initial commit to create baseline
//	4. Gradually migrate applications to use LakeFS S3 gateway
//	5. Update data pipelines to commit changes
//	6. Implement branching workflow for new development
//	7. Add hooks for data quality validation
//	8. Train team on Git-like workflows
//
// High Availability:
//
//	For production HA setup:
//	- Run multiple LakeFS instances behind load balancer
//	- Use external PostgreSQL with replication
//	- Use distributed object storage (S3, GCS, Azure)
//	- Configure health checks on load balancer
//	- Implement automatic failover
//	- Monitor all instances
//	- Use connection pooling for database
//	- Configure proper timeouts and retries
//
// Scalability:
//
//	LakeFS can scale to handle large workloads:
//	- Horizontal scaling: Add more LakeFS instances
//	- Vertical scaling: Increase CPU and memory
//	- Database scaling: Use PostgreSQL with read replicas
//	- Storage scaling: Unlimited with S3/GCS/Azure
//	- Optimize block size for your data patterns
//	- Use caching for frequently accessed metadata
//	- Configure connection pooling
//	- Monitor and tune database queries
//
// Configuration:
//
//	This deployment uses local storage and embedded database for simplicity.
//	For production, configure external storage and database via environment variables:
//
//	Database configuration (PostgreSQL):
//	LAKEFS_DATABASE_TYPE=postgres
//	LAKEFS_DATABASE_CONNECTION_STRING=postgres://user:pass@host:5432/lakefs
//	LAKEFS_DATABASE_MAX_OPEN_CONNECTIONS=25
//	LAKEFS_DATABASE_MAX_IDLE_CONNECTIONS=5
//
//	Block storage configuration (S3):
//	LAKEFS_BLOCKSTORE_TYPE=s3
//	LAKEFS_BLOCKSTORE_S3_REGION=us-east-1
//	LAKEFS_BLOCKSTORE_S3_CREDENTIALS_ACCESS_KEY_ID=...
//	LAKEFS_BLOCKSTORE_S3_CREDENTIALS_SECRET_ACCESS_KEY=...
//
//	Authentication configuration (OpenID Connect):
//	LAKEFS_AUTH_OIDC_ENABLED=true
//	LAKEFS_AUTH_OIDC_URL=https://auth.example.com
//	LAKEFS_AUTH_OIDC_CLIENT_ID=lakefs
//
//	Logging configuration:
//	LAKEFS_LOGGING_LEVEL=info
//	LAKEFS_LOGGING_OUTPUT=stdout
//	LAKEFS_LOGGING_FORMAT=json
//
// Best Practices:
//
//	1. Always use meaningful commit messages
//	2. Create branches for all development and testing
//	3. Never commit directly to main branch
//	4. Use pull request workflow for merges
//	5. Implement data quality hooks
//	6. Tag releases for reproducibility
//	7. Clean up old branches regularly
//	8. Monitor repository size and growth
//	9. Use external storage for production
//	10. Document branching strategy for team
//	11. Automate with hooks and CI/CD
//	12. Regular backups of metadata
//
// Error Handling:
//
//	Returns error if:
//	- Container with same name already exists
//	- Network or volume creation fails
//	- Docker API errors occur
//	- Invalid configuration provided
func DeployLakeFS(ctx context.Context, cli common.DockerClient, config LakeFSProductionConfig) (string, error) {
	// Check if container already exists
	exists, err := common.ContainerExistsWithClient(ctx, cli, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return "", fmt.Errorf("container %s already exists", config.ContainerName)
	}

	// Prepare production environment (network and volume)
	if err := PrepareProductionEnvironment(ctx, cli, config.Production); err != nil {
		return "", fmt.Errorf("failed to prepare environment: %w", err)
	}

	// Pull LakeFS image
	if err := common.ImagePullWithClient(ctx, cli, config.Image, &common.ImagePullOptions{Silent: true}); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: config.Port,
	}
	portMap := nat.PortMap{
		"8000/tcp": []nat.PortBinding{portBinding},
	}

	// Configure volume mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: config.DataVolume,
			Target: "/data",
		},
	}

	// Container configuration
	containerConfig := container.Config{
		Image: config.Image,
		Cmd:   []string{"run", "--local-settings"},
		Env: []string{
			"LAKEFS_DATABASE_TYPE=local",
			"LAKEFS_BLOCKSTORE_TYPE=local",
			"LAKEFS_BLOCKSTORE_LOCAL_PATH=/data",
			fmt.Sprintf("LAKEFS_AUTH_ENCRYPT_SECRET_KEY=%s", config.SecretAccessKey),
			"LAKEFS_STATS_ENABLED=false",
			"LAKEFS_INSTALLATION_USER_NAME=admin",
			fmt.Sprintf("LAKEFS_INSTALLATION_ACCESS_KEY_ID=%s", config.AccessKeyID),
			fmt.Sprintf("LAKEFS_INSTALLATION_SECRET_ACCESS_KEY=%s", config.SecretAccessKey),
		},
		ExposedPorts: nat.PortSet{
			"8000/tcp": struct{}{},
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --spider --quiet http://localhost:8000 || exit 1"},
			Interval: 30000000000, // 30 seconds
			Timeout:  10000000000, // 10 seconds
			Retries:  3,
		},
	}

	// Host configuration
	hostConfig := container.HostConfig{
		PortBindings: portMap,
		Mounts:       mounts,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Deploy container
	err = common.CreateAndStartContainerWithClient(ctx, cli, containerConfig, hostConfig, config.ContainerName, config.Production.NetworkName)
	if err != nil {
		return "", fmt.Errorf("failed to create and start container: %w", err)
	}

	// Get container ID
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+config.ContainerName {
				return cont.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container created but ID not found")
}

// StopLakeFS stops a running LakeFS container.
//
// Performs graceful shutdown to ensure data integrity and proper flushing of metadata.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the LakeFS container to stop
//
// Returns:
//   - error: Stop errors
//
// Example:
//
//	err := StopLakeFS(ctx, cli, "lakefs")
//	if err != nil {
//	    log.Printf("Failed to stop LakeFS: %v", err)
//	}
func StopLakeFS(ctx context.Context, cli common.DockerClient, containerName string) error {
	timeout := 30 // 30 seconds for graceful shutdown
	return cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
}

// RemoveLakeFS removes a LakeFS container and optionally its volume.
//
// WARNING: Removing the volume will DELETE ALL DATA, COMMITS, and BRANCHES permanently!
// Always backup data before removing volumes.
//
// Parameters:
//   - ctx: Context for Docker operations
//   - cli: Docker API client
//   - containerName: Name of the LakeFS container to remove
//   - removeVolume: Whether to also remove the data volume (DANGEROUS!)
//   - volumeName: Name of the data volume (required if removeVolume is true)
//
// Returns:
//   - error: Removal errors
//
// Example:
//
//	// Remove container but keep data (safe)
//	err := RemoveLakeFS(ctx, cli, "lakefs", false, "")
//
//	// Remove container and data (DANGEROUS - backup first!)
//	err := RemoveLakeFS(ctx, cli, "lakefs", true, "lakefs-data")
func RemoveLakeFS(ctx context.Context, cli common.DockerClient, containerName string, removeVolume bool, volumeName string) error {
	// Remove container
	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove volume if requested (DANGEROUS - data loss!)
	if removeVolume && volumeName != "" {
		if err := cli.VolumeRemove(ctx, volumeName, true); err != nil {
			return fmt.Errorf("failed to remove volume: %w", err)
		}
	}

	return nil
}

// GetLakeFSURL returns the LakeFS HTTP endpoint URL for the deployed container.
//
// This is a convenience function that formats the URL for LakeFS UI and API.
//
// Parameters:
//   - config: LakeFS production configuration
//
// Returns:
//   - string: LakeFS HTTP endpoint URL
//
// Example:
//
//	config := DefaultLakeFSProductionConfig()
//	lakeFSURL := GetLakeFSURL(config)
//	// http://localhost:8000
func GetLakeFSURL(config LakeFSProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s", config.Port)
}

// GetLakeFSAPIURL returns the LakeFS API base URL for the deployed container.
//
// This is a convenience function that formats the API base URL.
//
// Parameters:
//   - config: LakeFS production configuration
//
// Returns:
//   - string: LakeFS API base URL
//
// Example:
//
//	config := DefaultLakeFSProductionConfig()
//	apiURL := GetLakeFSAPIURL(config)
//	// http://localhost:8000/api/v1
func GetLakeFSAPIURL(config LakeFSProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api/v1", config.Port)
}

// GetLakeFSHealthURL returns the LakeFS health check URL for the deployed container.
//
// This is a convenience function for monitoring and health checks.
//
// Parameters:
//   - config: LakeFS production configuration
//
// Returns:
//   - string: LakeFS health check URL
//
// Example:
//
//	config := DefaultLakeFSProductionConfig()
//	healthURL := GetLakeFSHealthURL(config)
//	// http://localhost:8000/api/v1/healthcheck
func GetLakeFSHealthURL(config LakeFSProductionConfig) string {
	return fmt.Sprintf("http://localhost:%s/api/v1/healthcheck", config.Port)
}
