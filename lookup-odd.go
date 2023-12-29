package lookupodd

import (
	"bytes"
	_ "embed"
	"encoding/gob"
	"fmt"

	"github.com/Manbeardo/lookup-odd/layer"
)

//go:generate go run -mod=mod ./cmd/generate-lookup-table

//go:embed lookup_table
var lookupTableBytes []byte

func loadLookupTable() (layer.Section, error) {
	table := layer.Section{}
	decoder := gob.NewDecoder(bytes.NewReader(lookupTableBytes))
	err := decoder.Decode(&table)
	if err != nil {
		return layer.Section{}, err
	}
	return table, nil
}

func IsOdd(num uint64) (bool, error) {
	sectionStart := uint64(0)

	section, err := loadLookupTable()
	if err != nil {
		return false, fmt.Errorf("loading embedded lookup table: %w", err)
	}
	for section.Layer > 1 {
		subSections, err := section.DecodeSubSections()
		if err != nil {
			return false, fmt.Errorf("decoding layer %d's subsections: %w", section.Layer, err)
		}
		for _, subSection := range subSections {
			section = subSection
			// this weird style of check avoids overflowing uint64
			if num <= sectionStart+(section.NumberCount-1) {
				break
			}
			sectionStart += section.NumberCount
		}
	}
	var byteToCheck byte
	for _, byteToCheck = range section.Content {
		if num <= sectionStart+7 {
			break
		}
		sectionStart += 8
	}
	switch num - sectionStart {
	case 0:
		return (byteToCheck & layer.BitMask0) > 0, nil
	case 1:
		return (byteToCheck & layer.BitMask1) > 0, nil
	case 2:
		return (byteToCheck & layer.BitMask2) > 0, nil
	case 3:
		return (byteToCheck & layer.BitMask3) > 0, nil
	case 4:
		return (byteToCheck & layer.BitMask4) > 0, nil
	case 5:
		return (byteToCheck & layer.BitMask5) > 0, nil
	case 6:
		return (byteToCheck & layer.BitMask6) > 0, nil
	case 7:
		return (byteToCheck & layer.BitMask7) > 0, nil
	default:
		return false, fmt.Errorf("something went very wrong")
	}
}
