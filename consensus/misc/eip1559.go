package misc

import (
	"fmt"
	"math/big"

	"github.com/ledgerwatch/turbo-geth/common/math"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/params"
)

func VerifyEip1559Header(parent, header *types.Header, notFirst bool) error {
	// Verify that the gasUsed is <= gasTarget*elasticityMultiplier
	if header.GasUsed > header.GasLimit*params.ElasticityMultiplier {
		return fmt.Errorf("exceeded elasticity multiplier: gasUsed %d, gasTarget*elasticityMultiplier %d", header.GasUsed, header.GasLimit*params.ElasticityMultiplier)
	}

	// Verify the baseFee is correct based on the parent header.
	expectedBaseFee := new(big.Int).SetUint64(params.InitialBaseFee)
	if notFirst {
		// Only calculate the correct baseFee if the parent header is
		// also a EIP-1559 header.
		expectedBaseFee = CalcBaseFee(parent)
	}
	if header.BaseFee.Cmp(expectedBaseFee) != 0 {
		var parentBaseFee big.Int
		if parent.BaseFee != nil {
			parentBaseFee.Set(parent.BaseFee)
		}
		return fmt.Errorf("invalid baseFee: expected: %d, have %d", &parentBaseFee, header.BaseFee.Int64())
	}

	return nil
}

func CalcBaseFee(parent *types.Header) *big.Int {
	// If the parent gasUsed is the same as the target, the baseFee remains unchanged.
	if parent.GasUsed == parent.GasLimit {
		return new(big.Int).Set(parent.BaseFee)
	}

	var (
		gasLimit                 = new(big.Int).SetUint64(parent.GasLimit)
		baseFeeChangeDenominator = new(big.Int).SetUint64(params.BaseFeeChangeDenominator)
	)

	var parentBaseFee big.Int
	if parent.BaseFee != nil {
		parentBaseFee.Set(parent.BaseFee)
	}
	if parent.GasUsed > parent.GasLimit {
		// If the parent block used more gas than its target, the baseFee should increase.
		gasUsedDelta := new(big.Int).SetUint64(parent.GasUsed - parent.GasLimit)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := new(big.Int).Div(x, gasLimit)
		baseFeeDelta := math.BigMax(
			new(big.Int).Div(y, baseFeeChangeDenominator),
			big.NewInt(1),
		)

		return new(big.Int).Add(parent.BaseFee, baseFeeDelta)
	} else {
		// Otherwise if the parent block used less gas than its target, the baseFee should decrease.
		gasUsedDelta := new(big.Int).SetUint64(parent.GasLimit - parent.GasUsed)
		x := new(big.Int).Mul(&parentBaseFee, gasUsedDelta)
		y := new(big.Int).Div(x, gasLimit)
		baseFeeDelta := new(big.Int).Div(y, baseFeeChangeDenominator)

		return math.BigMax(
			new(big.Int).Sub(&parentBaseFee, baseFeeDelta),
			big.NewInt(0),
		)
	}
}
