package recoverylock

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainnetWalletManifest(t *testing.T) {
	input, err := os.ReadFile("mainnet-wallets.json")
	require.NoError(t, err)
	manifest, err := parseParams(input)
	require.NoError(t, err)
	require.Equal(t, params{
		ChainID:         "atomone-1",
		RecoveryAddress: "atone1ng0kmvd2rz7mzh9x6227n8mqc43whn58xnf6e2",
		FeePayers:       []string{"atone1mhwea2pnk7td6pt44fwl2r8vx4nhm2dc03mldr"},
		ListCount:       5725,
		ListSHA256:      "ad653663e3b9d4afba1784d1d9f9e8d0e0ac15a75e2787586b205d014e3c680b",
	}, manifest)

	listed, err := os.ReadFile("mainnet-wallets.txt")
	require.NoError(t, err)
	require.Equal(t, manifest.ListCount, bytes.Count(listed, []byte("\n")))
	require.Equal(t, manifest.ListSHA256, fmt.Sprintf("%x", sha256.Sum256(listed)))
}
