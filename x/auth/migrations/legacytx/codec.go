package legacytx

import (
	"github.com/atomone-hub/atomone/codec"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(StdTx{}, "atomone/StdTx", nil)
}
