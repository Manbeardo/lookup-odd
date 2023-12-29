# lookup-odd - the oddness of every uint64 number compressed into a 5MB binary

![Screenshot_20231228-215510](https://github.com/Manbeardo/lookup-odd/assets/698833/fea0079b-f4a2-43f3-85f3-b421f5104abe)

`lookup-odd` is a tool and library that can determine whether a uint64 is odd without resorting to using the modulus operator and it's only 86 million times slower!

Why use the modulus operator for fizzbuzz when you can import a library instead?

Credit to [@andreasjhkarlsson's blog post](https://andreasjhkarlsson.github.io/jekyll/update/2023/12/27/4-billion-if-statements.html) and [/u/joske79's comment](https://www.reddit.com/r/programming/comments/18s69kd/comment/kf5gt3o/?utm_source=share&utm_medium=web3x&utm_name=web3xcss&utm_term=1&utm_content=share_button) for inspiration!

## how it works

The `lookup-odd` binary has an embedded 13kB lookup table that contains a precomputed result for every possible uint64. The lookup table is a search tree where each generation is compressed with the codec from our ensemble that produces the best results. In order to figure out whether a number is odd, we use the search tree, decompressing each node's contents as we encounter it until we eventually find the bit that tells us whether the number is odd or even!

Each node of the search tree contains:

- a count of its child nodes
- a count of the integers it covers
- which layer it is in (I need this because the leaf nodes are encoded differently)
- which codec was used to compress its child nodes
- its encoded and compressed child nodes

## how I built the lookup table

Luckily for us, the oddness of numbers happens in such a predictable pattern that we're guaranteed that every node's value within a generation will be identical so long as they all cover the same number of numbers and that number is divisible by 2! I'm able to take advantage of that principle by memoizing the nodes for each generation as I build the table for every uint64.

In order to ensure the table is of a reasonable size, I run an ensemble of compression algorithms (zlib, gzip, lzw, and bzip2) on each generation and use the result which achieved the highest compression ratio.

In order to build the table within a reasonable amount of time, I have a hand-tuned number of bits that each generation covers. A generation that covers 6 bits has 64 (2^64) nodes. The bit depth for each generation is hand-tuned because I need the total number of bits to be 64 (we're covering uint64 after all) and ideally, each generation should meet these size guidelines:

- expanded: <2MB
- compressed: <16kB

The expanded size guideline is boring. It's just there to keep the library from using too much memory at runtime. 

The interesting one is the 16kB limit on compressed generation size. 

Most compression algorithms use a 32kB window to search for substrings, so keeping my individual nodes smaller than 16kB lets the algorithm's window see two copies of the node at a time. When the algorithm can see two copies of the node simultaneously, it's able to load all of the node's contents into its dictionary, which unlocks huge compression ratios. By tuning the bit depth of each generation, I was able to get the final table size down all the way to 18kB. That tuning is **absolutely critical**. If a single generation exceeds the 16kB size target by too much (e.g. 40kB), the next generation baloons to a few MB, then the next generation to a few GB, then my attention span runs out and I hit ctrl+C.

## install instructions

requires some version (who cares?) of Go installed locally

```bash
go install github.com/Manbeardo/lookup-odd/cmd/lookup-odd@latest
```

## regenerate the lookup table yourself

from the directory where you've cloned the repo, run

```bash
go generate ./...
```

## CLI usage

`lookup-odd <number> ...`

```bash
lookup-odd 0 1 2 3 18446744073709551615
# output:
# no
# yes
# no
# yes
# yes
```

## Benchmark results

run on an m2 macbook pro

```text
goos: darwin
goarch: arm64
pkg: github.com/Manbeardo/lookup-odd
BenchmarkIsOdd-12                     46          24702452 ns/op
BenchmarkModulusOperator-12   1000000000          0.2872 ns/op
PASS
ok      github.com/Manbeardo/lookup-odd 12.953s
```

2.4ms per operation should be plenty fast to check whether a number is odd. How many times do you need to check oddness of a number per day? Twice? Here's 10ms, go see a Star War.
