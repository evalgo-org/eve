package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPtrInt32 tests the int32 pointer helper function
func TestPtrInt32(t *testing.T) {
	tests := []struct {
		name  string
		input int32
		want  int32
	}{
		{
			name:  "ZeroValue",
			input: 0,
			want:  0,
		},
		{
			name:  "PositiveValue",
			input: 42,
			want:  42,
		},
		{
			name:  "NegativeValue",
			input: -100,
			want:  -100,
		},
		{
			name:  "MaxInt32",
			input: 2147483647,
			want:  2147483647,
		},
		{
			name:  "MinInt32",
			input: -2147483648,
			want:  -2147483648,
		},
		{
			name:  "SmallPositive",
			input: 10,
			want:  10,
		},
		{
			name:  "SmallNegative",
			input: -10,
			want:  -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ptrInt32(tt.input)
			assert.NotNil(t, result, "ptrInt32 should return non-nil pointer")
			assert.Equal(t, tt.want, *result, "dereferenced value should match input")

			// Verify it's actually a pointer to a different memory location
			differentValue := ptrInt32(tt.input)
			assert.NotSame(t, result, differentValue, "each call should return a new pointer")
			assert.Equal(t, *result, *differentValue, "but values should be equal")
		})
	}
}

// TestPtrInt32UniquePointers verifies that ptrInt32 creates new pointers each time
func TestPtrInt32UniquePointers(t *testing.T) {
	value := int32(100)

	ptr1 := ptrInt32(value)
	ptr2 := ptrInt32(value)
	ptr3 := ptrInt32(value)

	// All should point to the same value
	assert.Equal(t, value, *ptr1)
	assert.Equal(t, value, *ptr2)
	assert.Equal(t, value, *ptr3)

	// But should be different pointers
	assert.NotSame(t, ptr1, ptr2)
	assert.NotSame(t, ptr2, ptr3)
	assert.NotSame(t, ptr1, ptr3)

	// Modifying one should not affect others
	*ptr1 = 200
	assert.Equal(t, int32(200), *ptr1)
	assert.Equal(t, int32(100), *ptr2)
	assert.Equal(t, int32(100), *ptr3)
}

// TestHetznerServerCreate_InvalidToken tests error handling with invalid credentials
func TestHetznerServerCreate_InvalidToken(t *testing.T) {
	// This test verifies the function handles invalid tokens gracefully
	// Since the function logs errors internally and doesn't return anything,
	// we can't easily verify the error without dependency injection
	// This is a smoke test to ensure the function doesn't panic

	tests := []struct {
		name   string
		token  string
		sName  string
		sType  string
		expect string
	}{
		{
			name:   "EmptyToken",
			token:  "",
			sName:  "test-server",
			sType:  "default",
			expect: "should handle empty token",
		},
		{
			name:   "InvalidToken",
			token:  "invalid-token-12345",
			sName:  "test-server",
			sType:  "default",
			expect: "should handle invalid token",
		},
		{
			name:   "EmptyServerName",
			token:  "test-token",
			sName:  "",
			sType:  "default",
			expect: "should handle empty server name",
		},
		{
			name:   "NonDefaultType",
			token:  "test-token",
			sName:  "test-server",
			sType:  "custom",
			expect: "should handle non-default server type (no-op)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic even with invalid inputs
			assert.NotPanics(t, func() {
				HetznerServerCreate(tt.token, tt.sName, tt.sType)
			}, tt.expect)
		})
	}
}

// TestHetznerServerDelete_InvalidToken tests error handling with invalid credentials
func TestHetznerServerDelete_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		sName string
	}{
		{
			name:  "EmptyToken",
			token: "",
			sName: "test-server",
		},
		{
			name:  "InvalidToken",
			token: "invalid-token-12345",
			sName: "test-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic even with invalid inputs
			assert.NotPanics(t, func() {
				HetznerServerDelete(tt.token, tt.sName)
			})
		})
	}
}

// TestHetznerServerDelete_EmptyServerName tests the nil pointer scenario
func TestHetznerServerDelete_EmptyServerName(t *testing.T) {
	// This test documents that HetznerServerDelete panics when server is not found
	// The function attempts to delete a nil server pointer which causes a panic
	assert.Panics(t, func() {
		HetznerServerDelete("test-token", "")
	}, "function panics when server is not found (nil pointer)")
}

// TestHetznerServers_InvalidToken tests error handling with invalid credentials
func TestHetznerServers_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "EmptyToken",
			token: "",
		},
		{
			name:  "InvalidToken",
			token: "invalid-token-12345",
		},
		{
			name:  "ShortToken",
			token: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic even with invalid token
			assert.NotPanics(t, func() {
				HetznerServers(tt.token)
			})
		})
	}
}

