package ledger

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdkledger "github.com/cosmos/cosmos-sdk/crypto/ledger"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestOpenApp(t *testing.T) {
	app, err := openApp(newFakeCosmosDevice())
	require.NoError(t, err)
	require.IsType(t, &CosmosApp{}, app)

	_, err = openApp(errorDevice{sw: 0x6E00})
	require.ErrorContains(t, err, "open the Cosmos app on the Ledger device")

	_, err = openApp(errorDevice{sw: swDeviceLocked})
	require.ErrorContains(t, err, "unlock")

	_, err = openApp(failingDevice{})
	require.ErrorContains(t, err, "transfer failed")
}

// TestSDKLedger runs the Cosmos SDK's own Ledger key code, as the keyring
// does for --ledger keys, against this package's client.
func TestSDKLedger(t *testing.T) {
	d := newFakeCosmosDevice()
	sdkledger.SetDiscoverLedger(func() (sdkledger.SECP256K1, error) { return openApp(d) })
	t.Cleanup(Register)

	params := *hd.NewFundraiserParams(0, sdk.CoinType, 0)
	priv, addr, err := sdkledger.NewPrivKeySecp256k1(params, "atone")
	require.NoError(t, err)
	require.Equal(t, d.address("atone"), addr)
	require.Equal(t, d.key.PubKey().Bytes(), priv.PubKey().Bytes())

	msg := []byte(`{"account_number":"1","chain_id":"atomone-1","msgs":[]}`)
	for _, sign := range []func([]byte) ([]byte, error){
		priv.Sign, // SIGN_MODE_TEXTUAL
		priv.(interface {
			SignLedgerAminoJSON([]byte) ([]byte, error)
		}).SignLedgerAminoJSON,
	} {
		sig, err := sign(msg)
		require.NoError(t, err)
		require.True(t, priv.PubKey().VerifySignature(msg, sig))
	}
}
