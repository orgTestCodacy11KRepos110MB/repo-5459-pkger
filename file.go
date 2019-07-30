package pkger

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gobuffalo/here"
)

const timeFmt = time.RFC3339Nano

type File struct {
	info   *FileInfo
	her    here.Info
	path   Path
	data   []byte
	index  *index
	Source io.ReadCloser
}

func (f *File) Open(name string) (http.File, error) {
	if f.index == nil {
		f.index = &index{
			Files: map[Path]*File{},
		}
	}
	pt, err := Parse(name)
	if err != nil {
		return nil, err
	}

	if len(pt.Pkg) == 0 {
		pt.Pkg = f.path.Pkg
	}

	h := httpFile{}

	if pt == f.path {
		h.File = f
	} else {
		of, err := f.index.Open(pt)
		if err != nil {
			return nil, err
		}
		defer of.Close()
		h.File = of
	}

	if len(f.data) > 0 {
		h.crs = &byteCRS{bytes.NewReader(f.data)}
		return h, nil
	}

	bf, err := os.Open(h.File.Path())
	if err != nil {
		return h, err
	}
	fi, err := bf.Stat()
	if err != nil {
		return h, err
	}
	if fi.IsDir() {
		return h, nil
	}

	if err != nil {
		return nil, err
	}

	h.crs = bf
	return h, nil
}

func (f File) Stat() (os.FileInfo, error) {
	if f.info == nil {
		return nil, os.ErrNotExist
	}
	return f.info, nil
}

func (f *File) Close() error {
	if f.Source == nil {
		return nil
	}
	if c, ok := f.Source.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (f File) Name() string {
	return f.info.Name()
}

func (f File) Path() string {
	dir := f.her.Dir
	if filepath.Base(dir) == f.Name() {
		return dir
	}
	fp := filepath.Join(dir, f.Name())
	return fp
}

func (f File) String() string {
	if f.info == nil {
		return ""
	}
	b, _ := json.MarshalIndent(f.info, "", "  ")
	return string(b)
}

func (f *File) Read(p []byte) (int, error) {
	if f.Source != nil {
		return f.Source.Read(p)
	}

	of, err := os.Open(f.Path())
	if err != nil {
		return 0, err
	}
	f.Source = of
	return f.Source.Read(p)
}

// Readdir reads the contents of the directory associated with file and returns a slice of up to n FileInfo values, as would be returned by Lstat, in directory order. Subsequent calls on the same file will yield further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case, if Readdir returns an empty slice, it will return a non-nil error explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in a single slice. In this case, if Readdir succeeds (reads all the way to the end of the directory), it returns the slice and a nil error. If it encounters an error before the end of the directory, Readdir returns the FileInfo read until that point and a non-nil error.
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	of, err := os.Open(f.Path())
	if err != nil {
		return nil, err
	}
	defer of.Close()
	return of.Readdir(count)
}