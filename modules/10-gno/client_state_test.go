package gno_test

import (
	ics23 "github.com/cosmos/ics23/go"

	ibcgno "github.com/atomone-hub/atomone/modules/10-gno"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
)

const (
	// Do not change the length of these variables
	fiftyCharChainID    = "12345678901234567890123456789012345678901234567890"
	fiftyOneCharChainID = "123456789012345678901234567890123456789012345678901"
)

var invalidProof = []byte("invalid proof")

func (suite *GnoTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *ibcgno.ClientState
		expErr      error
	}{
		{
			name:        "valid client",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      nil,
		},
		{
			name:        "valid client with nil upgrade path",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), nil),
			expErr:      nil,
		},
		{
			name:        "invalid chainID",
			clientState: ibcgno.NewClientState("  ", ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidChainID,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "valid chainID - chainID validation did not fail for chainID of length 50! ",
			clientState: ibcgno.NewClientState(fiftyCharChainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      nil,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "invalid chainID - chainID validation failed for chainID of length 51! ",
			clientState: ibcgno.NewClientState(fiftyOneCharChainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidChainID,
		},
		{
			name:        "invalid trust level",
			clientState: ibcgno.NewClientState(chainID, ibcgno.Fraction{Numerator: 0, Denominator: 1}, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidTrustLevel,
		},
		{
			name:        "invalid zero trusting period",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, 0, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidTrustingPeriod,
		},
		{
			name:        "invalid negative trusting period",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, -1, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidTrustingPeriod,
		},
		{
			name:        "invalid zero unbonding period",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, 0, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidUnbondingPeriod,
		},
		{
			name:        "invalid negative unbonding period",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, -1, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidUnbondingPeriod,
		},
		{
			name:        "invalid zero max clock drift",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, 0, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidMaxClockDrift,
		},
		{
			name:        "invalid negative max clock drift",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, -1, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidMaxClockDrift,
		},
		{
			name:        "invalid revision number",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidHeaderHeight,
		},
		{
			name:        "invalid revision height",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, clienttypes.ZeroHeight(), commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidHeaderHeight,
		},
		{
			name:        "trusting period not less than unbonding period",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, ubdPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expErr:      ibcgno.ErrInvalidTrustingPeriod,
		},
		{
			name:        "proof specs is nil",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, nil, upgradePath),
			expErr:      ibcgno.ErrInvalidProofSpecs,
		},
		{
			name:        "proof specs contains nil",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, []*ics23.ProofSpec{ics23.TendermintSpec, nil}, upgradePath),
			expErr:      ibcgno.ErrInvalidProofSpecs,
		},
		{
			name:        "invalid upgrade path",
			clientState: ibcgno.NewClientState(chainID, ibcgno.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), invalidUpgradePath),
			expErr:      clienttypes.ErrInvalidClient,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.clientState.Validate()

			if tc.expErr == nil {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
