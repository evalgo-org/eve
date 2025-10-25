// Package deploy provides comprehensive Docker container orchestration for infrastructure deployment.
// This package implements Docker-based deployment automation with specialized support for
// workflow orchestration services, focusing on Kestra deployment patterns and enterprise
// data pipeline infrastructure requiring scalable task execution and workflow management.
//
// Workflow Orchestration Integration:
//
//	The package provides deployment automation for Kestra:
//	- Scalable workflow execution and task orchestration
//	- Multi-language support for diverse data processing tasks
//	- Real-time monitoring and workflow visualization
//	- Integration with databases, APIs, and cloud services
//	- Event-driven architecture with conditional execution
//
// Data Pipeline Infrastructure Patterns:
//
//	Implements enterprise data pipeline deployment scenarios:
//	- ETL/ELT workflow automation and scheduling
//	- Multi-environment pipeline deployment and management
//	- Microservice orchestration with dependency management
//	- Real-time and batch processing pipeline execution
//	- Data quality validation and error handling workflows
//
// Container Orchestration for Data Processing:
//
//	Provides specialized deployment functions for:
//	- Workflow orchestration services with persistent storage
//	- Database-backed workflow state management
//	- Multi-worker task execution environments
//	- Integration platforms connecting diverse data sources
//	- Enterprise scheduling and monitoring infrastructure
//
// Production Data Pipeline Considerations:
//
//	Designed with enterprise data processing requirements:
//	- High-throughput task execution with worker scaling
//	- Persistent storage for workflow definitions and execution history
//	- Database integration for state management and metadata storage
//	- Network connectivity for multi-service data pipeline architectures
//	- Monitoring and alerting for pipeline health and performance
package deploy

import (
	"context"
	eve "eve.evalgo.org/common"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"time"
)

