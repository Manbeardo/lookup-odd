package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Manbeardo/lookup-odd/layer"
	"github.com/dustin/go-humanize"
	"github.com/schollz/progressbar/v3"
)

const (
	progressInteval       time.Duration = 50 * time.Millisecond
	Layer0BitDepth                      = uint64(14)
	BytesPerLayer0Section               = uint64(1) << Layer0BitDepth
	BitsPerLayer0Section                = BytesPerLayer0Section * 8
)

var layerBitDepths = []uint64{
	Layer0BitDepth,
	6,
	7,
	11,
	7,
	7,
	7,
	5,
}

func init() {
	totalDepth := uint64(0)
	for _, bits := range layerBitDepths {
		totalDepth += bits
	}
	if totalDepth != 64 {
		log.Fatalf("expected total bit depth to be 64, but it was %d!", totalDepth)
	}
}

func main() {
	err := os.RemoveAll(layer.LayersDir)
	if err != nil {
		log.Fatalf("error deleting layers dir: %s", err)
	}

	err = os.MkdirAll(layer.LayersDir, 0o777)
	if err != nil {
		log.Fatalf("error creating layers dir: %s", err)
	}

	table, err := buildLookupTable()
	if err != nil {
		log.Fatalf("error building lookup table: %s", err)
	}
	file, err := os.OpenFile("lookup_table", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatalf("error opening lookup table file: %s", err)
	}
	defer file.Close()
	enc := gob.NewEncoder(file)
	err = enc.Encode(table)
	if err != nil {
		log.Fatalf("error encoding lookup table: %s", err)
	}
}

func buildLookupTable() (*layer.Section, error) {
	prevCodec := "no compression"
	prevLayer := []byte{}
	for i := uint64(0); i < BytesPerLayer0Section; i++ {
		prevLayer = append(prevLayer, layer.BitMask1|layer.BitMask3|layer.BitMask5|layer.BitMask7)
	}
	prevSectionCount := uint64(0)
	prevNumberCount := BitsPerLayer0Section
	for i := 1; i < len(layerBitDepths); i++ {
		bitDepth := layerBitDepths[i]
		sectionCount := uint64(1) << bitDepth
		log.Printf("layer %d (%d bits, %s values)", i, bitDepth, humanize.SIWithDigits(float64(sectionCount), 2, ""))

		ce, err := layer.NewEncoder(i)
		if err != nil {
			return nil, fmt.Errorf("creating compressing writer: %w", err)
		}
		j := uint64(0)
		progressDone := watchProgress(int(sectionCount), func() int { return int(j) })
		for ; j < sectionCount; j++ {
			ce.Encode(layer.Section{
				Layer:           i,
				Codec:           prevCodec,
				SubSectionCount: prevSectionCount,
				NumberCount:     prevNumberCount,
				Content:         prevLayer,
			})
		}
		progressDone()
		err = ce.Close()
		if err != nil {
			return nil, fmt.Errorf("closing encoder: %w", err)
		}
		prevSectionCount = sectionCount
		prevNumberCount = sectionCount * prevNumberCount
		prevLayer, prevCodec, err = ce.GetSmallestResult()
		if err != nil {
			return nil, fmt.Errorf("finding smallest result file: %w", err)
		}
	}
	return &layer.Section{
		Layer:           len(layerBitDepths),
		SubSectionCount: prevSectionCount,
		NumberCount:     0,
		Codec:           prevCodec,
		Content:         prevLayer,
	}, nil
}

func watchProgress(max int, checker func() int) func() {
	mutex := sync.Mutex{}
	p := progressbar.New(max)
	_ = p.RenderBlank()
	go func() {
		for val := checker(); val < max; val = checker() {
			mutex.Lock()
			if p.IsFinished() {
				mutex.Unlock()
				break
			}
			_ = p.Set(val)
			mutex.Unlock()

			time.Sleep(progressInteval)
		}
	}()
	return func() {
		mutex.Lock()
		_ = p.Finish()
		fmt.Println()
		mutex.Unlock()
	}
}
