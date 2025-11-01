package network

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectMajor int
		expectMinor int
		expectPatch int
	}{
		{"Standard format", "v1.6.5", 1, 6, 5},
		{"Without v prefix", "1.6.8", 1, 6, 8},
		{"Latest SDK", "v1.2.10", 1, 2, 10},
		{"Current SDK", "v1.2.2", 1, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch := parseVersion(tt.version)
			if major != tt.expectMajor || minor != tt.expectMinor || patch != tt.expectPatch {
				t.Errorf("parseVersion(%s) = %d.%d.%d, want %d.%d.%d",
					tt.version, major, minor, patch,
					tt.expectMajor, tt.expectMinor, tt.expectPatch)
			}
		})
	}
}

func TestCheckVersionCompatibility(t *testing.T) {
	tests := []struct {
		name              string
		controllerVersion string
		sdkVersion        string
		expectCompatible  bool
		expectWarnings    int
	}{
		{
			name:              "SDK v1.2.2 with Controller v1.6.5",
			controllerVersion: "v1.6.5",
			sdkVersion:        "v1.2.2",
			expectCompatible:  true,
			expectWarnings:    0,
		},
		{
			name:              "SDK v1.2.3 with Controller v1.6.5",
			controllerVersion: "v1.6.5",
			sdkVersion:        "v1.2.3",
			expectCompatible:  false,
			expectWarnings:    2, // Requires v1.6.8, Known issues
		},
		{
			name:              "SDK v1.2.3 with Controller v1.6.8",
			controllerVersion: "v1.6.8",
			sdkVersion:        "v1.2.3",
			expectCompatible:  true,
			expectWarnings:    0,
		},
		{
			name:              "SDK v1.2.10 with Controller v1.6.5",
			controllerVersion: "v1.6.5",
			sdkVersion:        "v1.2.10",
			expectCompatible:  false,
			expectWarnings:    2,
		},
		{
			name:              "SDK v1.2.10 with Controller v1.6.9",
			controllerVersion: "v1.6.9",
			sdkVersion:        "v1.2.10",
			expectCompatible:  true,
			expectWarnings:    0,
		},
		{
			name:              "SDK v1.2.2 with Controller v1.6.8",
			controllerVersion: "v1.6.8",
			sdkVersion:        "v1.2.2",
			expectCompatible:  true,
			expectWarnings:    0, // Compatible, will have recommendation to upgrade
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compatible, warnings, _ := checkVersionCompatibility(
				tt.controllerVersion,
				tt.sdkVersion,
			)

			if compatible != tt.expectCompatible {
				t.Errorf("Expected compatible=%v, got %v", tt.expectCompatible, compatible)
			}

			if len(warnings) != tt.expectWarnings {
				t.Errorf("Expected %d warnings, got %d: %v",
					tt.expectWarnings, len(warnings), warnings)
			}
		})
	}
}

func TestVersionComparisonEdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		controllerVersion string
		sdkVersion        string
		expectCompatible  bool
	}{
		{
			name:              "Exact version match for breaking change",
			controllerVersion: "v1.6.7",
			sdkVersion:        "v1.2.3",
			expectCompatible:  false, // v1.2.3 needs v1.6.8+
		},
		{
			name:              "One version below requirement",
			controllerVersion: "v1.6.7",
			sdkVersion:        "v1.2.3",
			expectCompatible:  false,
		},
		{
			name:              "Exact minimum requirement",
			controllerVersion: "v1.6.8",
			sdkVersion:        "v1.2.3",
			expectCompatible:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compatible, _, _ := checkVersionCompatibility(
				tt.controllerVersion,
				tt.sdkVersion,
			)

			if compatible != tt.expectCompatible {
				t.Errorf("Expected compatible=%v, got %v", tt.expectCompatible, compatible)
			}
		})
	}
}

// TestVersionCheckerIntegration tests the full version checking flow
// This test requires a real Ziti controller to be available
func TestVersionCheckerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	identityFile := "/home/opunix/caches/fs.test.user.json"

	info, err := CheckCompatibility(identityFile)
	if err != nil {
		t.Skipf("Cannot run integration test: %v", err)
	}

	t.Logf("Controller Version: %s", info.ControllerVersion)
	t.Logf("SDK Version: %s", info.SDKVersion)
	t.Logf("Compatible: %v", info.Compatible)

	if len(info.Warnings) > 0 {
		t.Log("Warnings:")
		for _, w := range info.Warnings {
			t.Logf("  - %s", w)
		}
	}

	if len(info.Recommendations) > 0 {
		t.Log("Recommendations:")
		for _, r := range info.Recommendations {
			t.Logf("  - %s", r)
		}
	}

	// For our current setup, we expect controller v1.6.5 and SDK v1.2.2 to be compatible
	if info.ControllerVersion == "v1.6.5" && info.SDKVersion == "v1.2.2" {
		if !info.Compatible {
			t.Error("Expected controller v1.6.5 and SDK v1.2.2 to be compatible")
		}
	}
}