// DeployKestraContainer deploys a Kestra workflow orchestration container with comprehensive configuration.
// This function creates and starts a Kestra instance optimized for enterprise workflow execution,
// providing scalable task orchestration, data pipeline management, and real-time workflow monitoring.
//
// Kestra Overview:
//
//	Kestra is an open-source workflow orchestration platform that provides:
//	- Declarative workflow definition using YAML syntax
//	- Multi-language task execution support (Python, R, SQL, Shell, etc.)
//	- Real-time workflow monitoring and visualization
//	- Event-driven execution with conditional logic and branching
//	- Integration with databases, APIs, file systems, and cloud services
//	- Scalable execution with parallel task processing and worker management
//
// Parameters:
//   - ctx: Context for operation cancellation and timeout control
//   - cli: Docker client for container management API operations
//   - imageVersion: Kestra Docker image version tag for deployment control
//   - containerName: Unique container identifier for management operations
//   - volumeName: Named volume for persistent workflow storage and execution data
//   - postgresLink: Legacy Docker link to PostgreSQL container for database connectivity
//   - envVars: Environment variables for Kestra configuration and database connection
//
// Returns:
//   - error: Container creation, volume management, or startup failures
//
// Container Configuration:
//
//	Image: docker.io/kestra/kestra:{imageVersion}
//	- Official Kestra image with full feature set including all plugins
//	- Version control through imageVersion parameter for deployment consistency
//	- Includes built-in connectors for databases, cloud services, and APIs
//	- Pre-configured with optimizations for container environments
//
//	Command: ["server", "standalone", "--worker-thread=128"]
//	- Runs Kestra in standalone mode with embedded worker execution
//	- Configures 128 worker threads for high-throughput task execution
//	- Combines web server, scheduler, and executor in single container
//	- Suitable for development and medium-scale production workloads
//
// Storage Management:
//
//	Volume Configuration:
//	- Creates named volume for persistent workflow and execution data storage
//	- Mounts volume to /app/storage for workflow artifacts and temporary files
//	- Ensures data persistence across container restarts and updates
//	- Supports backup and disaster recovery for workflow execution history
//
//	Storage Use Cases:
//	- Workflow definition files and templates
//	- Execution logs and audit trails
//	- Temporary files and intermediate processing results
//	- Plugin configurations and custom extensions
//	- Execution artifacts and output files
//
// Network Configuration:
//
//	Port Exposure:
//	- Exposes port 8080 for Kestra web interface and API access
//	- Maps container port to host port 8080 for external connectivity
//	- Provides access to workflow designer, execution monitoring, and REST API
//	- Supports integration with external monitoring and management tools
//
//	Database Connectivity:
//	- Uses legacy Docker links for PostgreSQL database connection
//	- Enables Kestra to store workflow metadata, execution state, and logs
//	- Supports transaction management and consistency for workflow execution
//	- Provides scalability and reliability for enterprise deployments
//
// Worker Thread Configuration:
//
//	Thread Pool: 128 worker threads
//	- Configures high-concurrency execution for parallel task processing
//	- Enables simultaneous execution of multiple workflow tasks
//	- Optimized for CPU-intensive and I/O-bound workflow operations
//	- Scalable based on container resource allocation and workload requirements
//
// Use Cases:
//
//	Primary applications requiring Kestra workflow orchestration:
//	- ETL/ELT data pipeline automation and scheduling
//	- Microservice orchestration with complex dependency management
//	- Business process automation with human task integration
//	- Data quality validation and monitoring workflows
//	- API integration and data synchronization processes
//	- Machine learning pipeline orchestration and model deployment
//
// Error Conditions:
//
//	Volume Creation Failures:
//	- Insufficient disk space for volume creation
//	- Docker daemon connectivity issues
//	- Permission errors for volume management
//	- Volume name conflicts with existing storage
//
//	Container Creation Failures:
//	- Image pull failures due to network or authentication issues
//	- Port binding conflicts with existing services
//	- Resource constraints (memory, CPU) preventing container creation
//	- Invalid environment variable or configuration format
//
//	Startup Failures:
//	- Database connectivity issues preventing Kestra initialization
//	- Invalid configuration causing application startup errors
//	- Network connectivity problems affecting service integration
//	- Insufficient resources for worker thread pool initialization
//
// Example Usage:
//
//	ctx := context.Background()
//	cli, err := client.NewClientWithOpts(client.FromEnv)
//	if err != nil {
//	    log.Fatal("Failed to create Docker client:", err)
//	}
//	defer cli.Close()
//
//	// Configure Kestra environment variables
//	config := `
//	datasources:
//	  postgres:
//	    url: jdbc:postgresql://postgres:5432/kestra
//	    driverClassName: org.postgresql.Driver
//	    username: kestra
//	    password: kestrapassword
//	server:
//	  port: 8080
//	kestra:
//	  repository:
//	    type: postgres
//	  queue:
//	    type: postgres
//	  storage:
//	    type: local
//	    local:
//	      basePath: "/app/storage"
//	`
//
//	envVars := []string{"KESTRA_CONFIGURATION=" + config}
//
//	// Deploy Kestra workflow orchestration service
//	err = DeployKestraContainer(ctx, cli, "latest-full", "kestra-workflow",
//	                           "kestra-storage", "postgres:postgres", envVars)
//	if err != nil {
//	    log.Printf("Kestra deployment failed: %v", err)
//	    return
//	}
//
//	log.Println("Kestra workflow orchestration deployed successfully")
//
// Production Deployment Enhancements:
//
//	High Availability Configuration:
//	- Deploy multiple Kestra instances with shared database backend
//	- Use external PostgreSQL cluster for scalability and reliability
//	- Implement load balancing for web interface and API access
//	- Configure health checks and monitoring for service availability
//
//	Performance Optimization:
//	- Adjust worker thread count based on workload characteristics
//	- Configure JVM settings for optimal memory management
//	- Use dedicated worker nodes for compute-intensive tasks
//	- Implement resource quotas and limits for workflow execution
//
//	Storage and Backup:
//	- Use external storage systems for workflow artifacts and logs
//	- Implement backup strategies for workflow definitions and execution history
//	- Configure log rotation and archival for long-term retention
//	- Use distributed storage for high availability and performance
//
// Configuration Management:
//
//	The envVars parameter typically includes KESTRA_CONFIGURATION with YAML content:
//
//	Database Configuration:
//	- PostgreSQL connection details for metadata storage
//	- Connection pooling and transaction management settings
//	- Database migration and schema management configuration
//
//	Storage Configuration:
//	- Local storage path for workflow artifacts and temporary files
//	- External storage integration (S3, GCS, Azure Blob) for scalability
//	- File retention policies and cleanup automation
//
//	Queue Configuration:
//	- Task queue backend for workflow execution coordination
//	- Queue persistence and reliability settings
//	- Worker communication and task distribution mechanisms
//
//	Security Configuration:
//	- Authentication and authorization settings
//	- API security and access control
//	- Integration with external identity providers
//	- Audit logging and compliance monitoring
func DeployKestraContainer(ctx context.Context, cli *client.Client, imageVersion, containerName, volumeName, postgresLink string, envVars []string) error {
	// Create persistent volume for workflow storage and execution data
	if err := CreateVolume(ctx, cli, volumeName); err != nil {
		return err
	}

	// Create Kestra container with comprehensive workflow orchestration configuration
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "docker.io/kestra/kestra:" + imageVersion, // Official Kestra image with specified version
		Env:   envVars,                                   // Environment configuration including database settings
		ExposedPorts: nat.PortSet{ // Port exposure for web interface and API
			"8080/tcp": struct{}{},
		},
		Cmd: []string{"server", "standalone", "--worker-thread=128"}, // Standalone mode with high-concurrency workers
	}, &container.HostConfig{
		PortBindings: nat.PortMap{ // Host-to-container port mapping
			"8080/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
		},
		Mounts: []mount.Mount{ // Persistent storage mounting
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/app/storage",
			},
		},
		Links: []string{postgresLink}, // Database connectivity through Docker links
	}, nil, nil, "kestra")

	if err != nil {
		return err
	}

	// Start the Kestra workflow orchestration container
	return cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
}

