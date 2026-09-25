package gno

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	"github.com/stretchr/testify/require"

	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// TestConvertToGnoCommit_AbsentValidators tests that ConvertToGnoCommit correctly
// handles absent validators represented as zero-value CommitSig structs.
// Proto3 repeated message fields always deserialize as non-nil pointers, so absent
// validators appear as zero-value CommitSig{} rather than nil entries. Without the
// fix, this would panic due to nil BlockId dereference and MustAddressFromString("").
func TestConvertToGnoCommit_AbsentValidators(t *testing.T) {
	chainID := testChainID
	height := int64(10)
	blockTime := time.Now().UTC()

	// Create a validator set with 3 validators, but only 2 will sign
	valSet, privKeys := createTestValidatorSet(3, 100)
	signedHeader := createTestSignedHeader(chainID, height, blockTime, valSet, privKeys)

	// Simulate what a proto3 deserialization produces for absent validators:
	// replace the 3rd precommit with a zero-value CommitSig (not nil).
	signedHeader.Commit.Precommits[2] = &CommitSig{}

	// This must not panic
	gnoCommit, err := ConvertToGnoCommit(signedHeader.Commit)
	require.NoError(t, err)
	require.NotNil(t, gnoCommit)

	// The absent validator's slot should be nil in the converted commit
	require.Nil(t, gnoCommit.Precommits[2], "absent validator should produce nil precommit entry")

	// The present validators should have been converted correctly
	require.NotNil(t, gnoCommit.Precommits[0])
	require.NotNil(t, gnoCommit.Precommits[1])
}

// TestConvertToGnoCommit_AllAbsent tests that a commit with all zero-value
// CommitSig entries (all absent) does not panic.
func TestConvertToGnoCommit_AllAbsent(t *testing.T) {
	commit := &Commit{
		BlockId: createTestBlockID(),
		Precommits: []*CommitSig{
			{}, // zero-value: absent validator
			{}, // zero-value: absent validator
			{}, // zero-value: absent validator
		},
	}

	gnoCommit, err := ConvertToGnoCommit(commit)
	require.NoError(t, err)
	require.NotNil(t, gnoCommit)

	for i, pc := range gnoCommit.Precommits {
		require.Nil(t, pc, "precommit %d should be nil for absent validator", i)
	}
}

// TestConvertToGnoValidatorSet_RejectsMalformedSets ensures that
// ConvertToGnoValidatorSet validates relayer-supplied validator sets the same way
// Gno's NewValidatorSet constructor does. A set containing negative voting power must
// be rejected: otherwise the total voting power can become negative, making the +2/3
// commit threshold negative and satisfiable by a single signature, which would allow a
// forged light-client update.
func TestConvertToGnoValidatorSet_RejectsMalformedSets(t *testing.T) {
	valA, keyA := createTestValidator(1)
	valB, keyB := createTestValidator(10)
	// A duplicate of valA (same address) for the duplicate-address case.
	dupA := createTestValidatorWithKey(5, keyA)

	t.Run("negative voting power", func(t *testing.T) {
		neg := createTestValidatorWithKey(-10, keyB)
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{valA, neg}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)

		// Sanity check: Gno's own constructor also rejects this set.
		require.Panics(t, func() {
			bfttypes.NewValidatorSet([]*bfttypes.Validator{
				toBftValidator(valA), toBftValidator(neg),
			})
		})
	})

	t.Run("zero voting power", func(t *testing.T) {
		zero := createTestValidatorWithKey(0, keyB)
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{valA, zero}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
	})

	t.Run("duplicate address", func(t *testing.T) {
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{valA, dupA}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
	})

	t.Run("voting power exceeds max", func(t *testing.T) {
		tooBig := createTestValidatorWithKey(bfttypes.MaxTotalVotingPower+1, keyB)
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{tooBig}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
	})

	t.Run("total voting power exceeds max", func(t *testing.T) {
		half := createTestValidatorWithKey(bfttypes.MaxTotalVotingPower-1, keyA)
		half2 := createTestValidatorWithKey(bfttypes.MaxTotalVotingPower-1, keyB)
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{half, half2}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
	})

	t.Run("address not derived from pubkey", func(t *testing.T) {
		// valB's address paired with valA's pubkey.
		unbound := &Validator{Address: valB.Address, PubKey: valA.PubKey, VotingPower: 10}
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{unbound}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
		require.Contains(t, err.Error(), "does not match pubkey")

		// Sanity check: Gno's own constructor also rejects this set.
		require.Panics(t, func() {
			bfttypes.NewValidatorSet([]*bfttypes.Validator{toBftValidator(unbound)})
		})
	})

	t.Run("one pubkey under several distinct addresses", func(t *testing.T) {
		// Shape of a forged set that reuses a single key, and therefore a
		// single commit signature, across several validator slots. Every
		// address is syntactically valid and distinct; only the binding to
		// the pubkey is wrong. The first entry is legitimately bound, so the
		// converter must stop at the second and name it.
		valC, _ := createTestValidator(10)
		forged := []*Validator{
			createTestValidatorWithKey(100, keyA),
			{Address: valB.Address, PubKey: valA.PubKey, VotingPower: 100},
			{Address: valC.Address, PubKey: valA.PubKey, VotingPower: 100},
		}
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: forged})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
		require.Contains(t, err.Error(), valB.Address)
	})

	t.Run("duplicate address differing only in case", func(t *testing.T) {
		// bech32 decoding is case-insensitive, so both strings denote the
		// same address and the duplicate check must key on the parsed value.
		upper := createTestValidatorWithKey(5, keyA)
		upper.Address = strings.ToUpper(upper.Address)
		_, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{valA, upper}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
		require.Contains(t, err.Error(), "duplicate")
	})

	t.Run("pubkey with invalid length is rejected without panicking", func(t *testing.T) {
		short := createTestValidatorWithKey(10, keyA)
		short.PubKey = &cmtcrypto.PublicKey{Sum: &cmtcrypto.PublicKey_Ed25519{Ed25519: []byte{1, 2, 3}}}
		var err error
		require.NotPanics(t, func() {
			_, err = ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{short}})
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
	})

	t.Run("nil validator entry is rejected", func(t *testing.T) {
		var err error
		require.NotPanics(t, func() {
			_, err = ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{valA, nil}})
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidValidatorSet)
	})

	t.Run("valid set is accepted with order and total preserved", func(t *testing.T) {
		vals, err := ConvertToGnoValidatorSet(&ValidatorSet{Validators: []*Validator{valA, valB}})
		require.NoError(t, err)
		require.Equal(t, int64(11), vals.TotalVotingPower())
		require.Len(t, vals.Validators, 2)
		// Input order must be preserved (commit verification relies on index ordering).
		require.Equal(t, valA.Address, vals.Validators[0].Address.String())
		require.Equal(t, valB.Address, vals.Validators[1].Address.String())
	})
}

