//go:build main

package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dsnet/compress/bzip2"
	"github.com/dustin/go-humanize"
	"github.com/schollz/progressbar/v3"
)

const (
	layerBits       uint64        = 28
	layerSize       uint64        = 1 << layerBits
	progressInteval time.Duration = 150 * time.Millisecond
)

const (
	Mask0 byte = 1 << iota
	Mask1
	Mask2
	Mask3
	Mask4
	Mask5
	Mask6
	Mask7
)

const OddByte = Mask1 | Mask3 | Mask5 | Mask7

var layerBitSizes = []uint64{
	33,
	17,
	8,
	6,
}

func main() {
	table, err := buildLookupTable()
	if err != nil {
		log.Fatalf("error building lookup table: %s", err)
	}
	err = os.WriteFile("../lookup_table", table, 0o666)
	if err != nil {
		log.Fatalf("error writing lookup table: %s", err)
	}
}

func buildLookupTable() ([]byte, error) {
	layer := 0
	layerBits := layerBitSizes[layer]
	layerSize := uint64(1) << layerBits
	log.Printf("layer 0 (%d bits, %d values)", layerBits, layerSize)
	sectionSize := uint64(0)
	ce, err := newCompressingEncoder()
	if err != nil {
		return nil, fmt.Errorf("creating compressing encoder: %w", err)
	}
	done := watchProgress(int(layerSize), func() int { return int(sectionSize) })
	for ; sectionSize <= layerSize; sectionSize += 8 {
		ce.Encode(OddByte)
	}
	done()
	prevLayer := ce.Close()

	prevSectionSize := sectionSize
	for layer := 1; layer < len(layerBitSizes); layer++ {
		layerBits := layerBitSizes[layer]
		layerSize := 1 << layerBits
		log.Printf("layer %d (%d bits, %d values)", layer, layerBits, layerSize)
		ce, err := newCompressingEncoder()
		if err != nil {
			return nil, fmt.Errorf("creating compressing writer: %w", err)
		}
		i := uint64(0)
		done := watchProgress(layerSize, func() int { return int(i) })
		for ; i < uint64(layerSize); i++ {
			ce.Encode(layerSection{
				Layer:   layer,
				Start:   i * prevSectionSize,
				End:     (i * prevSectionSize) + (sectionSize - 1),
				Content: prevLayer,
			})
			sectionSize += prevSectionSize
		}
		done()
		prevLayer = ce.Close()
	}
	return prevLayer, nil
}

func watchProgress(max int, checker func() int) func() {
	mutex := sync.Mutex{}
	p := progressbar.New(max)
	go func() {
		for val := checker(); val < max; val = checker() {
			mutex.Lock()
			if p.IsFinished() {
				mutex.Unlock()
				break
			}
			p.Set(val)
			mutex.Unlock()

			time.Sleep(progressInteval)
		}
	}()
	return func() {
		mutex.Lock()
		p.Finish()
		fmt.Println()
		mutex.Unlock()
	}
}

type layerSection struct {
	Layer   int
	Start   uint64
	End     uint64
	Content []byte
}

type compressingEncoder struct {
	done    chan struct{}
	objects chan []any
	buf     bytes.Buffer
}

func newCompressingEncoder() (*compressingEncoder, error) {
	ce := &compressingEncoder{
		done:    make(chan struct{}),
		objects: make(chan []any),
	}
	rEnc, wEnc := io.Pipe()
	rCom, wCom := io.Pipe()

	bufWEnc := bufio.NewWriter(wEnc)
	encoder := gob.NewEncoder(bufWEnc)
	bzip2Writer, err := bzip2.NewWriter(wCom, &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	})
	if err != nil {
		return nil, fmt.Errorf("creating bzip2 writer: %w", err)
	}

	go func() {
		defer wEnc.Close()
		defer bufWEnc.Flush()
		for arr := range ce.objects {
			for _, obj := range arr {
				err := encoder.Encode(obj)
				if err != nil {
					panic(fmt.Errorf("encoder routine: %w", err))
				}
			}
		}
	}()
	go func() {
		defer wCom.Close()
		defer bzip2Writer.Close()
		n, err := io.Copy(bzip2Writer, rEnc)
		if err != nil {
			panic(fmt.Errorf("compression routine: %w", err))
		}
		log.Printf("encoded size: %s", humanize.Bytes(uint64(n)))
	}()
	go func() {
		defer close(ce.done)
		n, err := io.Copy(&ce.buf, rCom)
		if err != nil {
			panic(fmt.Errorf("buffer building routine: %w", err))
		}
		log.Printf("compressed size: %s", humanize.Bytes(uint64(n)))
	}()

	return ce, nil
}

func (ce *compressingEncoder) Encode(arr ...any) {
	ce.objects <- arr
}

func (ce *compressingEncoder) Close() []byte {
	close(ce.objects)
	<-ce.done
	return ce.buf.Bytes()
}
