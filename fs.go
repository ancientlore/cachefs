/*
Package cachefs implements a read-only cache around a fs.FS, using groupcache.

Using cachefs is straightforward:

	// Setup groupcache (in this example with no peers)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })

	// Create the cached file system with group name "groupName", a 10MB cache, and a ten second expiration
	cachedFileSystem := cachefs.New(os.DirFS("."), &cachefs.Config{GroupName: "groupName", SizeInBytes: 10*1024*1024, Duration: 10*time.Second})

	// Use the file system as usual...

cachefs "wraps" the underlying file system with caching. You can specify groupcache parameters - the group name
and the cache size.

groupcache does not support expiration, but cachefs supports quantizing values so that expiration happens
around the expiration duration provided. Expiration can be disabled by specifying 0 for the duration.

See https://pkg.go.dev/github.com/golang/groupcache for more information on groupcache.
*/
package cachefs

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/groupcache"
	"github.com/google/uuid"
)

// Config stores the configuration settings of your cache.
type Config struct {
	GroupName   string        // Name of the groupcache group
	SizeInBytes int64         // Size of the cache
	Duration    time.Duration // Duration after which items can expire
	NoStat      bool          // Don't do extra file Stat calls in ReadDir
}

// An cacheFS provides cached access to a hierarchical file system.
type cacheFS struct {
	fs       fs.FS
	duration time.Duration
	cache    *groupcache.Group
}

// Open opens the named file.
//
// When Open returns an error, it should be of type *fs.PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// fs.ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (cfs *cacheFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	var (
		buf groupcache.ByteView
		q   = make(url.Values, 2)
		f   file
	)
	t := quantize(time.Now(), cfs.duration, name)
	q.Set("t", strconv.FormatInt(t, 10))
	q.Set("path", name)
	ctx := context.Background()
	err := cfs.cache.Get(ctx, q.Encode(), groupcache.ByteViewSink(&buf))
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	rdr := countingReader{Reader: buf.Reader()}
	decoder := gob.NewDecoder(&rdr)
	err = decoder.Decode(&f)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	// rest of the slice is the file data
	f.ReadSeeker = buf.SliceFrom(rdr.Count()).Reader()

	return &f, nil
}

// New creates a new cached FS around innerFS using groupcache with the given
// configuration. The returned FS is read-only. If config is nil, it defaults
// to a 1MB cache using a random GUID as a name.
func New(innerFS fs.FS, config *Config) fs.FS {
	if config == nil {
		config = &Config{
			GroupName:   uuid.NewString(),
			SizeInBytes: 1024 * 1024,
		}
	}
	noStat := config.NoStat
	return &cacheFS{
		duration: config.Duration,
		cache: groupcache.NewGroup(config.GroupName, config.SizeInBytes, groupcache.GetterFunc(
			func(ctx context.Context, key string, dest groupcache.Sink) error {
				// Parse query which contains quantize info and path
				q, err := url.ParseQuery(key)
				if err != nil {
					return fmt.Errorf("invalid cache key: %w", err)
				}
				// Open file
				f, err := innerFS.Open(q.Get("path"))
				if err != nil {
					return err
				}
				defer f.Close()
				// Get file info
				info, err := f.Stat()
				if err != nil {
					return err
				}
				// setup result data
				resultFile := file{
					FI: fileInfo{
						Nm: info.Name(),
						Sz: info.Size(),
						Md: info.Mode(),
						Mt: info.ModTime(),
					},
				}
				var data []byte
				if info.IsDir() {
					// Read directory info
					entries, err := f.(fs.ReadDirFile).ReadDir(-1)
					if err != nil {
						return err
					}
					resultFile.Dirs = make([]dirEntry, 0, len(entries))
					for _, entry := range entries {
						if !noStat {
							fi, err := entry.Info()
							if err != nil {
								// Pretend it doesn't exist, like (*os.File).Readdir does.
								continue
							}
							resultFile.Dirs = append(resultFile.Dirs, dirEntry{
								FI: fileInfo{
									Nm: fi.Name(),
									Md: fi.Mode(),
									Sz: fi.Size(),
									Mt: fi.ModTime(),
								},
							})
						} else {
							resultFile.Dirs = append(resultFile.Dirs, dirEntry{
								FI: fileInfo{
									Nm: entry.Name(),
									Md: entry.Type(),
								},
							})
						}
					}
				} else {
					// Read file
					data, err = io.ReadAll(f)
					if err != nil {
						return err
					}
				}
				// Encode the result
				var buf bytes.Buffer
				encoder := gob.NewEncoder(&buf)
				err = encoder.Encode(resultFile)
				if err != nil {
					return err
				}
				// Write data afterward to avoid extra copies of large stuff
				n, err := buf.Write(data)
				if err != nil {
					return err
				}
				if n != len(data) {
					return fmt.Errorf("wrote incorrect number of  bytes: %d of %d", n, len(data))
				}
				return dest.SetBytes(buf.Bytes())
			})),
	}
}
