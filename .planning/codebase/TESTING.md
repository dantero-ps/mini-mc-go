# Testing Patterns

**Analysis Date:** 2026-02-08

## Test Framework

**Runner:**
- Go's built-in `testing` package (Go standard)
- No external test framework (no testify, ginkgo, etc.)

**Run Commands:**
```bash
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test -cover ./...             # Coverage report
go test ./internal/world         # Run specific package tests
```

**Assertion Library:**
- No external assertion library; uses manual comparisons with `if` statements and `t.Errorf()`
- Standard Go testing patterns: `*testing.T`, `t.Fatalf()`, `t.Errorf()`

## Test File Organization

**Location:**
- Co-located pattern: Test files live in same package as source code
- Example: `internal/world/generator_test.go` tests `generator.go` in same package
- Example: `pkg/blockmodel/loader_test.go` tests `loader.go` in same package

**Naming:**
- Standard Go pattern: `{source}_test.go` suffix
- Tests found at:
  - `internal/world/generator_test.go`
  - `pkg/blockmodel/loader_test.go`
  - `pkg/blockmodel/loader_sharing_test.go`

**Structure:**
```
internal/world/
├── generator.go        # Source code
├── generator_test.go   # Tests for generator
├── chunk.go
├── chunk_store.go
├── ...

pkg/blockmodel/
├── loader.go
├── loader_test.go
├── loader_sharing_test.go
├── ...
```

## Test Structure

**Suite Organization:**
Tests use table-driven testing and simple sequential test functions (not grouped into suites):

```go
func TestFlatGeneratorHeight(t *testing.T) {
	g := NewFlatGenerator(10)
	if h := g.HeightAt(0, 0); h != 10 {
		t.Errorf("Expected height 10, got %d", h)
	}
	if h := g.HeightAt(100, -50); h != 10 {
		t.Errorf("Expected height 10, got %d", h)
	}
}
```

**Patterns:**
- No setUp/tearDown per test (use TestMain for global setup/teardown)
- Each test is independent and self-contained
- Test names start with `Test` prefix
- Test function signature: `func TestNameOfTest(t *testing.T)`

## Global Setup/Teardown

**Pattern (from `loader_test.go`):**
```go
func TestMain(m *testing.M) {
	// Setup: Create dummy test files
	os.MkdirAll("assets-test/models/block", 0755)
	writeTestFile("assets-test/models/block/test_cube.json", `{...}`)

	// Run all tests
	exitCode := m.Run()

	// Teardown: Clean up
	os.RemoveAll("assets-test")
	os.Exit(exitCode)
}
```

This pattern:
- Creates test assets before running tests
- Cleans up after all tests complete
- Used in `loader_test.go` to set up JSON fixture files for block model testing

## Assertion Patterns

**Manual assertions (no assertion library):**
```go
if len(model.Elements) != 1 {
	t.Errorf("Expected 1 element, got %d", len(model.Elements))
}

if model.Textures["all"] != "block/stone" {
	t.Errorf("Expected texture 'all' to be 'block/stone', got '%s'", model.Textures["all"])
}
```

**Fatal errors (stop test immediately):**
```go
loader := NewLoader("assets-test")
model, err := loader.LoadModel("block/test_cube")
if err != nil {
	t.Fatalf("Failed to load model: %v", err)
}
```

**Error checking pattern:**
```go
if condition != expected {
	t.Errorf("Expected X, got Y")  // Continue with other assertions
}

if err != nil {
	t.Fatalf("Critical failure: %v", err)  // Stop test immediately
}
```

## Test Categories

**Interface Implementation Tests:**
- Verify that types implement expected interfaces using blank assignment
- Example from `generator_test.go`:
```go
func TestStandardGeneratorImplementsInterface(t *testing.T) {
	var _ TerrainGenerator = NewGenerator(123)
}

func TestFlatGeneratorImplementsInterface(t *testing.T) {
	var _ TerrainGenerator = NewFlatGenerator(10)
}
```
These compile-time checks ensure the types satisfy the interface contract.