// toBftValidator converts a proto test Validator to a bfttypes.Validator for
// cross-checking against Gno's native constructor.
func toBftValidator(v *Validator) *bfttypes.Validator {
	addr, err := crypto.AddressFromString(v.Address)
	if err != nil {
		panic(err)
	}
	return &bfttypes.Validator{
		Address:     addr,
		PubKey:      ed25519.PubKeyEd25519(v.PubKey.GetEd25519()),
		VotingPower: v.VotingPower,
	}
}

// TestConvertToGnoHeader_AppVersion tests that ConvertToGnoHeader preserves
// the AppVersion field. AppVersion is included in the header's Merkle hash
// (the 7th leaf in Header.Hash). If it were dropped, the converted header's
// hash would not match the original commit's BlockID.Hash, and
// SignedHeader.ValidateBasic would reject every header.
func TestConvertToGnoHeader_AppVersion(t *testing.T) {
	appHash := make([]byte, 32)
	rand.Read(appHash)
	consensusHash := make([]byte, 32)
	rand.Read(consensusHash)
	valsHash := make([]byte, 32)
	rand.Read(valsHash)

	protoHeader := &GnoHeader{
		Version:    "1.0.0",
		ChainId:    testChainID,
		Height:     100,
		Time:       time.Now().UTC(),
		AppVersion: "v1.2.3",
		LastBlockId: &BlockID{
			Hash:        make([]byte, 32),
			PartsHeader: &PartSetHeader{Total: 1, Hash: make([]byte, 32)},
		},
		ValidatorsHash:     valsHash,
		NextValidatorsHash: valsHash,
		ConsensusHash:      consensusHash,
		AppHash:            appHash,
		ProposerAddress:    "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
	}

	bftHeader, err := ConvertToGnoHeader(protoHeader)
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", bftHeader.AppVersion,
		"AppVersion must be preserved; it is the 7th Merkle leaf in Header.Hash")

	// Verify the hash changes when AppVersion differs
	hashWith := bftHeader.Hash()

	bftHeader.AppVersion = ""
	hashWithout := bftHeader.Hash()

	require.NotEqual(t, hashWith, hashWithout,
		"header hash must differ when AppVersion changes, proving it participates in the Merkle tree")
}

