// Package testing provides testcontainers-based stack setup for integration tests.
//
// This package uses testcontainers-go to deploy multi-container stacks for testing
// purposes. Stacks are automatically orchestrated with proper dependency ordering,
// health checks, and post-start actions. All containers are ephemeral and cleaned
// up after tests complete.
//
// Key Features:
//   - Multi-container stack deployment
//   - Automatic dependency ordering (by position)
//   - Health check waiting for dependencies
//   - Post-start action execution (migrations, initialization)
//   - Automatic cleanup of all containers
//   - Network creation and isolation
//
// Build Tags:
//
//	Integration tests using this package should use the integration build tag:
//	//go:build integration
//
// Example Usage:
//
//	func TestInfisicalStack(t *testing.T) {
//	    ctx := context.Background()
//	    stack, err := stacks.LoadStackFromFile("../definitions/infisical.json")
//	    require.NoError(t, err)
//
//	    deployment, cleanup, err := SetupStack(ctx, t, stack)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Use the deployed stack for testing
//	    infisicalURL := fmt.Sprintf("http://localhost:%s", deployment.Ports["infisical"])
//	}
package testing

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"eve.evalgo.org/containers/stacks"
)

// StackDeployment represents a deployed test stack with container references and ports.
type StackDeployment struct {
	// Stack is the stack definition
	Stack *stacks.Stack
	// Containers maps container names to testcontainer instances
	Containers map[string]testcontainers.Container
	// Ports maps container names to their exposed host ports (first exposed port)
	Ports map[string]string
	// Network is the testcontainer network
	Network *testcontainers.DockerNetwork
}

// StackCleanup is a function type for cleaning up test stacks.
// Call this function in defer to ensure all containers and networks are terminated after tests.
type StackCleanup func()