**Functional Tests:**
- Test actual behavior and state changes
- Example: `TestFlatGeneratorHeight()` verifies height computation
- Example: `TestFlatGeneratorPopulate()` verifies chunk population with correct block types
- Example: `TestLoadChildModel()` verifies model inheritance and texture resolution

**Data Integrity Tests:**
- Verify correct handling of cached data and shared state
- Example: `TestSharedParentMutation()` verifies that parent model cache isn't corrupted when children modify it
- Example: `TestCache()` verifies same object returned from cache on second load

## Fixture Management

**Test Data Organization:**
```go
func TestMain(m *testing.M) {
	os.MkdirAll("assets-test/models/block", 0755)

	// Create test fixture files
	writeTestFile("assets-test/models/block/test_cube.json", `{
		"textures": { "all": "block/stone" },
		"elements": [ { "from": [0,0,0], "to": [16,16,16], "faces": { "down": { "texture": "#all" } } } ]
	}`)

	exitCode := m.Run()
	os.RemoveAll("assets-test")
	os.Exit(exitCode)
}

func writeTestFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(err)
	}
}
```

**Helper function pattern:**
- `writeTestFile()` - Writes test fixture files to disk
- Fixtures are ephemeral (created at test start, deleted at end)
- Fixtures use real file I/O (tests actual file loading)

## Data-Driven Tests

**Pattern not extensively used in discovered tests:**
- Most tests are single-case or small manual assertion sets
- Table-driven approach could reduce duplication but not currently applied
- Example that could be table-driven: Multiple texture resolution tests in `loader_test.go`

## Mocking

**Strategy:**
- Minimal mocking; tests use real implementations where possible
- File-based fixtures instead of mocks for asset loading tests
- No mock/stub library detected (gomock, testify/mock, etc.)

**Example from `loader_test.go`:**
- Tests use real file system with temporary test directories
- No mock loaders or mock file systems
- Real JSON parsing and inheritance logic tested end-to-end

## Coverage

**Requirements:** Not detected - no coverage thresholds or enforcement in place
- No `.coveragerc`, coverage badge, or CI integration visible
- Tests appear written for correctness, not coverage targets

**Current coverage:**
- Only 3 test files found in codebase (limited coverage)
- `internal/world/generator_test.go` - Tests terrain generation
- `pkg/blockmodel/loader_test.go` - Tests block model loading
- `pkg/blockmodel/loader_sharing_test.go` - Tests model inheritance

## Test Types

**Unit Tests:**
- Scope: Single component/function behavior
- Example: `TestFlatGeneratorHeight()` tests single method in isolation
- Approach: Create component, call method, verify result
- No external dependencies except file system for asset loaders

**Integration Tests:**
- Scope: Multiple components working together
- Example: `TestFlatGeneratorPopulate()` tests generator creates correct blocks in chunk
- Approach: Create full objects (generator + chunk), execute workflow, verify state
- Example: `TestSharedParentMutation()` tests loader + cache + inheritance system

**E2E Tests:**
- Not found in codebase
- Game functionality not covered by automated E2E tests

## Testing Gaps

**Areas not tested:**
- Player movement and physics (`internal/player/movement.go` - 426 lines, no tests)
- Player mining and interaction (`internal/player/mining.go`, `interaction.go` - no tests)
- Input handling (`internal/input/input.go` - 269 lines, no tests)
- Rendering pipeline (graphics package - no tests)
- World generation (only flat and standard generators partially tested)
- Entity management and item entity behavior (no tests found)
- UI and menu systems (no tests)
- Game session and main loop (no tests)

**High-risk untested areas:**
- Physics calculations (gravity, collision detection)
- Save/load systems (if any)
- Multiplayer networking (if any)
- Complex pathfinding or AI (if any)

---

*Testing analysis: 2026-02-08*
