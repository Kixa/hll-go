package hll

import "testing"

func TestRollup_Nil(t *testing.T) {
	_, err := Rollup(nil)

	if err == nil {
		t.Logf("rollup - expected rollup to error with nil, but did not")
		t.Fail()
	}
}

func TestRollup_Empty(t *testing.T) {
	_, err := Rollup([]Sketch{})

	if err == nil {
		t.Logf("rollup - expected rollup to error with empty list, but did not")
		t.Fail()
	}
}

func TestRollup_DiffVersion(t *testing.T) {
	s0 := createSketch()
	s0.version = "TEST0"

	s1 := createSketch()
	s1.version = "TEST1"

	_, err := Rollup([]Sketch{s0, s1})

	if err == nil {
		t.Logf("rollup - expected rollup to error with different versions, but did not")
		t.Fail()
	}
}

func TestRollup_DiffPrecision(t *testing.T) {
	s0 := createSketch()

	s1 := createSketch()
	s1.registers = make([]uint8, 1)

	_, err := Rollup([]Sketch{s0, s1})

	if err == nil {
		t.Logf("rollup - expected rollup to error with different precision/malformed registers, but did not")
		t.Fail()
	}
}

func TestRollup(t *testing.T) {
	s0 := createSketch()
	s0.registers[0] = 1

	s1 := createSketch()
	s1.registers[1] = 1

	res, err := Rollup([]Sketch{s0, s1})

	if err != nil {
		t.Fatalf("rollup - unexpected error for valid rollup: %v", err)
	}

	resRegisters := res.getRegisters()

	if resRegisters[0] != 1 || resRegisters[1] != 1 {
		t.Logf("rollup - expected rollup to contain both set registers (0 & 1), but did not")
		t.Fail()
	}
}

func TestRollup_DiffVersionToStandard(t *testing.T) {
	expectedVersion := "TEST"

	s0 := createSketch()
	s0.version = expectedVersion
	s0.registers[0] = 1

	s1 := createSketch()
	s1.version = expectedVersion
	s1.registers[1] = 1

	res, err := Rollup([]Sketch{s0, s1})

	if err != nil {
		t.Fatalf("rollup - unexpected error for valid rollup (diff version): %v", err)
	}

	resVersion := res.getVersion()

	if resVersion != expectedVersion {
		t.Logf("rollup - expected rollup version to be set to common (diff) version: %s, but got: %s", expectedVersion, resVersion)
		t.Fail()
	}

}
