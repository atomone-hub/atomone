package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cosmossdk.io/math"

	icagen "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func getGenDoc(path string) (*genutiltypes.AppGenesis, error) {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config
	config.SetRoot(path)

	genFile := config.GenesisFile()

	if _, err := os.Stat(genFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		return &genutiltypes.AppGenesis{}, nil
	}

	var err error
	doc, err := genutiltypes.AppGenesisFromFile(genFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis doc from file: %w", err)
	}

	return doc, nil
}

func modifyGenesis(cdc codec.Codec, path, moniker, amountStr string, addrAll []sdk.AccAddress, denom string) error {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config
	config.SetRoot(path)
	config.Moniker = moniker

	//-----------------------------------------
	// Modifying auth genesis

	coins, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return fmt.Errorf("failed to parse coins: %w", err)
	}

	var balances []banktypes.Balance
	var genAccounts []*authtypes.BaseAccount
	for _, addr := range addrAll {
		balance := banktypes.Balance{Address: addr.String(), Coins: coins.Sort()}
		balances = append(balances, balance)
		genAccount := authtypes.NewBaseAccount(addr, nil, 0, 0)
		genAccounts = append(genAccounts, genAccount)
	}

	genFile := config.GenesisFile()
	appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}

	authGenState := authtypes.GetGenesisStateFromAppState(cdc, appState)
	accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
	if err != nil {
		return fmt.Errorf("failed to get accounts from any: %w", err)
	}

	for _, addr := range addrAll {
		if accs.Contains(addr) {
			return fmt.Errorf("failed to add account to genesis state; account already exists: %s", addr)
		}
	}

	// Add the new account to the set of genesis accounts and sanitize the
	// accounts afterwards.
	for _, genAcct := range genAccounts {
		accs = append(accs, genAcct)
		accs = authtypes.SanitizeGenesisAccounts(accs)
	}

	genAccs, err := authtypes.PackAccounts(accs)
	if err != nil {
		return fmt.Errorf("failed to convert accounts into any's: %w", err)
	}

	authGenState.Accounts = genAccs

	authGenStateBz, err := cdc.MarshalJSON(&authGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal auth genesis state: %w", err)
	}
	appState[authtypes.ModuleName] = authGenStateBz

	//-----------------------------------------
	// Modifying bank genesis

	bankGenState := banktypes.GetGenesisStateFromAppState(cdc, appState)
	bankGenState.Balances = append(bankGenState.Balances, balances...)
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)

	bankGenStateBz, err := cdc.MarshalJSON(bankGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal bank genesis state: %w", err)
	}
	appState[banktypes.ModuleName] = bankGenStateBz

	//-----------------------------------------
	// Modifying interchain accounts genesis

	// add ica host allowed msg types
	var icaGenesisState icagen.GenesisState

	if appState[icatypes.ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[icatypes.ModuleName], &icaGenesisState)
	}

	icaGenesisState.HostGenesisState.Params.AllowMessages = []string{
		"/cosmos.authz.v1beta1.MsgExec",
		"/cosmos.authz.v1beta1.MsgGrant",
		"/cosmos.authz.v1beta1.MsgRevoke",
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/cosmos.distribution.v1beta1.MsgSetWithdrawAddress",
		"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission",
		"/cosmos.distribution.v1beta1.MsgFundCommunityPool",
		"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
		"/cosmos.feegrant.v1beta1.MsgGrantAllowance",
		"/cosmos.feegrant.v1beta1.MsgRevokeAllowance",
		"/cosmos.gov.v1beta1.MsgVoteWeighted",
		"/cosmos.gov.v1beta1.MsgSubmitProposal",
		"/cosmos.gov.v1beta1.MsgDeposit",
		"/cosmos.gov.v1beta1.MsgVote",
		"/cosmos.staking.v1beta1.MsgEditValidator",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/cosmos.staking.v1beta1.MsgCreateValidator",
		"/cosmos.vesting.v1beta1.MsgCreateVestingAccount",
		"/ibc.applications.transfer.v1.MsgTransfer",
		"/tendermint.liquidity.v1beta1.MsgCreatePool",
		"/tendermint.liquidity.v1beta1.MsgSwapWithinBatch",
		"/tendermint.liquidity.v1beta1.MsgDepositWithinBatch",
		"/tendermint.liquidity.v1beta1.MsgWithdrawWithinBatch",
	}

	icaGenesisStateBz, err := cdc.MarshalJSON(&icaGenesisState)
	if err != nil {
		return fmt.Errorf("failed to marshal interchain accounts genesis state: %w", err)
	}
	appState[icatypes.ModuleName] = icaGenesisStateBz

	//-----------------------------------------
	// Modifying staking genesis

	stakingGenState := stakingtypes.GetGenesisStateFromAppState(cdc, appState)
	stakingGenState.Params.BondDenom = denom
	stakingGenStateBz, err := cdc.MarshalJSON(stakingGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal staking genesis state: %s", err)
	}
	appState[stakingtypes.ModuleName] = stakingGenStateBz

	var mintGenState minttypes.GenesisState
	cdc.MustUnmarshalJSON(appState[minttypes.ModuleName], &mintGenState)
	mintGenState.Params.MintDenom = denom
	mintGenStateBz, err := cdc.MarshalJSON(&mintGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal mint genesis state: %s", err)
	}
	appState[minttypes.ModuleName] = mintGenStateBz

	//-----------------------------------------
	// Modifying gov genesis

	// Refactor to separate method
	threshold := "0.000000000000000001"
	lawThreshold := "0.000000000000000001"
	amendmentsThreshold := "0.000000000000000001"
	minQuorum := "0.2"
	maxQuorum := "0.8"
	participationEma := "0.25"
	minConstitutionAmendmentQuorum := "0.2"
	maxConstitutionAmendmentQuorum := "0.8"
	minLawQuorum := "0.2"
	maxLawQuorum := "0.8"

	maxDepositPeriod := 10 * time.Minute
	votingPeriod := 15 * time.Second

	govGenState := govv1.NewGenesisState(1,
		participationEma, participationEma, participationEma,
		govv1.NewParams(
			// sdk.NewCoins(sdk.NewCoin(denom, depositAmount.Amount)),
			maxDepositPeriod,
			votingPeriod,
			threshold, amendmentsThreshold, lawThreshold,
			// sdk.ZeroDec().String(),
			false, false, govv1.DefaultMinDepositRatio.String(),
			govv1.DefaultQuorumTimeout, govv1.DefaultMaxVotingPeriodExtension, govv1.DefaultQuorumCheckCount,
			sdk.NewCoins(sdk.NewCoin(denom, depositAmount.Amount)), govv1.DefaultMinDepositUpdatePeriod,
			govv1.DefaultMinDepositDecreaseSensitivityTargetDistance,
			govv1.DefaultMinDepositIncreaseRatio.String(), govv1.DefaultMinDepositDecreaseRatio.String(),
			govv1.DefaultTargetActiveProposals, sdk.NewCoins(sdk.NewCoin(denom, initialDepositAmount.Amount)), govv1.DefaultMinInitialDepositUpdatePeriod,
			govv1.DefaultMinInitialDepositDecreaseSensitivityTargetDistance, govv1.DefaultMinInitialDepositIncreaseRatio.String(),
			govv1.DefaultMinInitialDepositDecreaseRatio.String(), govv1.DefaultTargetProposalsInDepositPeriod,
			govv1.DefaultBurnDepositNoThreshold.String(),
			maxQuorum, minQuorum,
			maxConstitutionAmendmentQuorum, minConstitutionAmendmentQuorum,
			maxLawQuorum, minLawQuorum,
		),
	)
	govGenState.Constitution = "This is a test constitution"
	govGenStateBz, err := cdc.MarshalJSON(govGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal gov genesis state: %w", err)
	}
	appState[govtypes.ModuleName] = govGenStateBz

	//-----------------------------------------
	// Modifying feemarket genesis

	feemarketGenState := feemarkettypes.GetGenesisStateFromAppState(cdc, appState)
	baseGasPrice := math.LegacyMustNewDecFromStr("0.00001")
	feemarketGenState.Params.MinBaseGasPrice = baseGasPrice
	feemarketGenState.State.BaseGasPrice = baseGasPrice
	feemarketGenStateBz, err := cdc.MarshalJSON(&feemarketGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal feemarket genesis state: %w", err)
	}
	appState[feemarkettypes.ModuleName] = feemarketGenStateBz

	//-----------------------------------------
	// Record final genesis

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("failed to marshal application genesis state: %w", err)
	}
	genDoc.AppState = appStateJSON

	return genutil.ExportGenesisFile(genDoc, genFile)
}
