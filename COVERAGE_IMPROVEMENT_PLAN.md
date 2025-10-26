# Code Coverage Improvement Plan

## Current Status

### Coverage Progress
- **Initial Coverage**: 40.1% (GitHub display) / 42.3% (actual)
- **After Refactoring**: 43.4% (+1.1 percentage points)
- **After Go 1.25 Upgrade**: **44.6%** (+2.3 percentage points total)
- **After Logger.Fatal Refactoring**: **44.8%** (+2.5 percentage points total)
- **After Error Path Tests**: **45.8%** (+3.5 percentage points total)
- **Target**: 60%+
- **Gap**: 14.2 percentage points remaining

---

## Accomplishments

### 1. Architecture Refactoring for Testability ✅

**Introduced Interfaces:**
- `queue.MessagePublisher` - Interface for message publishing operations
- `db.DocumentStore` - Interface for document storage operations

**Benefits:**
- Enables comprehensive unit testing without external dependencies
- Allows easy mocking of RabbitMQ and CouchDB
- Better separation of concerns
- Foundation for future testability improvements

**Files Modified:**
- `queue/rabbit.go` - Added MessagePublisher interface
- `db/couchdb.go` - Added DocumentStore interface
- `api/jwt.go` - Updated Handlers struct to use interfaces

### 2. Comprehensive Test Coverage Added

**API Module Tests** (13 new tests):
- PublishMessage handler: Success, validation errors, malformed JSON, publish failures
- GetProcess handler: Success, not found, empty ID, database errors
- GetProcessesByState handler: All processes, filtered by state, invalid state, database errors

**Result**: API module coverage jumped from 20.9% to **80.6%** (+59.7%)

**Security Module Tests** (15 new tests):
- File encryption/decryption roundtrip tests
- Wrong password handling
- Corrupted data detection
- Edge cases (empty password, long password, short ciphertext)
- Different nonce verification
- Performance benchmarks

**Result**: Security module coverage improved from 23.2% to **37.4%** (+14.2%)

### 3. Logger.Fatal Refactoring ✅

**Refactored 48 Functions Across 10 Files:**
- `common/shell.go`: ShellExecute, ShellSudoExecute
- `common/docker.go`: ImageBuild, ImagePush, CopyRenameToContainer, ContainerExists
- `forge/gitea.go`: GiteaGetRepo
- `forge/gitlab.go`: GitlabRunners, GitlabRegisterNewRunner
- `db/graphdb.go`: 6 functions (GraphDBRepositoryConf, GraphDBRepositoryBrf, etc.)
- `network/http_client.go`: HttpClientDownloadFile
- `network/zti_conf.go`: WriteZitiRouterConfig, WriteZitiControllerConfig, ZitiGenerateCtrlConfig, ZitiGenerateRouterConfig
- `network/ziti.go`: postWithAuthMap, ZitiServicePolicies, ZitiIdentities
- `network/ssh.go`: SshExec
- `security/certs.go`: ZitiCreateCSR

**Benefits:**
- Functions now return errors instead of terminating execution
- Error paths can be tested without program termination
- Better error handling and debugging
- Follows Go best practices for error handling

### 4. Error Path Tests Added ✅

**Network Module Tests** (network/zti_conf_test.go - 335 lines):
- TestWriteZitiRouterConfig (3 test cases)
- TestWriteZitiControllerConfig (2 test cases)
- TestZitiGenerateCtrlConfig (2 test cases)
- TestZitiGenerateRouterConfig (2 test cases)
- TestZitiConfigStructures (3 test cases)
- TestZitiConfigYAMLEncoding (2 test cases)
- 2 benchmarks

**Result**: Network module coverage improved from 17.6% to **27.0%** (+9.4%)

**Security Module Tests** (security/security_test.go - +137 lines):
- TestZitiCreateCSR_ErrorPaths (4 test scenarios)
- Error handling for invalid paths, permission errors, etc.

**Result**: Security module coverage improved from 37.4% to **38.7%** (+1.3%)

**Overall Impact**: Coverage improved from 44.8% to **45.8%** (+1.0%)

---

## Current Coverage by Module

