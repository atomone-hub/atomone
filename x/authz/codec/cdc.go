package codec

import (
	"github.com/atomone-hub/atomone/codec"
	cryptocodec "github.com/atomone-hub/atomone/crypto/codec"
	sdk "github.com/atomone-hub/atomone/types"
)

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(Amino)
)

func init() {
	cryptocodec.RegisterCrypto(Amino)
	codec.RegisterEvidences(Amino)
	sdk.RegisterLegacyAminoCodec(Amino)
}
