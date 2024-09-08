package rosetta

import (
	"github.com/atomone-hub/atomone/codec"
	codectypes "github.com/atomone-hub/atomone/codec/types"
	cryptocodec "github.com/atomone-hub/atomone/crypto/codec"
	authcodec "github.com/atomone-hub/atomone/x/auth/types"
	bankcodec "github.com/atomone-hub/atomone/x/bank/types"
)

// MakeCodec generates the codec required to interact
// with the cosmos APIs used by the rosetta gateway
func MakeCodec() (*codec.ProtoCodec, codectypes.InterfaceRegistry) {
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	authcodec.RegisterInterfaces(ir)
	bankcodec.RegisterInterfaces(ir)
	cryptocodec.RegisterInterfaces(ir)

	return cdc, ir
}
