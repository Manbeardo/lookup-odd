package main

import (
	"fmt"
	"log"

	lookupodd "github.com/Manbeardo/lookup-odd"
	"github.com/alecthomas/kong"
)

type CLI struct {
	Numbers []uint64 `arg:""`
	Verbose bool     `short:"v"`
}

func main() {
	cli := CLI{}
	kong.Parse(&cli)

	lookupodd.Verbose = cli.Verbose

	for _, num := range cli.Numbers {
		isOdd, err := lookupodd.IsOdd(num)
		if err != nil {
			log.Fatalf("error looking up %d: %s", num, err)
		}
		if isOdd {
			fmt.Println("odd")
		} else {
			fmt.Println("even")
		}
	}
}
