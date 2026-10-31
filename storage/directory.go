package storage

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
)

// Directory is a writable world directory. Read-only worlds can use os.DirFS.
// It tracks modified paths so Commit also synchronizes files whose handles closed.
type Directory struct {
	root  string
	dirty map[string]bool
}

func OpenDirectory(root string) (*Directory, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	return &Directory{root: root, dirty: map[string]bool{}}, nil
}

func (d *Directory) local(name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "path", Path: name, Err: fs.ErrInvalid}
	}
	return filepath.Join(d.root, filepath.FromSlash(name)), nil
}
func (d *Directory) Open(name string) (fs.File, error) {
	p, err := d.local(name)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}
func (d *Directory) OpenFile(name string, flag int) (File, error) {
	p, err := d.local(name)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p, flag, 0644)
	if err == nil && flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) != 0 {
		d.dirty[name] = true
	}
	return f, err
}
func (d *Directory) MkdirAll(name string) error {
	p, err := d.local(name)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(p, 0755); err == nil {
		d.dirty[name] = true
	}
	return err
}
func (d *Directory) Remove(name string) error {
	p, err := d.local(name)
	if err != nil {
		return err
	}
	if err = os.Remove(p); err == nil {
		d.dirty[name] = true
	}
	return err
}
func (d *Directory) Rename(oldName, newName string) error {
	oldPath, err := d.local(oldName)
	if err != nil {
		return err
	}
	newPath, err := d.local(newName)
	if err != nil {
		return err
	}
	if err = os.Rename(oldPath, newPath); err == nil {
		d.dirty[oldName] = true
		d.dirty[newName] = true
	}
	return err
}

// Commit synchronizes changed files and their parent directories. Ordinary
// filesystem commits do not provide PackFS's atomic multi-file publication.
func (d *Directory) Commit() error {
	paths := map[string]bool{}
	for name := range d.dirty {
		for {
			paths[name] = true
			if name == "." {
				break
			}
			name = path.Dir(name)
		}
	}
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	// Descendants precede parents when synchronizing newly created paths.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		p, _ := d.local(name)
		f, err := os.Open(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		syncErr := f.Sync()
		closeErr := f.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	clear(d.dirty)
	return nil
}
