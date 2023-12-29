package layer

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

type Section struct {
	Layer           int
	SubSectionCount uint64
	NumberCount     uint64
	Codec           string
	Content         []byte
}

func (s Section) DecodeSubSections() ([]Section, error) {
	if s.Layer == 1 {
		return nil, fmt.Errorf("layer 1 can not be decoded, read its contents directly!")
	}
	reader, err := NewCodecReader(bytes.NewReader(s.Content), s.Codec)
	if err != nil {
		return nil, fmt.Errorf("creating decompressing reader: %w", err)
	}
	decoder := gob.NewDecoder(reader)
	subsections := make([]Section, s.SubSectionCount)
	for i := range subsections {
		err := decoder.Decode(&subsections[i])
		if err != nil {
			return nil, fmt.Errorf("decoding subsection %d: %w", i, err)
		}
	}
	return subsections, nil
}
