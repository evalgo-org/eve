package security

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "simple password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "complex password with special chars",
			password: "P@ssw0rd!#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt can hash empty strings
		},
		{
			name:     "very long password (exceeds 72 bytes)",
			password: strings.Repeat("a", 100),
			wantErr:  true, // bcrypt has 72-byte limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify hash is not empty
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}

				// Verify hash starts with bcrypt prefix
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
					t.Errorf("HashPassword() hash doesn't have bcrypt prefix: %s", hash)
				}

				// Verify we can verify the password with the hash
				if err := VerifyPassword(hash, tt.password); err != nil {
					t.Errorf("VerifyPassword() failed for generated hash: %v", err)
				}
			}
		})
	}
}

func TestHashPasswordWithCost(t *testing.T) {
	tests := []struct {
		name     string
		password string
		cost     int
		wantErr  bool
	}{
		{
			name:     "minimum cost",
			password: "password",
			cost:     bcrypt.MinCost,
			wantErr:  false,
		},
		{
			name:     "default cost",
			password: "password",
			cost:     DefaultBcryptCost,
			wantErr:  false,
		},
		{
			name:     "high cost",
			password: "password",
			cost:     12,
			wantErr:  false,
		},
		{
			name:     "maximum cost",
			password: "password",
			cost:     bcrypt.MaxCost,
			wantErr:  false,
		},
		{
			name:     "cost too low",
			password: "password",
			cost:     bcrypt.MinCost - 1,
			wantErr:  true,
		},
		{
			name:     "cost too high",
			password: "password",
			cost:     bcrypt.MaxCost + 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPasswordWithCost(tt.password, tt.cost)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPasswordWithCost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify cost is correct
				actualCost, err := bcrypt.Cost([]byte(hash))
				if err != nil {
					t.Errorf("Failed to get cost from hash: %v", err)
				}
				if actualCost != tt.cost {
					t.Errorf("HashPasswordWithCost() cost = %d, want %d", actualCost, tt.cost)
				}
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	// Generate a test hash
	testPassword := "correctPassword123"
	testHash, err := HashPassword(testPassword)
	if err != nil {
		t.Fatalf("Failed to generate test hash: %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  bool
	}{
		{
			name:     "correct password",
			hash:     testHash,
			password: testPassword,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			hash:     testHash,
			password: "wrongPassword",
			wantErr:  true,
		},
		{
			name:     "empty password",
			hash:     testHash,
			password: "",
			wantErr:  true,
		},
		{
			name:     "case sensitive password",
			hash:     testHash,
			password: "CORRECTPASSWORD123",
			wantErr:  true,
		},
		{
			name:     "invalid hash format",
			hash:     "not-a-valid-hash",
			password: testPassword,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPassword(tt.hash, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Specifically check for mismatch error
			if tt.wantErr && err != nil && tt.hash != "not-a-valid-hash" {
				if err != bcrypt.ErrMismatchedHashAndPassword && tt.password != "" {
					// It's okay if it's not the mismatch error for invalid hash
					if tt.name != "invalid hash format" {
						t.Logf("Got error: %v (expected mismatch error)", err)
					}
				}
			}
		})
	}
}

func TestNeedsRehash(t *testing.T) {
	// Generate hashes with different costs
	password := "testpassword"
	hashCost4, _ := HashPasswordWithCost(password, 4)
	hashCost10, _ := HashPasswordWithCost(password, 10)
	hashCost12, _ := HashPasswordWithCost(password, 12)

	tests := []struct {
		name         string
		hash         string
		desiredCost  int
		wantRehash   bool
		wantErr      bool
	}{
		{
			name:        "hash with lower cost needs rehash",
			hash:        hashCost4,
			desiredCost: 10,
			wantRehash:  true,
			wantErr:     false,
		},
		{
			name:        "hash with same cost doesn't need rehash",
			hash:        hashCost10,
			desiredCost: 10,
			wantRehash:  false,
			wantErr:     false,
		},
		{
			name:        "hash with higher cost needs rehash",
			hash:        hashCost12,
			desiredCost: 10,
			wantRehash:  true,
			wantErr:     false,
		},
		{
			name:        "invalid hash",
			hash:        "not-a-valid-hash",
			desiredCost: 10,
			wantRehash:  false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsRehash, err := NeedsRehash(tt.hash, tt.desiredCost)
			if (err != nil) != tt.wantErr {
				t.Errorf("NeedsRehash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if needsRehash != tt.wantRehash {
				t.Errorf("NeedsRehash() = %v, want %v", needsRehash, tt.wantRehash)
			}
		})
	}
}

func TestPasswordHashingWorkflow(t *testing.T) {
	// Simulate a complete password workflow
	t.Run("user registration and login", func(t *testing.T) {
		password := "MySecureP@ssw0rd!"

		// Registration: hash password
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to hash password during registration: %v", err)
		}

		// Store hash in database (simulated)
		storedHash := hash

		// Login attempt with correct password
		if err := VerifyPassword(storedHash, password); err != nil {
			t.Errorf("Failed to verify correct password: %v", err)
		}

		// Login attempt with incorrect password
		if err := VerifyPassword(storedHash, "WrongPassword"); err == nil {
			t.Error("VerifyPassword() should fail for incorrect password")
		}
	})

	t.Run("password rehashing on login", func(t *testing.T) {
		password := "TestPassword123"
		oldCost := 4
		newCost := 12

		// Old hash with low cost
		oldHash, err := HashPasswordWithCost(password, oldCost)
		if err != nil {
			t.Fatalf("Failed to create old hash: %v", err)
		}

		// Verify password with old hash
		if err := VerifyPassword(oldHash, password); err != nil {
			t.Fatalf("Failed to verify password with old hash: %v", err)
		}

		// Check if rehash is needed
		needsRehash, err := NeedsRehash(oldHash, newCost)
		if err != nil {
			t.Fatalf("Failed to check rehash: %v", err)
		}
		if !needsRehash {
			t.Error("NeedsRehash() should return true for old hash")
		}

		// Generate new hash
		newHash, err := HashPasswordWithCost(password, newCost)
		if err != nil {
			t.Fatalf("Failed to create new hash: %v", err)
		}

		// Verify with new hash
		if err := VerifyPassword(newHash, password); err != nil {
			t.Errorf("Failed to verify password with new hash: %v", err)
		}

		// Verify new hash doesn't need rehash
		needsRehash, err = NeedsRehash(newHash, newCost)
		if err != nil {
			t.Fatalf("Failed to check rehash for new hash: %v", err)
		}
		if needsRehash {
			t.Error("NeedsRehash() should return false for new hash")
		}
	})
}

func TestDefaultBcryptCost(t *testing.T) {
	if DefaultBcryptCost != 10 {
		t.Errorf("DefaultBcryptCost = %d, want 10", DefaultBcryptCost)
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "BenchmarkPassword123!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "BenchmarkPassword123!"
	hash, _ := HashPassword(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(hash, password)
	}
}
