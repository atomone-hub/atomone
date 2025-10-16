package gno

import (
	"bytes"
	"errors"
	"fmt"
	math "math"
	"time"

	errorsmod "cosmossdk.io/errors"
	cmtmath "github.com/cometbft/cometbft/libs/math"
	"github.com/cometbft/cometbft/types"
	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
)

// DefaultTrustLevel - new header can be trusted if at least one correct
// validator signed it.
var LCDefaultTrustLevel = cmtmath.Fraction{Numerator: 1, Denominator: 3}

// VerifyNonAdjacent verifies non-adjacent untrustedHeader against
// trustedHeader. It ensures that:
//
//		a) trustedHeader can still be trusted (if not, ErrOldHeaderExpired is returned)
//		b) untrustedHeader is valid (if not, ErrInvalidHeader is returned)
//		c) trustLevel ([1/3, 1]) of trustedHeaderVals (or trustedHeaderNextVals)
//	 signed correctly (if not, ErrNewValSetCantBeTrusted is returned)
//		d) more than 2/3 of untrustedVals have signed h2
//	   (otherwise, ErrInvalidHeader is returned)
//	 e) headers are non-adjacent.
//
// maxClockDrift defines how much untrustedHeader.Time can drift into the
// future.
func VerifyNonAdjacent(
	trustedHeader *bfttypes.SignedHeader, // height=X
	trustedVals *bfttypes.ValidatorSet, // height=X or height=X+1
	untrustedHeader *bfttypes.SignedHeader, // height=Y
	untrustedVals *bfttypes.ValidatorSet, // height=Y
	trustingPeriod time.Duration,
	now time.Time,
	maxClockDrift time.Duration,
	trustLevel cmtmath.Fraction,
) error {
	if untrustedHeader.Height == trustedHeader.Height+1 {
		return errors.New("headers must be non adjacent in height")
	}

	if HeaderExpired(trustedHeader, trustingPeriod, now) {
		return errorsmod.Wrapf(ErrOldHeaderExpired, "trusted header expired at %v (now: %v)", trustedHeader.Time.Add(trustingPeriod), now)
	}

	if err := verifyNewHeaderAndVals(
		untrustedHeader, untrustedVals,
		trustedHeader,
		now, maxClockDrift); err != nil {
		return errorsmod.Wrapf(ErrInvalidHeader, "failed to verify new header and vals: %v", err)
	}

	// Ensure that +`trustLevel` (default 1/3) or more of last trusted validators signed correctly.
	err := VerifyLightCommit(trustedVals, trustedHeader.ChainID, untrustedHeader.Commit.BlockID, untrustedHeader.Height, untrustedHeader.Commit, trustLevel)
	if err != nil {
		return errorsmod.Wrapf(ErrNewValSetCantBeTrusted, "trusted validators failed to verify commit: %v", err)
	}

	// Ensure that +2/3 of new validators signed correctly.
	//
	// NOTE: this should always be the last check because untrustedVals can be
	// intentionally made very large to DOS the light client. not the case for
	// VerifyAdjacent, where validator set is known in advance.
	if err := untrustedVals.VerifyCommit(trustedHeader.ChainID, untrustedHeader.Commit.BlockID,
		untrustedHeader.Height, untrustedHeader.Commit); err != nil {
		return errorsmod.Wrapf(ErrInvalidHeader, "failed to verify commit: %v", err)
	}

	return nil
}

