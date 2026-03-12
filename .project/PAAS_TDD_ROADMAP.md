# PaaS TDD Development Roadmap

## Testing Strategy

### Testing Framework Selection
**Standard Go `testing` package + minimal assertions** - No heavy frameworks like testify to keep it lightweight like vitest for React. We use valyala libraries for performance, so testing should be equally lean.

**Why this approach:**
- Built-in `testing` package (zero dependencies)
- `httpexpect` for HTTP endpoint testing (lightweight HTTP testing)
- Focus on behavior testing, not implementation details
- Tests as documentation and specification

**Essential testing packages:**
```go
import (
    "testing"
    "github.com/gavv/httpexpect/v2"  // Lightweight HTTP testing
    "github.com/valyala/fasthttp"     // For fasthttp integration tests
)
```

### Local Development Constraints
**Windows + Limited Docker Desktop space = Mock-based testing strategy**

**Local Testing (Windows):**
- ✅ HTTP server + routing tests
- ✅ Database (BoltDB) tests
- ✅ Business logic tests
- ✅ Mocked Docker API calls
- ❌ Real Docker operations

**Server Testing (Ubuntu):**
- ✅ Full integration tests with real Docker
- ✅ End-to-end deployment tests
- ✅ Performance tests
- ✅ Real Docker API calls

**Testing Workflow:**
1. **Local**: Write code + mocked tests → commit
2. **Server**: Deploy → run integration tests → validate
3. **Iterate**: Fix issues found on server

## TDD Session Structure

### Core Principle: Tests First → Implementation → Refactor
**Every feature starts with failing tests, then minimal code to pass them.**

### Session 1: Core Infrastructure (Database + Web Server)
**Goal:** Working HTTP server with BoltDB persistence

**LLM Task:** Create tests first, then implement minimal web server and database layer. Focus on HTTP endpoints for basic CRUD operations.

**TDD Tests to Write First:**
```go
func TestWebServer_BasicEndpoints(t *testing.T) {
    // Test server starts and responds
}

func TestBoltDB_AppCRUD(t *testing.T) {
    // Test basic app creation/retrieval/deletion
}

func TestHTTP_AppAPI(t *testing.T) {
    // Test REST API endpoints with httpexpect
}
```

**Implementation:** Minimal fasthttp server + BoltDB models + basic handlers

### Session 2: Docker Integration (Mock-Based)
**Goal:** Container management interface (mocked for local development)

**Important:** Docker tests run separately on server due to Windows/Docker Desktop limitations.

**LLM Task:** Write unit tests with mocked Docker API first, then implement Docker client wrapper.

**Local TDD Tests to Write First (Mocked):**
```go
func TestDockerClient_ListContainers(t *testing.T) {
    // Test with mocked Docker API responses
}

func TestDockerClient_ExecuteCompose(t *testing.T) {
    // Test compose command generation (no actual execution)
}

func TestDockerClient_StatusParsing(t *testing.T) {
    // Test status parsing from Docker API responses
}
```

**Server-Only Integration Tests (Run on Ubuntu server):**
```go
func TestDocker_Integration_FullCycle(t *testing.T) {
    // Real Docker operations - run only on server
}
```

**Implementation:** Docker client wrapper + mock interfaces + command builders

### Session 3: Application Management
**Goal:** Full app lifecycle (create, deploy, monitor, destroy)

**LLM Task:** Behavior tests for app management with mocked Docker calls.

**TDD Tests to Write First:**
```go
func TestAppLifecycle_FullCycle(t *testing.T) {
    // Test complete create→deploy→monitor→destroy cycle (mocked Docker)
}

func TestAppDiscovery_FindExisting(t *testing.T) {
    // Test finding apps from filesystem (real) + Docker API (mocked)
}

func TestAppValidation_ConfigValidation(t *testing.T) {
    // Test compose file and config validation (no Docker needed)
}
```

**Implementation:** App lifecycle management + validation + discovery (with Docker mocks)

