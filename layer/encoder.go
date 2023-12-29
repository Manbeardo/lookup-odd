package layer

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sync"

	"github.com/dustin/go-humanize"
)

type Encoder struct {
	layerCount  int
	encodedSize int64
	objects     chan any
	err         error
	mutex       sync.Mutex
	wg          sync.WaitGroup
}

func NewEncoder(layerCount int) (*Encoder, error) {
	ce := &Encoder{
		layerCount: layerCount,
		objects:    make(chan any),
	}

	codecWriters := []io.Writer{}
	for _, codec := range Codecs {
		codecWriter, err := ce.startCodecGoroutine(codec)
		if err != nil {
			return nil, fmt.Errorf("starting %s goroutine: %w", codec, err)
		}
		codecWriters = append(codecWriters, codecWriter)
	}

	encReader, encWriter := io.Pipe()
	encoder := gob.NewEncoder(encWriter)

	go func() {
		defer func() {
			for _, w := range codecWriters {
				if w, ok := w.(io.Closer); ok {
					err := w.Close()
					if err != nil {
						ce.reportError(fmt.Errorf("closing codec writer: %w", err))
					}
				}
			}
		}()
		n, err := io.Copy(io.MultiWriter(codecWriters...), encReader)
		if err != nil {
			ce.reportError(fmt.Errorf("fanout: %w", err))
			return
		}
		log.Printf("encoded size: %s", humanize.Bytes(uint64(n)))
		ce.encodedSize = n
	}()

	go func() {
		defer encWriter.Close()
		for obj := range ce.objects {
			err := encoder.Encode(obj)
			if err != nil {
				ce.reportError(fmt.Errorf("encoding object: %w", err))
				return
			}
		}
	}()

	return ce, nil
}

func (ce *Encoder) startCodecGoroutine(codec string) (io.WriteCloser, error) {
	file, err := os.OpenFile(
		FileName(ce.layerCount, codec),
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0o666,
	)
	if err != nil {
		return nil, fmt.Errorf("opening layer file for %s: %w", codec, err)
	}
	codecWriter, err := NewCodecWriter(file, codec)
	if err != nil {
		return nil, fmt.Errorf("creating %s codec writer: %w", codec, err)
	}
	pipeReader, pipeWriter := io.Pipe()
	ce.wg.Add(1)
	go func() {
		defer func() {
			if err := codecWriter.Close(); err != nil {
				ce.reportError(fmt.Errorf("closing codec writer: %w", err))
			}
			if err := file.Close(); err != nil {
				ce.reportError(fmt.Errorf("closing file: %w", err))
			}

			ce.wg.Done()
		}()
		_, err := io.Copy(codecWriter, pipeReader)
		if err != nil {
			ce.reportError(fmt.Errorf("%s compression routine: %w", codec, err))
			return
		}
	}()
	return pipeWriter, nil
}

func (ce *Encoder) reportError(err error) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	ce.err = errors.Join(ce.err, err)
}

func (ce *Encoder) Encode(obj any) {
	ce.objects <- obj
}

func (ce *Encoder) Close() error {
	close(ce.objects)
	ce.wg.Wait()
	return ce.err
}

func (ce *Encoder) GetSmallestResult() ([]byte, string, error) {
	smallestCodec := ""
	smallestSize := int64(math.MaxInt64)
	for _, codec := range Codecs {
		fileInfo, err := os.Stat(FileName(ce.layerCount, codec))
		if err != nil {
			return nil, "", fmt.Errorf("getting %s file info: %w", codec, err)
		}
		log.Printf(
			"%s: compressed: %s, ratio: %.2f",
			codec,
			humanize.Bytes(uint64(fileInfo.Size())),
			float64(ce.encodedSize)/float64(fileInfo.Size()),
		)
		if fileInfo.Size() < smallestSize {
			smallestCodec = codec
			smallestSize = fileInfo.Size()
		}
	}
	log.Printf("%s won the compression competition with a %s file", smallestCodec, humanize.Bytes(uint64(smallestSize)))
	buf, err := os.ReadFile(FileName(ce.layerCount, smallestCodec))
	if err != nil {
		return nil, "", fmt.Errorf("reading %s file: %w", smallestCodec, err)
	}
	return buf, smallestCodec, nil
}