// SetupStack deploys a multi-container stack for integration testing.
//
// This function orchestrates the deployment of a complete stack with proper dependency
// ordering, health checks, and post-start actions. All containers are ephemeral and
// automatically cleaned up after tests complete.
//
// Orchestration Process:
//  1. Create dedicated network for stack isolation
//  2. Sort containers by position (startup order)
//  3. For each container in order:
//     - Wait for all dependencies to be healthy
//     - Start the container
//     - Wait for the container's health check to pass
//     - Execute post-start actions sequentially
//  4. Return deployment info with container references and ports
//
// Health Check Support:
//   - command: Execute command in container (e.g., ["pg_isready", "-U", "postgres"])
//   - http: HTTP GET request to path (e.g., /health)
//   - tcp: TCP connection check on port
//   - postgres: PostgreSQL connection check using pg_isready
//   - redis: Redis PING command check
//
// Post-Start Actions:
//   - Executed after container is healthy
//   - Run sequentially in order defined
//   - Supports migrations, data seeding, initialization
//   - Configurable timeout per action
//
// Parameters:
//   - ctx: Context for container operations
//   - t: Testing context for requirement checks and logging
//   - stack: Stack definition loaded from JSON-LD
//
// Returns:
//   - *StackDeployment: Deployment info with container references and ports
//   - StackCleanup: Function to terminate all containers and network
//   - error: Deployment errors (validation, container creation, health checks)
//
// Example Usage:
//
//	func TestPostgresRedisStack(t *testing.T) {
//	    ctx := context.Background()
//
//	    // Define stack
//	    stack := &stacks.Stack{
//	        Name: "test-stack",
//	        Network: stacks.NetworkConfig{
//	            Name: "test-network",
//	            CreateIfNotExists: true,
//	        },
//	        ItemListElement: []stacks.StackItemElement{
//	            {
//	                Position: 1,
//	                Name: "postgres",
//	                Image: "postgres:17",
//	                Environment: map[string]string{
//	                    "POSTGRES_PASSWORD": "test",
//	                },
//	                Ports: []stacks.PortMapping{{ContainerPort: 5432}},
//	                HealthCheck: stacks.HealthCheckConfig{
//	                    Type: "command",
//	                    Command: []string{"pg_isready", "-U", "postgres"},
//	                },
//	            },
//	            {
//	                Position: 2,
//	                Name: "redis",
//	                Image: "redis:7-alpine",
//	                SoftwareRequirements: []stacks.SoftwareRequirement{
//	                    {Name: "postgres", WaitForHealthy: true},
//	                },
//	                Ports: []stacks.PortMapping{{ContainerPort: 6379}},
//	            },
//	        },
//	    }
//
//	    deployment, cleanup, err := SetupStack(ctx, t, stack)
//	    require.NoError(t, err)
//	    defer cleanup()
//
//	    // Access containers by name
//	    pgPort := deployment.Ports["postgres"]
//	    redisPort := deployment.Ports["redis"]
//	    t.Logf("PostgreSQL: localhost:%s", pgPort)
//	    t.Logf("Redis: localhost:%s", redisPort)
//	}
//
// Network Isolation:
//
//	Each stack gets its own Docker network, ensuring isolation between
//	concurrent test runs. Containers can communicate using their names
//	as hostnames (Docker DNS resolution).
//
// Cleanup:
//
//	Always defer the cleanup function to ensure containers are terminated:
//	defer cleanup()
//
//	The cleanup function terminates all containers and removes the network.
//	It's safe to call even if setup fails (it's a no-op).
//
// Error Handling:
//
//	Returns error if:
//	- Stack validation fails
//	- Network creation fails
//	- Container creation or startup fails
//	- Health checks timeout or fail
//	- Post-start actions fail or timeout
//	- Circular dependencies detected
//
// Performance:
//
//	Stack startup time depends on:
//	- Number of containers
//	- Image pull time (first run)
//	- Container initialization time
//	- Health check intervals and retries
//	- Post-start action execution time
//
//	Typical times:
//	- Simple 2-container stack: 5-10 seconds
//	- Complex 5-container stack: 20-30 seconds
//	- Includes image pulls on first run
//
// Debugging:
//
//	Use t.Logf() to log deployment progress:
//
//	deployment, cleanup, err := SetupStack(ctx, t, stack)
//	if err != nil {
//	    t.Logf("Stack deployment failed: %v", err)
//	}
//
//	Check container logs for troubleshooting:
//	logs, _ := deployment.Containers["postgres"].Logs(ctx)
func SetupStack(ctx context.Context, t *testing.T, stack *stacks.Stack) (*StackDeployment, StackCleanup, error) {
	// Validate stack
	if err := stack.Validate(); err != nil {
		return nil, func() {}, fmt.Errorf("stack validation failed: %w", err)
	}

	// Create deployment tracking
	deployment := &StackDeployment{
		Stack:      stack,
		Containers: make(map[string]testcontainers.Container),
		Ports:      make(map[string]string),
	}

	// Create network if specified
	var networkName string
	if stack.Network.Name != "" {
		networkName = fmt.Sprintf("%s-%s", stack.Network.Name, time.Now().Format("20060102150405"))
		dockerNetwork, err := network.New(ctx, network.WithDriver(stack.Network.Driver))
		if err != nil {
			return nil, func() {}, fmt.Errorf("failed to create network: %w", err)
		}
		deployment.Network = dockerNetwork
		networkName = dockerNetwork.Name
		t.Logf("Created network: %s", networkName)
	}

	// Cleanup function to terminate all containers
	cleanup := func() {
		for name, container := range deployment.Containers {
			if err := container.Terminate(ctx); err != nil {
				t.Logf("Warning: Failed to terminate container %s: %v", name, err)
			}
		}
		if deployment.Network != nil {
			if err := deployment.Network.Remove(ctx); err != nil {
				t.Logf("Warning: Failed to remove network: %v", err)
			}
		}
	}

	// Get containers in startup order
	orderedContainers := stack.GetStartupOrder()

	// Deploy containers in order
	for _, containerDef := range orderedContainers {
		t.Logf("Starting container %s (position %d)...", containerDef.Name, containerDef.Position)

		// Wait for dependencies to be healthy
		if err := waitForDependencies(ctx, t, stack, &containerDef, deployment); err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("dependency wait failed for %s: %w", containerDef.Name, err)
		}

		// Create and start container
		container, err := createAndStartContainer(ctx, t, &containerDef, networkName)
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("failed to start container %s: %w", containerDef.Name, err)
		}

		deployment.Containers[containerDef.Name] = container

		// Get mapped port for first exposed port
		if len(containerDef.Ports) > 0 {
			portStr := fmt.Sprintf("%d/tcp", containerDef.Ports[0].ContainerPort)
			mappedPort, err := container.MappedPort(ctx, nat.Port(portStr))
			if err == nil {
				deployment.Ports[containerDef.Name] = mappedPort.Port()
			}
		}

		// Wait for health check
		if containerDef.HealthCheck.Type != "" {
			t.Logf("Waiting for health check: %s", containerDef.Name)
			if err := waitForHealthCheck(ctx, t, container, &containerDef); err != nil {
				cleanup()
				return nil, func() {}, fmt.Errorf("health check failed for %s: %w", containerDef.Name, err)
			}
		}

		// Execute post-start actions
		if len(containerDef.PotentialAction) > 0 {
			t.Logf("Executing %d post-start action(s) for %s", len(containerDef.PotentialAction), containerDef.Name)
			if err := executePostStartActions(ctx, t, container, &containerDef); err != nil {
				cleanup()
				return nil, func() {}, fmt.Errorf("post-start actions failed for %s: %w", containerDef.Name, err)
			}
		}

		t.Logf("Container %s is ready", containerDef.Name)
	}

	t.Logf("Stack %s deployed successfully with %d containers", stack.Name, len(deployment.Containers))
	return deployment, cleanup, nil
}

