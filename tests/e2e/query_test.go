package e2e

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

// queryAtomOneTx returns an error if the tx is not found or is failed.
func queryAtomOneTx(endpoint, txHash string) error {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", endpoint, txHash))
	if err != nil {
		return err
	}

	var resp tx.GetTxResponse
	if err := cdc.UnmarshalJSON(body, &resp); err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.TxResponse.Code != 0 {
		return fmt.Errorf("tx %s is failed with code=%d log='%s'", txHash, resp.TxResponse.Code, resp.TxResponse.RawLog)
	}
	return nil
}

// if coin is zero, return empty coin.
func getSpecificBalance(endpoint, addr, denom string) (amt sdk.Coin, err error) {
	balances, err := queryAtomOneAllBalances(endpoint, addr)
	if err != nil {
		return amt, err
	}
	for _, c := range balances {
		if strings.Contains(c.Denom, denom) {
			amt = c
			break
		}
	}
	return amt, nil
}

func queryAtomOneAllBalances(endpoint, addr string) (sdk.Coins, error) {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", endpoint, addr))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	var balancesResp banktypes.QueryAllBalancesResponse
	if err := cdc.UnmarshalJSON(body, &balancesResp); err != nil {
		return nil, err
	}

	return balancesResp.Balances, nil
}

func queryStakingParams(endpoint string) (stakingtypes.QueryParamsResponse, error) { //nolint:unused
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/params", endpoint))
	if err != nil {
		return stakingtypes.QueryParamsResponse{}, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	var params stakingtypes.QueryParamsResponse
	if err := cdc.UnmarshalJSON(body, &params); err != nil {
		return stakingtypes.QueryParamsResponse{}, err
	}

	return params, nil
}

func queryDelegation(endpoint string, validatorAddr string, delegatorAddr string) (stakingtypes.QueryDelegationResponse, error) {
	var res stakingtypes.QueryDelegationResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s/delegations/%s", endpoint, validatorAddr, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func queryUnbondingDelegation(endpoint string, validatorAddr string, delegatorAddr string) (stakingtypes.QueryUnbondingDelegationResponse, error) {
	var res stakingtypes.QueryUnbondingDelegationResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s/delegations/%s/unbonding_delegation", endpoint, validatorAddr, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func queryDelegatorWithdrawalAddress(endpoint string, delegatorAddr string) (disttypes.QueryDelegatorWithdrawAddressResponse, error) {
	var res disttypes.QueryDelegatorWithdrawAddressResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/distribution/v1beta1/delegators/%s/withdraw_address", endpoint, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func queryDelegatorTotalRewards(endpoint, delegatorAddr string) (disttypes.QueryDelegationTotalRewardsResponse, error) { //nolint:unused
	var res disttypes.QueryDelegationTotalRewardsResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/distribution/v1beta1/delegators/%s/rewards", endpoint, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}

	return res, nil
}

func queryGovProposal(endpoint string, proposalID int) (govtypesv1beta1.QueryProposalResponse, error) {
	var govProposalResp govtypesv1beta1.QueryProposalResponse

	path := fmt.Sprintf("%s/atomone/gov/v1beta1/proposals/%d", endpoint, proposalID)

	body, err := httpGet(path)
	if err != nil {
		return govProposalResp, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	if err := cdc.UnmarshalJSON(body, &govProposalResp); err != nil {
		return govProposalResp, err
	}

	return govProposalResp, nil
}

func queryAccount(endpoint, address string) (acc authtypes.AccountI, err error) {
	var res authtypes.QueryAccountResponse
	resp, err := http.Get(fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", endpoint, address))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := cdc.UnmarshalJSON(bz, &res); err != nil {
		return nil, err
	}
	return acc, cdc.UnpackAny(res.Account, &acc)
}

func queryDelayedVestingAccount(endpoint, address string) (authvesting.DelayedVestingAccount, error) {
	baseAcc, err := queryAccount(endpoint, address)
	if err != nil {
		return authvesting.DelayedVestingAccount{}, err
	}
	acc, ok := baseAcc.(*authvesting.DelayedVestingAccount)
	if !ok {
		return authvesting.DelayedVestingAccount{},
			fmt.Errorf("cannot cast %v to DelayedVestingAccount", baseAcc)
	}
	return *acc, nil
}

func queryContinuousVestingAccount(endpoint, address string) (authvesting.ContinuousVestingAccount, error) {
	baseAcc, err := queryAccount(endpoint, address)
	if err != nil {
		return authvesting.ContinuousVestingAccount{}, err
	}
	acc, ok := baseAcc.(*authvesting.ContinuousVestingAccount)
	if !ok {
		return authvesting.ContinuousVestingAccount{},
			fmt.Errorf("cannot cast %v to ContinuousVestingAccount", baseAcc)
	}
	return *acc, nil
}

func queryPeriodicVestingAccount(endpoint, address string) (authvesting.PeriodicVestingAccount, error) { //nolint:unused // this is called during e2e tests
	baseAcc, err := queryAccount(endpoint, address)
	if err != nil {
		return authvesting.PeriodicVestingAccount{}, err
	}
	acc, ok := baseAcc.(*authvesting.PeriodicVestingAccount)
	if !ok {
		return authvesting.PeriodicVestingAccount{},
			fmt.Errorf("cannot cast %v to PeriodicVestingAccount", baseAcc)
	}
	return *acc, nil
}

func queryValidator(endpoint, address string) (stakingtypes.Validator, error) {
	var res stakingtypes.QueryValidatorResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s", endpoint, address))
	if err != nil {
		return stakingtypes.Validator{}, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if err := cdc.UnmarshalJSON(body, &res); err != nil {
		return stakingtypes.Validator{}, err
	}
	return res.Validator, nil
}

func queryValidators(endpoint string) (stakingtypes.Validators, error) {
	var res stakingtypes.QueryValidatorsResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators", endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if err := cdc.UnmarshalJSON(body, &res); err != nil {
		return nil, err
	}
	return res.Validators, nil
}

func queryEvidence(endpoint, hash string) (evidencetypes.QueryEvidenceResponse, error) { //nolint:unused // this is called during e2e tests
	var res evidencetypes.QueryEvidenceResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/evidence/v1beta1/evidence/%s", endpoint, hash))
	if err != nil {
		return res, err
	}

	if err = cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func queryAllEvidence(endpoint string) (evidencetypes.QueryAllEvidenceResponse, error) {
	var res evidencetypes.QueryAllEvidenceResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/evidence/v1beta1/evidence", endpoint))
	if err != nil {
		return res, err
	}

	if err = cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *IntegrationTestSuite) queryStakingParams(endpoint string) stakingtypes.QueryParamsResponse {
	var res stakingtypes.QueryParamsResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/params", endpoint))
	s.Require().NoError(err)
	err = cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}