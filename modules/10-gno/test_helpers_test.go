package gno

import (
	"crypto/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ics23 "github.com/cosmos/ics23/go"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	testChainID         = "gno-test-1"
	testClientID        = "10-gno-0"
	testTrustingPeriod  = time.Hour * 24 * 14 // 14 days
	testUnbondingPeriod = time.Hour * 24 * 21 // 21 days
	testMaxClockDrift   = time.Second * 10
	testStoreKey        = "test-store"
)

// makePrivKey creates a test ed25519 private key using proper key generation
func makePrivKey() ed25519.PrivKeyEd25519 {
	return ed25519.GenPrivKey()
}

// makePubKey creates a public key from a private key
func makePubKey(privKey ed25519.PrivKeyEd25519) ed25519.PubKeyEd25519 {
	return privKey.PubKey().(ed25519.PubKeyEd25519)
}

// makeAddress returns the address from the public key
func makeAddress(pubKey ed25519.PubKeyEd25519) crypto.Address {
	return pubKey.Address()
}

// signVoteBytes signs the canonical vote bytes with the given private key
func signVoteBytes(privKey ed25519.PrivKeyEd25519, chainID string, vote *bfttypes.Vote) []byte {
	signBytes := vote.SignBytes(chainID)
	sig, err := privKey.Sign(signBytes)
	if err != nil {
		panic(err)
	}
	return sig
}

