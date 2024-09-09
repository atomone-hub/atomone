package legacybech32

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/crypto/hd"
	"github.com/atomone-hub/atomone/crypto/ledger"
	"github.com/atomone-hub/atomone/testutil/testdata"
	sdk "github.com/atomone-hub/atomone/types"
)

func TestBeach32ifPbKey(t *testing.T) {
	t.Skip() // TODO: fix! getting "ledger nano S: support for ledger devices is not available in this executable"
	require := require.New(t)
	path := *hd.NewFundraiserParams(0, sdk.CoinType, 0)
	priv, err := ledger.NewPrivKeySecp256k1Unsafe(path)
	require.Nil(err, "%s", err)
	require.NotNil(priv)

	pubKeyAddr, err := MarshalPubKey(AccPK, priv.PubKey())
	require.NoError(err)
	require.Equal("cosmospub1addwnpepqd87l8xhcnrrtzxnkql7k55ph8fr9jarf4hn6udwukfprlalu8lgw0urza0",
		pubKeyAddr, "Is your device using test mnemonic: %s ?", testdata.TestMnemonic)
}
