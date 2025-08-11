package e2e

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cosmossdk.io/math"
	evidencetypes "cosmossdk.io/x/evidence/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	disttypes "github.com/atomone-hub/atomone/x/distribution/types"
	dynamicfeetypes "github.com/atomone-hub/atomone/x/dynamicfee/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// queryAtomOneTx returns an error if the tx is not found or is failed.
func (s *IntegrationTestSuite) queryAtomOneTx(endpoint, txHash string, msgResp codec.ProtoMarshaler) (height int, err error) {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", endpoint, txHash))
	if err != nil {
		return 0, err
	}

	var resp tx.GetTxResponse
	if err := s.cdc.UnmarshalJSON(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.TxResponse.Code != 0 {
		return 0, fmt.Errorf("tx %s is failed with code=%d log='%s'", txHash, resp.TxResponse.Code, resp.TxResponse.RawLog)
	}
	if msgResp != nil {
		// msgResp is provided, try to decode the tx response
		data, err := hex.DecodeString(resp.TxResponse.Data)
		if err != nil {
			return 0, err
		}
		var txMsgData sdk.TxMsgData
		if err := s.cdc.Unmarshal(data, &txMsgData); err != nil {
			return 0, err
		}
		if err := s.cdc.Unmarshal(txMsgData.MsgResponses[0].Value, msgResp); err != nil {
			return 0, err
		}
	}
	return int(resp.TxResponse.Height), nil
}

// if coin is zero, return empty coin.
func (s *IntegrationTestSuite) getSpecificBalance(endpoint, addr, denom string) (amt sdk.Coin, err error) {
	balances, err := s.queryAtomOneAllBalances(endpoint, addr)
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

func (s *IntegrationTestSuite) queryAtomOneAllBalances(endpoint, addr string) (sdk.Coins, error) {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", endpoint, addr))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	var balancesResp banktypes.QueryAllBalancesResponse
	if err := s.cdc.UnmarshalJSON(body, &balancesResp); err != nil {
		return nil, err
	}

	return balancesResp.Balances, nil
}

func (s *IntegrationTestSuite) queryBankSupply(endpoint string) sdk.Coins {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/bank/v1beta1/supply", endpoint))
	s.Require().NoError(err)
	var resp banktypes.QueryTotalSupplyResponse
	err = s.cdc.UnmarshalJSON(body, &resp)
	s.Require().NoError(err)
	return resp.Supply
}

func (s *IntegrationTestSuite) queryDelegation(endpoint string, validatorAddr string, delegatorAddr string) (stakingtypes.QueryDelegationResponse, error) {
	var res stakingtypes.QueryDelegationResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s/delegations/%s", endpoint, validatorAddr, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = s.cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *IntegrationTestSuite) queryUnbondingDelegation(endpoint string, validatorAddr string, delegatorAddr string) (stakingtypes.QueryUnbondingDelegationResponse, error) {
	var res stakingtypes.QueryUnbondingDelegationResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s/delegations/%s/unbonding_delegation", endpoint, validatorAddr, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = s.cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *IntegrationTestSuite) queryDelegatorWithdrawalAddress(endpoint string, delegatorAddr string) (disttypes.QueryDelegatorWithdrawAddressResponse, error) {
	var res disttypes.QueryDelegatorWithdrawAddressResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/distribution/v1beta1/delegators/%s/withdraw_address", endpoint, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = s.cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *IntegrationTestSuite) queryDelegatorTotalRewards(endpoint, delegatorAddr string) (disttypes.QueryDelegationTotalRewardsResponse, error) { //nolint:unused
	var res disttypes.QueryDelegationTotalRewardsResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/distribution/v1beta1/delegators/%s/rewards", endpoint, delegatorAddr))
	if err != nil {
		return res, err
	}

	if err = s.cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}

	return res, nil
}

func (s *IntegrationTestSuite) queryGovProposal(endpoint string, proposalID int) (govtypesv1beta1.QueryProposalResponse, error) {
	var govProposalResp govtypesv1beta1.QueryProposalResponse

	path := fmt.Sprintf("%s/atomone/gov/v1beta1/proposals/%d", endpoint, proposalID)

	body, err := httpGet(path)
	if err != nil {
		return govProposalResp, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	if err := s.cdc.UnmarshalJSON(body, &govProposalResp); err != nil {
		return govProposalResp, err
	}

	return govProposalResp, nil
}

func (s *IntegrationTestSuite) queryGovMinInitialDeposit(endpoint string) (govtypesv1.QueryMinInitialDepositResponse, error) {
	var govMinInitialDepositResp govtypesv1.QueryMinInitialDepositResponse
	path := fmt.Sprintf("%s/atomone/gov/v1/mininitialdeposit", endpoint)
	body, err := httpGet(path)
	if err != nil {
		return govMinInitialDepositResp, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	if err := s.cdc.UnmarshalJSON(body, &govMinInitialDepositResp); err != nil {
		return govMinInitialDepositResp, err
	}
	return govMinInitialDepositResp, nil
}

func (s *IntegrationTestSuite) queryGovMinDeposit(endpoint string) (govtypesv1.QueryMinDepositResponse, error) {
	var govMinDepositResp govtypesv1.QueryMinDepositResponse
	path := fmt.Sprintf("%s/atomone/gov/v1/mindeposit", endpoint)
	body, err := httpGet(path)
	if err != nil {
		return govMinDepositResp, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	if err := s.cdc.UnmarshalJSON(body, &govMinDepositResp); err != nil {
		return govMinDepositResp, err
	}
	return govMinDepositResp, nil
}

func (s *IntegrationTestSuite) queryGovQuorums(endpoint string) govtypesv1.QueryQuorumsResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/gov/v1/quorums", endpoint))
	s.Require().NoError(err)
	var res govtypesv1.QueryQuorumsResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryGovParams(endpoint string, param string) govtypesv1.QueryParamsResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/gov/v1/params/%s", endpoint, param))
	s.Require().NoError(err)
	var res govtypesv1.QueryParamsResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryAccount(endpoint, address string) (acc sdk.AccountI, err error) {
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
	if err := s.cdc.UnmarshalJSON(bz, &res); err != nil {
		return nil, err
	}
	return acc, s.cdc.UnpackAny(res.Account, &acc)
}

func (s *IntegrationTestSuite) queryDelayedVestingAccount(endpoint, address string) (authvesting.DelayedVestingAccount, error) {
	baseAcc, err := s.queryAccount(endpoint, address)
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

func (s *IntegrationTestSuite) queryContinuousVestingAccount(endpoint, address string) (authvesting.ContinuousVestingAccount, error) {
	baseAcc, err := s.queryAccount(endpoint, address)
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

func (s *IntegrationTestSuite) queryPeriodicVestingAccount(endpoint, address string) (authvesting.PeriodicVestingAccount, error) { //nolint:unused // this is called during e2e tests
	baseAcc, err := s.queryAccount(endpoint, address)
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

func (s *IntegrationTestSuite) queryValidator(endpoint, address string) (stakingtypes.Validator, error) {
	var res stakingtypes.QueryValidatorResponse

	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s", endpoint, address))
	if err != nil {
		return stakingtypes.Validator{}, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if err := s.cdc.UnmarshalJSON(body, &res); err != nil {
		return stakingtypes.Validator{}, err
	}
	return res.Validator, nil
}

func (s *IntegrationTestSuite) queryValidators(endpoint string) ([]stakingtypes.Validator, error) {
	var res stakingtypes.QueryValidatorsResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/validators", endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if err := s.cdc.UnmarshalJSON(body, &res); err != nil {
		return nil, err
	}
	return res.Validators, nil
}

func (s *IntegrationTestSuite) queryStakingPool(endpoint string) stakingtypes.QueryPoolResponse {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/pool", endpoint))
	s.Require().NoError(err)
	var res stakingtypes.QueryPoolResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryAllEvidence(endpoint string) (evidencetypes.QueryAllEvidenceResponse, error) {
	var res evidencetypes.QueryAllEvidenceResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/evidence/v1beta1/evidence", endpoint))
	if err != nil {
		return res, err
	}

	if err = s.cdc.UnmarshalJSON(body, &res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *IntegrationTestSuite) queryStakingParams(endpoint string) stakingtypes.QueryParamsResponse {
	var res stakingtypes.QueryParamsResponse
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/params", endpoint))
	s.Require().NoError(err)
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryConstitution(endpoint string) govtypesv1.QueryConstitutionResponse {
	var res govtypesv1.QueryConstitutionResponse
	body, err := httpGet(fmt.Sprintf("%s/atomone/gov/v1/constitution", endpoint))
	s.Require().NoError(err)
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryPhotonConversionRate(endpoint string) math.LegacyDec {
	body, err := httpGet(fmt.Sprintf("%s/atomone/photon/v1/conversion_rate", endpoint))
	s.Require().NoError(err)
	var resp photontypes.QueryConversionRateResponse
	err = s.cdc.UnmarshalJSON(body, &resp)
	s.Require().NoError(err)
	return math.LegacyMustNewDecFromStr(resp.ConversionRate)
}

func (s *IntegrationTestSuite) queryPhotonParams(endpoint string) photontypes.QueryParamsResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/photon/v1/params", endpoint))
	s.Require().NoError(err)
	var res photontypes.QueryParamsResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryDynamicfeeParams(endpoint string) dynamicfeetypes.ParamsResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/dynamicfee/v1/params", endpoint))
	s.Require().NoError(err)
	var res dynamicfeetypes.ParamsResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryDynamicfeeState(endpoint string) dynamicfeetypes.StateResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/dynamicfee/v1/state", endpoint))
	s.Require().NoError(err)
	var res dynamicfeetypes.StateResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryDynamicfeeStateAtHeight(endpoint string, height string) dynamicfeetypes.StateResponse {
	headers := addHeader(nil, "x-cosmos-block-height", height)
	body, err := httpGetWithHeader(fmt.Sprintf("%s/atomone/dynamicfee/v1/state", endpoint), headers)
	s.Require().NoError(err)
	var res dynamicfeetypes.StateResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryDynamicfeeGasPrice(endpoint string, denom string) dynamicfeetypes.GasPriceResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/dynamicfee/v1/gas_price/%s", endpoint, denom))
	s.Require().NoError(err)
	var res dynamicfeetypes.GasPriceResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryDynamicfeeGasPrices(endpoint string) dynamicfeetypes.GasPricesResponse {
	body, err := httpGet(fmt.Sprintf("%s/atomone/dynamicfee/v1/gas_prices", endpoint))
	s.Require().NoError(err)
	var res dynamicfeetypes.GasPricesResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}

func (s *IntegrationTestSuite) queryUpgradePlan(endpoint string) upgradetypes.QueryCurrentPlanResponse {
	body, err := httpGet(fmt.Sprintf("%s/cosmos/upgrade/v1beta1/current_plan", endpoint))
	s.Require().NoError(err)
	var res upgradetypes.QueryCurrentPlanResponse
	err = s.cdc.UnmarshalJSON(body, &res)
	s.Require().NoError(err)
	return res
}
