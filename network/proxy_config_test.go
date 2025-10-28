package network

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "string duration",
			input:    `"30s"`,
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "numeric duration",
			input:    `30000000000`,
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "minutes duration",
			input:    `"5m"`,
			expected: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "invalid duration",
			input:    `"invalid"`,
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && d.Duration != tt.expected {
				t.Errorf("UnmarshalJSON() = %v, want %v", d.Duration, tt.expected)
			}
		})
	}
}

func TestLoadProxyConfig(t *testing.T) {
	// Create temporary config file
	configJSON := `{
		"server": {
			"host": "0.0.0.0",
			"port": 8880,
			"read_timeout": "30s",
			"write_timeout": "30s",
			"idle_timeout": "60s"
		},
		"auth": {
			"type": "api-key",
			"header": "X-API-Key",
			"keys": ["test-key"],
			"bypass": ["/health"]
		},
		"routes": [
			{
				"path": "/api/v1/*",
				"methods": ["GET", "POST"],
				"backends": [
					{
						"ziti_service": "test-service",
						"identity_file": "./test.json",
						"timeout": "30s"
					}
				],
				"load_balancing": "round-robin"
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "proxy-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading config
	config, err := LoadProxyConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadProxyConfig() error = %v", err)
	}

	// Verify server config
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %v, want 0.0.0.0", config.Server.Host)
	}
	if config.Server.Port != 8880 {
		t.Errorf("Server.Port = %v, want 8880", config.Server.Port)
	}

	// Verify auth config
	if config.Auth.Type != "api-key" {
		t.Errorf("Auth.Type = %v, want api-key", config.Auth.Type)
	}
	if len(config.Auth.Keys) != 1 || config.Auth.Keys[0] != "test-key" {
		t.Errorf("Auth.Keys = %v, want [test-key]", config.Auth.Keys)
	}

	// Verify routes
	if len(config.Routes) != 1 {
		t.Fatalf("len(Routes) = %v, want 1", len(config.Routes))
	}

	route := config.Routes[0]
	if route.Path != "/api/v1/*" {
		t.Errorf("Route.Path = %v, want /api/v1/*", route.Path)
	}
	if route.LoadBalancing != "round-robin" {
		t.Errorf("Route.LoadBalancing = %v, want round-robin", route.LoadBalancing)
	}

	// Verify backends
	if len(route.Backends) != 1 {
		t.Fatalf("len(Backends) = %v, want 1", len(route.Backends))
	}

	backend := route.Backends[0]
	if backend.ZitiService != "test-service" {
		t.Errorf("Backend.ZitiService = %v, want test-service", backend.ZitiService)
	}
	if backend.Timeout.Duration != 30*time.Second {
		t.Errorf("Backend.Timeout = %v, want 30s", backend.Timeout.Duration)
	}
}

func TestLoadProxyConfigWithDefaults(t *testing.T) {
	configJSON := `{
		"server": {
			"host": "0.0.0.0",
			"port": 8880
		},
		"defaults": {
			"timeout": "60s",
			"max_retries": 5,
			"load_balancing": "weighted-round-robin"
		},
		"routes": [
			{
				"path": "/test",
				"backends": [
					{
						"ziti_service": "service1",
						"identity_file": "./id.json"
					}
				]
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "proxy-config-defaults-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	config, err := LoadProxyConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadProxyConfig() error = %v", err)
	}

	// Verify defaults were applied to route
	route := config.Routes[0]
	if route.LoadBalancing != "weighted-round-robin" {
		t.Errorf("Route.LoadBalancing = %v, want weighted-round-robin (from defaults)", route.LoadBalancing)
	}
	if route.Timeout.Duration != 60*time.Second {
		t.Errorf("Route.Timeout = %v, want 60s (from defaults)", route.Timeout.Duration)
	}

	// Verify defaults were applied to backend
	backend := route.Backends[0]
	if backend.MaxRetries != 5 {
		t.Errorf("Backend.MaxRetries = %v, want 5 (from defaults)", backend.MaxRetries)
	}
	if backend.Timeout.Duration != 60*time.Second {
		t.Errorf("Backend.Timeout = %v, want 60s (from defaults)", backend.Timeout.Duration)
	}
	if backend.Weight != 1 {
		t.Errorf("Backend.Weight = %v, want 1 (default)", backend.Weight)
	}
}

func TestLoadProxyConfigFileNotFound(t *testing.T) {
	_, err := LoadProxyConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("LoadProxyConfig() expected error for nonexistent file, got nil")
	}
}

func TestLoadProxyConfigInvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "proxy-config-invalid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("invalid json")); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadProxyConfig(tmpFile.Name())
	if err == nil {
		t.Error("LoadProxyConfig() expected error for invalid JSON, got nil")
	}
}

func TestLoadBalancingStrategy(t *testing.T) {
	strategies := []LoadBalancingStrategy{
		RoundRobin,
		WeightedRoundRobin,
		LeastConnections,
	}

	expected := []string{
		"round-robin",
		"weighted-round-robin",
		"least-connections",
	}

	for i, strategy := range strategies {
		if string(strategy) != expected[i] {
			t.Errorf("Strategy %d = %v, want %v", i, strategy, expected[i])
		}
	}
}

func TestBackendConfigDefaults(t *testing.T) {
	configJSON := `{
		"server": {"host": "0.0.0.0", "port": 8880},
		"routes": [{
			"path": "/test",
			"backends": [{
				"ziti_service": "service1",
				"identity_file": "./id.json"
			}]
		}]
	}`

	tmpFile, err := os.CreateTemp("", "proxy-config-backend-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	config, err := LoadProxyConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadProxyConfig() error = %v", err)
	}

	backend := config.Routes[0].Backends[0]

	// Check defaults
	if backend.Weight != 1 {
		t.Errorf("Backend.Weight = %v, want 1 (default)", backend.Weight)
	}
	if backend.MaxRetries != 3 {
		t.Errorf("Backend.MaxRetries = %v, want 3 (default)", backend.MaxRetries)
	}
	if backend.Timeout.Duration != 30*time.Second {
		t.Errorf("Backend.Timeout = %v, want 30s (default)", backend.Timeout.Duration)
	}
}

func TestRouteConfigWithHealthCheck(t *testing.T) {
	configJSON := `{
		"server": {"host": "0.0.0.0", "port": 8880},
		"routes": [{
			"path": "/test",
			"backends": [{
				"ziti_service": "service1",
				"identity_file": "./id.json"
			}],
			"health_check": {
				"enabled": true,
				"interval": "15s",
				"timeout": "5s",
				"path": "/health",
				"expected_status": 200,
				"failure_count": 3,
				"success_count": 2
			}
		}]
	}`

	tmpFile, err := os.CreateTemp("", "proxy-config-health-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	config, err := LoadProxyConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadProxyConfig() error = %v", err)
	}

	healthCheck := config.Routes[0].HealthCheck
	if healthCheck == nil {
		t.Fatal("HealthCheck is nil")
	}
	if !healthCheck.Enabled {
		t.Error("HealthCheck.Enabled = false, want true")
	}
	if healthCheck.Interval.Duration != 15*time.Second {
		t.Errorf("HealthCheck.Interval = %v, want 15s", healthCheck.Interval.Duration)
	}
	if healthCheck.Path != "/health" {
		t.Errorf("HealthCheck.Path = %v, want /health", healthCheck.Path)
	}
	if healthCheck.ExpectedStatus != 200 {
		t.Errorf("HealthCheck.ExpectedStatus = %v, want 200", healthCheck.ExpectedStatus)
	}
}
