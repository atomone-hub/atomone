package gno

import (
	"testing"
	"time"

	cmtmath "github.com/cometbft/cometbft/libs/math"
	"github.com/stretchr/testify/require"
)

func TestLCDefaultTrustLevel(t *testing.T) {
	require.Equal(t, uint64(1), LCDefaultTrustLevel.Numerator)
	require.Equal(t, uint64(3), LCDefaultTrustLevel.Denominator)
}

func TestValidateTrustLevel(t *testing.T) {
	testCases := []struct {
		name      string
		trustLevel cmtmath.Fraction
		expectErr bool
	}{
		{
			name:      "valid - exactly 1/3",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 3},
			expectErr: false,
		},
		{
			name:      "valid - 2/3",
			trustLevel: cmtmath.Fraction{Numerator: 2, Denominator: 3},
			expectErr: false,
		},
		{
			name:      "valid - exactly 1",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 1},
			expectErr: false,
		},
		{
			name:      "valid - 1/2",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 2},
			expectErr: false,
		},
		{
			name:      "invalid - less than 1/3 (1/4)",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 4},
			expectErr: true,
		},
		{
			name:      "invalid - numerator > denominator",
			trustLevel: cmtmath.Fraction{Numerator: 4, Denominator: 3},
			expectErr: true,
		},
		{
			name:      "invalid - denominator is zero",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 0},
			expectErr: true,
		},
		{
			name:      "invalid - both zero",
			trustLevel: cmtmath.Fraction{Numerator: 0, Denominator: 0},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTrustLevel(tc.trustLevel)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSafeMul(t *testing.T) {
	testCases := []struct {
		name          string
		a             int64
		b             int64
		expectedResult int64
		expectedOverflow bool
	}{
		{
			name:          "no overflow - positive numbers",
			a:             100,
			b:             200,
			expectedResult: 20000,
			expectedOverflow: false,
		},
		{
			name:          "no overflow - zero and positive",
			a:             0,
			b:             100,
			expectedResult: 0,
			expectedOverflow: false,
		},
		{
			name:          "no overflow - negative numbers",
			a:             -100,
			b:             200,
			expectedResult: -20000,
			expectedOverflow: false,
		},
		{
			name:          "no overflow - both negative",
			a:             -100,
			b:             -200,
			expectedResult: 20000,
			expectedOverflow: false,
		},
		{
			name:          "overflow - large positive numbers",
			a:             9223372036854775807, // max int64
			b:             2,
			expectedResult: 0,
			expectedOverflow: true,
		},
		{
			name:          "no overflow - one is 1",
			a:             9223372036854775807,
			b:             1,
			expectedResult: 9223372036854775807,
			expectedOverflow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, overflow := safeMul(tc.a, tc.b)
			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedOverflow, overflow)
		})
	}
}

