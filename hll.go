package hll

import (
	"errors"
	"fmt"
	"math"
	"math/bits"

	hllProto "github.com/kixa/hll-go/proto"
	"github.com/zeebo/xxh3"
	"google.golang.org/protobuf/proto"
)

const (
	hashLength = 64

	precision = 14
	remnant   = hashLength - precision

	currentVersion = "1"
)

var (
	m  = uint64(math.Pow(2, precision))
	mf = float64(m)

	maxLinearCounting = uint64(11500)
)

var (
	// ErrorMismatchedVersion is returned from a Merge when two sketch versions do not match.
	ErrorMismatchedVersion = errors.New("sketch version mismatch")

	// ErrorMalformedPrecision is returned from a Merge when a sketch is found to have differing precisions (or
	// is malformed/incorrectly deserialized).
	ErrorMalformedPrecision = errors.New("sketch precision mismatch")
)

// Sketch is an interface that wraps a HyperLogLog implementation for counting unique elements.
type Sketch interface {
	// Insert inserts element into the Sketch.
	Insert(element []byte)

	// Estimate returns an estimate (+/-3%) of the number of uniques (cardinality) of everything that
	//has been inserted.
	Estimate() uint64

	// Merge merges this Sketch with other, returning itself (now combined with other) and an non-nil
	// error if the Merge could not be completed.
	Merge(other Sketch) (Sketch, error)

	// ProtoSerialize returns []byte representing this Sketch. The serialization format can be found
	// at: https://github.com/kixa/hll-protobuf
	ProtoSerialize() ([]byte, error)

	getRegisters() []uint8
	getVersion() string
}

type sketch struct {
	biasSet   *biases
	registers []uint8

	version string
}

// Insert inserts element into the Sketch.
func (s *sketch) Insert(element []byte) {
	h := xxh3.Hash(element)
	s.addHash(h)
}

func (s *sketch) addHash(h uint64) {
	register, zeros := getRegisterAndLeadingZeros(h)

	// Avoid 0's for the harmonic mean...
	// (As in: 1/0 is sadtimes, so we need to know whether to include this or not in estimate calculation).
	zeros += 1

	if s.registers[register] >= zeros {
		return
	}

	s.registers[register] = zeros
}

// (This translates to 14 0's, followed by 50 1's)
const bitMask = (1 << remnant) - 1

// getRegisterAndLeadingZeros returns the register to inc (bits [0..14]) and
// the number of leading zeros in the remnant [14..].
//
// Assumes: Most Significant Bit (MSB) is at index 0.
//
// Bit Tricks:
// - First 14 found by right shifting the length of remnant (50).
// - Remnant found by using bitMask and logical 'and', i.e: 0 and X = 0, 1 and X = X.
func getRegisterAndLeadingZeros(hash uint64) (uint64, uint8) {
	return hash >> remnant, uint8(bits.LeadingZeros(uint(hash&bitMask)) - precision)
}

// Estimate returns the estimated cardinality (number of unique items) inserted into this Sketch.
// It is accurate to +/-3% of the 'true' value, however in practice, it performs significantly better than that.
func (s *sketch) Estimate() uint64 {
	rawEstimate := s.rawHarmonicEstimate()

	// Bigger than largest elem in bias set, just use raw.
	if rawEstimate > s.biasSet.maxTick {
		return rawEstimate
	}

	// Less than 11,500, use LinearCount.
	if rawEstimate < maxLinearCounting {
		return s.linearCounting()
	}

	// Anything else, return interpolated bias.
	return uint64(s.biasSet.getInterpolatedBias(int(rawEstimate)) * float64(rawEstimate))
}

// This is a 'predictable' bias correction constant.
// tl;dr: http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf
var alpha = 1 / (2 * math.Log(2))

// rawHarmonicEstimate returns a harmonic average across each registers raw estimate.
func (s *sketch) rawHarmonicEstimate() uint64 {
	var sum float64
	var registersUsed float64

	for _, n := range s.registers {
		// Don't count any registers that haven't been touched.
		if n <= 0 {
			continue
		}

		registersUsed += 1

		// Classic estimate of cardinality is: 2^n (where n is number of leading 0's).
		// However, since we want the reciprocal for the harmonic case, we use 2^(-1*n)
		sum += math.Pow(2, float64(n)*-1)
	}

	// Special case: No registers used; nothing added, so no cardinality.
	if registersUsed <= 0 {
		return 0
	}

	// Bias corrected calculation is: alpha * registersUsed^2 * (sum^-1), but we can re-write that as
	// (alpha * registersUsed^2) / sum.
	return uint64((alpha * registersUsed * registersUsed) / sum)
}

func (s *sketch) linearCounting() uint64 {
	var registersUsed float64

	for _, n := range s.registers {
		if n <= 0 {
			registersUsed += 1
		}
	}

	return uint64(mf * math.Log(mf/registersUsed))
}

// Merge merges s with other, returning s for convenience. It will error if there is a version
// mismatch, or either Sketch's underlying registers are incompatible.
func (s *sketch) Merge(other Sketch) (Sketch, error) {
	if s.version != other.getVersion() {
		return nil, ErrorMismatchedVersion
	}

	otherRegisters := other.getRegisters()

	if len(s.registers) != len(otherRegisters) {
		return nil, ErrorMalformedPrecision
	}

	for i, thisZeros := range s.registers {
		otherZeros := otherRegisters[i]

		if otherZeros > thisZeros {
			s.registers[i] = otherZeros
		}
	}

	return s, nil
}

// ProtoSerialize returns an encoded protobuf version of this Sketch. The proto schema used can be
// found in the companion repository: https://github.com/kixa/hll-protobuf
func (s *sketch) ProtoSerialize() ([]byte, error) {
	registerspb := make([]uint32, len(s.registers))

	for i, r := range s.registers {
		registerspb[i] = uint32(r)
	}

	return proto.Marshal(&hllProto.Sketch{
		Version:   s.version,
		Registers: registerspb,
	})
}

func (s *sketch) getRegisters() []uint8 {
	return s.registers
}

func (s *sketch) getVersion() string {
	return s.version
}

// NewSketch returns a new Sketch using the default biases.
func NewSketch() Sketch {
	return createSketch()
}

// NewCustomSketch returns a new Sketch using the biases registered under biasKey. If these biases are not
// found (previously registered via RegisterBiases), an error will be returned.
func NewCustomSketch(biasKey string) (Sketch, error) {
	bs, exist := biasStore[biasKey]

	if !exist {
		return nil, fmt.Errorf("requested biases %s were not found - they may not have not been registered", biasKey)
	}

	s := createSketch()
	s.biasSet = bs

	return s, nil
}

func createSketch() *sketch {
	return &sketch{
		biasSet:   defaultBiases,
		registers: make([]uint8, m),

		version: currentVersion,
	}
}

// ProtoDeserialize returns a Sketch from an encoded protobuf version. The proto schema used can be
// found in the companion repository: https://github.com/kixa/hll-protobuf
func ProtoDeserialize(protoBs []byte) (Sketch, error) {
	if protoBs == nil {
		return nil, fmt.Errorf("cannot deserialize nil proto")
	}

	var sketchpb hllProto.Sketch

	err := proto.Unmarshal(protoBs, &sketchpb)

	if err != nil {
		return nil, err
	}

	s := createSketch()
	s.version = sketchpb.Version

	for i, registerpb := range sketchpb.Registers {
		s.registers[i] = uint8(registerpb)
	}

	return s, nil
}
