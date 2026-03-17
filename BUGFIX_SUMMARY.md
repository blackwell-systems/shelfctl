# Bug Fixes Summary - Wave 2 Agent I

## Files Modified
1. `internal/migrate/scan.go`
2. `internal/ingest/resolver.go`

## Bugs Fixed

### BUG 20 - http.Client{} has no Timeout (MEDIUM)
**Location:** `internal/migrate/scan.go:32`
**Problem:** HTTP client had no timeout, could hang forever on slow networks
**Fix:** Added 30-second timeout to the HTTP client
```go
client := &http.Client{Timeout: 30 * time.Second}
```
**Also required:** Added `"time"` import

### BUG 21 - http.NewRequest error ignored, req can be nil → panic (MEDIUM)
**Location:** `internal/migrate/scan.go:42`
**Problem:** Error from `http.NewRequest` was ignored with `req, _`, allowing nil req which would panic when setting headers
**Fix:** Properly handle the error:
```go
req, err := http.NewRequest(http.MethodGet, url, nil)
if err != nil {
    return fmt.Errorf("building request for %s: %w", url, err)
}
```

### BUG 22 - http.NewRequest error ignored, req can be nil → panic (MEDIUM)
**Location:** `internal/ingest/resolver.go:107` (inside `resolveGitHub` Open closure)
**Problem:** Same as BUG 21 - error ignored, potential nil panic
**Fix:** Properly handle the error:
```go
req, err := http.NewRequest(http.MethodGet, contentURL, nil)
if err != nil {
    return nil, fmt.Errorf("building request: %w", err)
}
```

### BUG 19 - eager HEAD request wastes 15s on slow servers (LOW)
**Location:** `internal/ingest/resolver.go:60-68` (resolveHTTP function)
**Problem:** HEAD request made immediately at resolve time even if file never downloaded (e.g., dry-run mode)
**Fix:** Removed eager HEAD request entirely, setting Size to -1 by default. This avoids blocking on slow servers and simplifies the code. Progress bars will show indeterminate progress, which is acceptable.

## Testing

### New Test Added
`internal/migrate/scan_test.go` - Contains:
1. `TestScanDir_RequestError` - Verifies malformed URLs return errors instead of panicking (exercises BUG 21 fix)
2. `TestScanRepo_Timeout` - Documents that timeout is set (BUG 20 fix)
3. `TestRouteSource_EmptyMapping` - Additional coverage test

### Verification Results
```bash
✓ go build ./internal/migrate/... ./internal/ingest/...
✓ go vet ./internal/migrate/... ./internal/ingest/...
✓ go test ./internal/migrate/... ./internal/ingest/...
```

All tests pass successfully.

## Impact Assessment

### BUG 20 (Timeout)
- **Severity:** Medium
- **Impact:** Prevents indefinite hangs on network issues
- **Risk:** Low - adds safety with reasonable 30s timeout

### BUG 21 & 22 (Nil Request Panic)
- **Severity:** Medium
- **Impact:** Prevents panics from malformed URLs
- **Risk:** Low - proper error handling is standard practice

### BUG 19 (Eager HEAD)
- **Severity:** Low
- **Impact:** Eliminates 15s delay on slow servers, improves UX in dry-run scenarios
- **Risk:** Low - Size=-1 is already handled by callers; progress bars just show indeterminate

## Trade-offs

For BUG 19, removing the eager HEAD means:
- **Pro:** No blocking wait at resolve time
- **Pro:** Better dry-run performance
- **Pro:** Simpler code
- **Con:** Progress bars can't show file size/percentage for HTTP sources

The trade-off was deemed acceptable because:
1. The eager HEAD could block for 15+ seconds on slow servers
2. In dry-run mode, the file is never downloaded anyway
3. Progress bars can still function with indeterminate progress
4. Local files and GitHub sources still provide size information
