package cachefs_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/ancientlore/cachefs"
)

func TestReadDir(t *testing.T) {
	rootFS := os.DirFS(".")
	fileSys := cachefs.New(rootFS, nil)

	f, err := fileSys.Open(".")
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Error("Root is not a ReadDirFile")
		return
	}

	dirs, err := rdf.ReadDir(0)
	if err != nil {
		t.Error(err)
		return
	}

	rf, err := rootFS.Open(".")
	if err != nil {
		t.Error(err)
		return
	}
	defer rf.Close()

	rootDirs, err := rf.(fs.ReadDirFile).ReadDir(0)
	if err != nil {
		t.Error(err)
		return
	}

	if len(rootDirs) != len(dirs) {
		t.Errorf("rootDirs has length %d but dirs has length %d", len(rootDirs), len(dirs))
		return
	}

	for i := range dirs {
		if dirs[i].Name() != rootDirs[i].Name() {
			t.Errorf("Entry %d of %q does not match %q", i, dirs[i].Name(), rootDirs[i].Name())
		}
	}
}

func TestReadDirLoop(t *testing.T) {
	rootFS := os.DirFS(".")
	fileSys := cachefs.New(rootFS, nil)

	f, err := fileSys.Open(".")
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Error("Root is not a ReadDirFile")
		return
	}

	var dirs []fs.DirEntry
	for {
		dirs, err = rdf.ReadDir(2)
		if errors.Is(err, io.EOF) {
			if len(dirs) != 0 {
				t.Errorf("Expected empty directory at EOF")
			}
			break
		}
		if err != nil {
			t.Error(err)
			break
		}
		if len(dirs) == 0 {
			t.Errorf("Should not return empty directory if not EOF")
			break
		}
		if len(dirs) > 2 {
			t.Errorf("Returned more than 2 entries: %d", len(dirs))
		}
		t.Log(dirs)
	}
}

func TestSeek(t *testing.T) {
	rootFS := os.DirFS(".")
	fileSys := cachefs.New(rootFS, nil)

	f, err := fileSys.Open("file_test.go")
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	seek, ok := f.(io.ReadSeeker)
	if !ok {
		t.Error("Seek is not implemented")
	}

	b := make([]byte, 16)

	// Read at end
	N, err := seek.Seek(0, io.SeekEnd)
	if err != nil {
		t.Error(err)
		return
	}
	if N <= 0 {
		t.Errorf("Seek end: Expected non-zero location: %d", N)
	}
	n, err := seek.Read(b)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Errorf("Seek end: Expected EOF: %d %v", n, err)
	}

	// Read at start
	N, err = seek.Seek(0, io.SeekStart)
	if err != nil {
		t.Error(err)
		return
	}
	if N != 0 {
		t.Errorf("Seek start: Expected zero location: %d", N)
	}
	n, err = seek.Read(b)
	if n != len(b) || err != nil {
		t.Errorf("Seek start: Expected to read data: %d %v", n, err)
	}
	if !strings.HasPrefix(string(b), "package") {
		t.Errorf("Seek start: Expected to read the prefix \"package\": %q", b)
	}

	// Read at middle somewhere
	N, err = seek.Seek(-8, io.SeekCurrent)
	if err != nil {
		t.Error(err)
		return
	}
	if N != 8 {
		t.Errorf("Seek current: Expected location 8: %d", N)
	}
	n, err = seek.Read(b)
	if n != len(b) || err != nil {
		t.Errorf("Expected to read data: %d %v", n, err)
	}
	if !strings.HasPrefix(string(b), "cachefs_test") {
		t.Errorf("Expected to read the prefix \"cachefs_test\": %q", b)
	}

	// Test seek before start
	_, err = seek.Seek(-800, io.SeekCurrent)
	if err == nil {
		t.Error("Seek before start: Expected error")
		return
	}

	// Test seek after end
	N, err = seek.Seek(800000, io.SeekCurrent)
	if err != nil {
		t.Error("Seek after ebdL Expected no error")
		return
	}
	if N <= 0 {
		t.Errorf("Seek after end: Expected non-zero location: %d", N)
	}
}
