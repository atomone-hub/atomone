//go:build test_amino
// +build test_amino

package params

import (
	"github.com/atomone-hub/atomone/codec"
	cdctypes "github.com/atomone-hub/atomone/codec/types"
	"github.com/atomone-hub/atomone/x/auth/migrations/legacytx"
)

func MakeTestEncodingConfig() EncodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         codec,
		TxConfig:          legacytx.StdTxConfig{Cdc: cdc},
		Amino:             cdc,
	}
}