### Excellent Coverage (>70%)
- notification: 87.5%
- archive: 85.2%
- assets: 83.8%
- **api: 80.6%** ⬆️ (was 20.9%)
- media: 71.2%

### Good Coverage (60-70%)
- cloud: 65.7%
- **kvm: 61.8%** ⬆️ (was 44.7%, +17.1%!)

### Needs Improvement (<60%)
- **db: 57.9%** ⬆️ (was 57.7%)
- hr: 47.9%
- **security: 38.7%** ⬆️ (was 23.2%, +15.5%!)
- **network: 27.0%** ⬆️ (was 17.6%, +9.4%!)
- **forge: 26.2%** ⬇️ (was 27.2%)
- **common: 26.6%** ⬆️ (was 22.6%)
- queue: 20.0%
- **storage: 7.1%** ⬅️ Critical gap (requires complex mocking)

---

## Why We Haven't Reached 60% Yet

### 1. External Service Dependencies ✅ Partially Addressed
- **Storage (7.1%)**: Requires mocking AWS S3, MinIO, LakeFS, Hetzner APIs - VERY COMPLEX
- **Network (27.0%)**: SSH functions tested where possible, Ziti configuration tested ✅
- Remaining gaps require actual service connections or complex mocking frameworks

### 2. Functions That Terminate ✅ RESOLVED
- ~~`ShellExecute`, `ShellSudoExecute` use `Logger.Fatal`~~ ✅ Refactored
- ~~`GiteaGetRepo` terminates on error~~ ✅ Refactored
- **All 48 Logger.Fatal calls have been refactored to return errors**
- Error paths now testable

### 3. Database and Message Queue Integration
- **CouchDB (0% on core functions)**: Requires CouchDB test instance or mocking Kivik client
- **PostgreSQL (0% on all functions)**: Requires PostgreSQL test instance or GORM mocking
- **RabbitMQ (PublishMessage at 0%)**: Requires RabbitMQ test instance or AMQP mocking
- **Docker client (most at 0%)**: Requires Docker API mocking

### 4. Test Infrastructure Gap
To achieve the remaining 14.2% to reach 60%, we need:
- Mocking frameworks for HTTP clients, Docker client, database clients
- Test containers for real service testing (preferred approach)
- Significant investment in test infrastructure setup

---

## Roadmap to 60%+ Coverage

### Phase 1: Quick Wins - REVISED ⚠️

**Status**: Partially completed. Coverage improved from 44.6% to **45.8%** (+1.2%)

**Key Finding**: Most "quick wins" require test infrastructure (mocking or test containers) that wasn't initially accounted for.

#### 1. ✅ Network Module Error Path Tests (COMPLETED)
- Added 18 test cases for Ziti configuration functions
- Coverage: 17.6% → 27.0% (+9.4%)
- **Overall impact**: +0.5% to total coverage

#### 2. ✅ Security Module Error Path Tests (COMPLETED)
- Added 4 error path test scenarios for CSR creation
- Coverage: 37.4% → 38.7% (+1.3%)
- **Overall impact**: +0.1% to total coverage

#### 3. ❌ DB Module Tests (REQUIRES INFRASTRUCTURE)
**Status**: Not feasible without test infrastructure
**Current Coverage**: 57.9%

**Findings:**
- CouchDB functions (0% coverage): Require Kivik client mocking or CouchDB test container
- PostgreSQL functions (0% coverage): Require GORM mocking or PostgreSQL test container
- Helper functions already well tested (sanitizeFilename: 100%, saveDocumentToFile: 88.9%)

**To Implement:**
- Option A: Use `testcontainers-go` with real CouchDB/PostgreSQL instances
- Option B: Create mock implementations of Kivik and GORM interfaces
- **Estimated Effort**: High (requires test infrastructure setup)

#### 4. ❌ Common Module Tests (REQUIRES INFRASTRUCTURE)
**Status**: Not feasible without Docker client mocking
**Current Coverage**: 26.6%

**Findings:**
- Shell functions: 100% coverage ✅ (already tested)
- Docker functions (0% coverage): Require Docker client API mocking
- URLToFilePath: 100% coverage ✅ (already tested)
- FlowConfig: Type definitions only, no executable code