// TestVerifyLightCommit tests that the VerifyLightCommit function correctly
// verifies signatures from validators
func TestVerifyLightCommit(t *testing.T) {
	chainID := testChainID
	height := int64(10)
	blockTime := time.Now().UTC()

	// Create a validator set with 3 validators
	valSet, privKeys := createTestValidatorSet(3, 100)
	signedHeader := createTestSignedHeader(chainID, height, blockTime, valSet, privKeys)

	// Convert to bft types for verification
	bftValSet := createBftValidatorSet(valSet.Validators, privKeys)
	bftCommit := toBftCommit(signedHeader.Commit)

	testCases := []struct {
		name       string
		trustLevel cmtmath.Fraction
		expectErr  bool
	}{
		{
			name:       "valid - 1/3 trust level",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 3},
			expectErr:  false,
		},
		{
			name:       "valid - 2/3 trust level",
			trustLevel: cmtmath.Fraction{Numerator: 2, Denominator: 3},
			expectErr:  false,
		},
		{
			name:       "valid - 1/2 trust level",
			trustLevel: cmtmath.Fraction{Numerator: 1, Denominator: 2},
			expectErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyLightCommit(
				bftValSet,
				chainID,
				bftCommit.BlockID,
				height,
				bftCommit,
				tc.trustLevel,
			)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestVerifyLightCommit_InvalidSignature tests that VerifyLightCommit fails
// when signatures are invalid
func TestVerifyLightCommit_InvalidSignature(t *testing.T) {
	chainID := testChainID
	height := int64(10)
	blockTime := time.Now().UTC()

	// Create a validator set
	valSet, privKeys := createTestValidatorSet(1, 100)
	signedHeader := createTestSignedHeader(chainID, height, blockTime, valSet, privKeys)

	// Corrupt the signature
	signedHeader.Commit.Precommits[0].Signature[0] ^= 0xFF

	// Convert to bft types for verification
	bftValSet := createBftValidatorSet(valSet.Validators, privKeys)
	bftCommit := toBftCommit(signedHeader.Commit)

	err := VerifyLightCommit(
		bftValSet,
		chainID,
		bftCommit.BlockID,
		height,
		bftCommit,
		cmtmath.Fraction{Numerator: 1, Denominator: 3},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid signature")
}

// TestVerifyAdjacent tests adjacent header verification
func TestVerifyAdjacent(t *testing.T) {
	chainID := testChainID
	trustedHeight := int64(10)
	untrustedHeight := int64(11) // adjacent
	trustedTime := time.Now().UTC()
	untrustedTime := trustedTime.Add(time.Second * 5)

	trustedHeader, untrustedHeader, _, privKeys := createChainedTestHeaders(
		t, chainID, trustedHeight, untrustedHeight, trustedTime, untrustedTime, 3, 100,
	)

	// Convert to bft types
	bftTrustedHeader := toBftSignedHeader(trustedHeader.SignedHeader)
	bftUntrustedHeader := toBftSignedHeader(untrustedHeader.SignedHeader)
	bftUntrustedVals := createBftValidatorSet(untrustedHeader.ValidatorSet.Validators, privKeys)

	// For adjacent verification, the untrusted header's ValidatorsHash must match
	// the trusted header's NextValidatorsHash. Since we use the same val set,
	// we need to ensure this consistency
	bftTrustedHeader.NextValidatorsHash = bftUntrustedHeader.ValidatorsHash

	err := VerifyAdjacent(
		bftTrustedHeader,
		bftUntrustedHeader,
		bftUntrustedVals,
		testTrustingPeriod,
		untrustedTime.Add(time.Second), // "now" is slightly after untrusted time
		testMaxClockDrift,
	)
	require.NoError(t, err)
}

// TestVerifyNonAdjacent tests non-adjacent header verification
func TestVerifyNonAdjacent(t *testing.T) {
	chainID := testChainID
	trustedHeight := int64(10)
	untrustedHeight := int64(20) // non-adjacent (gap of 10)
	trustedTime := time.Now().UTC()
	untrustedTime := trustedTime.Add(time.Hour)

	trustedHeader, untrustedHeader, _, privKeys := createChainedTestHeaders(
		t, chainID, trustedHeight, untrustedHeight, trustedTime, untrustedTime, 3, 100,
	)

	// Convert to bft types
	bftTrustedHeader := toBftSignedHeader(trustedHeader.SignedHeader)
	bftTrustedVals := createBftValidatorSet(trustedHeader.ValidatorSet.Validators, privKeys)
	bftUntrustedHeader := toBftSignedHeader(untrustedHeader.SignedHeader)
	bftUntrustedVals := createBftValidatorSet(untrustedHeader.ValidatorSet.Validators, privKeys)

	err := VerifyNonAdjacent(
		bftTrustedHeader,
		bftTrustedVals,
		bftUntrustedHeader,
		bftUntrustedVals,
		testTrustingPeriod,
		untrustedTime.Add(time.Second), // "now" is slightly after untrusted time
		testMaxClockDrift,
		cmtmath.Fraction{Numerator: 1, Denominator: 3},
	)
	require.NoError(t, err)
}

// TestVerify tests the combined verify function for both adjacent and non-adjacent
func TestVerify(t *testing.T) {
	chainID := testChainID
	trustedTime := time.Now().UTC()

	testCases := []struct {
		name            string
		trustedHeight   int64
		untrustedHeight int64
		timeDelta       time.Duration
		expectErr       bool
	}{
		{
			name:            "adjacent headers",
			trustedHeight:   10,
			untrustedHeight: 11,
			timeDelta:       time.Second * 5,
			expectErr:       false,
		},
		{
			name:            "non-adjacent headers",
			trustedHeight:   10,
			untrustedHeight: 20,
			timeDelta:       time.Hour,
			expectErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			untrustedTime := trustedTime.Add(tc.timeDelta)

			trustedHeader, untrustedHeader, _, privKeys := createChainedTestHeaders(
				t, chainID, tc.trustedHeight, tc.untrustedHeight, trustedTime, untrustedTime, 3, 100,
			)

			// Convert to bft types
			bftTrustedHeader := toBftSignedHeader(trustedHeader.SignedHeader)
			bftTrustedVals := createBftValidatorSet(trustedHeader.ValidatorSet.Validators, privKeys)
			bftUntrustedHeader := toBftSignedHeader(untrustedHeader.SignedHeader)
			bftUntrustedVals := createBftValidatorSet(untrustedHeader.ValidatorSet.Validators, privKeys)

			// For adjacent verification, ensure hash consistency
			if tc.untrustedHeight == tc.trustedHeight+1 {
				bftTrustedHeader.NextValidatorsHash = bftUntrustedHeader.ValidatorsHash
			}

			err := Verify(
				bftTrustedHeader,
				bftTrustedVals,
				bftUntrustedHeader,
				bftUntrustedVals,
				testTrustingPeriod,
				untrustedTime.Add(time.Second),
				testMaxClockDrift,
				cmtmath.Fraction{Numerator: 1, Denominator: 3},
			)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestHeaderExpired tests the HeaderExpired function
func TestHeaderExpired(t *testing.T) {
	chainID := testChainID
	height := int64(10)
	headerTime := time.Now().UTC()

	valSet, privKeys := createTestValidatorSet(1, 100)
	signedHeader := createTestSignedHeader(chainID, height, headerTime, valSet, privKeys)
	bftHeader := toBftSignedHeader(signedHeader)

	testCases := []struct {
		name           string
		trustingPeriod time.Duration
		now            time.Time
		expectExpired  bool
	}{
		{
			name:           "not expired - within trusting period",
			trustingPeriod: time.Hour * 24,
			now:            headerTime.Add(time.Hour),
			expectExpired:  false,
		},
		{
			name:           "expired - past trusting period",
			trustingPeriod: time.Hour,
			now:            headerTime.Add(time.Hour * 2),
			expectExpired:  true,
		},
		{
			name:           "not expired - exactly at expiry boundary",
			trustingPeriod: time.Hour,
			now:            headerTime.Add(time.Hour),
			expectExpired:  true, // at boundary is considered expired
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expired := HeaderExpired(bftHeader, tc.trustingPeriod, tc.now)
			require.Equal(t, tc.expectExpired, expired)
		})
	}
}