// waitForDependencies waits for all dependencies of a container to be healthy.
func waitForDependencies(ctx context.Context, t *testing.T, stack *stacks.Stack, containerDef *stacks.StackItemElement, deployment *StackDeployment) error {
	// Collect all dependencies
	var deps []string
	deps = append(deps, containerDef.Requirements...)
	for _, swReq := range containerDef.SoftwareRequirements {
		deps = append(deps, swReq.Name)
	}

	if len(deps) == 0 {
		return nil // No dependencies
	}

	t.Logf("Waiting for dependencies of %s: %v", containerDef.Name, deps)

	// Wait for each dependency
	for _, depName := range deps {
		depContainer, exists := deployment.Containers[depName]
		if !exists {
			return fmt.Errorf("dependency %s not found (should have been started first)", depName)
		}

		// Check if we need to wait for health
		waitForHealth := false
		for _, swReq := range containerDef.SoftwareRequirements {
			if swReq.Name == depName && swReq.WaitForHealthy {
				waitForHealth = true
				break
			}
		}

		// Get dependency definition for health check
		if waitForHealth {
			depDef, err := stack.GetContainerByName(depName)
			if err != nil {
				return err
			}

			if depDef.HealthCheck.Type != "" {
				t.Logf("Waiting for dependency %s to be healthy...", depName)
				if err := waitForHealthCheck(ctx, t, depContainer, depDef); err != nil {
					return fmt.Errorf("dependency %s health check failed: %w", depName, err)
				}
			}
		}
	}

	return nil
}

// createAndStartContainer creates and starts a container from stack definition.
func createAndStartContainer(ctx context.Context, t *testing.T, containerDef *stacks.StackItemElement, networkName string) (testcontainers.Container, error) {
	// Build environment variables
	env := make(map[string]string)
	for k, v := range containerDef.Environment {
		env[k] = v
	}

	// Build exposed ports
	exposedPorts := []string{}
	for _, port := range containerDef.Ports {
		exposedPorts = append(exposedPorts, fmt.Sprintf("%d/tcp", port.ContainerPort))
	}

	// Build wait strategy (basic - health checks handled separately)
	var waitStrategy wait.Strategy
	if len(containerDef.Ports) > 0 {
		waitStrategy = wait.ForListeningPort(nat.Port(fmt.Sprintf("%d/tcp", containerDef.Ports[0].ContainerPort))).
			WithStartupTimeout(60 * time.Second)
	} else {
		// If no ports, just wait for container to start
		waitStrategy = wait.ForLog("").WithStartupTimeout(30 * time.Second)
	}

	// Build container request
	req := testcontainers.ContainerRequest{
		Image:        containerDef.Image,
		ExposedPorts: exposedPorts,
		Env:          env,
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {containerDef.Name},
		},
		Name:       containerDef.Name,
		WaitingFor: waitStrategy,
	}

	// Add command override if specified
	if len(containerDef.Command) > 0 {
		req.Cmd = containerDef.Command
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return container, nil
}