// VerifyAdjacent verifies directly adjacent untrustedHeader against
// trustedHeader. It ensures that:
//
//	a) trustedHeader can still be trusted (if not, ErrOldHeaderExpired is returned)
//	b) untrustedHeader is valid (if not, ErrInvalidHeader is returned)
//	c) untrustedHeader.ValidatorsHash equals trustedHeader.NextValidatorsHash
//	d) more than 2/3 of new validators (untrustedVals) have signed h2
//	  (otherwise, ErrInvalidHeader is returned)
//	e) headers are adjacent.
//
// maxClockDrift defines how much untrustedHeader.Time can drift into the
// future.
func VerifyAdjacent(
	trustedHeader *bfttypes.SignedHeader, // height=X
	untrustedHeader *bfttypes.SignedHeader, // height=X+1
	untrustedVals *bfttypes.ValidatorSet, // height=X+1
	trustingPeriod time.Duration,
	now time.Time,
	maxClockDrift time.Duration,
) error {
	if untrustedHeader.Height != trustedHeader.Height+1 {
		return errors.New("headers must be adjacent in height")
	}

	if HeaderExpired(trustedHeader, trustingPeriod, now) {
		return errorsmod.Wrapf(ErrOldHeaderExpired, "trusted header expired at %v (now: %v)", trustedHeader.Time.Add(trustingPeriod), now)
	}

	if err := verifyNewHeaderAndVals(
		untrustedHeader, untrustedVals,
		trustedHeader,
		now, maxClockDrift); err != nil {
		return errorsmod.Wrapf(ErrInvalidHeader, "failed to verify new header and vals: %v", err)
	}

	// Check the validator hashes are the same
	if !bytes.Equal(untrustedHeader.ValidatorsHash, trustedHeader.NextValidatorsHash) {
		err := fmt.Errorf("expected old header next validators (%X) to match those from new header (%X)",
			trustedHeader.NextValidatorsHash,
			untrustedHeader.ValidatorsHash,
		)
		return err
	}

	// Ensure that +2/3 of new validators signed correctly.
	if err := untrustedVals.VerifyCommit(trustedHeader.ChainID, untrustedHeader.Commit.BlockID,
		untrustedHeader.Height, untrustedHeader.Commit); err != nil {
		return errorsmod.Wrapf(ErrInvalidHeader, "failed to verify commit: %v", err)
	}

	return nil
}

// Verify combines both VerifyAdjacent and VerifyNonAdjacent functions.
func Verify(
	trustedHeader *bfttypes.SignedHeader, // height=X
	trustedVals *bfttypes.ValidatorSet, // height=X or height=X+1
	untrustedHeader *bfttypes.SignedHeader, // height=Y
	untrustedVals *bfttypes.ValidatorSet, // height=Y
	trustingPeriod time.Duration,
	now time.Time,
	maxClockDrift time.Duration,
	trustLevel cmtmath.Fraction,
) error {
	if untrustedHeader.Height != trustedHeader.Height+1 {
		return VerifyNonAdjacent(trustedHeader, trustedVals, untrustedHeader, untrustedVals,
			trustingPeriod, now, maxClockDrift, trustLevel)
	}

	return VerifyAdjacent(trustedHeader, untrustedHeader, untrustedVals, trustingPeriod, now, maxClockDrift)
}

func verifyNewHeaderAndVals(
	untrustedHeader *bfttypes.SignedHeader,
	untrustedVals *bfttypes.ValidatorSet,
	trustedHeader *bfttypes.SignedHeader,
	now time.Time,
	maxClockDrift time.Duration,
) error {
	if err := untrustedHeader.ValidateBasic(trustedHeader.ChainID); err != nil {
		return fmt.Errorf("untrustedHeader.ValidateBasic failed: %w", err)
	}

	if untrustedHeader.Height <= trustedHeader.Height {
		return fmt.Errorf("expected new header height %d to be greater than one of old header %d",
			untrustedHeader.Height,
			trustedHeader.Height)
	}

	if !untrustedHeader.Time.After(trustedHeader.Time) {
		return fmt.Errorf("expected new header time %v to be after old header time %v",
			untrustedHeader.Time,
			trustedHeader.Time)
	}

	if !untrustedHeader.Time.Before(now.Add(maxClockDrift)) {
		return fmt.Errorf("new header has a time from the future %v (now: %v; max clock drift: %v)",
			untrustedHeader.Time,
			now,
			maxClockDrift)
	}

	if !bytes.Equal(untrustedHeader.ValidatorsHash, untrustedVals.Hash()) {
		return fmt.Errorf("expected new header validators (%X) to match those that were supplied (%X) at height %d",
			untrustedHeader.ValidatorsHash,
			untrustedVals.Hash(),
			untrustedHeader.Height,
		)
	}

	return nil
}

