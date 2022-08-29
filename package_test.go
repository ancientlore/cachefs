package cachefs_test

import (
	"log"
	"testing"

	"github.com/mailgun/groupcache/v2"
)

func TestMain(t *testing.M) {
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })
	t.Run()
	g := groupcache.GetGroup("TestFS")
	if g != nil {
		s := g.CacheStats(groupcache.HotCache)
		log.Printf("Hot Cache  %#v", s)
		s = g.CacheStats(groupcache.MainCache)
		log.Printf("Main Cache %#v", s)
	}
}
