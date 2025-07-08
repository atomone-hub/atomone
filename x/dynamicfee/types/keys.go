package types

const (
	// ModuleName is the name of the dynamicfee module.
	ModuleName = "dynamicfee"
	// StoreKey is the store key string for the dynamicfee module.
	StoreKey = ModuleName
)

const (
	prefixParams = iota + 1
	prefixState
	prefixEnableHeight = 3
)

var (
	// KeyParams is the store key for the dynamicfee module's parameters.
	KeyParams = []byte{prefixParams}

	// KeyState is the store key for the dynamicfee module's data.
	KeyState = []byte{prefixState}

	// KeyEnabledHeight is the store key for the dynamicfee module's enabled height.
	KeyEnabledHeight = []byte{prefixEnableHeight}
)
