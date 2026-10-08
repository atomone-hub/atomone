package spectre

import (
	errorsmod "cosmossdk.io/errors"
)

// Light client sentinel errors.
var (
	ErrInvalidClientID            = errorsmod.Register(ModuleName, 2, "invalid client identifier")
	ErrInvalidHeader              = errorsmod.Register(ModuleName, 3, "invalid header")
	ErrInvalidMisbehaviour        = errorsmod.Register(ModuleName, 4, "invalid misbehaviour")
	ErrMustBeElectraOrLater       = errorsmod.Register(ModuleName, 5, "fork must be electra or later")
	ErrFrozen                     = errorsmod.Register(ModuleName, 6, "client is frozen")
	ErrInvalidConsensusState      = errorsmod.Register(ModuleName, 7, "invalid consensus state")
	ErrInvalidClientState         = errorsmod.Register(ModuleName, 8, "invalid client state")
	ErrInsufficientParticipants   = errorsmod.Register(ModuleName, 9, "insufficient sync committee participants")
	ErrNotEnoughSignatures        = errorsmod.Register(ModuleName, 10, "not enough sync committee signatures")
	ErrInvalidSlots               = errorsmod.Register(ModuleName, 11, "invalid slot ordering in light client update")
	ErrInvalidSignaturePeriod     = errorsmod.Register(ModuleName, 12, "invalid signature period")
	ErrIrrelevantUpdate           = errorsmod.Register(ModuleName, 13, "irrelevant light client update")
	ErrInvalidMerkleBranch        = errorsmod.Register(ModuleName, 14, "invalid merkle branch")
	ErrBLSVerificationFailed      = errorsmod.Register(ModuleName, 15, "bls signature verification failed")
	ErrSyncCommitteeMismatch      = errorsmod.Register(ModuleName, 16, "sync committee mismatch")
	ErrInvalidSyncCommittee       = errorsmod.Register(ModuleName, 17, "invalid sync committee")
	ErrInvalidCommitmentPath      = errorsmod.Register(ModuleName, 18, "invalid commitment path")
	ErrInvalidProof               = errorsmod.Register(ModuleName, 19, "invalid proof")
	ErrValueMismatch              = errorsmod.Register(ModuleName, 20, "stored value mismatch")
	ErrUpdateNotValid             = errorsmod.Register(ModuleName, 21, "light client update failed verification")
	ErrClientAndConsensusMismatch = errorsmod.Register(ModuleName, 22, "client state and consensus state slot mismatch")
	ErrUnsupportedOperation       = errorsmod.Register(ModuleName, 23, "unsupported operation")
)
