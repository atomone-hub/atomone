package tx

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/atomone-hub/atomone/codec"
	codectypes "github.com/atomone-hub/atomone/codec/types"
	"github.com/atomone-hub/atomone/std"
	"github.com/atomone-hub/atomone/testutil/testdata"
	sdk "github.com/atomone-hub/atomone/types"
	txtestutil "github.com/atomone-hub/atomone/x/auth/tx/testutil"
)

func TestGenerator(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &testdata.TestMsg{})
	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	suite.Run(t, txtestutil.NewTxConfigTestSuite(NewTxConfig(protoCodec, DefaultSignModes)))
}