**To Implement:**
- Mock Docker client API for functions like ContainerRun, ImageBuild, etc.
- **Estimated Effort**: High (complex Docker API mocking)

#### 5. ❌ Forge Module Tests (REQUIRES INFRASTRUCTURE)
**Status**: Not feasible without HTTP client mocking
**Current Coverage**: 26.2%

**Findings:**
- All functions make HTTP calls to external GitLab/Gitea APIs
- Require `httptest` mocking or VCR-style HTTP recording
- **Estimated Effort**: Medium-High

#### 6. ❌ Queue Module Tests (REQUIRES INFRASTRUCTURE)
**Status**: Not feasible without RabbitMQ mocking
**Current Coverage**: 20.0%

**Findings:**
- Existing tests cover error cases and nil safety
- PublishMessage (0% coverage): Requires AMQP protocol mocking or RabbitMQ test container
- **Estimated Effort**: Medium-High

**Phase 1 Actual Coverage**: **44.6% → 45.8%** (+1.2% instead of estimated +8-13%)

---

### Phase 2: Reaching 60% - REVISED

**Revised Strategy**: Focus on test infrastructure setup to unlock coverage improvements

#### ✅ Option B: Refactor Logger.Fatal (COMPLETED +0.2% coverage)
**Status**: COMPLETED
**Actual Impact**: +0.2% overall coverage (refactoring added code, tests added +1.0%)

- ✅ Refactored 48 functions across 10 files to return errors
- ✅ Added error path tests for network and security modules
- ✅ Coverage: 44.6% → 45.8%

#### Option A: Test Infrastructure Setup (+8-12% coverage)
**Effort**: High
**Requirements**: Test containers or mocking framework
**Priority**: HIGH - Unlocks most remaining coverage gains

**Approach 1: Test Containers** (Recommended)
- Use `testcontainers-go` library
- Real service instances: CouchDB, PostgreSQL, RabbitMQ
- Pros: Tests real behavior, catches integration issues
- Cons: Slower test execution, requires Docker

**Implementation:**
```bash
go get github.com/testcontainers/testcontainers-go
```

Tests to add:
- DB module: CouchDB integration tests (+3-4% overall)
- DB module: PostgreSQL integration tests (+2-3% overall)
- Queue module: RabbitMQ integration tests (+1-2% overall)
- **Estimated total**: +6-9% overall coverage

**Approach 2: Comprehensive Mocking**
- Mock Docker client API for common module
- Mock HTTP clients for forge module
- Mock database clients for db module
- Pros: Fast test execution
- Cons: More brittle, may miss integration issues

**Estimated total**: +8-12% overall coverage

#### Option C: Storage Module Tests (Deferred - Very Complex)
**Effort**: Very High
**Current Coverage**: 7.1%
**Priority**: LOW

The storage module requires mocking multiple cloud provider APIs:
- AWS S3 SDK
- MinIO client
- LakeFS client
- Hetzner storage API

**Recommendation**: Defer until other modules reach good coverage levels

**Phase 2 Revised Target**: **45.8% + 8-12% = 53-57% coverage**

---

## Recommended Implementation Order

### ✅ Completed Work
1. ✅ **Interface Refactoring**: Added MessagePublisher and DocumentStore interfaces
2. ✅ **API Module Tests**: Coverage 20.9% → 80.6% (+59.7%)
3. ✅ **Security Module Tests**: Coverage 23.2% → 38.7% (+15.5%)
4. ✅ **Logger.Fatal Refactoring**: 48 functions across 10 files
5. ✅ **Network Module Tests**: Coverage 17.6% → 27.0% (+9.4%)
6. ✅ **Coverage Analysis**: Identified infrastructure requirements

**Current Coverage**: **45.8%** (Target: 60%, Gap: 14.2%)

### Next Steps to Reach 60%

#### Step 1: Set Up Test Infrastructure (Week 1-2)
**Priority**: CRITICAL - Blocks most coverage improvements
**Effort**: 2-3 days

Choose an approach:
- **Option A** (Recommended): Test Containers
  - Install `testcontainers-go`
  - Create test helpers for CouchDB, PostgreSQL, RabbitMQ
  - Write example integration test

