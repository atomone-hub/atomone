package cmd

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/atomone-hub/atomone/app/params"
)

func InitSDKConfig() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(appparams.Bech32PrefixAccAddr, appparams.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(appparams.Bech32PrefixValAddr, appparams.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(appparams.Bech32PrefixConsAddr, appparams.Bech32PrefixConsPub)
	cfg.Seal()
}