// DeployKestra orchestrates the complete deployment of a Kestra workflow environment.
// This function provides end-to-end deployment automation including database setup,
// network configuration, and Kestra application deployment with proper dependency management.
//
// Complete Environment Deployment:
//
//	The function orchestrates deployment of a complete Kestra environment:
//	- PostgreSQL database for workflow metadata and execution state storage
//	- Custom Docker network for secure service-to-service communication
//	- Kestra application server with workflow execution capabilities
//	- Proper startup sequencing with dependency management
//	- Network integration for database connectivity and service discovery
//
// Deployment Architecture:
//
//	Network Infrastructure:
//	- Creates custom bridge network named "kestra" for service isolation
//	- Enables container-to-container communication using service names
//	- Provides network security and traffic isolation from other applications
//	- Supports service discovery and automatic DNS resolution
//
//	Database Layer:
//	- Deploys PostgreSQL 15 with Kestra-specific configuration
//	- Creates dedicated database, user, and credentials for Kestra
//	- Provides persistent storage for workflow definitions and execution history
//	- Ensures data consistency and reliability for workflow state management
//
//	Application Layer:
//	- Deploys Kestra with latest-full image containing all plugins and features
//	- Configures database connectivity and storage integration
//	- Sets up high-concurrency worker execution environment
//	- Provides web interface for workflow design and monitoring
//
// Deployment Sequence:
//  1. Network Creation: Establishes isolated network for service communication
//  2. Database Deployment: Creates and starts PostgreSQL with Kestra configuration
//  3. Database Readiness: Waits for database initialization and availability
//  4. Network Integration: Connects database container to Kestra network
//  5. Application Configuration: Prepares Kestra configuration with database settings
//  6. Application Deployment: Creates and starts Kestra workflow orchestration service
//  7. Service Integration: Connects Kestra container to shared network
//
// Database Configuration:
//
//	PostgreSQL Setup:
//	- Version: PostgreSQL 15 for stability and performance
//	- Database: "kestra" dedicated database for workflow metadata
//	- User: "kestra" with appropriate permissions for application access
//	- Password: "kestrapassword" (should be randomized for production)
//	- Volume: "kestra-postgres-data" for persistent database storage
//
// Kestra Configuration:
//
//	Application Settings:
//	- Datasource: PostgreSQL connection for metadata and state storage
//	- Repository: PostgreSQL-backed workflow definition storage
//	- Queue: PostgreSQL-backed task queue for workflow execution
//	- Storage: Local filesystem storage for workflow artifacts
//	- Server: Web interface and API server on port 8080
//
//	YAML Configuration Structure:
//	```yaml
//	datasources:
//	  postgres:
//	    url: jdbc:postgresql://kestra-postgres:5432/kestra
//	    driverClassName: org.postgresql.Driver
//	    username: kestra
//	    password: kestrapassword
//	server:
//	  port: 8080
//	kestra:
//	  repository:
//	    type: postgres
//	  queue:
//	    type: postgres
//	  storage:
//	    type: local
//	    local:
//	      basePath: "/app/storage"
//	```
//
// Network Integration:
//
//	Service Discovery:
//	- Database accessible via hostname "kestra-postgres"
//	- Kestra application accessible via hostname "kestra"
//	- Internal communication through custom bridge network
//	- External access to Kestra web interface through port 8080
//
// Use Cases:
//
//	Complete workflow orchestration environments for:
//	- Development teams building and testing data pipelines
//	- Production data processing workflows with enterprise requirements
//	- ETL/ELT automation with complex dependency management
//	- Business process automation and integration workflows
//	- Machine learning pipeline orchestration and model deployment
//
// Example Usage:
//
//	ctx := context.Background()
//	cli, err := client.NewClientWithOpts(client.FromEnv)
//	if err != nil {
//	    log.Fatal("Failed to create Docker client:", err)
//	}
//	defer cli.Close()
//
//	// Deploy complete Kestra environment
//	DeployKestra(ctx, cli)
//
//	// Wait for services to be fully ready
//	time.Sleep(30 * time.Second)
//
//	log.Println("Kestra workflow environment deployed successfully")
//	log.Println("Access Kestra web interface at: http://localhost:8080")
//
// Production Considerations:
//
//	Security Enhancements:
//	- Use strong, randomized passwords for database authentication
//	- Implement SSL/TLS encryption for database connections
//	- Configure network policies for access control and traffic isolation
//	- Enable audit logging and compliance monitoring
//
//	High Availability:
//	- Deploy PostgreSQL in clustered configuration for database reliability
//	- Use external load balancers for Kestra web interface access
//	- Implement health checks and automatic restart policies
//	- Configure backup and disaster recovery procedures
//
//	Performance Optimization:
//	- Adjust PostgreSQL configuration for workflow workload characteristics
//	- Configure Kestra worker thread pools based on processing requirements
//	- Use external storage systems for large workflow artifacts
//	- Implement monitoring and alerting for performance optimization
//
//	Scalability Planning:
//	- Design for horizontal scaling with multiple Kestra instances
//	- Use external databases and storage for shared state management
//	- Implement container orchestration platforms (Kubernetes) for scaling
//	- Plan for geographic distribution and multi-region deployments
//
// Monitoring and Operations:
//
//	Health Monitoring:
//	- Database connectivity and performance monitoring
//	- Workflow execution success rates and error tracking
//	- Resource utilization monitoring for capacity planning
//	- Service availability and response time monitoring
//
//	Operational Procedures:
//	- Regular database maintenance and optimization
//	- Workflow definition backup and version control
//	- Log aggregation and analysis for troubleshooting
//	- Performance tuning and capacity scaling procedures
//
// Development Workflow:
//
//	The deployed environment supports complete workflow development:
//	- Web-based workflow designer for visual pipeline creation
//	- REST API for programmatic workflow management
//	- Real-time execution monitoring and debugging
//	- Version control integration for workflow definitions
//	- Testing and validation tools for pipeline development
func DeployKestra(ctx context.Context, cli *client.Client) {
	// Create isolated network for Kestra service communication
	CreateNetwork(ctx, cli, "kestra")

	// Deploy PostgreSQL database with Kestra-specific configuration
	DeployPostgres(ctx, cli, "15", "kestra-postgres", "kestra-postgres-data",
		[]string{
			"POSTGRES_PASSWORD=kestrapassword",
			"POSTGRES_USER=kestra",
			"POSTGRES_DB=kestra",
		}, &PostgresDeployOptions{
			PullImage:    true,
			CreateVolume: false,
		})

	// Wait for database initialization and readiness
	time.Sleep(5 * time.Second)

	// Connect PostgreSQL container to Kestra network
	eve.AddContainerToNetwork(ctx, cli, "kestra-postgres", "kestra")

	// Configure Kestra application with database connectivity and storage settings
	config := `
datasources:
  postgres:
    url: jdbc:postgresql://kestra-postgres:5432/kestra
    driverClassName: org.postgresql.Driver
    username: kestra
    password: kestrapassword
server:
  port: 8080
kestra:
  repository:
    type: postgres
  queue:
    type: postgres
  storage:
    type: local
    local:
      basePath: "/app/storage"
`

	// Prepare environment variables with Kestra configuration
	envVars := []string{"KESTRA_CONFIGURATION=" + config}

	// Deploy Kestra workflow orchestration container
	DeployKestraContainer(ctx, cli, "latest-full", "kestra", "kestra-data",
		"kestra-postgres:kestra-postgres", envVars)

	// Connect Kestra container to shared network for service integration
	eve.AddContainerToNetwork(ctx, cli, "kestra", "kestra")
}

