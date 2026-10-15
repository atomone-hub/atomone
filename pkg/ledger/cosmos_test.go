package ledger

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"testing"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

// path is m/44'/118'/3'/0/7 as hd.BIP44Params.DerivationPath returns it.
var path = []uint32{44, 118, 3, 0, 7}

// swError is how the transport reports a status word other than 0x9000.
func swError(sw uint16) error {
	return &statusError{sw: sw}
}

// fakeCosmosDevice answers like the Ledger Cosmos app (as ledger-cosmos-go
// drives it) for a single secp256k1 key, and as the transport reports errors.
type fakeCosmosDevice struct {
	version [3]byte
	key     *secp256k1.PrivKey
	reject  bool
	apdus   [][]byte
	signing []byte // payload of the signature in progress
	open    bool
}

func newFakeCosmosDevice() *fakeCosmosDevice {
	return &fakeCosmosDevice{version: [3]byte{2, 39, 1}, key: secp256k1.GenPrivKey()}
}

func (d *fakeCosmosDevice) address(hrp string) string {
	addr, err := bech32.ConvertAndEncode(hrp, d.key.PubKey().Address())
	if err != nil {
		panic(err)
	}
	return addr
}

// checkCosmosPath accepts m/44'/118'/a'/c/i in the app's little-endian layout.
func checkCosmosPath(p []byte) error {
	if len(p) != 20 {
		return swError(0x6A80)
	}
	c := func(i int) uint32 { return binary.LittleEndian.Uint32(p[4*i:]) }
	if c(0) != 44|0x8000_0000 || c(1) != 118|0x8000_0000 || c(2) < 0x8000_0000 || c(3) >= 0x8000_0000 || c(4) >= 0x8000_0000 {
		return swError(0x6A80)
	}
	return nil
}

func (d *fakeCosmosDevice) Exchange(apdu []byte) ([]byte, error) {
	d.apdus = append(d.apdus, apdu)
	if apdu[0] != cosmosCLA {
		return nil, swError(0x6E00)
	}
	ins, p1, p2, data := apdu[1], apdu[2], apdu[3], apdu[5:]
	switch ins {
	case cosmosInsGetVersion:
		return []byte{0, d.version[0], d.version[1], d.version[2]}, nil
	case cosmosInsGetAddress:
		hrp := string(data[1 : 1+data[0]])
		if err := checkCosmosPath(data[1+data[0]:]); err != nil {
			return nil, err
		}
		if p1 == 1 && d.reject {
			return nil, swError(0x6986)
		}
		return append(d.key.PubKey().Bytes(), d.address(hrp)...), nil
	case cosmosInsSign:
		if p2 > cosmosSignModeTextual {
			return nil, swError(0x6B00)
		}
		switch p1 {
		case cosmosP1SignInit:
			if err := checkCosmosPath(data); err != nil {
				return nil, err
			}
			d.signing, d.open = nil, true
			return nil, nil
		case cosmosP1SignAdd, cosmosP1SignLast:
			if !d.open || len(data) > cosmosChunkSize {
				return nil, swError(0x6984)
			}
			d.signing = append(d.signing, data...)
			if p1 == cosmosP1SignAdd {
				return nil, nil
			}
		}
		d.open = false
		if d.reject {
			return nil, swError(0x6986)
		}
		if string(d.signing) == "not json" {
			return nil, &statusError{sw: 0x6A80, data: []byte("PARSER ERROR: JSMN_ERROR_INVAL")}
		}
		hash := sha256.Sum256(d.signing)
		return ecdsa.Sign(secp.PrivKeyFromBytes(d.key.Key), hash[:]).Serialize(), nil
	}
	return nil, swError(0x6D00)
}

func (*fakeCosmosDevice) Close() error { return nil }

// errorDevice answers every command with the same status word.
type errorDevice struct{ sw uint16 }

func (d errorDevice) Exchange([]byte) ([]byte, error) { return nil, swError(d.sw) }
func (errorDevice) Close() error                      { return nil }

// failingDevice fails every transfer, as an unresponsive device does.
type failingDevice struct{}

func (failingDevice) Exchange([]byte) ([]byte, error) { return nil, errors.New("transfer failed") }
func (failingDevice) Close() error                    { return nil }

