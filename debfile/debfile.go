package debfile

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ar "github.com/blakesmith/ar"
	"github.com/lxq/lzma"
	"github.com/pkg/errors"
	xz "github.com/smira/go-xz"
)

type DebFile interface {
	Control() Tarball
	Data() Tarball
}

type debFile struct {
	control Tarball
	data    Tarball
}

var _ DebFile = (*debFile)(nil)

func (d *debFile) Control() Tarball {
	return d.control
}

func (d *debFile) Data() Tarball {
	return d.data
}

func LoadFromFile(path string) (DebFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return Load(f)
}

func Load(r io.Reader) (DebFile, error) {
	d := &debFile{
		control: Tarball{Contents: make(map[string]TarballEntry)},
		data:    Tarball{Contents: make(map[string]TarballEntry)},
	}

	arReader := ar.NewReader(r)
	var arFiles []*ar.Header
	var i int
	for {
		h, err := arReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "failed to get next part from package archive")
		}

		buf := make([]byte, h.Size)
		n, err := arReader.Read(buf)
		if int64(n) != h.Size {
			return nil, errors.New("read unexpected number of bytes")
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to read part from package archive")
		}

		switch i {
		case 0:
			// debian header
			if err := d.loadFormat(h, buf); err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to load %q from package archive", h.Name))
			}
		case 1:
			// debiaan control file
			if err := d.loadControl(h, buf); err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to load %q from package archive", h.Name))
			}
		case 2:
			// data-file
			if err := d.loadData(h, buf); err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to load %q from package archive", h.Name))
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
		return errors.Wrap(err, "failed to create gzip reader")
	}

	d.control, err = loadTarball(tar.NewReader(r))
	if err != nil {
		return errors.Wrap(err, "failed to load tarball")
	}

	return nil
}

func (d *debFile) loadData(h *ar.Header, buf []byte) error {
	if !strings.HasPrefix(h.Name, "data.tar") {
		return errors.New("unexpected filename for data component")
	}

	var err error
	var r io.Reader = bytes.NewReader(buf)

	// TODO: Make compression type visible somehow?
	dataFileExt := filepath.Ext(h.Name)
	switch dataFileExt {
	case ".tar":
	case ".xz":
		r, err = xz.NewReader(r)
		if err != nil {
			return errors.Wrap(err, "failed to create xz reader")
		}
	case ".gz":
		r, err = gzip.NewReader(r)
		if err != nil {
			return errors.Wrap(err, "failed to create gzip reader")
		}
	case ".bz2":
		r = bzip2.NewReader(r)
	case ".lz":
		r = lzma.NewReader(r)
	default:
		return fmt.Errorf("unsupported compression method: %v (extension: %v)", h.Name, dataFileExt)
	}

	d.data, err = loadTarball(tar.NewReader(r))
	if err != nil {
		return err
	}

	return nil
}