// Additional deployment patterns and considerations for Kestra:
//
// Enterprise Deployment Patterns:
//   Multi-Instance Deployment:
//   - Deploy multiple Kestra instances for high availability and load distribution
//   - Use external PostgreSQL cluster for shared state management
//   - Implement load balancing for web interface and API access
//   - Configure service discovery and health monitoring
//
//   Microservice Integration:
//   - Deploy Kestra as part of larger microservice architectures
//   - Integrate with API gateways and service mesh technologies
//   - Configure distributed tracing and observability
//   - Implement circuit breakers and fault tolerance patterns
//
// Storage and Data Management:
//   External Storage Integration:
//   - Configure S3, GCS, or Azure Blob storage for workflow artifacts
//   - Implement data lifecycle management and archival policies
//   - Use distributed storage for high availability and performance
//   - Configure backup and disaster recovery procedures
//
//   Database Optimization:
//   - Tune PostgreSQL for workflow metadata and execution logging
//   - Implement connection pooling and query optimization
//   - Configure database monitoring and performance analysis
//   - Plan for database scaling and maintenance procedures
//
// Security and Compliance:
//   Authentication and Authorization:
//   - Integrate with enterprise identity providers (LDAP, SAML, OIDC)
//   - Implement role-based access control for workflow management
//   - Configure API security and access token management
//   - Enable audit logging and compliance monitoring
//
//   Network Security:
//   - Implement network policies for traffic isolation and access control
//   - Configure SSL/TLS encryption for all communications
//   - Use VPN or private networks for database connectivity
//   - Implement security scanning and vulnerability management
//
// Monitoring and Observability:
//   Comprehensive Monitoring:
//   - Monitor workflow execution performance and success rates
//   - Track resource utilization and capacity planning metrics
//   - Implement alerting for workflow failures and system issues
//   - Configure distributed tracing for complex workflow debugging
//
//   Integration with Monitoring Platforms:
//   - Export metrics to Prometheus, Grafana, or other monitoring systems
//   - Configure log aggregation with ELK stack or similar platforms
//   - Implement custom dashboards for workflow operations teams
//   - Set up automated incident response and escalation procedures
