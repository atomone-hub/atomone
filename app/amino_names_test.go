package atomone_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"cosmossdk.io/api/amino"
	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"

	atomone "github.com/atomone-hub/atomone/app"
)

// maxAminoNameLength is the length limit of the amino names, as enforced by legacy.RegisterAminoMsg.
const maxAminoNameLength = 39

// The nodes derive the SIGN_MODE_LEGACY_AMINO_JSON sign bytes from the amino.name
// option of the proto messages, the LegacyAmino codec is what the clients using it
// to encode messages rely on. The two must agree, or a message signed against one
// is not valid against the other.
func TestLegacyAminoNamesMatchProto(t *testing.T) {
	app := atomone.NewAtomOneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, atomone.EmptyAppOptions{})
	registry := app.InterfaceRegistry()
	legacyAmino := app.LegacyAmino()

	var checked int
	for _, typeURL := range registry.ListImplementations(sdk.MsgInterfaceProtoName) {
		if !strings.HasPrefix(typeURL, "/atomone.") {
			continue
		}
		t.Run(typeURL, func(t *testing.T) {
			desc, err := proto.HybridResolver.FindDescriptorByName(protoreflect.FullName(strings.TrimPrefix(typeURL, "/")))
			require.NoError(t, err)
			protoName, _ := protov2.GetExtension(desc.(protoreflect.MessageDescriptor).Options(), amino.E_Name).(string)
			if protoName == "" {
				// the nodes encode a message without amino.name as a bare object, there is
				// no name to agree on
				t.Skip("no amino.name")
			}

			// longer names cannot be signed with a Ledger, see cosmos/cosmos-sdk#10870
			require.LessOrEqual(t, len(protoName), maxAminoNameLength)

			msg, err := registry.Resolve(typeURL)
			require.NoError(t, err)
			bz, err := legacyAmino.Amino.MarshalJSON(msg)
			require.NoError(t, err, "not registered in the LegacyAmino codec, amino.name is %q", protoName)
			var enc struct {
				Type string `json:"type"`
			}
			require.NoError(t, json.Unmarshal(bz, &enc))
			require.Equal(t, protoName, enc.Type)
		})
		checked++
	}
	require.Greater(t, checked, 10, "the atomone messages were not found")
}
