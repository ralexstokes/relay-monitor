package consensus

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestComputeBaseF(t *testing.T) {
	for _, tc := range []struct {
		parentGasTarget  uint64
		parentGasUsed    uint64
		baseFeeAsHex     string
		nextBaseFeeAsHex string
	}{
		{
			parentGasTarget:  15_000_000,
			parentGasUsed:    15_000_000,
			baseFeeAsHex:     "0x7",
			nextBaseFeeAsHex: "0x7",
		},
		{
			// from geth tests
			parentGasTarget:  10000000,
			parentGasUsed:    9000000,
			baseFeeAsHex:     "0x3b9aca00",
			nextBaseFeeAsHex: "0x3adc0de0",
		},
		{
			parentGasTarget:  15_000_000,
			parentGasUsed:    20_000_000,
			baseFeeAsHex:     "0x7",
			nextBaseFeeAsHex: "0x8",
		},
	} {
		baseFeeValue := uint256.NewInt(0)
		err := baseFeeValue.UnmarshalText([]byte(tc.baseFeeAsHex))
		if err != nil {
			t.Fatal(err)
		}
		baseFee := baseFeeValue.ToBig()
		newBaseFee := computeBaseFee(tc.parentGasTarget, tc.parentGasUsed, baseFee)

		expectedBaseFee := uint256.NewInt(0)
		err = expectedBaseFee.UnmarshalText([]byte(tc.nextBaseFeeAsHex))
		if err != nil {
			t.Fatal(err)
		}
		if !newBaseFee.Eq(expectedBaseFee) {
			t.Fatal("wrong base fee computed:", newBaseFee, "but expected", expectedBaseFee)
		}
	}
}
