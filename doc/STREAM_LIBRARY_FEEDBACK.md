# Feedback for github.com/fogfish/stream Library

## Issue: Path Handling Documentation and API Clarity

### Context
While implementing source/sink builders for the `iq` project, we encountered significant confusion around path handling in the `lfs` (local filesystem) wrapper, particularly around:
1. When paths need leading slashes
2. How paths relate to mounted filesystem roots
3. Differences between S3 and local filesystem path conventions

### Background
The library is designed to provide unified access to S3 and local filesystems, using S3 conventions where trailing `/` distinguishes directories from files.

---

## Problems Encountered

### 1. **Leading Slash Ambiguity for lfs.Create()**

**Symptom:**
```go
fsys, _ := lfs.New("/parent/dir")
fsys.Create("filename.txt", nil)  // ❌ Fails with "invalid argument"
fsys.Create("/filename.txt", nil) // ✅ Works
```

**Confusion:**
- `fs.FS` from Go standard library uses relative paths (no leading slash)
- S3 keys don't use leading slashes either
- But `lfs` requires leading slashes for Create()

**Example from our tests:**
```go
// From internal/iosystem/sink/fs_test.go
fsys, _ := lfs.New(tmpDir)
snk, _ := sink.NewFS(fsys)

// Document path HAS leading slash, but file is created WITHOUT it
doc := iosystem.NewDocument("/test.txt", reader)
snk.Write(ctx, doc)  // Creates tmpDir/test.txt (not tmpDir//test.txt)

// Verify file (NO leading slash in os.ReadFile path)
content, _ := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
```

**Our workaround:**
```go
// Had to add leading slash explicitly for lfs.Create()
func mountFile(path string) (stream.CreateFS[struct{}], string, error) {
    dir := filepath.Dir(absPath)
    base := filepath.Base(absPath)
    
    fsys, _ := lfs.New(dir)
    
    // Add leading slash for lfs - discovered through trial and error!
    return fsys, "/" + base, nil
}
```

---

### 2. **Inconsistency with Standard fs.FS Pattern**

**Expected behavior (based on os.DirFS):**
```go
fsys := os.DirFS("/parent/dir")
file, _ := fsys.Open("filename.txt")  // ✅ Relative path, no leading slash
```

**Actual lfs behavior:**
```go
fsys, _ := lfs.New("/parent/dir")
file, _ := fsys.Open("filename.txt")  // ✅ Works for Open()
file, _ := fsys.Create("filename.txt", nil) // ❌ Fails for Create()
file, _ := fsys.Create("/filename.txt", nil) // ✅ Works for Create()
```

**Why this is confusing:**
- `Open()` and `Create()` have different path expectations
- This breaks the mental model of "mount a directory, use relative paths"
- Code that works with `os.DirFS` doesn't work with `lfs` without modification

---

### 3. **Lack of Documentation on Path Conventions**

**What we needed to know:**
1. When mounting `lfs.New("/some/dir")`, what is the root?
2. Should paths to `Create()` be:
   - Relative to mount point? (like `os.DirFS`)
   - Absolute from filesystem root?
   - Relative but with leading `/`? (actual answer)
3. How does this relate to S3 conventions?
4. Why does `Open()` work differently than `Create()`?

**Current documentation gaps:**
- No examples of `Create()` usage in lfs package
- No explanation of leading slash requirement
- No comparison with `os.DirFS` behavior
- No guidance on path handling differences between S3 and local

---

### 4. **S3 Path Mapping Unclear**

**Questions we had:**
1. For S3 path `s3://bucket/prefix/file.txt`:
   - Do we mount with `stream.NewFS("bucket")` or `stream.NewFS("bucket/")`?
   - Is the path then `prefix/file.txt` or `/prefix/file.txt`?
   - Does trailing `/` in mount point affect behavior?

2. Directory semantics:
   - S3 doesn't have real directories, just key prefixes
   - How does `lfs` map this to local filesystem directories?
   - When do we need trailing `/`?

**Our confusion:**
```go
// Which is correct for s3://bucket/path/to/file.txt?
fsys, _ := stream.NewFS("bucket")
fsys.Create("/path/to/file.txt", nil)  // Option A?
fsys.Create("path/to/file.txt", nil)   // Option B?

// Or do we mount with prefix?
fsys, _ := stream.NewFS("bucket/path/to")
fsys.Create("/file.txt", nil)          // Option C?
```

---

## Recommendations for Library Improvement

### 1. **Explicit Documentation**

Add a "Path Handling Guide" section to README:

```markdown
## Path Handling

### Local Filesystem (lfs)

When using `lfs.New()`, paths must start with `/`:

```go
fsys, _ := lfs.New("/parent/dir")

// Reading (implements fs.FS - no leading slash needed)
file, _ := fsys.Open("file.txt")      // ✅ Opens /parent/dir/file.txt

// Writing (implements CreateFS - REQUIRES leading slash)
file, _ := fsys.Create("/file.txt", nil)  // ✅ Creates /parent/dir/file.txt
file, _ := fsys.Create("file.txt", nil)   // ❌ ERROR: invalid argument
```

**Why the difference?**
- `lfs` follows S3 conventions where paths starting with `/` are relative to the bucket root
- This ensures consistent behavior between S3 and local filesystems
- The leading `/` is stripped internally before filesystem operations

### S3 Filesystem (stream)

For S3, mount the bucket and use paths without the bucket name:

```go
// For s3://my-bucket/path/to/file.txt
fsys, _ := stream.NewFS("my-bucket")
file, _ := fsys.Create("/path/to/file.txt", nil)  // ✅

