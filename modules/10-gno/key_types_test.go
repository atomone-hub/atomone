package gno

import (
	"math/big"
	"testing"
	"time"

	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/crypto/secp256k1"
	"github.com/stretchr/testify/require"

	cmtmath "github.com/cometbft/cometbft/libs/math"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
)

// createSortedTestValidatorSet creates a validator set in address order, the
// order a Gno chain keeps its validators in, together with the private keys in
// the same order. Keys are of the current test key type.
func createSortedTestValidatorSet(numValidators int, votingPower int64) (*ValidatorSet, []crypto.PrivKey) {
	keys := make([]crypto.PrivKey, numValidators)
	for i := range keys {
		keys[i] = makePrivKey()
	}
	_, sortedKeys := sortedValidatorsAndKeys(keys, votingPower)
	vals := make([]*Validator, numValidators)
	for i, key := range sortedKeys {
		vals[i] = createTestValidatorWithKey(votingPower, key)
	}
	return &ValidatorSet{Validators: vals}, sortedKeys
}

// createTrustedHeader creates a header at the given height signed by every
// validator of valSet, which is also the trusted validator set.
func createTrustedHeader(height int64, trustedHeight clienttypes.Height, blockTime time.Time, valSet *ValidatorSet, keys []crypto.PrivKey) *Header {
	return &Header{
		SignedHeader:      createTestSignedHeader(testChainID, height, blockTime, valSet, keys),
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: valSet,
	}
}

// consensusStateTrusting creates a consensus state whose next validators are valSet.
func consensusStateTrusting(t *testing.T, valSet *ValidatorSet, timestamp time.Time) *ConsensusState {
	t.Helper()
	gnoVals, err := ConvertToGnoValidatorSet(valSet)
	require.NoError(t, err)
	return &ConsensusState{
		Timestamp:          timestamp,
		Root:               commitmenttypes.NewMerkleRoot([]byte("root")),
		NextValidatorsHash: gnoVals.Hash(),
		LcType:             Gno,
	}
}

// TestConvertToGnoValidatorSet_KeyTypes tests that validator sets with ed25519
// keys, secp256k1 keys or both convert to the set Gno builds from the same
// validators: same key types, addresses and hash.
func TestConvertToGnoValidatorSet_KeyTypes(t *testing.T) {
	forEachKeyType(t, func(t *testing.T) {
		valSet, keys := createSortedTestValidatorSet(4, 100)

		gnoVals, err := ConvertToGnoValidatorSet(valSet)
		require.NoError(t, err)
		require.Len(t, gnoVals.Validators, 4)

		want := make([]*bfttypes.Validator, len(keys))
		for i, key := range keys {
			pubKey := key.PubKey()
			require.Equal(t, pubKey, gnoVals.Validators[i].PubKey)
			require.Equal(t, pubKey.Address(), gnoVals.Validators[i].Address)
			want[i] = &bfttypes.Validator{Address: pubKey.Address(), PubKey: pubKey, VotingPower: 100}
		}
		// The hash is what binds the set to the header, so it has to be the
		// one a Gno node computes.
		require.Equal(t, bfttypes.NewValidatorSet(want).Hash(), gnoVals.Hash())
	})
}

// TestConvertToGnoValidatorSet_KeyTypesInSet tests that the generated sets
// really hold the key types their name says, so that the tests running over
// every key type cover secp256k1.
func TestConvertToGnoValidatorSet_KeyTypesInSet(t *testing.T) {
	count := func(t *testing.T) (ed, secp int) {
		valSet, _ := createSortedTestValidatorSet(4, 100)
		gnoVals, err := ConvertToGnoValidatorSet(valSet)
		require.NoError(t, err)
		for _, val := range gnoVals.Validators {
			switch val.PubKey.(type) {
			case ed25519.PubKeyEd25519:
				ed++
			case secp256k1.PubKeySecp256k1:
				secp++
			}
		}
		return ed, secp
	}

	useKeyType(t, keyTypeSecp256k1)
	ed, secp := count(t)
	require.Equal(t, 0, ed)
	require.Equal(t, 4, secp)

	useKeyType(t, keyTypeMixed)
	ed, secp = count(t)
	require.Equal(t, 2, ed)
	require.Equal(t, 2, secp)
}

