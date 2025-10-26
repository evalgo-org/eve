/*
Package security provides utilities for integrating with secret management systems
and performing encryption-related operations.

This file includes integration with the Infisical secrets management service,
allowing retrieval of environment-specific secrets using API credentials.

Usage Example:

	package main

	import (
		"fmt"
		"myapp/security"
	)

	func main() {
		output := security.InfisicalSecrets(
			"app.infisical.com",
			"your-client-id",
			"your-client-secret",
			"your-project-id",
			"dev",
			"env",
		)
		fmt.Println(output)
	}

The function retrieves all secrets from the specified Infisical project and environment.
Secrets can be formatted as environment variables or in `.netrc` style depending on the format parameter.
*/

package security

import (
	"context"
	"os"
	"strings"

	infisical "github.com/infisical/go-sdk"

	eve "eve.evalgo.org/common"
)

// InfisicalSecrets retrieves secrets from an Infisical project environment.
//
// It authenticates using the provided client ID and secret, fetches secrets
// for the given project and environment, and returns them formatted as either
// environment variable declarations or `.netrc` entries.
//
// Parameters:
//   - host:           The Infisical host domain (e.g. "app.infisical.com").
//   - client_id:      The Infisical client ID for authentication.
//   - client_secret:  The Infisical client secret for authentication.
//   - project_id:     The project identifier from which to fetch secrets.
//   - environment:    The target environment name (e.g. "dev", "prod").
//   - format:         Output format, either "env" (default) or "netrc".
//
// Returns:
//
//	A string containing either key=value pairs (one per line) or `.netrc`
//	formatted credentials if format == "netrc".
//
// Behavior:
//   - On authentication or retrieval failure, the program logs the error
//     using eve.Logger and exits with status code 1.
//   - If format == "netrc", it looks for secrets with keys "MACHINE", "LOGIN",
//     and "PASSWORD" to construct the .netrc entry.
//
// Example Output (env format):
//
//	MACHINE=github.com
//	LOGIN=myuser
//	PASSWORD=mytoken
//
// Example Output (netrc format):
//
//	machine github.com
//	login myuser
//	password mytoken
func InfisicalSecrets(host, client_id, client_secret, project_id, environment, format string) string {
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          "https://" + host,
		AutoTokenRefresh: false,
	})

	_, err := client.Auth().UniversalAuthLogin(client_id, client_secret)
	if err != nil {
		eve.Logger.Info("Authentication failed:", err)
		os.Exit(1)
	}

	apiKeySecrets, err := client.Secrets().List(infisical.ListSecretsOptions{
		AttachToProcessEnv: false,
		Environment:        environment,
		ProjectID:          project_id,
		SecretPath:         "/",
		IncludeImports:     true,
	})
	if err != nil {
		eve.Logger.Info("Error:", err)
		os.Exit(1)
	}

	if format == "netrc" {
		var machine, login, password string
		for _, secret := range apiKeySecrets {
			if secret.SecretKey == "MACHINE" {
				machine = "machine " + secret.SecretValue
			}
			if secret.SecretKey == "LOGIN" {
				login = "login " + secret.SecretValue
			}
			if secret.SecretKey == "PASSWORD" {
				password = "password " + secret.SecretValue
			}
		}
		return machine + "\n" + login + "\n" + password + "\n"
	}

	secs := make([]string, len(apiKeySecrets))
	for idx, secret := range apiKeySecrets {
		secs[idx] = secret.SecretKey + "=" + secret.SecretValue
	}
	return strings.Join(secs, "\n")
}
