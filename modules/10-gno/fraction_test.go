package gno

import (
	"testing"

	cmtmath "github.com/cometbft/cometbft/libs/math"
	"github.com/stretchr/testify/require"
)

func TestNewFractionFromTm(t *testing.T) {
	testCases := []struct {
		name        string
		tmFraction  cmtmath.Fraction
		expectedNum uint64
		expectedDen uint64
	}{
		{
			name:        "default trust level 1/3",
			tmFraction:  cmtmath.Fraction{Numerator: 1, Denominator: 3},
			expectedNum: 1,
			expectedDen: 3,
		},
		{
			name:        "two thirds",
			tmFraction:  cmtmath.Fraction{Numerator: 2, Denominator: 3},
			expectedNum: 2,
			expectedDen: 3,
		},
		{
			name:        "one",
			tmFraction:  cmtmath.Fraction{Numerator: 1, Denominator: 1},
			expectedNum: 1,
			expectedDen: 1,
		},
		{
			name:        "zero numerator",
			tmFraction:  cmtmath.Fraction{Numerator: 0, Denominator: 1},
			expectedNum: 0,
			expectedDen: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fraction := NewFractionFromTm(tc.tmFraction)
			require.Equal(t, tc.expectedNum, fraction.Numerator)
			require.Equal(t, tc.expectedDen, fraction.Denominator)
		})
	}
}

func TestFraction_ToTendermint(t *testing.T) {
	testCases := []struct {
		name        string
		fraction    Fraction
		expectedNum uint64
		expectedDen uint64
	}{
		{
			name:        "one third",
			fraction:    Fraction{Numerator: 1, Denominator: 3},
			expectedNum: 1,
			expectedDen: 3,
		},
		{
			name:        "two thirds",
			fraction:    Fraction{Numerator: 2, Denominator: 3},
			expectedNum: 2,
			expectedDen: 3,
		},
		{
			name:        "large values",
			fraction:    Fraction{Numerator: 1000000, Denominator: 3000000},
			expectedNum: 1000000,
			expectedDen: 3000000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmFraction := tc.fraction.ToTendermint()
			require.Equal(t, tc.expectedNum, tmFraction.Numerator)
			require.Equal(t, tc.expectedDen, tmFraction.Denominator)
		})
	}
}

func TestFraction_RoundTrip(t *testing.T) {
	original := cmtmath.Fraction{Numerator: 2, Denominator: 3}

	// Convert to Fraction
	fraction := NewFractionFromTm(original)

	// Convert back to CometBFT fraction
	roundTripped := fraction.ToTendermint()

	require.Equal(t, original.Numerator, roundTripped.Numerator)
	require.Equal(t, original.Denominator, roundTripped.Denominator)
}

func TestDefaultTrustLevel(t *testing.T) {
	require.Equal(t, uint64(1), DefaultTrustLevel.Numerator)
	require.Equal(t, uint64(3), DefaultTrustLevel.Denominator)
}
