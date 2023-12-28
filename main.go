package main

import "github.com/alecthomas/kong"

//go:generate protoc --go_out=. --go_opt=paths=source_relative pb/OddLookup.proto

//go:generate go run -mod=mod scripts/generate_lookup_table.go

type CLI struct {
	Numbers []uint64 `arg:""`
}

func main() {
	cli := CLI{}
	kong.Parse(&cli)
}
