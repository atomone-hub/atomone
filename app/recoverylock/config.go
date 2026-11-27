// Package recoverylock enforces the release-pinned AtomOne recovery policy.
package recoverylock

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	appparams "github.com/atomone-hub/atomone/app/params"
	"github.com/atomone-hub/atomone/app/recoverylock/core"
)

const (
	expiryMarginSeconds        = 7 * 24 * 3600
	minReleaseUnbondingSeconds = 21 * 24 * 3600
	minMillisPerBlock          = 500
	maxMillisPerBlock          = 30000
)

// Fixture identities are address bytes derived from hashes, without signing keys.
var fixtureAddresses = func() map[string]bool {
	labels := []string{"listed-0", "listed-1", "recovery", "payer"}
	addresses := make(map[string]bool, len(labels))
	for _, label := range labels {
		sum := sha256.Sum256([]byte("atomone-recovery-fixture-" + label))
		addresses[string(sum[:core.AddrLen])] = true
	}
	return addresses
}()

type params struct {
	ChainID                string   `json:"chain_id"`
	RecoveryAddress        string   `json:"recovery_address"`
	FeePayers              []string `json:"fee_payers"`
	ActivationHeight       int64    `json:"activation_height"`
	ExpiryTime             int64    `json:"expiry_time"`
	ActivationTimeEstimate int64    `json:"activation_time_estimate"`
	UnbondingSeconds       int64    `json:"unbonding_seconds"`
	EstimateRefHeight      int64    `json:"estimate_ref_height"`
	EstimateRefTime        int64    `json:"estimate_ref_time"`
	ListSHA256             string   `json:"list_sha256"`
	ListCount              int      `json:"list_count"`
	TestValues             bool     `json:"test_values"`
}

// paramKeys are the exact JSON keys of params (case sensitive).
var paramKeys = func() map[string]bool {
	out := map[string]bool{}
	t := reflect.TypeOf(params{})
	for i := 0; i < t.NumField(); i++ {
		out[strings.Split(t.Field(i).Tag.Get("json"), ",")[0]] = true
	}
	return out
}()

