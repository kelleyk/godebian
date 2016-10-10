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

func (d *debFile) loadData(h *ar.Header, buf []byte) error {
	if !strings.HasPrefix(h.Name, "data.tar") {
		return errors.New("unexpected filename for data component")
	}

	var err error
	var r io.Reader

	r = bytes.NewReader(buf)

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
		if err != nil {
			return err
		}
	case ".lz":
		r = lzma.NewReader(r)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported compression method: %v (extension: %v)", h.Name, dataFileExt)
	}

	tarReader := tar.NewReader(r)
	for {
		h, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		fmt.Printf("  --- %v\n", h.Name)
	}

	// TODO: Need to extract the file by full path.  If the file is a symlink, resolve the symlink and look in this same
	// package.

	return nil
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
		case 1:
			// debian packaging
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