- **Option B**: Mocking Framework
  - Install `gomock` or `testify/mock`
  - Create mock implementations
  - More maintenance overhead

#### Step 2: Database Module Tests (Week 2-3)
**Priority**: HIGH
**Estimated Impact**: +5-7% overall coverage
**Effort**: 2-3 days

1. CouchDB integration tests (+3-4%)
   - SaveDocument, GetDocument, DeleteDocument
   - GetDocumentsByState, GetAllDocuments
   - Error handling scenarios

2. PostgreSQL integration tests (+2-3%)
   - PGRabbitLogNew, PGRabbitLogList
   - PGRabbitLogUpdate
   - Connection pooling tests

#### Step 3: Queue Module Tests (Week 3)
**Priority**: MEDIUM
**Estimated Impact**: +1-2% overall coverage
**Effort**: 1 day

- PublishMessage integration tests
- Connection error handling
- Message acknowledgment scenarios

#### Step 4: Common/Forge Module Tests (Week 4)
**Priority**: MEDIUM
**Estimated Impact**: +2-4% overall coverage
**Effort**: 2-3 days

- Docker client mocking (if feasible)
- HTTP client mocking for GitLab/Gitea APIs
- Additional edge case coverage

**Expected Final Coverage**: **45.8% + 14.2% = 60%**

---

## Test Coverage Guidelines

### What Makes a Good Test

1. **Tests one specific behavior**
   - Each test should have a clear, single purpose
   - Use descriptive test names: `TestFunctionName_Scenario`

2. **Independent and isolated**
   - Tests should not depend on each other
   - Use mocks to isolate external dependencies

3. **Covers edge cases**
   - Empty inputs
   - Nil values
   - Error conditions
   - Boundary values

4. **Fast execution**
   - Unit tests should run in milliseconds
   - Use mocks instead of real services

### Testing Pattern

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange - Set up test data and mocks
    mock := &MockService{
        MethodFunc: func(input string) (string, error) {
            return "expected", nil
        },
    }

    // Act - Execute the function under test
    result, err := FunctionName(mock, "input")

    // Assert - Verify the results
    assert.NoError(t, err)
    assert.Equal(t, "expected", result)
}
```

---

## Files Modified in This Initiative

1. `queue/rabbit.go` - Added MessagePublisher interface (12 lines)
2. `db/couchdb.go` - Added DocumentStore interface (28 lines)
3. `api/jwt.go` - Refactored Handlers to use interfaces (10 lines changed)
4. `api/jwt_test.go` - Added 13 comprehensive handler tests (427 lines)
5. `security/security_test.go` - Added 15 encryption/decryption tests (347 lines)

**Total**: 819 lines added, architecture significantly improved for testability.

---

## Success Metrics

### Coverage Targets
- ✅ API Module: 80%+ (achieved 80.6%)
- ✅ KVM Module: 60%+ (achieved 61.8%)
- ⏳ Overall Project: 60%+ (current 44.6%)
- 🎯 DB Module: 70%+
- 🎯 Common Module: 35%+
- 🎯 Forge Module: 40%+

### Quality Metrics
- All tests pass in CI/CD
- No flaky tests
- Test execution time < 30 seconds for full suite
- Code maintains production quality

---

## Maintenance Plan

### Ongoing Practices
1. **Write tests for new features** - Aim for 80%+ coverage on new code
2. **Refactor when adding tests** - Improve architecture while testing
3. **Monitor coverage trends** - Use codecov.io or similar tools
4. **Review test quality** - Tests should be maintainable and readable

### Quarterly Reviews
- Review coverage gaps
- Prioritize high-impact, low-coverage modules
- Update this plan with new targets

---

## Contact & Questions

For questions about this coverage improvement plan:
- Review the test examples in `api/jwt_test.go` and `security/security_test.go`
- See interface definitions in `queue/rabbit.go` and `db/couchdb.go`
- Consult CONTRIBUTING.md for development guidelines

---

*Last Updated: 2025-10-26 (Post Go 1.25 Upgrade)*
*Current Coverage: 44.6%*
*Target Coverage: 60%+*
*Phase 1 in Progress*
