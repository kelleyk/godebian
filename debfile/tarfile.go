package debfile

import (
	"archive/tar"
	"fmt"
	"io"
)

type Tarball struct {
	Contents map[string]TarballEntry
}

type TarballEntry struct {
	Header *tar.Header
	Data   []byte
}

func (e *TarballEntry) IsReg() bool {
	switch e.Header.Typeflag {
	case tar.TypeReg, tar.TypeRegA:
		return true
	default:
		return false
	}
}

func (e *TarballEntry) IsSymlink() bool {
	return e.Header.Typeflag == tar.TypeSymlink
}

func loadTarball(r *tar.Reader) (Tarball, error) {
	tarball := Tarball{Contents: make(map[string]TarballEntry)}

	for {
		h, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return tarball, err
		}

		buf := make([]byte, h.Size)
		switch h.Typeflag {
		case tar.TypeSymlink, tar.TypeLink, tar.TypeChar, tar.TypeBlock, tar.TypeDir, tar.TypeFifo:
		case tar.TypeReg, tar.TypeRegA:
			if h.Size > 0 {
				if err := tarReadAll(r, buf); err != nil {
					return tarball, err
				}
			}
		default:
			return tarball, fmt.Errorf("unexpected type flag for entry: %v", h.Name)
		}

		if len(h.Name) < 1 || h.Name[0] != '.' {
			return tarball, fmt.Errorf("unexpected filename in tarball: %v", h.Name)
		}
		tarball.Contents[h.Name[1:]] = TarballEntry{Header: h, Data: buf}
	}

	return tarball, nil
}

// TODO: Clean this up.
func tarReadAll(r *tar.Reader, buf []byte) error {
	for {
		if len(buf) == 0 {
			return nil
		}
		n, err := r.Read(buf)
		if n == 0 && err == io.EOF {
			return fmt.Errorf("got unexpectedly early EOF")
		}
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
}
