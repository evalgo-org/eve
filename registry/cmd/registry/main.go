package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"eve.evalgo.org/registry"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Get registry path from environment or use default
	registryPath := os.Getenv("SERVICE_REGISTRY_PATH")
	if registryPath == "" {
		registryPath = "/home/opunix/registry.json"
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load registry: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "list":
		listServices(reg)
	case "get":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: service ID required\n")
			os.Exit(1)
		}
		getService(reg, os.Args[2])
	case "url":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: service ID required\n")
			os.Exit(1)
		}
		getURL(reg, os.Args[2])
	case "health":
		if len(os.Args) < 3 {
			// Check all services
			healthCheckAll(reg)
		} else {
			// Check specific service
			healthCheck(reg, os.Args[2])
		}
	case "find":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: capability required\n")
			os.Exit(1)
		}
		findByCapability(reg, os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: registry <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list                    List all registered services")
	fmt.Println("  get <service-id>        Get details for a specific service")
	fmt.Println("  url <service-id>        Get URL for a specific service")
	fmt.Println("  health [service-id]     Check health of service(s)")
	fmt.Println("  find <capability>       Find services by capability")
	fmt.Println("")
	fmt.Println("Environment Variables:")
	fmt.Println("  SERVICE_REGISTRY_PATH   Path to registry file (default: /home/opunix/registry.json)")
}

func listServices(reg *registry.Registry) {
	services := reg.List()

	if len(services) == 0 {
		fmt.Println("No services registered")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tURL\tPORT")
	fmt.Fprintln(w, "==\t====\t===\t====")

	for _, svc := range services {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
			svc.ID,
			svc.Name,
			svc.URL,
			svc.Properties.Port)
	}

	w.Flush()
}

func getService(reg *registry.Registry, serviceID string) {
	svc, err := reg.Get(serviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print as JSON
	data, err := json.MarshalIndent(svc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}

func getURL(reg *registry.Registry, serviceID string) {
	url, err := reg.GetServiceURL(serviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(url)
}

func healthCheck(reg *registry.Registry, serviceID string) {
	healthy, err := reg.HealthCheck(serviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if healthy {
		fmt.Printf("%s: ✓ healthy\n", serviceID)
		os.Exit(0)
	} else {
		fmt.Printf("%s: ✗ unhealthy\n", serviceID)
		os.Exit(1)
	}
}

func healthCheckAll(reg *registry.Registry) {
	results := reg.HealthCheckAll()

	if len(results) == 0 {
		fmt.Println("No services registered")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tSTATUS")
	fmt.Fprintln(w, "=======\t======")

	allHealthy := true
	for serviceID, healthy := range results {
		status := "✓ healthy"
		if !healthy {
			status = "✗ unhealthy"
			allHealthy = false
		}
		fmt.Fprintf(w, "%s\t%s\n", serviceID, status)
	}

	w.Flush()

	if !allHealthy {
		os.Exit(1)
	}
}

func findByCapability(reg *registry.Registry, capability string) {
	services := reg.FindByCapability(capability)

	if len(services) == 0 {
		fmt.Printf("No services found with capability: %s\n", capability)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tURL")
	fmt.Fprintln(w, "==\t====\t===")

	for _, svc := range services {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			svc.ID,
			svc.Name,
			svc.URL)
	}

	w.Flush()
}
