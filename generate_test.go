package hll

import "testing"

func TestDefaultGenerationOptions(t *testing.T) {
	opts := DefaultGenerationOptions()

	if opts == nil {
		t.Fatal("default generation options - expected some options, got nil")
	}
}

var testBiasFn = func() []byte {
	return []byte(genPseudoRandomStr())
}

func TestGenerateBiases_NilFn(t *testing.T) {
	_, err := GenerateBiases(nil, nil)

	if err == nil {
		t.Fatal("generate biases - expected to error given nil fn, did not")
	}
}

func TestGenerateBiases_InvalidMaxCardinality(t *testing.T) {
	_, err := GenerateBiases(testBiasFn, &GenerationOptions{
		MaxCardinality: 0,
		Repeats:        1,
		InitialStep:    1,
		StepRate:       1,
	})

	if err == nil {
		t.Fatal("generate biases - expected to error given bad step rate option, did not")
	}
}

func TestGenerateBiases_InvalidRepeats(t *testing.T) {
	_, err := GenerateBiases(testBiasFn, &GenerationOptions{
		MaxCardinality: m * 2,
		Repeats:        0,
		InitialStep:    1,
		StepRate:       1,
	})

	if err == nil {
		t.Fatal("generate biases - expected to error given bad repeat option, did not")
	}
}

func TestGenerateBiases_InvalidInitialStep(t *testing.T) {
	_, err := GenerateBiases(testBiasFn, &GenerationOptions{
		MaxCardinality: m * 2,
		Repeats:        1,
		InitialStep:    0,
		StepRate:       1,
	})

	if err == nil {
		t.Fatal("generate biases - expected to error given bad initial step option, did not")
	}
}

func TestGenerateBiases_InvalidStepRate(t *testing.T) {
	_, err := GenerateBiases(testBiasFn, &GenerationOptions{
		MaxCardinality: m * 2,
		Repeats:        1,
		InitialStep:    1,
		StepRate:       0,
	})

	if err == nil {
		t.Fatal("generate biases - expected to error given bad step rate option, did not")
	}
}

func TestGenerateBiases(t *testing.T) {
	bs, err := GenerateBiases(testBiasFn, &GenerationOptions{
		MaxCardinality: m + 1,
		Repeats:        1,
		InitialStep:    100,
		StepRate:       1,
	})

	if err != nil {
		t.Fatalf("generate biases - unexpected error generating biases: %v", err)
	}

	expectedBiases := (m + 1) / 100

	if uint64(len(bs)) != expectedBiases {
		t.Fatalf("generate biases - expected %d biases to be returned, received list with len: %d", expectedBiases, len(bs))
	}
}

type interpolationTestParams struct {
	maxCardinality uint64
	initialStep    int
	stepRate       float64

	expectedPoints int
}

var interpolationPointsTestInput = []interpolationTestParams{
	{100, 10, 1, 9},
	{100, 10, 1.5, 5},
	{100, 1, 2, 22},
}

func TestCalculateInterpolationPoints(t *testing.T) {
	for _, input := range interpolationPointsTestInput {
		res := calculateInterpolationPoints(input.maxCardinality, input.initialStep, input.stepRate)

		if len(res) != input.expectedPoints {
			t.Logf("calculate interpolation points - expected: %d, got: %d (max cardinality: %d, initial step: %d, step rate: %f",
				input.expectedPoints, len(res), input.maxCardinality, input.initialStep, input.stepRate)
			t.Fail()
		}
	}
}

func TestGenerateSets(t *testing.T) {
	res := generateSets(testBiasFn, 10, 2, false)

	if res == nil {
		t.Fatalf("generate sets - expected sets to be returned, got nil")
	}

	if len(res) != 2 {
		t.Fatalf("generate sets - expected 2 sets to be returned, got: %d", len(res))
	}

	for i, set := range res {
		if len(set) != 10 {
			t.Logf("generate sets - expected set %d to have len %d, got: %d", i, 10, len(set))
		}
	}
}
