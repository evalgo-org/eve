# KVM Library Testing Guide

This document describes the test suite for the KVM library.

## Test Files

| Test File | Coverage | Lines | Tests |
|-----------|----------|-------|-------|
| `types_test.go` | Type definitions and state conversions | ~180 | 8 |
| `validation_test.go` | VM name validation | ~170 | 30+ |
| `xml_test.go` | XML generation and parsing | ~280 | 15 |
| `cloudinit_test.go` | Cloud-init ISO creation | ~280 | 12 |
| `connection_test.go` | Libvirt connection management | ~220 | 15 |
| `network_test.go` | Network and IP detection | ~230 | 12 |
| `domain_test.go` | High-level VM operations | ~380 | 18 |

**Total:** ~1,740 lines of test code covering all 7 library modules

---

## Running Tests

### Run All Tests

```bash
cd kvm
go test -v
```

### Run Specific Test File

```bash
go test -v -run TestValidation
go test -v -run TestXML
```

### Run Short Tests (Skip Integration)

Many tests require libvirt to be running. Use `-short` to skip integration tests:

```bash
go test -short -v
```

### Run Tests with Coverage

```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Benchmarks

```bash
go test -bench=.
go test -bench=. -benchmem
```

---

## Test Categories

### Unit Tests (No External Dependencies)

These tests run without libvirt and test pure functions:

- ✅ `TestStateToString` - State enum conversions
- ✅ `TestIsValidVMName` - VM name validation
- ✅ `TestExtractMACFromXML` - XML parsing
- ✅ `TestGenerateDomainXML` - XML generation
- ✅ `TestVMResult` - Result structures
- ✅ All validation tests

**Run only unit tests:**
```bash
go test -short -v
```

### Integration Tests (Require Libvirt)

These tests connect to libvirt and may require VMs:

- 🔌 `TestConnect` - Connection to libvirt socket
- 🔌 `TestListVMsIntegration` - List actual VMs
- 🔌 `TestDeleteVMIntegration` - Delete operations
- 🔌 `TestGetVMIPAddressIntegration` - IP detection
- 🔌 `TestConnectionIntegration` - Full connection workflow

**Requirements:**
- Libvirt daemon running
- Socket at `/var/run/libvirt/libvirt-sock`
- User has permissions (member of `libvirt` group)

**Run integration tests:**
```bash
go test -v -run Integration
```

### File System Tests

These tests create temporary files and ISOs:

- 📁 `TestCreateCloudInitISO` - ISO generation
- 📁 Cloud-init user-data/meta-data validation

**Requirements:**
- `genisoimage` or `mkisofs` installed
- Write access to `/tmp`

---

## Test Results Interpretation

### Expected Results

**With libvirt available:**
```
PASS: TestStateToString
PASS: TestIsValidVMName (all 30+ cases)
PASS: TestExtractMACFromXML
PASS: TestGenerateDomainXML
PASS: TestConnect
PASS: TestListVMsIntegration
... (all tests pass)
```

**Without libvirt:**
```
PASS: TestStateToString
PASS: TestIsValidVMName
... (unit tests pass)
SKIP: TestConnect (libvirt socket not found)
SKIP: TestListVMsIntegration (libvirt socket not found)
... (integration tests skipped)
```

### Common Issues

#### "libvirt socket not found"

**Cause:** Libvirt daemon not running or socket at different location.

**Fix:**
```bash
sudo systemctl start libvirt
sudo systemctl status libvirt
```

#### "permission denied"

**Cause:** User not in libvirt group.

**Fix:**
```bash
sudo usermod -a -G libvirt $USER
# Log out and back in
```

#### "genisoimage: command not found"

**Cause:** ISO creation tools not installed.

**Fix:**
```bash
# Fedora/RHEL
sudo dnf install genisoimage

