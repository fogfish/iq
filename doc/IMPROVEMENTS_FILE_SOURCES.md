# File Sources Improvements

**Date:** 31 October 2025  
**Status:** ✅ Complete

## Summary

Refactored `FileSource` and renamed `FilesSource` to `FileSeqSource` with significant improvements to the implementation based on feedback.

## Changes Made

### 1. Extracted Shared Filesystem Mounting Logic

**Before:** Duplicated filesystem mounting code in both `FileSource` and `FilesSource`

**After:** Created a single `mountFS()` helper function that both sources use

**Benefits:**
- ✅ DRY principle - filesystem mounting logic defined once
- ✅ Consistent path handling across all file sources
- ✅ Easier to maintain and test
- ✅ Ensures stream library requirements are met uniformly

### 2. Renamed `FilesSource` → `FileSeqSource`

**Rationale:** 
- Clearer distinction from `FileSource` (singular)
- "Seq" explicitly indicates sequential processing
- Better naming convention for future parallel alternatives (e.g., `FileConcurrentSource`)

### 3. Fixed Path Handling for Stream Library

**Critical Fix:** Stream library requires all paths to start with `/`

**Implementation:**
```go
// S3 paths
filename = "/" + parts[1]  // Ensure leading /

// Local paths with slash
filename = path[lastSlash:]  // Keep the leading /

// Local paths without slash  
filename = "/" + path  // Prepend /
```

**Added Documentation:**
```go
// mountFS creates a filesystem for the given path and returns the filename for fs.Open().
// IMPORTANT: All paths returned by this function start with "/" as required by the stream library.
```

### 4. Moved Path Processing to Constructor (`New*`)

**Before:** Path extraction and validation happened in `Next()` method

**After:** All path processing happens once in constructor

**Benefits:**
- ✅ Better performance - no repeated path parsing
- ✅ Fail fast - errors discovered at construction time
- ✅ Cleaner `Next()` method - just opens and returns files
- ✅ Pre-computed filenames stored in struct

### 5. Removed Metadata Reading

**Decision:** Skip metadata reading in Phase 1

**Rationale:**
- Simplifies initial implementation
- Metadata can be added in later phase when requirements are clearer
- Avoids unnecessary I/O operations
- Keeps Phase 1 focused on core functionality

## File Structure

```
internal/iosystem/source/
├── file.go              # FileSource, FileSeqSource, mountFS helper
├── file_test.go         # Comprehensive tests (6 test functions)
└── stdin.go             # StdinSource
```

## API

### FileSource - Single File

```go
src, err := source.NewFileSource("/path/to/file.txt")
// or
src, err := source.NewFileSource("s3://bucket/key.txt")
```

### FileSeqSource - Multiple Files Sequentially

```go
src, err := source.NewFileSeqSource(
    "/path/to/file1.txt",
    "/path/to/file2.txt",
    "/path/to/file3.txt",
)
// or S3
src, err := source.NewFileSeqSource(
    "s3://bucket/file1.txt",
    "s3://bucket/file2.txt",  // Same bucket required
)
```

**Constraints:**
- All paths must be from same filesystem type (all local OR all S3)
- For S3 paths, all must use the same bucket
- Paths are processed sequentially in the order provided

## Testing

All tests use `github.com/fogfish/it` DSL style:

```go
func TestFileSource(t *testing.T)
func TestFileSource_EmptyPath(t *testing.T)
func TestFileSource_NonExistentFile(t *testing.T)
func TestFileSeqSource(t *testing.T)
func TestFileSeqSource_EmptyList(t *testing.T)
func TestFileSeqSource_MultipleEOF(t *testing.T)
```

**Test Results:** ✅ All passing

```bash
ok   github.com/fogfish/iq/internal/iosystem/source  0.226s
```

## Implementation Details

### mountFS Helper Function

```go
func mountFS(path string) (fsys fs.FS, filename string, err error)
```

**Responsibilities:**
1. Detect path type (S3 vs local)
2. Extract bucket/directory and filename
3. Create appropriate filesystem (stream.NewFS or lfs.New)
4. Ensure filename starts with `/` (stream library requirement)
5. Return mounted filesystem and normalized filename

**Used By:**
- `NewFileSource()` - mounts once for single file
- `NewFileSeqSource()` - mounts once, extracts all filenames

### FileSource Structure

```go
type FileSource struct {
    path     string // Original path for document naming
    filename string // Extracted filename for fs.Open (starts with /)
    fsys     fs.FS  // Mounted filesystem
    read     bool   // Track if file has been read
}
```

### FileSeqSource Structure

```go
type FileSeqSource struct {
    paths     []string // Original paths for document naming
    filenames []string // Extracted filenames for fs.Open (all start with /)
    fsys      fs.FS    // Mounted filesystem (shared)
    index     int      // Current position in sequence
}
```

## Key Insights

1. **Stream Library Requirements:**
   - All file paths MUST start with `/`
   - This applies to both S3 and local filesystem operations
   - Critical for proper fs.Open() calls

2. **Performance:**
   - Filesystem mounted once per source (not per file)
   - Path parsing done once in constructor
   - `Next()` method is optimized for repeated calls

3. **Error Handling:**
   - Construction errors fail fast
   - File open errors reported with original path for clarity
   - Clear error messages for common mistakes (empty paths, mixed filesystems)

4. **Design Pattern:**
   - Constructor does heavy lifting (validation, mounting, parsing)
   - Runtime methods (Next) are lightweight
   - Separation of concerns between setup and operation

## Phase 1 Deliverables ✅

- [x] FileSource with S3 and local support
- [x] FileSeqSource (renamed from FilesSource) with S3 and local support
- [x] Shared mountFS helper function
- [x] Correct path handling for stream library (leading /)
- [x] Path processing in constructors
- [x] Comprehensive tests using it DSL
- [x] Documentation and comments

## Future Enhancements (Phase 2+)

- [ ] Add metadata reading (size, modified time)
- [ ] FileConcurrentSource for parallel file processing
- [ ] FileGlobSource for pattern-based file selection
- [ ] S3 prefix-based file discovery
- [ ] Retry logic for transient S3 errors
- [ ] Progress tracking callbacks
