# cachefs

Package `cachefs` implements a read-only cache around a `fs.FS`, using `groupcache`.

Using `cachefs` is straightforward:

	// Setup groupcache (in this example with no peers)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })

	// Create the cached file system with group name "groupName", a 10MB cache, and a ten second expiration
	cachedFileSystem := cachefs.New(os.DirFS("."), &cachefs.Config{GroupName: "groupName", SizeInBytes: 10*1024*1024, Duration: 10*time.Second})

	// Use the file system as usual...

`cachefs` "wraps" the underlying file system with caching. You can specify groupcache parameters - the group name
and the cache size.

`groupcache` does not support expiration, but `cachefs` supports quantizing values so that expiration happens
around the expiration duration provided. Expiration can be disabled by specifying 0 for the duration.

See https://pkg.go.dev/github.com/golang/groupcache for more information on `groupcache`.