### Session 4: Nginx + SSL Automation
**Goal:** Domain routing and SSL certificates

**LLM Task:** Test configuration generation and SSL workflow first.

**TDD Tests to Write First:**
```go
func TestNginx_ConfigGeneration(t *testing.T) {
    // Test nginx config file generation
}

func TestSSL_LetsEncryptWorkflow(t *testing.T) {
    // Test certificate generation (mock/staging)
}

func TestDomain_WildcardRouting(t *testing.T) {
    // Test wildcard domain configuration
}
```

**Implementation:** Nginx config generator + Let's Encrypt integration

### Session 5: CRON + GUI Essentials
**Goal:** Cleanup automation + minimal working GUI

**LLM Task:** Test CRON logic and basic GUI rendering first.

**TDD Tests to Write First:**
```go
func TestCRON_CleanupLogic(t *testing.T) {
    // Test deployment cleanup rules
}

func TestGUI_BasicRendering(t *testing.T) {
    // Test quicktemplate rendering
}

func TestGUI_AppDashboard(t *testing.T) {
    // Test dashboard with mock data
}
```

**Implementation:** CRON system + essential GUI components

## TDD Rules for LLM

### Always Start with Tests
**If user forgets to mention TDD:**
1. **STOP** and remind: "Let's write tests first following TDD approach"
2. **Create failing tests** that define the expected behavior
3. **Write minimal code** to make tests pass
4. **Refactor** only after tests pass

### Testing Guidelines
- **No code without tests** - Every function/feature must have tests
- **Test behavior, not implementation** - Black box testing
- **Fast feedback** - Unit tests run in <1 second
- **Realistic test data** - Use actual compose files, domains, etc.
- **HTTP testing with httpexpect** - For API endpoints
- **Integration tests** - Test real Docker operations (with cleanup)

### Test Organization
```
paas/
├── main_test.go           # Integration tests (server only)
├── handlers/
│   ├── handlers_test.go   # Handler unit tests (local)
│   └── handlers_int_test.go # Integration tests (server only)
├── docker/
│   ├── docker_test.go     # Mocked Docker API tests (local)
│   └── docker_int_test.go # Real Docker tests (server only)
├── db/
│   └── db_test.go         # Database tests (local)
└── nginx/
    └── nginx_test.go      # Config generation tests (local)

# Build tags for conditional compilation
// +build !integration
// Local tests (no Docker dependency)

// +build integration
// Server integration tests (require Docker)
```

### Server Testing Setup
**For Ubuntu server testing:**

1. **Deploy code** to server with Docker
2. **Run integration tests:**
   ```bash
   go test -tags=integration ./...
   ```
3. **Test categories:**
   - `TestDocker_Integration_*` - Real Docker operations
   - `TestApp_EndToEnd_*` - Full deployment cycles
   - `TestPerformance_*` - Load and performance tests

4. **CI/CD approach:**
   - Local: `go test ./...` (mocked tests)
   - Server: `go test -tags=integration ./...` (real Docker)

### Quality Gates
- **All tests pass** before committing
- **Test coverage >80%** for critical paths
- **No flaky tests** - Tests must be deterministic
- **Fast execution** - Full test suite <30 seconds
- **CI/CD ready** - Tests work in automated environment

## MVP Validation Tests

### End-to-End Test Suite
```go
func TestMVP_EndToEnd(t *testing.T) {
    // 1. Create app via API
    // 2. Deploy (docker-compose up)
    // 3. Check nginx config generated
    // 4. Verify SSL certificate
    // 5. Test domain routing
    // 6. Destroy app
    // 7. Verify cleanup
}
```

### Performance Tests
```go
func TestPerformance_ResponseTime(t *testing.T) {
    // Ensure <100ms response times
}

func TestPerformance_ConcurrentRequests(t *testing.T) {
    // Handle 100+ concurrent requests
}
```

This TDD approach ensures **minimal, tested code** that delivers working MVP quickly. Tests serve as both specification and safety net for future changes.