// parseParams decodes params.json strictly: duplicate keys at any depth,
// unknown keys, keys in another case than the exact field names, values of
// the wrong type and trailing data are refused (encoding/json alone takes the
// last of duplicate keys, ignores unknown keys and matches keys case
// insensitively). A null value leaves the field at its zero value, which
// fails every later check that needs a value; for test_values it means false.
func parseParams(b []byte) (params, error) {
	var p params
	v, err := core.StrictJSON(b)
	if err != nil {
		return p, err
	}
	top, ok := v.(map[string]any)
	if !ok {
		return p, fmt.Errorf("params must be a JSON object")
	}
	for k := range top {
		if !paramKeys[k] {
			return p, fmt.Errorf("unknown key %q", k)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return p, err
	}
	return p, nil
}

// MustLoad validates the embedded manifest before the node starts.
func MustLoad(modulePermissions map[string][]string, logger log.Logger) (*core.Config, string) {
	ac := addresscodec.NewBech32Codec(appparams.Bech32PrefixAccAddr)
	blocked := make(map[string]bool, len(modulePermissions))
	for module := range modulePermissions {
		canonical, err := ac.BytesToString(authtypes.NewModuleAddress(module))
		if err != nil {
			panic(err)
		}
		blocked[canonical] = true
	}
	cfg, p, err := parseConfig(listedFile, paramsFile, ac, blocked, testBuild)
	if err != nil {
		panic("recovery lock config: " + err.Error())
	}
	mode := "release"
	if testBuild {
		mode = "test"
	}
	if logger != nil {
		start, expiry := cfg.Window()
		logger.Info("recovery lock config",
			"mode", mode,
			"chain_id", p.ChainID,
			"activation_height", start,
			"expiry_time", expiry,
			"expiry_utc", time.Unix(expiry, 0).UTC().Format(time.RFC3339),
			"recovery_address", p.RecoveryAddress,
			"fee_payers", len(cfg.FeePayers()),
			"list_count", p.ListCount,
			"list_sha256", strings.ToLower(p.ListSHA256),
		)
	}
	return cfg, p.ChainID
}

// checkEstimate cross-checks the activation height against the activation
// time estimate (release manifests).
func checkEstimate(p params) error {
	if p.EstimateRefHeight <= 0 || p.EstimateRefTime <= 0 {
		return fmt.Errorf("estimate_ref_height and estimate_ref_time are required")
	}
	if p.ActivationHeight <= p.EstimateRefHeight {
		return fmt.Errorf("activation_height %d must be above estimate_ref_height %d", p.ActivationHeight, p.EstimateRefHeight)
	}
	dh := p.ActivationHeight - p.EstimateRefHeight
	dt := p.ActivationTimeEstimate - p.EstimateRefTime
	// bounds keep the products below 2^63
	const limit = int64(1) << 40
	if dt <= 0 || dt >= limit || dh >= limit {
		return fmt.Errorf("activation_time_estimate %d is not plausible after estimate_ref_time %d", p.ActivationTimeEstimate, p.EstimateRefTime)
	}
	if dt*1000 < minMillisPerBlock*dh || dt*1000 > maxMillisPerBlock*dh {
		return fmt.Errorf("implied block time %d ms ((activation_time_estimate - estimate_ref_time) / (activation_height - estimate_ref_height)) is outside %d to %d ms",
			dt*1000/dh, minMillisPerBlock, maxMillisPerBlock)
	}
	return nil
}

func parseConfig(listed, paramsJSON []byte, ac address.Codec, blocked map[string]bool, allowFixtures bool) (*core.Config, params, error) {
	p, err := parseParams(paramsJSON)
	if err != nil {
		return nil, p, fmt.Errorf("params: %w", err)
	}
	cfg, err := build(listed, p, ac, blocked, allowFixtures)
	return cfg, p, err
}

func build(listed []byte, p params, ac address.Codec, blocked map[string]bool, allowFixtures bool) (*core.Config, error) {
	if len(p.FeePayers) != 1 {
		return nil, fmt.Errorf("exactly one fee payer is required")
	}
	if p.ChainID == "" || (p.TestValues && p.ChainID == "atomone-1") {
		return nil, fmt.Errorf("invalid recovery lock chain ID")
	}
	if p.TestValues && !allowFixtures {
		return nil, fmt.Errorf("test values outside a recoverylock_test build")
	}
	if !p.TestValues {
		if p.ChainID != "atomone-1" {
			return nil, fmt.Errorf("release recovery lock requires atomone-1")
		}
		if p.ActivationTimeEstimate <= 0 {
			return nil, fmt.Errorf("activation_time_estimate is required")
		}
		if p.UnbondingSeconds < minReleaseUnbondingSeconds {
			return nil, fmt.Errorf("unbonding_seconds %d is below the mainnet unbonding time %d", p.UnbondingSeconds, minReleaseUnbondingSeconds)
		}
		if p.UnbondingSeconds > 365*24*3600 {
			return nil, fmt.Errorf("unbonding_seconds %d is not plausible", p.UnbondingSeconds)
		}
		if err := checkEstimate(p); err != nil {
			return nil, err
		}
		if min := p.ActivationTimeEstimate + 2*p.UnbondingSeconds + expiryMarginSeconds; p.ExpiryTime < min {
			return nil, fmt.Errorf("expiry_time %d is earlier than activation estimate + 2 x unbonding + 7 days (%d)", p.ExpiryTime, min)
		}
	}
	sum := sha256.Sum256(listed)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, p.ListSHA256) {
		return nil, fmt.Errorf("listed.txt sha256 %s does not match params %s", got, p.ListSHA256)
	}
	decode := func(what, s string) ([]byte, error) {
		b, err := ac.StringToBytes(s)
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", what, s, err)
		}
		if len(b) != core.AddrLen {
			return nil, fmt.Errorf("%s %q has %d bytes, want %d", what, s, len(b), core.AddrLen)
		}
		if !p.TestValues && fixtureAddresses[string(b)] {
			return nil, fmt.Errorf("%s %q is a known test address", what, s)
		}
		canon, err := ac.BytesToString(b)
		if err != nil {
			return nil, err
		}
		if canon != s {
			return nil, fmt.Errorf("%s must be canonical", what)
		}
		if _, isBlocked := blocked[canon]; isBlocked {
			return nil, fmt.Errorf("%s %q is a module account or blocked address", what, s)
		}
		return b, nil
	}
	if !bytes.HasSuffix(listed, []byte("\n")) {
		return nil, fmt.Errorf("listed.txt must end with LF")
	}
	var list [][]byte
	prev := ""
	for i, line := range strings.Split(strings.TrimSuffix(string(listed), "\n"), "\n") {
		if line <= prev {
			return nil, fmt.Errorf("listed.txt line %d is empty, duplicated or out of order", i+1)
		}
		b, err := decode(fmt.Sprintf("list line %d", i+1), line)
		if err != nil {
			return nil, err
		}
		prev = line
		list = append(list, b)
	}
	if len(list) != p.ListCount {
		return nil, fmt.Errorf("listed.txt has %d addresses, params say %d", len(list), p.ListCount)
	}
	rec, err := decode("recovery address", p.RecoveryAddress)
	if err != nil {
		return nil, err
	}
	payer, err := decode("fee payer", p.FeePayers[0])
	if err != nil {
		return nil, err
	}
	return core.NewConfig(list, rec, [][]byte{payer}, p.ActivationHeight, p.ExpiryTime)
}
