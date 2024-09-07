package params

import (
	"github.com/atomone-hub/atomone/client"
	"github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/codec/types"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}
