package debfile

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	ar "github.com/blakesmith/ar"
	"github.com/lxq/lzma"
	xz "github.com/smira/go-xz"
)

type DebFile interface {
}

type debFile struct {
}

var _ DebFile = (*debFile)(nil)

func LoadFromFile(path string) (DebFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return Load(f)
}

func Load(r io.Reader) (DebFile, error) {
	d := &debFile{}

	arReader := ar.NewReader(r)
	var arFiles []*ar.Header
	var i int
	for {
		h, err := arReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		buf := make([]byte, h.Size)
		// func (rd *Reader) Read(b []byte) (n int, err error)
		n, err := arReader.Read(buf)
		if int64(n) != h.Size {
			return nil, fmt.Errorf("read unexpected number of bytes")
		}
		if err != nil {
			return nil, err
		}

		switch i {
		case 0:
			// debian header
			if err := d.loadFormat(h, buf); err != nil {
				return nil, err
			}
		case 1:
			// debiaan control file
			if err := d.loadControl(h, buf); err != nil {
				return nil, err
			}
		case 2:
			// data-file
			if err := d.loadData(h, buf); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unexpected number of files")
		}

		arFiles = append(arFiles, h)
		i++
	}

	if i != 3 {
		return nil, errors.New("unexpected number of files")
	}

	return d, nil
}

func (d *debFile) loadFormat(h *ar.Header, buf []byte) error {
	if h.Name != "debian-binary" {
		return errors.New("unexpected filename for format component")
	}

	if string(buf) != "2.0\n" {
		return fmt.Errorf("unexpected format version: %q", string(buf))
	}

	return nil
}

func (d *debFile) loadControl(h *ar.Header, buf []byte) error {
	if h.Name != "control.tar.gz" {
		return errors.New("unexpected filename for control component")
	}

	r, err := gzip.NewReader(bytes.NewReader(buf))
	if err != nil {
		return err
	}

	rr := tar.NewReader(r)
	for {
		h, err := rr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// TODO: store data ...
		_ = h
	}

	return nil
}

func (d *debFile) loadData(h *ar.Header, buf []byte) error {
	if !strings.HasPrefix(h.Name, "data.tar") {
		return errors.New("unexpected filename for data component")
	}

	var err error
	var r io.Reader = bytes.NewReader(buf)

	dataFileExt := filepath.Ext(h.Name)
	switch dataFileExt {
	case ".tar":
	case ".xz":
		r, err = xz.NewReader(r)
		if err != nil {
			return err
		}
	case ".gz":
		r, err = gzip.NewReader(r)
		if err != nil {
			return err
		}
	case ".bz2":
		r = bzip2.NewReader(r)
	case ".lz":
		r = lzma.NewReader(r)
	default:
		return fmt.Errorf("unsupported compression method: %v (extension: %v)", h.Name, dataFileExt)
	}

	rr := tar.NewReader(r)
	for {
		h, err := rr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// TODO: store data ...
		fmt.Printf("  - %v\n", h.Name)
		if h.Linkname != "" {
			path := "./" + path.Join(filepath.Dir(h.Name), h.Linkname)
			fmt.Printf("    - %v  =>  %v\n", h.Linkname, path)
		}
	}

	// TODO: Need to extract the file by full path.  If the file is a symlink, resolve the symlink and look in this same
	// package.

	return nil
}
