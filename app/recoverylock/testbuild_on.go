//go:build recoverylock_test

package recoverylock

import _ "embed"

// testBuild is true only when compiled with -tags recoverylock_test. Only such
// builds accept test values and the known test addresses.
const testBuild = true

//go:embed testdata/listed.txt
var listedFile []byte

//go:embed testdata/params.json
var paramsFile []byte