// TestConvertPartSetHeader_Bounds ensures every converter rejects a PartSetHeader whose
// Total lies outside gno's [0, MaxBlockPartsCount] bound, or whose hash has the wrong
// size, instead of forwarding it. gno's CanonicalizePartSetHeader panics on a
// Total outside the uint32 range when computing vote sign bytes, and nothing in the
// ValidateBasic chain caps Total, so the bound must be enforced at conversion.
func TestConvertPartSetHeader_Bounds(t *testing.T) {
	valSet, privKeys := createTestValidatorSet(1, 100)
	signed := createTestSignedHeader(testChainID, 10, time.Now().UTC(), valSet, privKeys)
	proposer := valSet.Validators[0].Address

	psh := func(total int64, hashLen int) *PartSetHeader {
		return &PartSetHeader{Total: total, Hash: make([]byte, hashLen)}
	}
	blockID := func(p *PartSetHeader) *BlockID {
		return &BlockID{Hash: make([]byte, 32), PartsHeader: p}
	}
	// One conversion per cast site, in a slice so subtest order is stable.
	converters := []struct {
		name    string
		convert func(*PartSetHeader) error
	}{
		{"commit block id", func(p *PartSetHeader) error {
			_, err := ConvertToGnoCommit(&Commit{BlockId: blockID(p), Precommits: signed.Commit.Precommits})
			return err
		}},
		{"precommit block id", func(p *PartSetHeader) error {
			sig := *signed.Commit.Precommits[0]
			sig.BlockId = blockID(p)
			_, err := ConvertToGnoCommit(&Commit{BlockId: signed.Commit.BlockId, Precommits: []*CommitSig{&sig}})
			return err
		}},
		{"header last block id", func(p *PartSetHeader) error {
			h := createTestGnoHeader(testChainID, 10, time.Now().UTC(), make([]byte, 32), proposer)
			h.LastBlockId = blockID(p)
			_, err := ConvertToGnoHeader(h)
			return err
		}},
		{"block id", func(p *PartSetHeader) error {
			_, err := ConvertToGnoBlockID(blockID(p))
			return err
		}},
	}

	badTotals := []int64{-1, bfttypes.MaxBlockPartsCount + 1, math.MaxUint32 + 1, math.MaxInt64}
	goodTotals := []int64{0, 1, bfttypes.MaxBlockPartsCount}

	for _, c := range converters {
		name, convert := c.name, c.convert
		for _, total := range badTotals {
			t.Run(fmt.Sprintf("%s rejects total %d", name, total), func(t *testing.T) {
				var err error
				require.NotPanics(t, func() { err = convert(psh(total, 32)) })
				require.ErrorIs(t, err, clienttypes.ErrInvalidHeader)
				require.Contains(t, err.Error(), "parts header total")
			})
		}
		for _, total := range goodTotals {
			t.Run(fmt.Sprintf("%s accepts total %d", name, total), func(t *testing.T) {
				require.NoError(t, convert(psh(total, 32)))
			})
		}
		t.Run(name+" rejects wrong hash size", func(t *testing.T) {
			err := convert(psh(1, 31))
			require.ErrorIs(t, err, clienttypes.ErrInvalidHeader)
			require.Contains(t, err.Error(), "parts header hash")
		})
		t.Run(name+" accepts empty hash", func(t *testing.T) {
			require.NoError(t, convert(psh(0, 0)))
		})
	}

	t.Run("nil parts header converts to zero value", func(t *testing.T) {
		id, err := ConvertToGnoBlockID(&BlockID{Hash: make([]byte, 32)})
		require.NoError(t, err)
		require.Equal(t, bfttypes.PartSetHeader{}, id.PartsHeader)
	})
}

// TestConvertToGnoCommit_RejectsOutOfRangePrecommitType ensures a precommit Type that
// does not fit gno's byte-sized SignedMsgType is rejected instead of being truncated
// by the conversion, which would otherwise let a wire value such as 258 pass gno's
// precommit type check as PrecommitType.
func TestConvertToGnoCommit_RejectsOutOfRangePrecommitType(t *testing.T) {
	valSet, privKeys := createTestValidatorSet(1, 100)
	signed := createTestSignedHeader(testChainID, 10, time.Now().UTC(), valSet, privKeys)

	// Sanity: the narrowing conversion alone would map this to PrecommitType.
	wireType := uint32(bfttypes.PrecommitType) + 256
	require.Equal(t, bfttypes.PrecommitType, bfttypes.SignedMsgType(wireType))

	sig := *signed.Commit.Precommits[0]
	sig.Type = wireType
	_, err := ConvertToGnoCommit(&Commit{BlockId: signed.Commit.BlockId, Precommits: []*CommitSig{&sig}})
	require.ErrorIs(t, err, clienttypes.ErrInvalidHeader)
	require.Contains(t, err.Error(), "type")

	// The in-range value is still accepted.
	_, err = ConvertToGnoCommit(signed.Commit)
	require.NoError(t, err)
}