// createBftValidatorSet creates a bfttypes.ValidatorSet from our test validators
// This is needed to compute the correct ValidatorsHash
// Note: NewValidatorSet sorts validators by address, so the returned order may differ
func createBftValidatorSet(validators []*Validator, privKeys []ed25519.PrivKeyEd25519) *bfttypes.ValidatorSet {
	bftVals := make([]*bfttypes.Validator, len(validators))
	for i, val := range validators {
		pubKey := privKeys[i].PubKey().(ed25519.PubKeyEd25519)
		bftVals[i] = &bfttypes.Validator{
			Address:          pubKey.Address(),
			PubKey:           pubKey,
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	return bfttypes.NewValidatorSet(bftVals)
}

// sortedValidatorsAndKeys returns validators and private keys sorted by address
// This matches the order that bfttypes.NewValidatorSet will use
func sortedValidatorsAndKeys(privKeys []ed25519.PrivKeyEd25519, votingPower int64) ([]*bfttypes.Validator, []ed25519.PrivKeyEd25519) {
	type valWithKey struct {
		val     *bfttypes.Validator
		privKey ed25519.PrivKeyEd25519
	}

	pairs := make([]valWithKey, len(privKeys))
	for i, privKey := range privKeys {
		pubKey := privKey.PubKey().(ed25519.PubKeyEd25519)
		pairs[i] = valWithKey{
			val: &bfttypes.Validator{
				Address:     pubKey.Address(),
				PubKey:      pubKey,
				VotingPower: votingPower,
			},
			privKey: privKey,
		}
	}

	// Sort by address (same as NewValidatorSet does)
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].val.Address.Compare(pairs[i].val.Address) < 0 {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	sortedVals := make([]*bfttypes.Validator, len(pairs))
	sortedKeys := make([]ed25519.PrivKeyEd25519, len(pairs))
	for i, pair := range pairs {
		sortedVals[i] = pair.val
		sortedKeys[i] = pair.privKey
	}

	return sortedVals, sortedKeys
}

// toBftBlockID converts a proto BlockID to bfttypes.BlockID
func toBftBlockID(blockID *BlockID) bfttypes.BlockID {
	if blockID == nil {
		return bfttypes.BlockID{}
	}
	return bfttypes.BlockID{
		Hash: blockID.Hash,
		PartsHeader: bfttypes.PartSetHeader{
			Total: int(blockID.PartsHeader.Total),
			Hash:  blockID.PartsHeader.Hash,
		},
	}
}

// toBftHeader converts a proto GnoHeader to bfttypes.Header
func toBftHeader(h *GnoHeader) *bfttypes.Header {
	return &bfttypes.Header{
		Version:            h.Version,
		ChainID:            h.ChainId,
		Height:             h.Height,
		Time:               h.Time,
		NumTxs:             h.NumTxs,
		TotalTxs:           h.TotalTxs,
		LastBlockID:        toBftBlockID(h.LastBlockId),
		LastCommitHash:     h.LastCommitHash,
		DataHash:           h.DataHash,
		ValidatorsHash:     h.ValidatorsHash,
		NextValidatorsHash: h.NextValidatorsHash,
		ConsensusHash:      h.ConsensusHash,
		AppHash:            h.AppHash,
		LastResultsHash:    h.LastResultsHash,
		ProposerAddress:    crypto.MustAddressFromString(h.ProposerAddress),
	}
}

// toBftCommit converts a proto Commit to bfttypes.Commit
func toBftCommit(c *Commit) *bfttypes.Commit {
	precommits := make([]*bfttypes.CommitSig, len(c.Precommits))
	for i, sig := range c.Precommits {
		if sig == nil {
			continue
		}
		vote := bfttypes.Vote{
			Type:             bfttypes.SignedMsgType(sig.Type),
			Height:           sig.Height,
			Round:            int(sig.Round),
			BlockID:          toBftBlockID(sig.BlockId),
			Timestamp:        sig.Timestamp,
			ValidatorAddress: crypto.MustAddressFromString(sig.ValidatorAddress),
			ValidatorIndex:   int(sig.ValidatorIndex),
			Signature:        sig.Signature,
		}
		cs := bfttypes.CommitSig(vote)
		precommits[i] = &cs
	}
	return &bfttypes.Commit{
		BlockID:    toBftBlockID(c.BlockId),
		Precommits: precommits,
	}
}

// toBftSignedHeader converts a proto SignedHeader to bfttypes.SignedHeader
func toBftSignedHeader(sh *SignedHeader) *bfttypes.SignedHeader {
	return &bfttypes.SignedHeader{
		Header: toBftHeader(sh.Header),
		Commit: toBftCommit(sh.Commit),
	}
}

// toBftValidatorSet converts a proto ValidatorSet to bfttypes.ValidatorSet
func toBftValidatorSet(vs *ValidatorSet) *bfttypes.ValidatorSet {
	if vs == nil {
		return nil
	}
	bftVals := make([]*bfttypes.Validator, len(vs.Validators))
	for i, val := range vs.Validators {
		key := val.PubKey.GetEd25519()
		bftVals[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
			PubKey:           ed25519.PubKeyEd25519(key),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	return bfttypes.NewValidatorSet(bftVals)
}

// createTestClientState creates a valid ClientState for testing
func createTestClientState(chainID string, height clienttypes.Height, frozen bool) *ClientState {
	cs := NewClientState(
		chainID,
		DefaultTrustLevel,
		testTrustingPeriod,
		testUnbondingPeriod,
		testMaxClockDrift,
		height,
		[]*ics23.ProofSpec{ics23.IavlSpec, ics23.TendermintSpec},
		[]string{"upgrade", "upgradedIBCState"},
	)
	if frozen {
		cs.FrozenHeight = FrozenHeight
	}
	return cs
}

// createTestConsensusState creates a valid ConsensusState for testing
func createTestConsensusState(timestamp time.Time) *ConsensusState {
	// Create a 32-byte hash for NextValidatorsHash
	nextValsHash := make([]byte, 32)
	rand.Read(nextValsHash)

	return NewConsensusState(
		timestamp,
		commitmenttypes.NewMerkleRoot([]byte("apphash")),
		nextValsHash,
	)
}

// createTestValidator creates a test Validator with the given voting power
// The private key is generated and returned for signing purposes
func createTestValidator(votingPower int64) (*Validator, ed25519.PrivKeyEd25519) {
	privKey := makePrivKey()
	return createTestValidatorWithKey(votingPower, privKey), privKey
}

// createTestValidatorWithKey creates a test Validator from the given private key
func createTestValidatorWithKey(votingPower int64, privKey ed25519.PrivKeyEd25519) *Validator {
	pubKey := privKey.PubKey().(ed25519.PubKeyEd25519)
	addr := pubKey.Address()

	return &Validator{
		Address:          addr.String(),
		PubKey:           &cmtcrypto.PublicKey{Sum: &cmtcrypto.PublicKey_Ed25519{Ed25519: pubKey[:]}},
		VotingPower:      votingPower,
		ProposerPriority: 0,
	}
}

// createTestValidatorSet creates a validator set with the given number of validators
func createTestValidatorSet(numValidators int, votingPower int64) (*ValidatorSet, []ed25519.PrivKeyEd25519) {
	validators := make([]*Validator, numValidators)
	privKeys := make([]ed25519.PrivKeyEd25519, numValidators)

	for i := 0; i < numValidators; i++ {
		val, privKey := createTestValidator(votingPower)
		validators[i] = val
		privKeys[i] = privKey
	}

	return &ValidatorSet{
		Validators: validators,
	}, privKeys
}

// createTestBlockID creates a test BlockID
func createTestBlockID() *BlockID {
	hash := make([]byte, 32)
	rand.Read(hash)

	partsHash := make([]byte, 32)
	rand.Read(partsHash)

	return &BlockID{
		Hash: hash,
		PartsHeader: &PartSetHeader{
			Total: 1,
			Hash:  partsHash,
		},
	}
}

// createTestGnoHeader creates a test GnoHeader
func createTestGnoHeader(chainID string, height int64, blockTime time.Time, nextValsHash []byte, proposerAddr string) *GnoHeader {
	lastBlockID := createTestBlockID()
	appHash := make([]byte, 32)
	rand.Read(appHash)

	valsHash := make([]byte, 32)
	rand.Read(valsHash)

	consensusHash := make([]byte, 32)
	rand.Read(consensusHash)

	return &GnoHeader{
		Version:            "1.0.0",
		ChainId:            chainID,
		Height:             height,
		Time:               blockTime,
		NumTxs:             0,
		TotalTxs:           0,
		LastBlockId:        lastBlockID,
		LastCommitHash:     []byte{},
		DataHash:           nil,
		ValidatorsHash:     valsHash,
		NextValidatorsHash: nextValsHash,
		ConsensusHash:      consensusHash,
		AppHash:            appHash,
		LastResultsHash:    nil,
		ProposerAddress:    proposerAddr,
	}
}

// createTestSignedHeader creates a test SignedHeader with valid signatures
// The header hash is computed and used as the BlockID in the commit
func createTestSignedHeader(chainID string, height int64, blockTime time.Time, valSet *ValidatorSet, privKeys []ed25519.PrivKeyEd25519) *SignedHeader {
	// Sort validators and keys to match the order NewValidatorSet will use
	sortedVals, sortedKeys := sortedValidatorsAndKeys(privKeys, valSet.Validators[0].VotingPower)

	// Create bft validator set (this will also sort by address)
	bftValSet := bfttypes.NewValidatorSet(sortedVals)
	valsHash := bftValSet.Hash()

	// Create random hashes for other fields
	appHash := make([]byte, 32)
	rand.Read(appHash)
	consensusHash := make([]byte, 32)
	rand.Read(consensusHash)
	lastCommitHash := make([]byte, 32)
	rand.Read(lastCommitHash)
	lastBlockHash := make([]byte, 32)
	rand.Read(lastBlockHash)
	partsHash := make([]byte, 32)
	rand.Read(partsHash)

	// Get proposer address from the first sorted validator
	proposerAddr := sortedVals[0].Address

	// Create the bft header first so we can compute its hash
	bftHeader := &bfttypes.Header{
		Version:  "1.0.0",
		ChainID:  chainID,
		Height:   height,
		Time:     blockTime,
		NumTxs:   0,
		TotalTxs: 0,
		LastBlockID: bfttypes.BlockID{
			Hash: lastBlockHash,
			PartsHeader: bfttypes.PartSetHeader{
				Total: 1,
				Hash:  partsHash,
			},
		},
		LastCommitHash:     lastCommitHash,
		DataHash:           nil,
		ValidatorsHash:     valsHash,
		NextValidatorsHash: valsHash, // Same val set for next block
		ConsensusHash:      consensusHash,
		AppHash:            appHash,
		LastResultsHash:    nil,
		ProposerAddress:    proposerAddr,
	}

	// Compute the header hash - this is what the commit must sign
	headerHash := bftHeader.Hash()

	// Create the BlockID that points to this header
	bftBlockID := bfttypes.BlockID{
		Hash: headerHash,
		PartsHeader: bfttypes.PartSetHeader{
			Total: 1,
			Hash:  partsHash,
		},
	}

	// Create and sign precommits using sorted order
	precommits := make([]*CommitSig, len(sortedVals))
	for i, val := range sortedVals {
		// Create a bfttypes.Vote to generate correct sign bytes
		vote := &bfttypes.Vote{
			Type:             bfttypes.PrecommitType,
			Height:           height,
			Round:            0,
			BlockID:          bftBlockID,
			Timestamp:        blockTime,
			ValidatorAddress: val.Address,
			ValidatorIndex:   i,
		}

		// Sign the vote with the corresponding private key (sorted order)
		signature := signVoteBytes(sortedKeys[i], chainID, vote)

		// Create the proto CommitSig
		precommits[i] = &CommitSig{
			Type:   2, // PrecommitType
			Height: height,
			Round:  0,
			BlockId: &BlockID{
				Hash: headerHash,
				PartsHeader: &PartSetHeader{
					Total: 1,
					Hash:  partsHash,
				},
			},
			Timestamp:        blockTime,
			ValidatorAddress: val.Address.String(),
			ValidatorIndex:   int64(i),
			Signature:        signature,
		}
	}

	// Create proto types from bft types
	gnoHeader := &GnoHeader{
		Version:  bftHeader.Version,
		ChainId:  bftHeader.ChainID,
		Height:   bftHeader.Height,
		Time:     bftHeader.Time,
		NumTxs:   bftHeader.NumTxs,
		TotalTxs: bftHeader.TotalTxs,
		LastBlockId: &BlockID{
			Hash: bftHeader.LastBlockID.Hash,
			PartsHeader: &PartSetHeader{
				Total: int64(bftHeader.LastBlockID.PartsHeader.Total),
				Hash:  bftHeader.LastBlockID.PartsHeader.Hash,
			},
		},
		LastCommitHash:     bftHeader.LastCommitHash,
		DataHash:           bftHeader.DataHash,
		ValidatorsHash:     bftHeader.ValidatorsHash,
		NextValidatorsHash: bftHeader.NextValidatorsHash,
		ConsensusHash:      bftHeader.ConsensusHash,
		AppHash:            bftHeader.AppHash,
		LastResultsHash:    bftHeader.LastResultsHash,
		ProposerAddress:    bftHeader.ProposerAddress.String(),
	}

	commit := &Commit{
		BlockId: &BlockID{
			Hash: headerHash,
			PartsHeader: &PartSetHeader{
				Total: 1,
				Hash:  partsHash,
			},
		},
		Precommits: precommits,
	}

	return &SignedHeader{
		Header: gnoHeader,
		Commit: commit,
	}
}

// createTestHeader creates a test Header for IBC updates
func createTestHeader(t *testing.T, chainID string, height int64, trustedHeight clienttypes.Height, blockTime time.Time) *Header {
	t.Helper()

	valSet, privKeys := createTestValidatorSet(1, 100)
	signedHeader := createTestSignedHeader(chainID, height, blockTime, valSet, privKeys)

	return &Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: valSet,
	}
}

// createTestHeaderWithKeys creates a test Header and returns the private keys for verification testing
func createTestHeaderWithKeys(t *testing.T, chainID string, height int64, trustedHeight clienttypes.Height, blockTime time.Time, numValidators int, votingPower int64) (*Header, *ValidatorSet, []ed25519.PrivKeyEd25519) {
	t.Helper()

	valSet, privKeys := createTestValidatorSet(numValidators, votingPower)
	signedHeader := createTestSignedHeader(chainID, height, blockTime, valSet, privKeys)

	header := &Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: valSet,
	}

	return header, valSet, privKeys
}

// createChainedTestHeaders creates two headers for adjacent verification testing
// The first header is the "trusted" header, the second is the "untrusted" header
func createChainedTestHeaders(t *testing.T, chainID string, trustedHeight int64, untrustedHeight int64, trustedTime time.Time, untrustedTime time.Time, numValidators int, votingPower int64) (*Header, *Header, *ValidatorSet, []ed25519.PrivKeyEd25519) {
	t.Helper()

	valSet, privKeys := createTestValidatorSet(numValidators, votingPower)

	// Create trusted header (at trustedHeight)
	trustedSignedHeader := createTestSignedHeader(chainID, trustedHeight, trustedTime, valSet, privKeys)
	trustedHeader := &Header{
		SignedHeader:      trustedSignedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     clienttypes.NewHeight(0, uint64(trustedHeight-1)), // previous height
		TrustedValidators: valSet,
	}

	// Create untrusted header (at untrustedHeight)
	untrustedSignedHeader := createTestSignedHeader(chainID, untrustedHeight, untrustedTime, valSet, privKeys)
	untrustedHeader := &Header{
		SignedHeader:      untrustedSignedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     clienttypes.NewHeight(0, uint64(trustedHeight)),
		TrustedValidators: valSet,
	}

	return trustedHeader, untrustedHeader, valSet, privKeys
}

// setupClientStore sets up a client store for testing
func setupClientStore(t *testing.T) storetypes.KVStore {
	t.Helper()

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(testStoreKey)
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	return ms.GetKVStore(storeKey)
}

// getTestCodec returns a codec for testing
func getTestCodec() codec.BinaryCodec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// getTestContext returns a test SDK context
func getTestContext(t *testing.T, blockTime time.Time) sdk.Context {
	t.Helper()

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(testStoreKey)
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, cmtproto.Header{Time: blockTime}, false, log.NewNopLogger())
	return ctx
}
