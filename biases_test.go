package hll

import (
	"testing"
)

func TestBiasTicksIsSorted(t *testing.T) {
	previous := 0
	for i, tick := range defaultBiases.ticks {
		if tick < previous {
			t.Logf("bias ticks is not sorted: element %d (%d) is smaller than prior (%d)", i, tick, previous)
			t.FailNow()
			return
		}
		previous = tick
	}
}

func TestGetInterpolatedBias(t *testing.T) {
	sum := 0.0

	for _, tick := range defaultBiases.ticks[0:4] {
		sum += defaultBiases.store[tick]
	}

	expected := sum / 4
	result := defaultBiases.getInterpolatedBias(defaultBiases.ticks[0])

	if result != expected {
		t.Fail()
		t.Logf("interpolated bias - expected: %f, got: %f (from elements [0:4])", expected, result)
	}
}

func TestGetNeighbourTicks_Standard(t *testing.T) {
	validateTicks(t, defaultBiases.getNeighbourTicks(10_000))
}

func TestGetNeighbourTicks_Underflow(t *testing.T) {
	validateTicks(t, defaultBiases.getNeighbourTicks(0))
}

func TestGetNeighbourTicks_Overflow(t *testing.T) {
	validateTicks(t, defaultBiases.getNeighbourTicks(1_000_000))
}

func validateTicks(t *testing.T, ticks []int) {
	if len(ticks) != 4 {
		t.Fail()
		t.Logf("validate ticks - expected len 6, got: %d", len(ticks))
	}
}

const (
	validBiasKey   = "custom"
	invalidBiasKey = ""
)

func TestRegisterBiases_InvalidKey(t *testing.T) {
	err := RegisterBiases(invalidBiasKey, nil)

	if err == nil {
		t.Fatalf("register biases - expected to error when given empty str for key, but did not")
	}
}

func TestRegisterBiases_InvalidEmptyBiases(t *testing.T) {
	err := RegisterBiases(validBiasKey, nil)

	if err == nil {
		t.Fatalf("register biases - expected to error when given empty biases, but did not")
	}
}

func TestRegisterBiases(t *testing.T) {
	testBiases := map[int]float64{
		0: 0.0,
		1: 0.0,
		2: 0.0,
		3: 0.0,
		4: 0.0,
	}

	err := RegisterBiases(validBiasKey, testBiases)

	if err != nil {
		t.Fatalf("register biases - errored when adding biases (shouldn't have): %v", err)
	}

	_, exists := biasStore[validBiasKey]

	if !exists {
		t.Fatalf("register biases - given bias key doesn't exist in store after registration")
	}
}