// waitForHealthCheck waits for a container's health check to pass.
func waitForHealthCheck(ctx context.Context, t *testing.T, container testcontainers.Container, containerDef *stacks.StackItemElement) error {
	hc := containerDef.HealthCheck

	// Set defaults
	interval := time.Duration(hc.Interval) * time.Second
	if interval == 0 {
		interval = 10 * time.Second
	}
	timeout := time.Duration(hc.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	retries := hc.Retries
	if retries == 0 {
		retries = 3
	}
	startPeriod := time.Duration(hc.StartPeriod) * time.Second
	if startPeriod == 0 {
		startPeriod = 10 * time.Second
	}

	// Wait for start period
	time.Sleep(startPeriod)

	// Perform health checks with retries
	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		if attempt > 0 {
			time.Sleep(interval)
		}

		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var err error
		switch hc.Type {
		case "command":
			err = healthCheckCommand(checkCtx, container, hc.Command)
		case "http":
			err = healthCheckHTTP(checkCtx, container, hc.Path, hc.Port)
		case "tcp":
			err = healthCheckTCP(checkCtx, container, hc.Port)
		case "postgres":
			err = healthCheckCommand(checkCtx, container, []string{"pg_isready", "-U", "postgres"})
		case "redis":
			err = healthCheckCommand(checkCtx, container, []string{"redis-cli", "ping"})
		default:
			return fmt.Errorf("unsupported health check type: %s", hc.Type)
		}

		if err == nil {
			t.Logf("Health check passed for %s", containerDef.Name)
			return nil
		}

		lastErr = err
		t.Logf("Health check attempt %d/%d failed for %s: %v", attempt+1, retries, containerDef.Name, err)
	}

	return fmt.Errorf("health check failed after %d attempts: %w", retries, lastErr)
}

// healthCheckCommand executes a command in the container for health check.
func healthCheckCommand(ctx context.Context, container testcontainers.Container, command []string) error {
	exitCode, output, err := container.Exec(ctx, command)
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}
	if exitCode != 0 {
		outputStr, _ := io.ReadAll(output)
		return fmt.Errorf("command exited with code %d: %s", exitCode, string(outputStr))
	}
	return nil
}

// healthCheckHTTP performs an HTTP GET request for health check.
func healthCheckHTTP(ctx context.Context, container testcontainers.Container, path string, port int) error {
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}

	var hostPort string
	if port == 0 {
		// Use first exposed port
		ports, err := container.Ports(ctx)
		if err != nil || len(ports) == 0 {
			return fmt.Errorf("no ports exposed")
		}
		for _, bindings := range ports {
			if len(bindings) > 0 {
				hostPort = bindings[0].HostPort
				break
			}
		}
	} else {
		hostPort = strconv.Itoa(port)
	}

	url := fmt.Sprintf("http://%s:%s%s", host, hostPort, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	return nil
}

// healthCheckTCP performs a TCP connection check for health check.
func healthCheckTCP(ctx context.Context, container testcontainers.Container, port int) error {
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}

	var hostPort string
	if port == 0 {
		// Use first exposed port
		ports, err := container.Ports(ctx)
		if err != nil || len(ports) == 0 {
			return fmt.Errorf("no ports exposed")
		}
		for _, bindings := range ports {
			if len(bindings) > 0 {
				hostPort = bindings[0].HostPort
				break
			}
		}
	} else {
		hostPort = strconv.Itoa(port)
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", host, hostPort))
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	conn.Close()

	return nil
}

// executePostStartActions executes all post-start actions for a container.
func executePostStartActions(ctx context.Context, t *testing.T, container testcontainers.Container, containerDef *stacks.StackItemElement) error {
	for i, action := range containerDef.PotentialAction {
		t.Logf("Executing action %d/%d: %s", i+1, len(containerDef.PotentialAction), action.Name)

		// Set timeout
		timeout := time.Duration(action.Timeout) * time.Second
		if timeout == 0 {
			timeout = 60 * time.Second // Default 60 seconds
		}

		actionCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Execute command
		exitCode, output, err := container.Exec(actionCtx, action.Command)
		if err != nil {
			return fmt.Errorf("action %s exec failed: %w", action.Name, err)
		}

		// Read output
		outputBytes, _ := io.ReadAll(output)
		outputStr := string(outputBytes)

		if exitCode != 0 {
			return fmt.Errorf("action %s exited with code %d: %s", action.Name, exitCode, outputStr)
		}

		t.Logf("Action %s completed successfully", action.Name)
		if len(outputStr) > 0 && len(outputStr) < 500 {
			t.Logf("Output: %s", strings.TrimSpace(outputStr))
		}
	}

	return nil
}
