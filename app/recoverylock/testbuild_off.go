//go:build !recoverylock_test

package recoverylock

import _ "embed"

// testBuild is false in every build without -tags recoverylock_test,
// including release builds. Such builds refuse test values.
const testBuild = false

// For release, copy mainnet-wallets.txt to listed.txt and populate
// params.json from mainnet-wallets.json and the chain team's release settings.
//
//go:embed listed.txt
var listedFile []byte

//go:embed params.json
var paramsFile []byte
