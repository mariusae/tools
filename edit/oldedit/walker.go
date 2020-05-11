package main

import (
	"os"
	"path/filepath"
	"sort"
)

var ignoreDirs = map[string]bool{
	".git":   true,
	".svn":   true,
	"_build": true,
}

// TODO: store errors
type Walker struct {
	err  error
	path string
	info os.FileInfo
	todo []string
}

func NewWalker(roots ...string) *Walker {
	return &Walker{todo: roots}
}

func (w *Walker) Next() bool {
Next:
	if len(w.todo) == 0 || w.err != nil {
		return false
		//		return io.EOF
	}

	w.path = w.todo[0]
	w.todo = w.todo[1:]
	var err error
	w.info, err = os.Lstat(w.path)
	if err != nil {
		goto Next
	}

	if w.info.IsDir() {
		if _, ok := ignoreDirs[filepath.Base(w.path)]; ok {
			goto Next
		}

		var paths []string
		paths, w.err = readDirNames(w.path)
		if w.err != nil {
			return false
		}
		for i := range paths {
			paths[i] = filepath.Join(w.path, paths[i])
		}
		w.todo = append(paths, w.todo...)
	}

	return true
}

func (w *Walker) Path() string {
	return w.path
}

func (w *Walker) Info() os.FileInfo {
	return w.info
}

func (w *Walker) Err() error {
	return w.err
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
