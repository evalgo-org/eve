# Code Coverage Improvement Plan

## Current Status

### Coverage Progress
- **Initial Coverage**: 40.1% (GitHub display) / 42.3% (actual)
- **Current Coverage**: **43.4%** (+1.1 percentage points)
- **Target**: 60%+
- **Gap**: 16.6 percentage points remaining

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

### Needs Improvement (<60%)
- db: 57.7%
- hr: 47.9%
- kvm: 44.7%
- **security: 37.4%** ⬆️ (was 23.2%)
- forge: 27.2%
- common: 22.6%
- queue: 20.0%
- network: 17.6%
- **storage: 7.1%** ⬅️ Critical gap

---

## Why We Haven't Reached 60% Yet

### 1. External Service Dependencies (storage: 7.1%, network: 17.6%)
- Require mocking AWS S3, MinIO, LakeFS, Hetzner APIs
- Need SSH server mocking
- Complex integration test scenarios

### 2. Functions That Terminate (common: 22.6%, forge: 27.2%)
- `ShellExecute`, `ShellSudoExecute` use `Logger.Fatal`
- `GiteaGetRepo` terminates on error
- Require refactoring to return errors instead

### 3. Large Modules (kvm: 44.7%, db: 57.7%)
- Many complex functions
- Would need significant test investment

---

## Roadmap to 60%+ Coverage

### Phase 1: Quick Wins (Estimated: 3-4 hours) 🎯

#### 1. Add DB Module Tests (+3-5% coverage)
**Priority**: HIGH
**Effort**: Medium

**Tests to Add:**
- `TestCouchDBService_SaveDocument` - Test document creation and updates
- `TestCouchDBService_DeleteDocument` - Test document deletion
- `TestNewCouchDBService` - Test service initialization
- `TestCouchDBService_SaveDocument_Errors` - Error handling
- PostgreSQL operation tests with mocks

**Expected Coverage**: 57.7% → 65-70%

#### 2. Add Common Module Tests (+2-3% coverage)
**Priority**: HIGH
**Effort**: Low-Medium

**Tests to Add:**
- Docker operation tests with mock Docker client
- More URLToFilePath edge cases
- FlowConfig validation tests

**Expected Coverage**: 22.6% → 30-35%

#### 3. Add Forge Module Tests (+2-3% coverage)
**Priority**: MEDIUM
**Effort**: Medium

**Tests to Add:**
- `TestGitlabCreateTag` - Tag creation with httptest
- GitLab job monitoring tests
- More GitLab API operation tests
- Gitea archive download tests

**Expected Coverage**: 27.2% → 35-40%

#### 4. Add Queue Module Tests (+1-2% coverage)
**Priority**: MEDIUM
**Effort**: Low

**Tests to Add:**
- `TestNewRabbitMQService` - Connection initialization
- `TestRabbitMQService_PublishMessage` - Message publishing
- `TestRabbitMQService_Close` - Resource cleanup

**Expected Coverage**: 20.0% → 25-30%

**Phase 1 Total Estimated Coverage**: **43.4% + 8-13% = 51-56%**

---

### Phase 2: Reaching 60% (Estimated: 2-3 hours)

Choose one or more approaches:

#### Option A: Integration Tests (+5-8% coverage)
**Effort**: High
**Requirements**: Docker Compose setup

- Set up Docker Compose with RabbitMQ, CouchDB, MinIO
- Write integration tests for handlers and storage operations
- Slower test execution but comprehensive coverage

#### Option B: Refactor Logger.Fatal (+3-4% coverage)
**Effort**: Medium
**Requirements**: Code refactoring

- Refactor shell execution functions to return errors
- Refactor Gitea functions to return errors
- Update callers to handle errors
- Add tests for error paths

#### Option C: Storage Module Mocking (+2-3% coverage)
**Effort**: Medium-High
**Requirements**: AWS SDK mocking

- Mock S3 client for basic operations
- Test file upload/download paths
- Test error handling

**Phase 2 Target**: **Reach 60-65% coverage**

---

## Recommended Implementation Order

### Week 1: Foundation (Phase 1)
1. **Day 1**: DB module tests → +3-5% coverage
2. **Day 2**: Common module tests → +2-3% coverage
3. **Day 3**: Forge module tests → +2-3% coverage
4. **Day 4**: Queue module tests → +1-2% coverage

**Expected Result**: 51-56% coverage

### Week 2: Push to 60% (Phase 2)
5. **Day 5-6**: Choose Option A, B, or C based on priorities
6. **Day 7**: Final verification and documentation

**Expected Result**: 60-65% coverage

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
- ⏳ Overall Project: 60%+ (current 43.4%)
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

*Last Updated: 2025-10-26*
*Current Coverage: 43.4%*
*Target Coverage: 60%+*
