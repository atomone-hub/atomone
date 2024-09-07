package tx

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/atomone-hub/atomone/codec"
	codectypes "github.com/atomone-hub/atomone/codec/types"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	txtestutil "github.com/cosmos/cosmos-sdk/x/auth/tx/testutil"
)

func TestGenerator(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &testdata.TestMsg{})
	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	suite.Run(t, txtestutil.NewTxConfigTestSuite(NewTxConfig(protoCodec, DefaultSignModes)))
}