// ValidateTrustLevel checks that trustLevel is within the allowed range [1/3,
// 1]. If not, it returns an error. 1/3 is the minimum amount of trust needed
// which does not break the security model.
func ValidateTrustLevel(lvl cmtmath.Fraction) error {
	if lvl.Numerator*3 < lvl.Denominator || // < 1/3
		lvl.Numerator > lvl.Denominator || // > 1
		lvl.Denominator == 0 {
		return fmt.Errorf("trustLevel must be within [1/3, 1], given %v", lvl)
	}
	return nil
}
func VerifyLightCommit(vals *bfttypes.ValidatorSet, chainID string, blockID bfttypes.BlockID, height int64, commit *bfttypes.Commit, trustLevel cmtmath.Fraction) error {
	if err := commit.ValidateBasic(); err != nil {
		return err
	}
	if vals.Size() != len(commit.Precommits) {
		return errorsmod.Wrapf(ErrNewValSetCantBeTrusted, bfttypes.NewErrInvalidCommitPrecommits(vals.Size(), len(commit.Precommits)).Error())
	}
	if height != commit.Height() {
		return errorsmod.Wrapf(ErrNewValSetCantBeTrusted, bfttypes.NewErrInvalidCommitHeight(height, commit.Height()).Error())
	}
	if !blockID.Equals(commit.BlockID) {
		return fmt.Errorf("invalid commit -- wrong block id: want %v got %v",
			blockID, commit.BlockID)
	}

	talliedVotingPower := int64(0)

	for idx, precommit := range commit.Precommits {
		if precommit == nil {
			continue // OK, some precommits can be missing.
		}
		_, val := vals.GetByIndex(idx)
		// Validate signature.
		precommitSignBytes := commit.VoteSignBytes(chainID, idx)
		if !val.PubKey.VerifyBytes(precommitSignBytes, precommit.Signature) {
			return fmt.Errorf("invalid commit -- invalid signature: %v", precommit)
		}
		// Good precommit!
		if blockID.Equals(precommit.BlockID) {
			talliedVotingPower += val.VotingPower
		}
		// else {
		// It's OK that the BlockID doesn't match.  We include stray
		// precommits to measure validator availability.
		// }
	}

	// safely calculate voting power needed.
	totalVotingPowerMulByNumerator, overflow := safeMul(vals.TotalVotingPower(), int64(trustLevel.Numerator))
	if overflow {
		return errorsmod.Wrapf(ErrNewValSetCantBeTrusted, "int64 overflow while calculating voting power needed. please provide smaller trustLevel numerator")
	}
	votingPowerNeeded := totalVotingPowerMulByNumerator / int64(trustLevel.Denominator)
	if talliedVotingPower > votingPowerNeeded {
		return nil
	}
	return errorsmod.Wrapf(ErrNewValSetCantBeTrusted, "Invalid commit -- insufficient old voting power: got %v, needed %v", talliedVotingPower, vals.TotalVotingPower()*2/3+1)
}

func safeMul(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, false
	}

	absOfB := b
	if b < 0 {
		absOfB = -b
	}

	absOfA := a
	if a < 0 {
		absOfA = -a
	}

	if absOfA > math.MaxInt64/absOfB {
		return 0, true
	}

	return a * b, false
}

// HeaderExpired return true if the given header expired.
func HeaderExpired(h *bfttypes.SignedHeader, trustingPeriod time.Duration, now time.Time) bool {
	expirationTime := h.Time.Add(trustingPeriod)
	return !expirationTime.After(now)
}

// VerifyBackwards verifies an untrusted header with a height one less than
// that of an adjacent trusted header. It ensures that:
//
//		a) untrusted header is valid
//	 b) untrusted header has a time before the trusted header
//	 c) that the LastBlockID hash of the trusted header is the same as the hash
//	 of the trusted header
//
//	 For any of these cases ErrInvalidHeader is returned.
func VerifyBackwards(untrustedHeader, trustedHeader *types.Header) error {
	if err := untrustedHeader.ValidateBasic(); err != nil {
		return errorsmod.Wrapf(ErrInvalidHeader, "untrustedHeader.ValidateBasic failed: %v", err)
	}

	if untrustedHeader.ChainID != trustedHeader.ChainID {
		return errorsmod.Wrapf(ErrInvalidHeader, "header belongs to another chain: %v", untrustedHeader.ChainID)
	}

	if !untrustedHeader.Time.Before(trustedHeader.Time) {
		return errorsmod.Wrapf(ErrInvalidHeader, "expected older header time %v to be before new header time %v",
			untrustedHeader.Time,
			trustedHeader.Time)
	}

	if !bytes.Equal(untrustedHeader.Hash(), trustedHeader.LastBlockID.Hash) {
		return errorsmod.Wrapf(ErrInvalidHeader, "expected older header hash %X to match trusted header's last block %X",
			untrustedHeader.Hash(),
			trustedHeader.LastBlockID.Hash)
	}

	return nil
}