// TestHetznerPrices_InvalidToken tests error handling with invalid credentials
func TestHetznerPrices_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "EmptyToken",
			token: "",
		},
		{
			name:  "InvalidToken",
			token: "invalid-token-12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic even with invalid token
			assert.NotPanics(t, func() {
				HetznerPrices(tt.token)
			})
		})
	}
}

// TestAzureEmails_InvalidCredentials tests error handling with invalid Azure credentials
func TestAzureEmails_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name         string
		tenantId     string
		clientId     string
		clientSecret string
		expectError  bool
	}{
		{
			name:         "EmptyTenantId",
			tenantId:     "",
			clientId:     "client-id",
			clientSecret: "secret",
			expectError:  true,
		},
		{
			name:         "EmptyClientId",
			tenantId:     "tenant-id",
			clientId:     "",
			clientSecret: "secret",
			expectError:  true,
		},
		{
			name:         "EmptyClientSecret",
			tenantId:     "tenant-id",
			clientId:     "client-id",
			clientSecret: "",
			expectError:  true,
		},
		{
			name:         "AllEmpty",
			tenantId:     "",
			clientId:     "",
			clientSecret: "",
			expectError:  true,
		},
		{
			name:         "InvalidTenantFormat",
			tenantId:     "not-a-guid",
			clientId:     "client-id",
			clientSecret: "secret",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AzureEmails(tt.tenantId, tt.clientId, tt.clientSecret)
			if tt.expectError {
				assert.Error(t, err, "should return error for invalid credentials")
			}
		})
	}
}

// TestAzureCalendar_InvalidCredentials tests error handling with invalid Azure credentials
func TestAzureCalendar_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name         string
		tenantId     string
		clientId     string
		clientSecret string
		email        string
		start        string
		end          string
		expectError  bool
	}{
		{
			name:         "EmptyTenantId",
			tenantId:     "",
			clientId:     "client-id",
			clientSecret: "secret",
			email:        "test@example.com",
			start:        "2024-01-01T00:00:00Z",
			end:          "2024-01-31T23:59:59Z",
			expectError:  true,
		},
		{
			name:         "EmptyClientSecret",
			tenantId:     "tenant-id",
			clientId:     "client-id",
			clientSecret: "",
			email:        "test@example.com",
			start:        "2024-01-01T00:00:00Z",
			end:          "2024-01-31T23:59:59Z",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AzureCalendar(tt.tenantId, tt.clientId, tt.clientSecret, tt.email, tt.start, tt.end)
			if tt.expectError {
				assert.Error(t, err, "should return error for invalid parameters")
			}
		})
	}
}

// TestAzureCalendar_PanicOnAPIError tests that the function panics on API errors
// Note: AzureCalendar has panic(err) in error paths instead of returning errors
func TestAzureCalendar_PanicOnAPIError(t *testing.T) {
	// This test documents that AzureCalendar panics when API calls fail
	// The function uses panic(err) instead of returning errors
	assert.Panics(t, func() {
		_ = AzureCalendar("tenant-id", "client-id", "secret", "test@example.com", "2024-01-01T00:00:00Z", "2024-01-31T23:59:59Z")
	}, "function panics on API errors instead of returning them")
}

// TestHetznerServerCreate_ServerTypes verifies server type handling
func TestHetznerServerCreate_ServerTypes(t *testing.T) {
	tests := []struct {
		name      string
		sType     string
		shouldRun bool
	}{
		{
			name:      "DefaultType",
			sType:     "default",
			shouldRun: true,
		},
		{
			name:      "CustomType",
			sType:     "custom",
			shouldRun: false, // function only handles "default"
		},
		{
			name:      "EmptyType",
			sType:     "",
			shouldRun: false,
		},
		{
			name:      "CCX23Type",
			sType:     "ccx23",
			shouldRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify function doesn't panic regardless of server type
			assert.NotPanics(t, func() {
				HetznerServerCreate("test-token", "test-server", tt.sType)
			})
		})
	}
}

// BenchmarkPtrInt32 benchmarks the ptrInt32 helper function
func BenchmarkPtrInt32(b *testing.B) {
	var result *int32
	for i := 0; i < b.N; i++ {
		result = ptrInt32(int32(i))
	}
	// Prevent compiler optimization
	_ = result
}

// BenchmarkPtrInt32Parallel benchmarks ptrInt32 with parallel execution
func BenchmarkPtrInt32Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var result *int32
		i := int32(0)
		for pb.Next() {
			result = ptrInt32(i)
			i++
		}
		_ = result
	})
}