// TestConvertToGnoValidatorSet_RejectsMalformedKeys tests the public keys the
// conversion refuses: a wrong length for the declared type, an address derived
// from another key, and a missing or unknown key type.
func TestConvertToGnoValidatorSet_RejectsMalformedKeys(t *testing.T) {
	secpKey := secp256k1.GenPrivKey()
	secpPub := secpKey.PubKey().(secp256k1.PubKeySecp256k1)
	edKey := ed25519.GenPrivKey()
	edPub := edKey.PubKey().(ed25519.PubKeyEd25519)

	secpProto := func(bz []byte) *cmtcrypto.PublicKey {
		return &cmtcrypto.PublicKey{Sum: &cmtcrypto.PublicKey_Secp256K1{Secp256K1: bz}}
	}
	edProto := func(bz []byte) *cmtcrypto.PublicKey {
		return &cmtcrypto.PublicKey{Sum: &cmtcrypto.PublicKey_Ed25519{Ed25519: bz}}
	}

	tests := []struct {
		name    string
		address string
		pubKey  *cmtcrypto.PublicKey
		wantErr error
	}{
		{"secp256k1 key too short", secpPub.Address().String(), secpProto(secpPub[:3]), ErrInvalidValidatorSet},
		{"secp256k1 key with 32 bytes", secpPub.Address().String(), secpProto(secpPub[:32]), ErrInvalidValidatorSet},
		{"secp256k1 key with 34 bytes", secpPub.Address().String(), secpProto(append(secpPub[:], 0)), ErrInvalidValidatorSet},
		{"secp256k1 uncompressed key", secpPub.Address().String(), secpProto(make([]byte, 65)), ErrInvalidValidatorSet},
		{"secp256k1 empty key", secpPub.Address().String(), secpProto([]byte{}), ErrInvalidValidatorSet},
		{"ed25519 bytes declared as secp256k1", edPub.Address().String(), secpProto(edPub[:]), ErrInvalidValidatorSet},
		{"secp256k1 bytes declared as ed25519", secpPub.Address().String(), edProto(secpPub[:]), ErrInvalidValidatorSet},
		{"ed25519 empty key", edPub.Address().String(), edProto([]byte{}), ErrInvalidValidatorSet},
		{"secp256k1 key with the address of another key", edPub.Address().String(), secpProto(secpPub[:]), ErrInvalidValidatorSet},
		{"ed25519 key with the address of another key", secpPub.Address().String(), edProto(edPub[:]), ErrInvalidValidatorSet},
		{"no key", secpPub.Address().String(), nil, clienttypes.ErrInvalidHeader},
		{"key without a type", secpPub.Address().String(), &cmtcrypto.PublicKey{}, clienttypes.ErrInvalidHeader},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valSet := &ValidatorSet{Validators: []*Validator{
				{Address: tc.address, PubKey: tc.pubKey, VotingPower: 10},
			}}
			var err error
			require.NotPanics(t, func() {
				_, err = ConvertToGnoValidatorSet(valSet)
			})
			require.ErrorIs(t, err, tc.wantErr)
		})
	}

	t.Run("one key under two key types is one validator each", func(t *testing.T) {
		// The same 32 bytes cannot be reused as a secp256k1 key, and a valid
		// key of each type gives two distinct addresses, so a signature is
		// never counted for two slots.
		valSet := &ValidatorSet{Validators: []*Validator{
			{Address: edPub.Address().String(), PubKey: edProto(edPub[:]), VotingPower: 10},
			{Address: secpPub.Address().String(), PubKey: secpProto(secpPub[:]), VotingPower: 10},
		}}
		gnoVals, err := ConvertToGnoValidatorSet(valSet)
		require.NoError(t, err)
		require.NotEqual(t, gnoVals.Validators[0].Address, gnoVals.Validators[1].Address)
	})
}

