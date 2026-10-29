package spectre

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	errorsmod "cosmossdk.io/errors"
)

// This file handles the execution block headers of Gloas-format light client
// data. A Gloas light client header no longer carries the execution payload
// header: it proves only execution_block_hash. The execution state root,
// timestamp and block number that the client stores in its consensus state
// therefore come from the RLP-encoded execution block header supplied beside
// the update, and are trusted only because keccak256(rlp) equals the block
// hash that the beacon block body proves. It mirrors
// packages/ethereum/light-client/src/execution.rs in the spectre repository.

// maxExecutionHeaderRLPBytes bounds the RLP-encoded execution block header
// before it is hashed. A post-Prague header is about 600 bytes with extra_data
// capped at 32 bytes; the bound leaves room for fields later forks add.
const maxExecutionHeaderRLPBytes = 2048

// Positions of the items the client reads; they are the same in every fork
// since genesis. Fields later forks append need no handling, because the hash
// check binds the whole encoding.
const (
	executionHeaderStateRootIndex = 3
	executionHeaderNumberIndex    = 8
	executionHeaderTimestampIndex = 11
)

// executionSummary holds the execution header fields the client stores per
// finalized header.
type executionSummary struct {
	stateRoot   []byte
	timestamp   uint64
	blockNumber uint64
}

// finalizedExecutionSummary returns the execution state root, timestamp and
// block number of the finalized header of a Header client message.
//
// A pre-Gloas-format header carries its execution payload header, which its
// merkle branch already proves. A Gloas-format header carries only the
// execution block hash, so the values come from header.ExecutionHeaderRlp,
// checked against that hash.
func finalizedExecutionSummary(header *Header) (executionSummary, error) {
	finalized := &header.ConsensusUpdate.FinalizedHeader
	hasExecution := finalized.Execution != nil
	hasBlockHash := len(finalized.ExecutionBlockHash) != 0
	hasRLP := len(header.ExecutionHeaderRlp) != 0

	switch {
	case hasExecution && !hasBlockHash && !hasRLP:
		return executionSummary{
			stateRoot:   finalized.Execution.StateRoot,
			timestamp:   finalized.Execution.Timestamp,
			blockNumber: finalized.Execution.BlockNumber,
		}, nil
	case hasExecution && !hasBlockHash && hasRLP:
		return executionSummary{}, errorsmod.Wrap(ErrInvalidExecutionHeader,
			"execution header rlp supplied for a pre-Gloas format finalized header")
	case !hasExecution && hasBlockHash && hasRLP:
		return verifyExecutionHeaderRLP(header.ExecutionHeaderRlp, bytes32(finalized.ExecutionBlockHash))
	case !hasExecution && hasBlockHash:
		return executionSummary{}, errorsmod.Wrap(ErrInvalidExecutionHeader,
			"execution header rlp is required for a Gloas format finalized header")
	default:
		return executionSummary{}, errorsmod.Wrap(ErrInvalidHeader,
			"light client header must set exactly one of execution and execution_block_hash")
	}
}

// verifyExecutionHeaderRLP checks that the RLP-encoded execution block header
// hashes to expectedHash and reads the state root, block number and timestamp
// out of it.
func verifyExecutionHeaderRLP(raw []byte, expectedHash [32]byte) (executionSummary, error) {
	if len(raw) > maxExecutionHeaderRLPBytes {
		return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader,
			"execution header rlp is %d bytes, maximum %d", len(raw), maxExecutionHeaderRLPBytes)
	}
	if got := crypto.Keccak256Hash(raw); got != expectedHash {
		return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader,
			"execution header rlp hashes to %s, but the proven execution block hash is %X", got, expectedHash)
	}

	payload, rest, err := rlp.SplitList(raw)
	if err != nil {
		return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader, "malformed list: %v", err)
	}
	if len(rest) != 0 {
		return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader, "%d trailing bytes after the list", len(rest))
	}

	var summary executionSummary
	for index := 0; index <= executionHeaderTimestampIndex; index++ {
		kind, content, tail, err := rlp.Split(payload)
		if err != nil {
			return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader, "item %d: %v", index, err)
		}
		if kind == rlp.List {
			return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader, "item %d is a list", index)
		}
		item := payload[:len(payload)-len(tail)]
		payload = tail

		switch index {
		case executionHeaderStateRootIndex:
			if len(content) != 32 {
				return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader,
					"state root is %d bytes, expected 32", len(content))
			}
			summary.stateRoot = append([]byte(nil), content...)
		case executionHeaderNumberIndex:
			if err := rlp.DecodeBytes(item, &summary.blockNumber); err != nil {
				return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader, "block number: %v", err)
			}
		case executionHeaderTimestampIndex:
			if err := rlp.DecodeBytes(item, &summary.timestamp); err != nil {
				return executionSummary{}, errorsmod.Wrapf(ErrInvalidExecutionHeader, "timestamp: %v", err)
			}
		}
	}
	return summary, nil
}
