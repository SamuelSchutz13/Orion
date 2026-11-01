//go:build !windows

package inventory

type snapEntry struct {
	name string
	ppid int32
}

func snapshotProcs() map[int32]snapEntry { return nil }