// NOT like this:
file, _ := fsys.Create("path/to/file.txt", nil)   // ❌ May fail
```

### Directory Semantics

Trailing `/` indicates a directory:
- `/files/` - directory
- `/files` - could be file or directory (ambiguous)

Use trailing slash for directory operations.
```

### 2. **Consistent API**

**Option A: Make Create() match Open() behavior**
```go
// Allow both forms, strip leading slash internally
fsys.Create("file.txt", nil)   // Works
fsys.Create("/file.txt", nil)  // Also works (slash stripped)
```

**Option B: Make Open() match Create() behavior**
```go
// Require leading slash for both (breaking change)
fsys.Open("/file.txt")    // Consistent with Create()
```

**Option C: Add helper methods**
```go
// Clear intent
fsys.CreateAtRoot("/file.txt", nil)      // Explicit: at filesystem root
fsys.CreateRelative("file.txt", nil)     // Explicit: relative to mount
```

### 3. **Better Error Messages**

Current:
```
create filename.txt: invalid argument
```

Suggested:
```
create filename.txt: invalid argument (paths must start with '/' when using lfs.Create, e.g., '/filename.txt')
```

Or:
```
create filename.txt: path must be absolute from mount point (hint: add leading '/', e.g., '/filename.txt')
```

### 4. **Comprehensive Examples**

Add examples showing:

```go
// Example 1: Single file operations
func ExampleLFS_SingleFile() {
    fsys, _ := lfs.New("/tmp")
    
    // Write
    file, _ := fsys.Create("/output.txt", nil)
    file.Write([]byte("content"))
    file.Close()
    
    // Read
    file, _ = fsys.Open("output.txt")  // Note: no leading slash for Open
    // ... read ...
}

// Example 2: Directory structure
func ExampleLFS_DirectoryStructure() {
    fsys, _ := lfs.New("/tmp")
    
    // Create nested directory structure
    file, _ := fsys.Create("/subdir/nested/file.txt", nil)
    // ... 
}

// Example 3: S3 equivalent
func ExampleStream_S3() {
    fsys, _ := stream.NewFS("my-bucket")
    
    // Same pattern as lfs
    file, _ := fsys.Create("/prefix/file.txt", nil)
    // ...
}

// Example 4: Portable code (works with both)
func ProcessWithFS(fsys CreateFS) {
    // Always use leading slash for Create
    file, _ := fsys.Create("/data.txt", nil)
    // ...
}
```

### 5. **Type-Safe Path Types**

Consider introducing path types to make requirements explicit:

```go
type AbsPath string    // Must start with /
type RelPath string    // No leading /

func (fs *LFS) Create(path AbsPath, attr *Attr) (File, error)
func (fs *LFS) Open(path RelPath) (fs.File, error)

// Usage is self-documenting:
fsys.Create(AbsPath("/file.txt"), nil)   // Clear requirement
fsys.Open(RelPath("file.txt"))            // Clear requirement
```

---

## Impact on Our Code

We had to learn these conventions through trial and error:

1. **Failed approaches tried:**
   - Mount "." and use filepath.Join (didn't work)
   - Mount "/" and use absolute paths (didn't work)
   - Use paths without leading slash (didn't work for Create)
   - Use os.DirFS pattern (only worked for Open, not Create)

2. **Working solution:**
   ```go
   // Mount parent directory
   dir := filepath.Dir(absPath)
   fsys, _ := lfs.New(dir)
   
   // Add leading slash for Create operations
   base := filepath.Base(absPath)
   fsys.Create("/" + base, nil)  // ✅ Works
   ```

3. **Time spent:** ~2 hours debugging path issues
4. **Tests needed:** 7+ test iterations to get it right

---

## Suggested Checklist for Library Documentation

- [ ] Document leading slash requirement for `Create()`
- [ ] Explain why `Open()` and `Create()` differ
- [ ] Add comparison with `os.DirFS` behavior
- [ ] Provide S3-to-local path mapping examples
- [ ] Show directory vs file path conventions (trailing `/`)
- [ ] Include error message improvements
- [ ] Add comprehensive examples covering common use cases
- [ ] Document the "S3 conventions on local FS" design decision
- [ ] Explain when to use `lfs` vs `os.DirFS` directly

---

## Summary

The `stream` library provides valuable abstraction over S3 and local filesystems, but the path handling conventions are non-intuitive for developers familiar with Go's standard `fs.FS` interface. Better documentation and possibly API improvements would significantly reduce integration friction.

**Key insight:** The library uses S3-like conventions (leading `/` for paths relative to mount) even for local filesystems. This is a valid design choice but needs to be explicitly documented and consistently applied across all methods.

---

**Environment:**
- Library: github.com/fogfish/stream
- Use case: Building unified I/O abstraction for LLM processing pipeline
- Developer background: Familiar with Go's fs.FS, os.DirFS, and filepath packages

**Would we still use the library?** Yes! Once understood, it provides excellent S3/local unification. But the learning curve was steeper than necessary due to documentation gaps.
