package types

const (
	// ModuleName defines the module name
	ModuleName = "photon"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var ParamsKey = []byte{0x00}
