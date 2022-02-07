package hll

import (
	"fmt"
	"math/bits"
	"math/rand"
	"strconv"
	"testing"

	"github.com/zeebo/xxh3"
)

const (
	acceptableHighBound = 1.03
	acceptableLowBound  = 0.97
)

func acceptableEstimate(trueCardinality, estimate uint64) bool {
	highBound := uint64(float64(trueCardinality) * acceptableHighBound)
	lowBound := uint64(float64(trueCardinality) * acceptableLowBound)

	if estimate < lowBound || estimate > highBound {
		return false
	}

	return true
}

var alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func genPseudoRandomStr() string {
	var str string

	for i := 0; i < 20; i++ {
		str += string(alphabet[rand.Intn(len(alphabet))])
	}

	return str
}

func TestSketch_Insert(t *testing.T) {
	rand.Seed(0)

	s := createSketch()
	s.Insert([]byte(genPseudoRandomStr()))

	// One register should be filled.
	regFillCount := 0

	for _, c := range s.registers {
		if c > 0 {
			regFillCount += 1
		}
	}

	if regFillCount != 1 {
		t.Logf("sketch insert - expected to have filled: %d register(s), got: %d", 1, regFillCount)
		t.Fail()
	}
}

func TestGetRegisterAndLeadingZeros(t *testing.T) {
	rand.Seed(0)

	for i := 0; i < 100; i++ {
		v := xxh3.Hash([]byte(genPseudoRandomStr()))
		runGetRegisterAndLeadingZeros(v, t)
	}
}

func runGetRegisterAndLeadingZeros(v uint64, t *testing.T) {
	// (Use str representation to do a manual/different count)
	bs := fmt.Sprintf("%064b", v)

	first14Str := bs[0:14]
	first14Int, err := strconv.ParseInt(first14Str, 2, 64)

	if err != nil {
		t.Fatal(err)
	}

	remnantStr := bs[14:]
	remnantInt, err := strconv.ParseInt(remnantStr, 2, 64)

	if err != nil {
		t.Fatal(err)
	}

	register, leadingZeros := getRegisterAndLeadingZeros(v)

	if uint64(first14Int) != register {
		t.Fail()
		t.Logf("register (%d) - wrong register, expected: %d, got: %d", v, first14Int, register)
	}

	expectedLeadingZeros := bits.LeadingZeros64(uint64(remnantInt)) - precision

	if leadingZeros != uint8(expectedLeadingZeros) {
		t.Fail()
		t.Logf("leadingZeros (%d) - wrong count, expected: %d, got: %d", v, expectedLeadingZeros, leadingZeros)
	}
}

func TestSketch_Estimate(t *testing.T) {
	rand.Seed(0)

	s := NewSketch()

	for i := 0; i < 100_000; i++ {
		s.Insert([]byte(genPseudoRandomStr()))
	}

	estimate := s.Estimate()

	if !acceptableEstimate(100_000, estimate) {
		t.Logf("sketch merge - expected a cardinality +/-3%% of: %d, got: %d", 100_000, estimate)
		t.Fail()
	}
}

func TestSketch_Merge(t *testing.T) {
	rand.Seed(0)

	sketchA := NewSketch()

	for i := 0; i < 100_000; i++ {
		sketchA.Insert([]byte(genPseudoRandomStr()))
	}

	sketchB := NewSketch()

	for i := 0; i < 100_000; i++ {
		sketchB.Insert([]byte(genPseudoRandomStr()))
	}

	_, err := sketchA.Merge(sketchB)

	if err != nil {
		t.Fatal(err)
	}

	estimate := sketchA.Estimate()

	if !acceptableEstimate(200_000, estimate) {
		t.Logf("sketch merge - expected a cardinality +/-3%% of: %d, got: %d", 200_000, estimate)
		t.Fail()
	}
}

func TestSketch_MergeBadVersion(t *testing.T) {
	s1 := createSketch()
	s2 := createSketch()
	s2.version = "vTEST"

	_, err := s1.Merge(s2)

	if err == nil {
		t.Logf("sketch merge - expected merge with mismatched version to fail, it did not")
		t.Fail()
	}
}

func TestSketch_MergeBadPrecision(t *testing.T) {
	s1 := createSketch()
	s2 := createSketch()
	s2.registers = make([]uint8, 0)

	_, err := s1.Merge(s2)

	if err == nil {
		t.Logf("sketch merge - expected merge with malformed registers/different precision to fail, it did not")
		t.Fail()
	}
}

func TestSketch_ProtoSerialize(t *testing.T) {
	s := NewSketch()
	_, err := s.ProtoSerialize()

	if err != nil {
		t.Fatal(err)
	}
}

func TestNewSketch(t *testing.T) {
	s := NewSketch()

	if s == nil {
		t.Logf("new sketch - expected a sketch, but got nil")
		t.Fail()
	}
}

func TestNewCustomSketch(t *testing.T) {
	testBiases := map[int]float64{
		0: 0.0,
		1: 0.0,
		2: 0.0,
		3: 0.0,
		4: 0.0,
	}

	err := RegisterBiases(validBiasKey, testBiases)

	if err != nil {
		t.Fatalf("custom sketch - errored when adding biases: %v", err)
	}

	s, err := NewCustomSketch(validBiasKey)

	if err != nil {
		t.Logf("custom sketch - unexpectedly errored creating sketch: %v", err)
		t.Fail()
	}

	if s == nil {
		t.Logf("custom sketch - expected a sketch, but got nil")
		t.Fail()
	}
}

func TestNewCustomSketch_InvalidBiasKey(t *testing.T) {
	_, err := NewCustomSketch(invalidBiasKey)

	if err == nil {
		t.Logf("custom sketch - expected custom sketch to error with invalid key, but did not")
		t.Fail()
	}
}

func TestProtoDeserialize_NilProto(t *testing.T) {
	_, err := ProtoDeserialize(nil)

	if err == nil {
		t.Logf("proto deserialize - expected to fail when called with nil proto, but did not")
		t.Fail()
	}
}

func TestProtoDeserialize_GarbageProto(t *testing.T) {
	_, err := ProtoDeserialize([]byte("garbage"))

	if err == nil {
		t.Logf("proto deserialize - expected to fail when called with non-valid proto, but did not")
		t.Fail()
	}
}

func TestProtoDeserialize_Empty(t *testing.T) {
	rand.Seed(0)
	runProtoDeserialize(t, 0)
}

func TestProtoDeserialize_WithEntries(t *testing.T) {
	rand.Seed(0)
	runProtoDeserialize(t, 100)
}

func runProtoDeserialize(t *testing.T, entries int) {
	preS := NewSketch()

	for i := 0; i < entries; i++ {
		preS.Insert([]byte(genPseudoRandomStr()))
	}

	preEstimate := preS.Estimate()

	sketchBs, err := preS.ProtoSerialize()

	if err != nil {
		t.Fatal(err)
	}

	postS, err := ProtoDeserialize(sketchBs)

	if err != nil {
		t.Fatal(err)
	}

	postEstimate := postS.Estimate()

	if preEstimate != postEstimate {
		t.Logf("proto deserialize - %d entries, pre-estimate: %d does not match post-estimate: %d", entries, preEstimate, postEstimate)
		t.Fail()
	}

	if len(preS.getRegisters()) != len(postS.getRegisters()) {
		t.Logf("proto deserialize - %d entries, pre-s register len: %d does not match post-s register len: %d", entries, len(preS.getRegisters()), len(postS.getRegisters()))
		t.Fail()
	}
}
