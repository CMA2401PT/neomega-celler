package storage

import packfs "neomega-packfs"

// Pack wraps a borrowed container. The caller owns Close, Snapshot and Restore.
// Use the container and all its handles from the same goroutine.
type Pack struct{ *packfs.PackFS }

func (p Pack) OpenFile(name string, flag int) (File, error) {
	return p.PackFS.OpenFile(name, flag)
}

var _ Writer = Pack{}
var _ Writer = (*Directory)(nil)
