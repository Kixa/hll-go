hll-go
---

[![GoDoc](https://godoc.org/github.com/kixa/hll-go?status.svg)](https://godoc.org/github.com/kixa/hll-go) 

An optimised and opinionated version of [HyperLogLog](https://en.wikipedia.org/wiki/HyperLogLog), an algorithm for counting unique elements.

This is uses many techniques from ["HyperLogLog in Practice"](https://research.google/pubs/pub40671), most notably bias correction. It also includes serialisation to [protobuf](https://github.com/kixa/hll-protobuf), as well as methods to aid pre-calculation of a custom set of biases if the defaults do not improve accuracy enough for a particular use-case.

## Differences

This implementation is currently used effectively in production against several billions of requests per day, with cardinalities ranging between 20,000 and >500,000,000. Significant testing shows accuracy has an upper-bound of +/-3%, but in practice it is almost always <+/-1%.

The chief design goals were:
* Ability to accurately handle a large range of cardinalities, starting at ~20,000
* Known up-front memory allocation
* Fast insert
* Fast merge
* Protobuf serialisation

As such, it contains several changes from general purpose implementations (such as [HyperLogLog](https://github.com/axiomhq/hyperloglog)). Namely:

* Fixed precision of 14 
* 8 bit registers, no tailcuts
* No sparse representation
* Fixed use of 16.4kb of memory*
* Built-in default bias correction
* Protobuf `[]byte` output
* Optimised merges, including a [rollup helper](utils.go) for merging several Sketches into one

(* Runtime dependent)

## Usage

First initialise a `Sketch`, then add `[]byte` via `.Insert(...)`. Estimates can be obtained from a `Sketch` at any point via `.Estimate()`.

A simple example follows:

```go
package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"

	"github.com/kixa/hll-go"
)

func main() {
	// Initialise sketch using default biases.
	sketch := hll.NewSketch()

	for i := 0; i < 100; i++ {
		bs, err := genPseudoRandomBytes()

		if err != nil {
			log.Fatalf("couldn't generate random bytes: %v", err)
		}

		// Insert bytes into the sketch.
		sketch.Insert(bs)
	}

	// Get an estimate.
	estimate := sketch.Estimate()

	fmt.Printf("inserted 100 elements, estimate: %d\n", estimate)
}

func genPseudoRandomBytes() ([]byte, error) {
	r := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, r)

	if err != nil {
		return nil, err
	}

	return r, nil
}

```

## Custom Biases

As described in ["HyperLogLog in Practice"](https://research.google/pubs/pub40671), interpolated bias correction can be applied at low cardinality estimates (<100,000) to improve accuracy. 

This is handled transparently in hll-go since the default biases provided in [biases.go](biases.go) work for almost all use-cases. However, if you wish to use your own custom biases, you can:
* (Optionally) Create a set of biases using your own parameters and generation `fn`.
* Register your bias map (`map[int]float64`) via `RegisterBiases(...)` with a custom `key`.
* Create any new sketches via `NewCustomSketch(...)` with the same `key`.

(**NOTE**: Protobuf serialized sketches **WILL NOT** contain any custom biases. To re-use a custom set for estimates after de-serialisation from protobuf, initialise an empty `Sketch` with the custom biases via `NewCustomSketch(...)`, then `Merge` in the de-serialized one)

## License

Distributed under MIT License. See [LICENSE.md](LICENSE.md) for more information.
