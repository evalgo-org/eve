package network

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openziti/sdk-golang/ziti"

	eve "eve.evalgo.org/common"
)

// VersionInfo holds controller and SDK version information
type VersionInfo struct {
	ControllerVersion string
	SDKVersion        string
	Compatible        bool
	Warnings          []string
	Recommendations   []string
}

// ControllerVersionResponse represents the controller version API response
type ControllerVersionResponse struct {
	Data struct {
		Version string `json:"version"`
	} `json:"data"`
}

// CheckCompatibility verifies OpenZiti controller and SDK version compatibility
// Returns version information, compatibility status, and recommendations
func CheckCompatibility(identityFile string) (*VersionInfo, error) {
	info := &VersionInfo{
		SDKVersion:      getSDKVersion(),
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// Load identity to get controller URL
	cfg, err := ziti.NewConfigFromFile(identityFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}

	// Get controller URL from identity
	controllerURL := cfg.ZtAPI
	if controllerURL == "" {
		return nil, fmt.Errorf("no controller URL found in identity file")
	}

	// Extract base URL (remove /edge/client/v1 if present)
	baseURL := strings.TrimSuffix(controllerURL, "/edge/client/v1")
	versionURL := baseURL + "/edge/client/v1/version"

	// Fetch controller version
	controllerVersion, err := fetchControllerVersion(versionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller version: %w", err)
	}

	info.ControllerVersion = controllerVersion

	// Check compatibility
	info.Compatible, info.Warnings, info.Recommendations = checkVersionCompatibility(
		controllerVersion,
		info.SDKVersion,
	)

	return info, nil
}

// fetchControllerVersion retrieves the controller version from the API
func fetchControllerVersion(versionURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(versionURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("version API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var versionResp ControllerVersionResponse
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return "", fmt.Errorf("failed to parse version response: %w", err)
	}

	return versionResp.Data.Version, nil
}

// getSDKVersion returns the current SDK version
// This should match the version in go.mod
func getSDKVersion() string {
	// In production, this would be set at build time
	// For now, we return the version we're using
	return "v1.2.2"
}

// checkVersionCompatibility determines if controller and SDK versions are compatible
func checkVersionCompatibility(controllerVersion, sdkVersion string) (bool, []string, []string) {
	warnings := []string{}
	recommendations := []string{}
	compatible := true

	// Parse SDK version
	sdkMajor, sdkMinor, sdkPatch := parseVersion(sdkVersion)
	ctrlMajor, ctrlMinor, ctrlPatch := parseVersion(controllerVersion)

	// SDK v1.2.3+ requires controller v1.6.8+
	if sdkMajor == 1 && sdkMinor == 2 && sdkPatch >= 3 {
		if ctrlMajor < 1 || (ctrlMajor == 1 && ctrlMinor < 6) || (ctrlMajor == 1 && ctrlMinor == 6 && ctrlPatch < 8) {
			compatible = false
			warnings = append(warnings, fmt.Sprintf(
				"SDK version %s requires controller v1.6.8 or later (found %s)",
				sdkVersion, controllerVersion,
			))
			warnings = append(warnings, "Known issues: UNAUTHORIZED errors, HA controller discovery failures")
			recommendations = append(recommendations, "Option 1: Upgrade controller to v1.6.8 or later")
			recommendations = append(recommendations, "Option 2: Downgrade SDK to v1.2.2")
			recommendations = append(recommendations, "See OPENZITI_COMPATIBILITY.md for details")
		}
	}

	// SDK v1.2.2 works with all v1.6.x controllers
	if sdkMajor == 1 && sdkMinor == 2 && sdkPatch == 2 {
		if ctrlMajor == 1 && ctrlMinor == 6 {
			// Perfect compatibility
			if ctrlPatch >= 8 {
				recommendations = append(recommendations,
					"Controller supports SDK v1.2.3+ features. Consider upgrading SDK for HA/OIDC support.")
			}
		}
	}

	// Warn if controller is too old
	if ctrlMajor < 1 || (ctrlMajor == 1 && ctrlMinor < 6) {
		warnings = append(warnings, fmt.Sprintf(
			"Controller version %s is older than recommended (v1.6.0+)",
			controllerVersion,
		))
		recommendations = append(recommendations, "Consider upgrading controller to latest stable version")
	}

	// Warn if SDK is too new for controller
	if sdkMajor == 1 && sdkMinor > 2 {
		warnings = append(warnings, "SDK version is newer than tested versions. Compatibility unknown.")
		recommendations = append(recommendations, "Test thoroughly before production use")
	}

	return compatible, warnings, recommendations
}

// parseVersion extracts major, minor, patch from version string
// Handles formats like "v1.6.5", "1.6.5", etc.
func parseVersion(version string) (major, minor, patch int) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Split by dots
	parts := strings.Split(version, ".")
	if len(parts) >= 1 {
		_, _ = fmt.Sscanf(parts[0], "%d", &major)
	}
	if len(parts) >= 2 {
		_, _ = fmt.Sscanf(parts[1], "%d", &minor)
	}
	if len(parts) >= 3 {
		_, _ = fmt.Sscanf(parts[2], "%d", &patch)
	}

	return
}

// LogCompatibilityCheck performs a compatibility check and logs the results
func LogCompatibilityCheck(identityFile string) error {
	eve.Logger.Info("Checking OpenZiti version compatibility...")

	info, err := CheckCompatibility(identityFile)
	if err != nil {
		eve.Logger.Error(fmt.Sprintf("Failed to check compatibility: %v", err))
		return err
	}

	eve.Logger.Info(fmt.Sprintf("Controller Version: %s", info.ControllerVersion))
	eve.Logger.Info(fmt.Sprintf("SDK Version: %s", info.SDKVersion))

	if info.Compatible {
		eve.Logger.Info("✓ Versions are compatible")
	} else {
		eve.Logger.Warn("✗ Version compatibility issues detected")
	}

	for _, warning := range info.Warnings {
		eve.Logger.Warn(fmt.Sprintf("⚠ %s", warning))
	}

	for _, rec := range info.Recommendations {
		eve.Logger.Info(fmt.Sprintf("→ %s", rec))
	}

	return nil
}

// MustBeCompatible checks compatibility and panics if incompatible
// Use this during initialization to enforce version requirements
func MustBeCompatible(identityFile string) {
	info, err := CheckCompatibility(identityFile)
	if err != nil {
		eve.Logger.Error(fmt.Sprintf("Version check failed: %v", err))
		panic(fmt.Sprintf("Cannot verify OpenZiti version compatibility: %v", err))
	}

	if !info.Compatible {
		eve.Logger.Error("OpenZiti version compatibility check failed:")
		for _, warning := range info.Warnings {
			eve.Logger.Error(fmt.Sprintf("  - %s", warning))
		}
		panic("Incompatible OpenZiti controller/SDK versions. See OPENZITI_COMPATIBILITY.md")
	}

	eve.Logger.Info("✓ OpenZiti version compatibility verified")
}
