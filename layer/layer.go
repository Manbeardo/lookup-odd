package layer

import (
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"fmt"
	"io"
	"path/filepath"

	"github.com/dsnet/compress/bzip2"
)

const (
	LayersDir = ".tmp-layers"
)

var Codecs = []string{
	"bzip2",
	"zlib",
	"gzip",
	"lzw",
}

const (
	BitMask0 byte = 1 << iota
	BitMask1
	BitMask2
	BitMask3
	BitMask4
	BitMask5
	BitMask6
	BitMask7
)

func FileName(layerCount int, codec string) string {
	return filepath.Join(LayersDir, fmt.Sprintf("layer%d.%s", layerCount, codec))
}

func NewCodecWriter(w io.Writer, codec string) (io.WriteCloser, error) {
	switch codec {
	case "bzip2":
		return bzip2.NewWriter(w, &bzip2.WriterConfig{
			Level: bzip2.BestCompression,
		})
	case "gzip":
		return gzip.NewWriterLevel(w, gzip.BestCompression)
	case "zlib":
		return zlib.NewWriterLevel(w, zlib.BestCompression)
	case "lzw":
		return lzw.NewWriter(w, lzw.LSB, 8), nil
	default:
		return nil, fmt.Errorf("unsupported codec: %s", codec)
	}
}

func NewCodecReader(r io.Reader, codec string) (io.ReadCloser, error) {
	switch codec {
	case "bzip2":
		return bzip2.NewReader(r, &bzip2.ReaderConfig{})
	case "gzip":
		return gzip.NewReader(r)
	case "zlib":
		return zlib.NewReader(r)
	case "lzw":
		return lzw.NewReader(r, lzw.LSB, 8), nil
	default:
		return nil, fmt.Errorf("unsupported codec: %s", codec)
	}
}