// TestVerifyClientMessage_Header_KeyTypes tests that a header signed by
// ed25519 validators, secp256k1 validators or both is verified and stored,
// for an adjacent and a non-adjacent update, and that a forged signature
// is rejected.
func TestVerifyClientMessage_Header_KeyTypes(t *testing.T) {
	forEachKeyType(t, func(t *testing.T) {
		for _, height := range []int64{101, 150} {
			cdc := getTestCodec()
			clientStore := setupClientStore(t)
			blockTime := time.Now().UTC()
			ctx := getTestContext(t, blockTime.Add(time.Second))

			trustedHeight := clienttypes.NewHeight(1, 100)
			valSet, keys := createSortedTestValidatorSet(4, 100)
			cs := createTestClientState(testChainID, trustedHeight, false)
			setClientState(clientStore, cdc, cs)
			setConsensusState(clientStore, cdc, consensusStateTrusting(t, valSet, blockTime.Add(-time.Hour)), trustedHeight)

			header := createTrustedHeader(height, trustedHeight, blockTime, valSet, keys)
			require.NoError(t, header.ValidateBasic())
			require.NoError(t, cs.VerifyClientMessage(ctx, cdc, clientStore, header))

			heights := cs.UpdateState(ctx, cdc, clientStore, header)
			require.Len(t, heights, 1)
			require.True(t, heights[0].EQ(clienttypes.NewHeight(1, uint64(height))))
			stored, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(1, uint64(height)))
			require.True(t, found)
			require.Equal(t, header.SignedHeader.Header.NextValidatorsHash, stored.NextValidatorsHash)

			// Every signature counts: forging any one of them fails the commit.
			for i := range header.SignedHeader.Commit.Precommits {
				forged := createTrustedHeader(height, trustedHeight, blockTime, valSet, keys)
				forged.SignedHeader.Commit.Precommits[i].Signature[10] ^= 0xFF
				require.Error(t, cs.VerifyClientMessage(ctx, cdc, clientStore, forged))
			}
		}
	})
}

// TestVerifyClientMessage_Header_KeyTypeChange tests a chain moving its
// validators from ed25519 to secp256k1: a header signed by a new secp256k1 set
// is accepted when enough of the trusted ed25519 validators are still in it.
func TestVerifyClientMessage_Header_KeyTypeChange(t *testing.T) {
	cdc := getTestCodec()
	clientStore := setupClientStore(t)
	blockTime := time.Now().UTC()
	ctx := getTestContext(t, blockTime.Add(time.Second))
	trustedHeight := clienttypes.NewHeight(1, 100)

	useKeyType(t, keyTypeEd25519)
	oldKeys := []crypto.PrivKey{makePrivKey(), makePrivKey(), makePrivKey()}
	useKeyType(t, keyTypeSecp256k1)
	newKeys := []crypto.PrivKey{makePrivKey(), makePrivKey()}

	sortedSet := func(keys []crypto.PrivKey) (*ValidatorSet, []crypto.PrivKey) {
		_, sorted := sortedValidatorsAndKeys(keys, 100)
		vals := make([]*Validator, len(sorted))
		for i, key := range sorted {
			vals[i] = createTestValidatorWithKey(100, key)
		}
		return &ValidatorSet{Validators: vals}, sorted
	}

	trustedSet, _ := sortedSet(oldKeys)
	cs := createTestClientState(testChainID, trustedHeight, false)
	setClientState(clientStore, cdc, cs)
	setConsensusState(clientStore, cdc, consensusStateTrusting(t, trustedSet, blockTime.Add(-time.Hour)), trustedHeight)

	// Two of the three trusted ed25519 validators remain beside two new
	// secp256k1 ones: 200 of the 300 trusted power signs, above the 1/3 level.
	newSet, newSorted := sortedSet(append(append([]crypto.PrivKey{}, oldKeys[:2]...), newKeys...))
	header := &Header{
		SignedHeader:      createTestSignedHeader(testChainID, 150, blockTime, newSet, newSorted),
		ValidatorSet:      newSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedSet,
	}
	require.NoError(t, header.ValidateBasic())
	require.NoError(t, cs.VerifyClientMessage(ctx, cdc, clientStore, header))

	// A set of secp256k1 validators only has none of the trusted power.
	onlyNew, onlyNewSorted := sortedSet(newKeys)
	header = &Header{
		SignedHeader:      createTestSignedHeader(testChainID, 150, blockTime, onlyNew, onlyNewSorted),
		ValidatorSet:      onlyNew,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedSet,
	}
	require.NoError(t, header.ValidateBasic())
	require.ErrorIs(t, cs.VerifyClientMessage(ctx, cdc, clientStore, header), ErrNewValSetCantBeTrusted)
}

