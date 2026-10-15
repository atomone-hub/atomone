// Package ledger reaches the Ledger Cosmos app without CGO.
//
// The Cosmos SDK talks to Ledger devices through cosmos/ledger-cosmos-go,
// zondax/ledger-go and zondax/hid, a CGO binding of hidapi, and only in
// binaries built with CGO and the ledger build tag. This package speaks the
// Cosmos app's protocol over a pure-Go HID backend instead; [Register] makes
// the SDK's keyring use it, in any build.
package ledger

import (
	"errors"
	"fmt"
	"runtime"

	sdkledger "github.com/cosmos/cosmos-sdk/crypto/ledger"
)

// Register makes the Cosmos SDK reach Ledger devices through [Discover].
// Call it before any command runs.
func Register() {
	sdkledger.SetDiscoverLedger(Discover)
}

// Discover connects to the first Ledger device and returns a client for the
// Cosmos app, which must be open on it.
func Discover() (sdkledger.SECP256K1, error) {
	if !hidSupported {
		return nil, fmt.Errorf("no Ledger support on %s", runtime.GOOS)
	}
	dev, err := connectHID()
	if err != nil {
		return nil, err
	}
	app, err := openApp(dev)
	if err != nil {
		_ = dev.Close()
		return nil, err
	}
	return app, nil
}

// openApp returns a client for the Cosmos app open on dev.
func openApp(dev device) (sdkledger.SECP256K1, error) {
	app, err := openCosmosApp(dev)
	switch {
	case err == nil:
		return app, nil
	case errors.Is(err, errNotCosmosApp):
		return nil, errors.New("open the Cosmos app on the Ledger device")
	default:
		return nil, err
	}
}
