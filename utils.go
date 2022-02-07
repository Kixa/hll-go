package hll

import "fmt"

// Rollup merges sketches into a single (new) Sketch that is slightly more efficient than
// successively merging each into a common base, one at a time.
func Rollup(sketches []Sketch) (Sketch, error) {
	if len(sketches) <= 0 || sketches == nil {
		return nil, fmt.Errorf("rollup requires a list of sketches")
	}

	// Validate version and len(registers).
	firstVersion := sketches[0].getVersion()
	firstLen := len(sketches[0].getRegisters())

	for i := 1; i < len(sketches); i++ {
		if sketches[i].getVersion() != firstVersion {
			return nil, fmt.Errorf("rollup requires a list of sketches with the same version")
		}

		if len(sketches[i].getRegisters()) != firstLen {
			return nil, fmt.Errorf("rollup requires a list of sketches with the same precision (len of registers)")
		}
	}

	base := createSketch()

	if base.version != firstVersion {
		base.version = firstVersion
	}

	// Pull the registers for sketches.
	registers := make([][]uint8, len(sketches))

	for i, sk := range sketches {
		registers[i] = sk.getRegisters()
	}

	// For each register, take the highest entry for this i across each other sketch.
	for i := 0; i < len(base.registers); i++ {
		max := uint8(0)

		for _, reg := range registers {
			if max < reg[i] {
				max = reg[i]
			}
		}

		base.registers[i] = max
	}

	return base, nil
}