// TestVerifyClientMessage_Misbehaviour_KeyTypes tests that two conflicting
// headers signed by ed25519 validators, secp256k1 validators or both are
// valid misbehaviour that freezes the client.
func TestVerifyClientMessage_Misbehaviour_KeyTypes(t *testing.T) {
	forEachKeyType(t, func(t *testing.T) {
		cdc := getTestCodec()
		clientStore := setupClientStore(t)
		blockTime := time.Now().UTC()
		ctx := getTestContext(t, blockTime.Add(time.Second))

		trustedHeight := clienttypes.NewHeight(1, 100)
		valSet, keys := createSortedTestValidatorSet(4, 100)
		cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 150), false)
		setClientState(clientStore, cdc, cs)
		setConsensusState(clientStore, cdc, consensusStateTrusting(t, valSet, blockTime.Add(-time.Hour)), trustedHeight)

		// Two different blocks at the same height, both signed by the set.
		header1 := createTrustedHeader(150, trustedHeight, blockTime, valSet, keys)
		header2 := createTrustedHeader(150, trustedHeight, blockTime, valSet, keys)
		misbehaviour := NewMisbehaviour(testClientID, header1, header2)

		require.NoError(t, misbehaviour.ValidateBasic())
		require.NoError(t, cs.VerifyClientMessage(ctx, cdc, clientStore, misbehaviour))
		require.True(t, cs.CheckForMisbehaviour(ctx, cdc, clientStore, misbehaviour))

		// A header the set did not sign is not misbehaviour.
		header2.SignedHeader.Commit.Precommits[0].Signature[10] ^= 0xFF
		require.Error(t, misbehaviour.ValidateBasic())
		require.Error(t, cs.VerifyClientMessage(ctx, cdc, clientStore, misbehaviour))
	})
}

// TestVerifyLightCommit_Secp256k1SignatureEncoding tests that a secp256k1
// precommit signature is accepted in one encoding only: 64 bytes r||s with a
// low s. A second encoding of the same signature would let two nodes disagree
// on, or a relayer alter, an otherwise valid commit.
func TestVerifyLightCommit_Secp256k1SignatureEncoding(t *testing.T) {
	useKeyType(t, keyTypeSecp256k1)
	blockTime := time.Now().UTC()
	height := int64(10)
	trustLevel := cmtmath.Fraction{Numerator: 1, Denominator: 3}

	valSet, keys := createSortedTestValidatorSet(1, 100)
	gnoVals, err := ConvertToGnoValidatorSet(valSet)
	require.NoError(t, err)

	verify := func(mutate func(sig []byte) []byte) error {
		signedHeader := createTestSignedHeader(testChainID, height, blockTime, valSet, keys)
		precommit := signedHeader.Commit.Precommits[0]
		precommit.Signature = mutate(precommit.Signature)
		commit, err := ConvertToGnoCommit(signedHeader.Commit)
		require.NoError(t, err)
		return VerifyLightCommit(gnoVals, testChainID, commit.BlockID, height, commit, trustLevel)
	}

	require.NoError(t, verify(func(sig []byte) []byte {
		require.Len(t, sig, 64)
		return sig
	}))

	// (r, n-s) is the other valid ECDSA signature for the same message.
	curveOrder, ok := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	require.True(t, ok)
	err = verify(func(sig []byte) []byte {
		s := new(big.Int).SetBytes(sig[32:])
		highS := new(big.Int).Sub(curveOrder, s)
		out := append([]byte{}, sig[:32]...)
		return append(out, highS.FillBytes(make([]byte, 32))...)
	})
	require.ErrorContains(t, err, "invalid signature")

	// Ethereum-style r||s||v.
	for _, v := range []byte{0, 1, 27, 28} {
		err = verify(func(sig []byte) []byte { return append(append([]byte{}, sig...), v) })
		require.ErrorContains(t, err, "invalid signature")
	}

	err = verify(func(sig []byte) []byte { return sig[:63] })
	require.ErrorContains(t, err, "invalid signature")
}