# Ubuntu/Debian
sudo apt install genisoimage
```

---

## Test Coverage Goals

| Module | Current Coverage | Goal |
|--------|-----------------|------|
| types.go | ~95% | 100% |
| validation.go | 100% | 100% |
| xml.go | ~90% | 95% |
| cloudinit.go | ~80% | 85% |
| connection.go | ~75% | 80% |
| network.go | ~70% | 75% |
| domain.go | ~65% | 70% |

Lower coverage for integration modules (connection, network, domain) is expected due to libvirt dependencies.

---

## Writing New Tests

### Test Naming Convention

```go
func TestFunctionName(t *testing.T) {
    t.Run("specific scenario", func(t *testing.T) {
        // Test code
    })
}
```

### Skip Tests When Dependencies Missing

```go
func TestSomething(t *testing.T) {
    if _, err := os.Stat("/required/file"); os.IsNotExist(err) {
        t.Skip("Skipping: required file not found")
    }
    // Test code
}
```

### Integration Test Pattern

```go
func TestSomethingIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    socketPath := "/var/run/libvirt/libvirt-sock"
    if _, err := os.Stat(socketPath); os.IsNotExist(err) {
        t.Skip("Skipping: libvirt socket not found")
    }

    // Integration test code
}
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"valid case", "test-vm", true},
        {"invalid case", "123-vm", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := IsValidVMName(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

---

## Benchmarking

### Run All Benchmarks

```bash
go test -bench=. -benchmem
```

### Example Benchmark Output

```
BenchmarkIsValidVMName-8           2000000    750 ns/op    0 B/op    0 allocs/op
BenchmarkExtractMACFromXML-8       500000    3200 ns/op  1024 B/op    5 allocs/op
BenchmarkGenerateDomainXML-8       100000   12000 ns/op  4096 B/op   12 allocs/op
```

### Write Benchmarks

```go
func BenchmarkFunction(b *testing.B) {
    // Setup
    input := "test-data"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Function(input)
    }
}
```

---

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Install dependencies
        run: sudo apt-get install -y genisoimage

      - name: Run unit tests
        run: cd kvm && go test -short -v

      - name: Run coverage
        run: cd kvm && go test -short -coverprofile=coverage.out
```

---

## Test Maintenance

### When Adding New Functions

1. Add unit tests in corresponding `*_test.go` file
2. Add table-driven tests for multiple scenarios
3. Add edge case tests (empty input, nil, etc.)
4. Add benchmark if performance-critical
5. Update coverage goals in this document

### When Modifying Existing Functions

1. Update existing tests
2. Add new test cases for new behavior
3. Ensure backward compatibility tests pass
4. Update benchmarks if performance changed

---

## Quick Reference

```bash
# Run all tests
go test -v

# Skip integration tests
go test -short -v

# Run specific test
go test -v -run TestValidation

# Run with coverage
go test -cover

# Run benchmarks
go test -bench=.

# Run verbose with race detection
go test -v -race

# Clean test cache
go clean -testcache
```

---

## Test Data

Tests use:
- **Temporary directories:** `t.TempDir()` for file operations
- **Fake data:** SSH keys like `"ssh-rsa AAAA test"`
- **Mock sockets:** Created in temp directories
- **Skip conditions:** Tests skip gracefully when dependencies missing

---

## Contributing Tests

When contributing tests:

1. ✅ Follow existing naming conventions
2. ✅ Use table-driven tests for multiple cases
3. ✅ Add skip conditions for optional dependencies
4. ✅ Document any special requirements
5. ✅ Keep tests independent (no global state)
6. ✅ Clean up resources (use `t.TempDir()`, `defer`)
7. ✅ Test both success and failure paths

---

## Test Statistics

**Total Test Functions:** ~120
**Total Assertions:** ~400+
**Lines of Test Code:** ~1,740
**Benchmark Functions:** 10
**Integration Tests:** ~25
**Unit Tests:** ~95

**Test Execution Time:**
- Unit tests only: ~0.5s
- With integration: ~5-10s (depends on VMs)
- With benchmarks: ~15-30s
