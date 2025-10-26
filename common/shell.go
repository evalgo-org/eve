// Package common provides system utility functions for shell command execution and data processing.
// This package includes functions for running shell commands, handling privileged operations,
// and performing common string transformations for system automation and integration tasks.
//
// The package focuses on simplifying system interactions while providing appropriate
// logging and error handling for operational visibility. It includes both basic shell
// execution and privileged operations for system administration tasks.
//
// Security Considerations:
//
//	All shell execution functions in this package should be used with extreme caution
//	in production environments. They execute arbitrary shell commands and can pose
//	significant security risks if used with untrusted input. Always validate and
//	sanitize input before passing to these functions.
//
// Command Injection Prevention:
//
//	The functions use bash -c for command execution, which means they are vulnerable
//	to command injection if used with unsanitized input. Applications using these
//	functions must implement proper input validation and consider using more secure
//	alternatives for production use.
//
// Logging Integration:
//
//	All functions integrate with the common logging system to provide operational
//	visibility into command execution, including both successful operations and
//	failures with detailed error information.
//
// Use Cases:
//   - Development and testing automation
//   - System administration scripts
//   - Build and deployment processes
//   - File system operations and maintenance
//   - URL processing and normalization
package common

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ShellExecute runs a shell command and returns output or error.
// This function provides a simple interface for executing shell commands with
// automatic output capture and error handling.
//
// Execution Process:
//  1. Creates a bash subprocess with the provided command
//  2. Captures both stdout and stderr output
//  3. Executes the command and waits for completion
//  4. Returns output and error for caller to handle
//
// Output Handling:
//   - Successful execution: Returns stdout output
//   - Failed execution: Returns error with stderr details
//   - Both stdout and stderr are captured for complete command output
//
// Parameters:
//   - cmdToRun: Shell command string to execute (passed to bash -c)
//
// Returns:
//   - string: stdout output from the command
//   - error: error with stderr details if command fails, nil on success
//
// Error Handling:
//
//	The function returns errors instead of terminating, allowing callers
//	to handle failures appropriately for their use case.
//
// Security Warnings:
//
//	CRITICAL: This function is vulnerable to command injection attacks.
//	- Never pass unsanitized user input to this function
//	- Validate all input parameters before execution
//	- Consider using exec.Command with separate arguments for safer execution
//	- Avoid using this function in web applications or with external input
//
// Shell Environment:
//
//	Commands are executed in a bash shell environment with:
//	- Full shell command syntax support (pipes, redirects, variables)
//	- Access to environment variables and PATH
//	- Working directory of the calling process
//	- Standard shell expansion and globbing
//
// Example Usage:
//
//	// Safe usage with controlled input
//	ShellExecute("ls -la /tmp")
//	ShellExecute("docker ps --format table")
//
//	// DANGEROUS - vulnerable to injection
//	userInput := getUserInput() // Could contain "; rm -rf /"
//	ShellExecute(userInput)     // DO NOT DO THIS
//
// Best Practices:
//   - Use only with trusted, validated input
//   - Consider exec.Command for safer execution
//   - Implement timeout mechanisms for long-running commands
//   - Use in development and automation contexts, not production APIs
//
// Performance Considerations:
//   - Each call creates a new bash process (overhead for frequent use)
//   - Output is buffered in memory (consider streaming for large outputs)
//   - Synchronous execution blocks until command completion
//
// Alternative Approaches:
//
//	For production use, consider:
//	- exec.Command with separate arguments
//	- Restricted command execution with allowlists
//	- Sandboxed execution environments
//	- Process isolation and resource limits
func ShellExecute(cmdToRun string) (string, error) {
	// Create bash subprocess for command execution
	cmd := exec.Command("bash", "-c", cmdToRun)

	// Prepare output capture buffers
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Execute command and handle results
	err := cmd.Run()
	if err != nil {
		// Return error with stderr details
		return "", fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	// Return successful command output
	return out.String(), nil
}

// ShellSudoExecute runs a shell command with sudo privileges using password authentication.
// This function enables execution of privileged commands by automatically providing
// the sudo password through stdin, useful for automation tasks requiring elevated privileges.
//
// Sudo Authentication Process:
//  1. Constructs a command that pipes the password to sudo -S
//  2. Uses sudo -S flag to read password from stdin
//  3. Executes the privileged command with elevated permissions
//  4. Relies on ShellExecute for actual command execution and logging
//
// Parameters:
//   - password: Sudo password for the current user
//   - cmdToRun: Shell command to execute with sudo privileges
//
// Security Risks:
//
//	EXTREME CAUTION REQUIRED: This function has severe security implications:
//	- Passwords are passed as command line arguments (visible in process lists)
//	- Command injection vulnerabilities apply to both password and command
//	- Password may be logged or stored in shell history
//	- Process monitoring tools can capture password in plain text
//
// Security Recommendations:
//   - Never use this function in production environments
//   - Use only in isolated development or testing scenarios
//   - Consider passwordless sudo configuration for automation
//   - Use dedicated service accounts with specific sudo permissions
//   - Implement secure credential management instead of hardcoded passwords
//
// Alternative Approaches:
//
//	Safer alternatives for privileged operations:
//	- Configure passwordless sudo for specific commands
//	- Use dedicated service accounts with limited privileges
//	- Implement proper credential management systems
//	- Use container-based isolation instead of sudo
//	- Deploy applications with appropriate user permissions
//
// Command Construction:
//
//	The function builds a command in the format:
//	echo <password> | sudo -S <command>
//
//	This approach has inherent security vulnerabilities and should be
//	replaced with more secure authentication mechanisms.
//
// Example Usage:
//
//	// DANGEROUS - for development/testing only
//	ShellSudoExecute("mypassword", "apt update")
//	ShellSudoExecute("mypassword", "systemctl restart nginx")
//
// Process Visibility Warning:
//
//	The constructed command line, including the password, may be visible to:
//	- Process monitoring tools (ps, top, htop)
//	- System administrators with process access
//	- Log files and audit systems
//	- Other users on multi-user systems
//
// Recommended Replacements:
//
//	Instead of this function, consider:
//	- Sudo configuration: user ALL=(ALL) NOPASSWD: /specific/command
//	- Service accounts with appropriate permissions
//	- Container-based privilege isolation
//	- SSH key-based authentication for remote operations
func ShellSudoExecute(password, cmdToRun string) (string, error) {
	// Construct sudo command with password input
	// WARNING: This approach exposes passwords in process lists
	return ShellExecute(fmt.Sprintf("echo %s | sudo -S %s", password, cmdToRun))
}

// URLToFilePath converts a URL to a filesystem-safe filename.
// This function transforms URLs into valid filenames by removing protocol
// prefixes and replacing path separators with underscores, useful for
// caching, logging, and file storage based on URL identifiers.
//
// Transformation Process:
//  1. Removes "https://" prefix if present
//  2. Removes "http://" prefix if present
//  3. Replaces all forward slashes (/) with underscores (_)
//  4. Returns the transformed string suitable for filesystem use
//
// Parameters:
//   - url: URL string to convert to filesystem-safe format
//
// Returns:
//   - string: Filesystem-safe filename derived from the URL
//
// Transformation Examples:
//
//	"https://example.com/path/to/resource" → "example.com_path_to_resource"
//	"http://api.service.com/v1/users"      → "api.service.com_v1_users"
//	"example.com/docs/guide.html"          → "example.com_docs_guide.html"
//	"ftp://files.example.com/data"         → "ftp:__files.example.com_data"
//
// Use Cases:
//   - Cache file naming based on URL
//   - Log file organization by endpoint
//   - Temporary file creation for URL-based downloads
//   - Configuration file naming for URL-specific settings
//
// Filename Safety:
//
//	The function handles common URL-to-filename conversion needs but
//	may not address all filesystem restrictions:
//	- Converts forward slashes to underscores
//	- Preserves other characters (including potentially problematic ones)
//	- Does not handle maximum filename length restrictions
//	- May not be safe for all filesystems (Windows reserved names, etc.)
//
// Limitations:
//   - Only handles HTTP/HTTPS protocols explicitly
//   - Other protocols (ftp, ssh, etc.) are not processed
//   - Does not handle URL parameters or fragments
//   - May create long filenames that exceed filesystem limits
//   - Does not sanitize other potentially problematic characters
//
// Enhanced Safety Considerations:
//
//	For production use, consider additional transformations:
//	- Replace or remove other special characters (:, ?, &, etc.)
//	- Implement filename length limits
//	- Handle Windows reserved filenames (CON, PRN, AUX, etc.)
//	- Add file extension based on content type
//	- Hash long URLs to prevent filesystem limitations
//
// Example Usage:
//
//	// Basic URL to filename conversion
//	filename := URLToFilePath("https://api.example.com/v1/data")
//	// Result: "api.example.com_v1_data"
//
//	// Use for cache file creation
//	cacheFile := "/tmp/cache_" + URLToFilePath(apiURL) + ".json"
//
//	// Log file naming
//	logFile := URLToFilePath(endpoint) + ".log"
//
// Alternative Implementations:
//
//	For more robust URL-to-filename conversion:
//	- Use URL parsing libraries for component extraction
//	- Implement comprehensive character sanitization
//	- Add hash-based truncation for long URLs
//	- Include file extension detection based on URL analysis
//
// Thread Safety:
//
//	This function is safe for concurrent use as it only performs
//	string operations without modifying shared state or resources.
func URLToFilePath(url string) string {
	// Remove common URL protocol prefixes
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Convert path separators to filesystem-safe characters
	return strings.ReplaceAll(url, "/", "_")
}
