package legacytx_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/atomone-hub/atomone/codec"
	cryptoAmino "github.com/atomone-hub/atomone/crypto/codec"
	"github.com/atomone-hub/atomone/testutil/testdata"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/auth/migrations/legacytx"
	txtestutil "github.com/atomone-hub/atomone/x/auth/tx/testutil"
)

func testCodec() *codec.LegacyAmino {
	cdc := codec.NewLegacyAmino()
	sdk.RegisterLegacyAminoCodec(cdc)
	cryptoAmino.RegisterCrypto(cdc)
	cdc.RegisterConcrete(&testdata.TestMsg{}, "atomone/Test", nil)
	return cdc
}

func TestStdTxConfig(t *testing.T) {
	cdc := testCodec()
	txGen := legacytx.StdTxConfig{Cdc: cdc}
	suite.Run(t, txtestutil.NewTxConfigTestSuite(txGen))
}
