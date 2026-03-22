# Test Harness

A pure Go test harness for shelfctl that provides a mock GitHub API server and test fixtures for both manual and automated testing.

## Architecture

The test harness is implemented in pure Go with no external dependencies beyond the standard library and shelfctl's existing packages. It consists of four main components:

- **test/mockserver**: A lightweight HTTP server that implements GitHub API endpoints needed by shelfctl (releases, assets, downloads). Serves mock responses with configurable fixtures.
- **test/fixtures**: Pre-configured test data including shelves, books, and PDF assets. Provides a default fixture set for common test scenarios.
- **cmd/test-harness**: Command-line coordinator that starts/stops the mock server and sets up the test environment.
- **test/scenarios**: Programmatic integration tests that exercise shelfctl commands against the mock server.

The architecture allows testing shelfctl's full functionality without hitting the real GitHub API, enabling fast, reliable tests and safe experimentation.

## Usage

### Starting the harness

Start the mock server and initialize test fixtures:

```bash
go run cmd/test-harness/main.go start
```

This will:
- Start the mock GitHub API server on `http://localhost:8080`
- Load default fixtures (shelves, books, assets)
- Print the configuration path to use with shelfctl

### Manual testing

Use shelfctl commands with the test harness by setting `SHELFCTL_CONFIG`:

```bash
# Point shelfctl at the test configuration
export SHELFCTL_CONFIG=/tmp/shelfctl-test/config.yaml

# List all shelves
shelfctl shelf list

# List books on a specific shelf
shelfctl shelf show my-shelf

# Add a book to a shelf
shelfctl book add my-shelf "The Go Programming Language" golang/go v1.20.0

# Move a book between shelves
shelfctl book move my-shelf another-shelf book-id

# Verify a book
shelfctl book verify my-shelf book-id
```

The mock server will respond to all GitHub API requests, simulating real GitHub behavior without requiring network access or authentication.

### Running scenarios

Run automated integration tests:

```bash
# Run all scenario tests
go test ./test/scenarios/...

# Run with verbose output
go test -v ./test/scenarios/...

# Run a specific scenario
go test -v ./test/scenarios/ -run TestAddBook
```

Scenarios test end-to-end workflows like adding books, moving books between shelves, verifying checksums, and error handling.

### Stopping

Stop the mock server and clean up:

```bash
go run cmd/test-harness/main.go stop
```

This will shut down the server and remove temporary files.

## Components

### test/mockserver

The mock GitHub API server (`mockserver.Server`) implements the GitHub API endpoints used by shelfctl:

- `GET /repos/:owner/:repo/releases` - List releases
- `GET /repos/:owner/:repo/releases/tags/:tag` - Get specific release
- `GET /repos/:owner/:repo/releases/:id/assets` - List release assets
- `GET /repos/:owner/:repo/releases/assets/:id` - Download asset

The server uses fixtures to provide canned responses and can be configured to simulate various GitHub states (releases with/without assets, download failures, etc.).

### test/fixtures

Test data structures (`fixtures.FixtureSet`) containing:

- **Shelves**: Pre-configured shelf definitions with owner/repo information
- **Books**: Catalog entries with metadata (title, version, checksums)
- **Assets**: Binary PDF content for testing downloads and verification

The default fixture set includes common test scenarios. You can create custom fixtures by implementing `FixtureSet` or modifying the defaults.

### cmd/test-harness

The coordinator binary provides:

- `start`: Launch mock server, load fixtures, create test configuration
- `stop`: Shutdown server and cleanup
- `status`: Check if harness is running

Configuration is written to a temporary directory and includes the mock server URL, enabling shelfctl to connect to the test environment.

### test/scenarios

Programmatic integration tests (`*_test.go`) that:

- Set up the test harness using `harness.Setup()`
- Execute shelfctl commands via internal APIs
- Verify expected behavior and outcomes
- Clean up using `harness.Teardown()`

Each scenario is self-contained and can run independently or as part of the full test suite.

## Troubleshooting

### Mock server fails to start

**Problem**: Port 8080 already in use.

**Solution**: Check if another process is using port 8080 and stop it, or modify the server port in `cmd/test-harness/main.go`.

```bash
# Check what's using port 8080
lsof -i :8080

# Kill the process if needed
kill -9 <PID>
```

### Fixtures not loading

**Problem**: Default fixtures fail to load or are missing expected data.

**Solution**: Verify fixture definitions in `test/fixtures/defaults.go`. The fixture set should include at least one shelf with books and assets. Check for initialization errors in the test output.

### Shelfctl commands fail with "connection refused"

**Problem**: Mock server is not running or shelfctl is not configured correctly.

**Solution**:
1. Verify the harness is running: `go run cmd/test-harness/main.go status`
2. Check `SHELFCTL_CONFIG` points to the test configuration file
3. Confirm the mock server URL in the config matches the running server

### Download/verification failures

**Problem**: Book downloads fail or checksum verification errors occur.

**Solution**: Ensure fixtures include asset data (PDF content) for the books being tested. The mock server needs both the catalog entry and the binary asset to serve downloads. Check `FixtureSet.Assets` map contains entries for all book IDs.

### Tests pass individually but fail in suite

**Problem**: Test scenarios interfere with each other when run together.

**Solution**: Ensure each test properly cleans up using `harness.Teardown()`. Tests should not share state. Use `t.Parallel()` cautiously, as the mock server is a shared resource.

### Permission errors in temp directory

**Problem**: Cannot write to `/tmp/shelfctl-test`.

**Solution**: Check directory permissions or set a custom temp directory using `harness.Config.TempDir`. The harness needs write access to store configuration and downloaded assets.
