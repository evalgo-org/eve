package security

import (
	"context"
	infisical "github.com/infisical/go-sdk"
	"os"
	"strings"

	eve "eve.evalgo.org/common"
)

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
		var machine string
		var login string
		var password string
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