func TestCosmosPath(t *testing.T) {
	got, err := cosmosPath(path)
	require.NoError(t, err)
	want := []byte{}
	for _, c := range []uint32{44 | 0x8000_0000, 118 | 0x8000_0000, 3 | 0x8000_0000, 0, 7} {
		want = binary.LittleEndian.AppendUint32(want, c)
	}
	require.Equal(t, want, got)

	_, err = cosmosPath(path[:4])
	require.Error(t, err)
	_, err = cosmosPath([]uint32{44, 118, 3 | 0x8000_0000, 0, 7})
	require.Error(t, err)
}

func TestOpenCosmosApp(t *testing.T) {
	tests := []struct {
		name    string
		dev     device
		wantErr string
		notApp  bool
	}{
		{name: "current version", dev: newFakeCosmosDevice()},
		{name: "oldest supported", dev: &fakeCosmosDevice{version: [3]byte{2, 1, 0}, key: secp256k1.GenPrivKey()}},
		{name: "too old", dev: &fakeCosmosDevice{version: [3]byte{2, 0, 9}, key: secp256k1.GenPrivKey()}, wantErr: "too old"},
		{name: "another app or the dashboard", dev: errorDevice{sw: 0x6E00}, notApp: true},
		{name: "locked", dev: errorDevice{sw: swDeviceLocked}, wantErr: "unlock"},
		{name: "transfer failure", dev: failingDevice{}, wantErr: "transfer failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := openCosmosApp(tc.dev)
			switch {
			case tc.notApp:
				require.ErrorIs(t, err, errNotCosmosApp)
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestCosmosAppPublicKey(t *testing.T) {
	d := newFakeCosmosDevice()
	app, err := openCosmosApp(d)
	require.NoError(t, err)
	want := d.key.PubKey().Bytes()

	pub, err := app.GetPublicKeySECP256K1(path)
	require.NoError(t, err)
	require.Equal(t, want, pub)
	require.Equal(t, byte(0), d.apdus[len(d.apdus)-1][2])

	pub, addr, err := app.GetAddressPubKeySECP256K1(path, "atone")
	require.NoError(t, err)
	require.Equal(t, want, pub)
	require.Equal(t, d.address("atone"), addr)
	require.Equal(t, byte(1), d.apdus[len(d.apdus)-1][2])

	for _, hrp := range []string{"", "ATONE", "atone\x7f"} {
		_, _, err = app.GetAddressPubKeySECP256K1(path, hrp)
		require.ErrorContains(t, err, "invalid address prefix", "%q", hrp)
	}

	d.reject = true
	_, _, err = app.GetAddressPubKeySECP256K1(path, "atone")
	require.ErrorContains(t, err, "rejected on the Ledger device")
}

func TestCosmosAppSign(t *testing.T) {
	d := newFakeCosmosDevice()
	app, err := openCosmosApp(d)
	require.NoError(t, err)
	msg := make([]byte, 2*cosmosChunkSize+5)
	for i := range msg {
		msg[i] = byte(i)
	}

	d.apdus = nil
	der, err := app.SignSECP256K1(path, msg, 0)
	require.NoError(t, err)

	// The path, then 48-byte chunks: every APDU fits one HID report.
	require.Len(t, d.apdus, 4)
	for i, want := range []struct{ p1, n byte }{
		{cosmosP1SignInit, 20}, {cosmosP1SignAdd, cosmosChunkSize}, {cosmosP1SignAdd, cosmosChunkSize}, {cosmosP1SignLast, 5},
	} {
		require.Equal(t, []byte{cosmosCLA, cosmosInsSign, want.p1, 0, want.n}, d.apdus[i][:5], "APDU %d", i)
		require.Len(t, frameAPDU(d.apdus[i]), 1, "APDU %d", i)
	}

	sig, err := ecdsa.ParseDERSignature(der)
	require.NoError(t, err)
	hash := sha256.Sum256(msg)
	require.True(t, sig.Verify(hash[:], secp.PrivKeyFromBytes(d.key.Key).PubKey()))

	_, err = app.SignSECP256K1(path, msg, 2)
	require.Error(t, err)

	_, err = app.SignSECP256K1(path, []byte("not json"), 0)
	require.ErrorContains(t, err, "JSMN_ERROR_INVAL")

	d.reject = true
	_, err = app.SignSECP256K1(path, msg, 0)
	require.ErrorContains(t, err, "rejected on the Ledger device")
}
