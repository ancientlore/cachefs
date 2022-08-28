package cachefs

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

// file represents a cached file
type file struct {
	io.ReadSeeker

	FI   fileInfo   // Metadata about the file
	pos  int        // Current position
	Dirs []dirEntry // Directory entries
}

// Stat returns information about the file.
func (f file) Stat() (fs.FileInfo, error) {
	return f.FI, nil
}

// Close closes the file. Cached files are in memory, so this function does nothing.
func (f *file) Close() error {
	return nil // nothing to do
}

// fileInfo holds the metadata about the cached file
type fileInfo struct {
	Nm string
	Sz int64
	Md fs.FileMode
	Mt time.Time
}

// Name returns the base name of the file.
func (fi fileInfo) Name() string {
	return fi.Nm
}

// Size returns the length in bytes for regular files; system-dependent for others.
func (fi fileInfo) Size() int64 {
	return fi.Sz
}

// Mode returns the file mode bits of the file.
func (fi fileInfo) Mode() fs.FileMode {
	return fi.Md
}

// ModTime returns the modification time of the file.
func (fi fileInfo) ModTime() time.Time {
	return fi.Mt
}

// IsDir is an abbreviation for Mode().IsDir().
func (fi fileInfo) IsDir() bool {
	return fi.Md.IsDir()
}

// Sys always returns nil because the cached file does not keep this information.
func (fi fileInfo) Sys() interface{} {
	return nil
}

// ReadDir reads the contents of the directory and returns
// a slice of up to n DirEntry values in directory order.
// Subsequent calls on the same file will yield further DirEntry values.
//
// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.FI.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: f.FI.Name(), Err: fmt.Errorf("not a directory: %w", fs.ErrInvalid)}
	}

	if n <= 0 {
		dest := make([]fs.DirEntry, len(f.Dirs))
		for i := range f.Dirs {
			dest[i] = f.Dirs[i]
		}
		return dest, nil
	}

	max := len(f.Dirs) - f.pos
	if n < max {
		max = n
	}
	if max == 0 {
		return nil, io.EOF
	}
	dest := make([]fs.DirEntry, max)
	for i := f.pos; i < f.pos+max; i++ {
		dest[i-f.pos] = f.Dirs[i]
	}
	f.pos += max
	return dest, nil
}

// dirEntry is a special version of fileInfo to represent directory entries.
// It is lightweight in that it isn't as filled out as if you called Stat
// on the file itself.
type dirEntry struct {
	FI fileInfo
}

// IsDir reports whether the entry describes a directory.
func (di dirEntry) IsDir() bool {
	return di.FI.IsDir()
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (di dirEntry) Type() fs.FileMode {
	return di.FI.Mode().Type()
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned info is from the time of the directory read and does not contain
// values for ModTime(), Sys(), or Size(). Additionally, the Mode() bits only
// contain the type. This is done to prevent additional reads on the file system
// when a directory is filled out.
func (di dirEntry) Info() (fs.FileInfo, error) {
	return di.FI, nil
}

// Name returns the name of the file (or subdirectory) described by the entry.
// This name is only the final element of the path (the base name), not the entire path.
// For example, Name would return "hello.go" not "home/gopher/hello.go".
func (di dirEntry) Name() string {
	return di.FI.Name()
}
