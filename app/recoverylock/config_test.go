package recoverylock

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/require"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
)

//go:embed testdata/listed.txt
var fixtureList []byte

//go:embed testdata/params.json
var fixtureParams []byte

func TestRecoveryManifestBuildInputs(t *testing.T) {
	if testBuild {
		require.True(t, bytes.Equal(listedFile, fixtureList), "fixture build embedded the release address list")
		require.True(t, bytes.Equal(paramsFile, fixtureParams), "fixture build embedded the release parameters")
		return
	}

	p, err := parseParams(paramsFile)
	require.NoError(t, err)
	if p.TestValues {
		require.PanicsWithValue(t, "recovery lock config: test values outside a recoverylock_test build", func() {
			MustLoad(nil, nil)
		})
		return
	}

	_, chainID := MustLoad(nil, nil)
	require.Equal(t, "atomone-1", chainID)
}

func TestRecoveryManifest(t *testing.T) {
	ac := addresscodec.NewBech32Codec("atone")
	cfg, p, err := parseConfig(fixtureList, fixtureParams, ac, nil, true)
	require.NoError(t, err)
	require.Len(t, cfg.Listed(), p.ListCount)
	require.True(t, p.TestValues)

	_, _, err = parseConfig(fixtureList, fixtureParams, ac, nil, false)
	require.ErrorContains(t, err, "test values outside")

	release := p
	release.TestValues = false
	release.ChainID = "atomone-1"
	release.ActivationTimeEstimate = 1800000000
	release.EstimateRefHeight = p.ActivationHeight - 100
	release.EstimateRefTime = release.ActivationTimeEstimate - 500
	release.UnbondingSeconds = minReleaseUnbondingSeconds
	t.Run("known test address", func(t *testing.T) {
		for _, allowFixtures := range []bool{false, true} {
			_, err := build(fixtureList, release, ac, nil, allowFixtures)
			require.ErrorContains(t, err, "known test address", "allowFixtures=%t", allowFixtures)
		}
	})

	required := 2*release.UnbondingSeconds + expiryMarginSeconds
	for _, tc := range []struct {
		name string
		edit func(*params)
		want string
	}{
		{
			name: "wrong chain",
			edit: func(p *params) { p.ChainID = "atomone-recovery-test" },
			want: "release recovery lock requires atomone-1",
		},
		{
			name: "expiry below activation margin",
			edit: func(p *params) { p.ExpiryTime = p.ActivationTimeEstimate + required - 1 },
			want: fmt.Sprintf("expiry_time %d is earlier than activation estimate + 2 x unbonding + 7 days (%d)",
				release.ActivationTimeEstimate+required-1, release.ActivationTimeEstimate+required),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := release
			tc.edit(&p)
			for _, allowFixtures := range []bool{false, true} {
				_, err := build(fixtureList, p, ac, nil, allowFixtures)
				require.EqualError(t, err, tc.want, "allowFixtures=%t", allowFixtures)
			}
		})
	}

	release.ExpiryTime = release.ActivationTimeEstimate + required
	_, err = build(fixtureList, release, ac, nil, false)
	require.ErrorContains(t, err, "known test address")

	_, err = build(fixtureList, p, ac, map[string]bool{p.RecoveryAddress: false}, true)
	require.ErrorContains(t, err, "module account")
	_, err = build([]byte("tampered\n"), p, ac, nil, true)
	require.ErrorContains(t, err, "sha256")

	lines := strings.Split(strings.TrimSuffix(string(fixtureList), "\n"), "\n")
	invalid := []struct {
		text string
		want string
	}{
		{lines[0] + "\n" + lines[1], "end with LF"},
		{lines[1] + "\n" + lines[0] + "\n", "out of order"},
		{lines[0] + "\n" + lines[0] + "\n", "duplicated"},
		{strings.ToUpper(lines[0]) + "\n" + lines[1] + "\n", "canonical"},
	}
	for _, tc := range invalid {
		p.ListSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(tc.text)))
		_, err := build([]byte(tc.text), p, ac, nil, true)
		require.ErrorContains(t, err, tc.want)
	}
}

func TestRecoveryManifestJSON(t *testing.T) {
	for _, invalid := range []string{
		`{"test_values": false, "test_values": true}`,
		`{"TEST_VALUES": true}`,
		`{"test_values": "true"}`,
		`{}{}`,
	} {
		_, err := parseParams([]byte(invalid))
		require.Error(t, err)
	}
}
