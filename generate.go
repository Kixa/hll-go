package hll

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/zeebo/xxh3"
)

const (
	genBiasVerboseFlag = "HLL_BIAS_LOG"
)

// BiasEstimate holds an average cardinality estimate and a average resulting bias. This is produced by running a number
// of simulations on a set containing exactly TrueCardinality. (I.e. If a Sketch produces this RawEstimatedCardinality
// (without BiasCorrection or LinearCounting, we need to weight by Bias to get close to TrueCardinality).
type BiasEstimate struct {
	TrueCardinality         uint64
	RawEstimatedCardinality uint64
	Bias                    float64
}

// GenerationOptions contains parameters used for GenerateBiases.
type GenerationOptions struct {
	MaxCardinality uint64

	Repeats int

	InitialStep int
	StepRate    float64
}

// DefaultGenerationOptions returns a copy of the default GenerationOptions.
func DefaultGenerationOptions() *GenerationOptions {
	return &GenerationOptions{
		MaxCardinality: m * 7,

		Repeats: 5_000,

		InitialStep: 50,
		StepRate:    1.25,
	}
}

// GenerateBiases can be used to create a list of BiasEstimate for arbitrary []byte generated by fn. It should be
// used if defaultGeneratedBiases do not produce accurate estimates for a specific use-case (or if you have the
// patience/compute to run more precise estimates). The options used to generate the default biases are returned
// from DefaultGenerationOptions() and these are used if options is nil.
//
// Depending on fn and options this can take some time and use significant memory...
//
// WARNING: If fn produces a set of unique values less than options.MaxCardinality, this will never return.
// NOTE: For periodic log.Printf output, set envvar HLL_BIAS_LOG=1.
func GenerateBiases(fn func() []byte, options *GenerationOptions) ([]*BiasEstimate, error) {
	if options == nil {
		options = DefaultGenerationOptions()
	}

	if fn == nil {
		return nil, errors.New("invalid fn: must not be nil")
	}

	if options.MaxCardinality <= m {
		return nil, fmt.Errorf("invalid options: maxCardinality must be greater than m (%d)", m)
	}

	if options.Repeats <= 0 {
		return nil, errors.New("invalid options: repeats must be greater than 0")
	}

	if options.InitialStep <= 0 {
		return nil, errors.New("invalid options: step must be greater than 0")
	}

	if options.StepRate <= 0 {
		return nil, errors.New("invalid options: step rate must be greater than 0")
	}

	verbose := false
	if os.Getenv(genBiasVerboseFlag) == "1" {
		verbose = true
	}

	cardinalities := calculateInterpolationPoints(options.MaxCardinality, options.InitialStep, options.StepRate)
	results := make([]*BiasEstimate, len(cardinalities))

	if verbose {
		log.Printf("generateBiases - Total interpolation points: %d", len(cardinalities))
		log.Printf("generateBiases - Generating test sets...")
	}

	sets := generateSets(fn, options.MaxCardinality, options.Repeats, verbose)

	for i, cardinality := range cardinalities {
		theseEstimates := make([]int, options.Repeats)
		theseBiases := make([]float64, options.Repeats)

		for r := 0; r < options.Repeats; r++ {
			s := createSketch()

			for _, h := range sets[r][0:cardinality] {
				s.addHash(h)
			}

			rawEstimate := s.rawHarmonicEstimate()

			theseEstimates[r] = int(rawEstimate)
			theseBiases[r] = float64(cardinality) / float64(rawEstimate)
		}

		sumEstimate := 0
		sumBias := 0.0

		for k := 0; k < options.Repeats; k++ {
			sumEstimate += theseEstimates[k]
			sumBias += theseBiases[k]
		}

		estimate := sumEstimate / options.Repeats
		bias := sumBias / float64(options.Repeats)

		results[i] = &BiasEstimate{
			TrueCardinality:         cardinality,
			RawEstimatedCardinality: uint64(estimate),
			Bias:                    bias,
		}

		if verbose {
			log.Printf("generateBiases - (%d/%d): True Cardinality: %d, RawEstimate: %d, Bias: %f", i+1, len(cardinalities), cardinality, estimate, bias)
		}
	}

	return results, nil
}

// Split full range into 10ths, with an increasing step for each range.
func calculateInterpolationPoints(maxCardinality uint64, initialStep int, stepRate float64) []uint64 {
	rangeLength := maxCardinality / 10

	step := uint64(initialStep)
	nextStepChange := rangeLength

	var ticks []uint64

	for i := uint64(0); i < maxCardinality; i += step {
		if i > nextStepChange {
			nextStepChange += rangeLength
			step = uint64(float64(step) * stepRate)
		}

		ticks = append(ticks, i)
	}

	// (Don't want 0 since it's a special case)
	return ticks[1:]
}

// generateSets returns repeats number of []uint64's containing maxCardinality unique hashes produced
// from fn.
// (As above, if fn produces less uniques than maxCardinality, this will never end)
func generateSets(fn func() []byte, maxCardinality uint64, repeats int, verbose bool) [][]uint64 {
	var sets [][]uint64

	for i := 0; i < repeats; i++ {
		if verbose && (i%100 == 0) {
			log.Printf("generateBiases - Generating set: %d/%d", i, repeats)
		}

		uniques := make(map[uint64]struct{})

		totalUniquesGenerated := uint64(0)
		for totalUniquesGenerated < maxCardinality {
			candidate := xxh3.Hash(fn())

			_, exists := uniques[candidate]

			if exists {
				continue
			}

			totalUniquesGenerated += 1
			uniques[candidate] = struct{}{}
		}

		set := make([]uint64, len(uniques))

		j := 0
		for h := range uniques {
			set[j] = h
			j += 1
		}

		sets = append(sets, set)
	}

	return sets
}