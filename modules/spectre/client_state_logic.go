package spectre

import (
	errorsmod "cosmossdk.io/errors"
)

// verifySupportedForkAtEpoch enforces that the epoch is in a fork supported
// by this light client (Electra or later).
func (cs *ClientState) verifySupportedForkAtEpoch(epoch uint64) error {
	if epoch < cs.ForkParameters.Electra.Epoch {
		return errorsmod.Wrapf(ErrMustBeElectraOrLater, "epoch %d is before electra epoch %d", epoch, cs.ForkParameters.Electra.Epoch)
	}
	return nil
}

// isGloasEpoch reports whether the epoch is at or after the Gloas fork.
func (cs *ClientState) isGloasEpoch(epoch uint64) bool {
	return epoch >= cs.ForkParameters.Gloas.Epoch
}

// finalizedRootGindexAtSlot returns the generalized index of the finalized
// checkpoint root in the beacon state at the given slot, which depends on the
// fork the slot belongs to.
func (cs *ClientState) finalizedRootGindexAtSlot(slot uint64) (uint64, error) {
	epoch := cs.computeEpochAtSlot(slot)
	if err := cs.verifySupportedForkAtEpoch(epoch); err != nil {
		return 0, err
	}
	if cs.isGloasEpoch(epoch) {
		return finalizedRootGindexGloas, nil
	}
	return finalizedRootGindexElectra, nil
}

// nextSyncCommitteeGindexAtSlot returns the generalized index of the next sync
// committee in the beacon state at the given slot.
func (cs *ClientState) nextSyncCommitteeGindexAtSlot(slot uint64) (uint64, error) {
	epoch := cs.computeEpochAtSlot(slot)
	if err := cs.verifySupportedForkAtEpoch(epoch); err != nil {
		return 0, err
	}
	if cs.isGloasEpoch(epoch) {
		return nextSyncCommitteeGindexGloas, nil
	}
	return nextSyncCommitteeGindexElectra, nil
}

// computeSlotAtTimestamp returns the beacon slot at a wall-clock timestamp,
// or an error if the timestamp precedes genesis.
func (cs *ClientState) computeSlotAtTimestamp(timestampSeconds uint64) (uint64, error) {
	if timestampSeconds < cs.GenesisTime {
		return 0, errorsmod.Wrapf(ErrInvalidHeader, "timestamp %d precedes genesis time %d", timestampSeconds, cs.GenesisTime)
	}
	if cs.SecondsPerSlot == 0 {
		return 0, errorsmod.Wrap(ErrInvalidClientState, "seconds per slot must be non-zero")
	}
	return (timestampSeconds-cs.GenesisTime)/cs.SecondsPerSlot + cs.GenesisSlot, nil
}

// computeEpochAtSlot returns the epoch containing the slot.
func (cs *ClientState) computeEpochAtSlot(slot uint64) uint64 {
	return slot / cs.SlotsPerEpoch
}

// computeSyncCommitteePeriodAtSlot returns the sync committee period
// containing the slot.
func (cs *ClientState) computeSyncCommitteePeriodAtSlot(slot uint64) (uint64, error) {
	if cs.EpochsPerSyncCommitteePeriod == 0 {
		return 0, errorsmod.Wrap(ErrInvalidClientState, "epochs per sync committee period must be non-zero")
	}
	return cs.computeEpochAtSlot(slot) / cs.EpochsPerSyncCommitteePeriod, nil
}

// computeForkVersion returns the fork version active at an epoch, per the
// consensus specs.
func (fp *ForkParameters) computeForkVersion(epoch uint64) []byte {
	switch {
	case epoch >= fp.Gloas.Epoch:
		return fp.Gloas.Version
	case epoch >= fp.Fulu.Epoch:
		return fp.Fulu.Version
	case epoch >= fp.Electra.Epoch:
		return fp.Electra.Version
	case epoch >= fp.Deneb.Epoch:
		return fp.Deneb.Version
	case epoch >= fp.Capella.Epoch:
		return fp.Capella.Version
	case epoch >= fp.Bellatrix.Epoch:
		return fp.Bellatrix.Version
	case epoch >= fp.Altair.Epoch:
		return fp.Altair.Version
	default:
		return fp.GenesisForkVersion
	}
}

// computeDomain derives the 32-byte signature domain for a domain type,
// fork version, and the genesis validators root, per the consensus specs.
func (cs *ClientState) computeDomain(domainType [4]byte, forkVersion []byte) [32]byte {
	root := forkDataRoot(forkVersion, cs.GenesisValidatorsRoot)
	var domain [32]byte
	copy(domain[:4], domainType[:])
	copy(domain[4:], root[:28])
	return domain
}

// forkVersionSlot returns the slot whose epoch selects the fork version for
// a signature created at signatureSlot: max(signatureSlot, 1) - 1.
func forkVersionSlot(signatureSlot uint64) uint64 {
	if signatureSlot == 0 {
		return 0
	}
	return signatureSlot - 1
}

// numParticipants counts the set bits in the sync committee bitfield.
// numParticipants counts the set bits of the bitvector over the first
// committeeSize members.
func (sa *SyncAggregate) numParticipants(committeeSize uint64) uint64 {
	var count uint64
	for i, b := range sa.SyncCommitteeBits {
		for j := range 8 {
			if uint64(i*8+j) >= committeeSize {
				return count
			}
			if b&(1<<uint(j)) != 0 {
				count++
			}
		}
	}
	return count
}

// hasSupermajority reports whether at least 2/3 of the sync committee signed.
// The committee size is the configured one, not the bitfield length: the
// bitvector's padding bits are not members.
func (sa *SyncAggregate) hasSupermajority(committeeSize uint64) bool {
	return sa.numParticipants(committeeSize)*3 >= committeeSize*2
}

// summarize returns the compressed representation of a full sync committee.
func (sc *SyncCommittee) summarize() *SummarizedSyncCommittee {
	pubkeysHash := pubkeyVectorRoot(sc.Pubkeys)
	agg := make([]byte, blsPubkeySize)
	copy(agg, sc.AggregatePubkey)
	return &SummarizedSyncCommittee{
		PubkeysHash:     pubkeysHash[:],
		AggregatePubkey: agg,
	}
}

// summarizeEqual compares two committee summaries by value.
func summarizeEqual(a, b *SummarizedSyncCommittee) bool {
	if a == nil || b == nil {
		return a == b
	}
	return string(a.PubkeysHash) == string(b.PubkeysHash) &&
		string(a.AggregatePubkey) == string(b.AggregatePubkey)
}

// trustedSyncCommittee validates the provided full sync committee against
// the stored consensus state summary: it must match the stored current or
// next committee, have the configured size, and its pubkeys must aggregate
// to the claimed aggregate pubkey.
func trustedSyncCommittee(cs *ClientState, consState *ConsensusState, committee *SyncCommittee, isNext bool) (*SyncCommittee, error) {
	if committee == nil {
		return nil, errorsmod.Wrap(ErrInvalidSyncCommittee, "missing active sync committee")
	}

	// Bound the committee size before any per-key hashing or BLS work.
	if uint64(len(committee.Pubkeys)) != cs.SyncCommitteeSize {
		return nil, errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"committee has %d pubkeys, expected %d", len(committee.Pubkeys), cs.SyncCommitteeSize)
	}
	// Keys are fixed 48-byte values; anything else must not enter hashing or
	// BLS work as a zero-padded encoding.
	for i, pk := range committee.Pubkeys {
		if len(pk) != blsPubkeySize {
			return nil, errorsmod.Wrapf(ErrInvalidSyncCommittee,
				"pubkey %d is %d bytes, expected %d", i, len(pk), blsPubkeySize)
		}
	}
	if len(committee.AggregatePubkey) != blsPubkeySize {
		return nil, errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"aggregate pubkey is %d bytes, expected %d", len(committee.AggregatePubkey), blsPubkeySize)
	}

	summary := committee.summarize()
	if isNext {
		if consState.NextSyncCommittee == nil {
			return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "next sync committee is unknown")
		}
		if !summarizeEqual(summary, consState.NextSyncCommittee) {
			return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "provided committee does not match stored next sync committee")
		}
	} else if !summarizeEqual(summary, &consState.CurrentSyncCommittee) {
		return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "provided committee does not match stored current sync committee")
	}

	// The summary comparison binds the pubkeys hash and the aggregate
	// pubkey to the stored committee, which signature verification already
	// checked when it was stored; the aggregate is not re-derived. Only the
	// participants' keys are ever parsed, in verifyUpdateSignature.
	return committee, nil
}

// isValidLightClientHeader validates a light client header. A header must set
// exactly one of execution (pre-Gloas format) and execution_block_hash (Gloas
// format). A full execution payload header is rejected at or after the Gloas
// fork, since Gloas blocks no longer carry the payload. The execution
// commitment is proven against the beacon body root: the payload header root
// with the execution payload index, or the block hash with the index of the
// header's own epoch (Gloas from the Gloas epoch, otherwise Deneb; every
// supported fork is Electra or later).
func (cs *ClientState) isValidLightClientHeader(header *LightClientHeader) error {
	epoch := cs.computeEpochAtSlot(header.Beacon.Slot)
	if err := cs.verifySupportedForkAtEpoch(epoch); err != nil {
		return err
	}

	// The body root anchors the execution commitment branch: it is a fixed
	// 32-byte hash, and a wrong-length input must be rejected instead of
	// zero-substituted into the zero root.
	if len(header.Beacon.BodyRoot) != 32 {
		return errorsmod.Wrapf(ErrInvalidHeader,
			"beacon body root is %d bytes, expected 32", len(header.Beacon.BodyRoot))
	}

	hasExecution := header.Execution != nil
	hasBlockHash := len(header.ExecutionBlockHash) != 0
	switch {
	case hasExecution && !hasBlockHash:
		if cs.isGloasEpoch(epoch) {
			return errorsmod.Wrap(ErrInvalidHeader, "execution payload header is not allowed from the gloas epoch")
		}
		if err := ensureBranchLength("execution branch", header.ExecutionBranch, executionPayloadGindex); err != nil {
			return err
		}
		branch, err := bytesTo32Array(header.ExecutionBranch)
		if err != nil {
			return err
		}
		return validateMerkleBranch(
			executionPayloadHeaderRoot(header.Execution),
			branch,
			executionPayloadGindex,
			bytes32(header.Beacon.BodyRoot),
		)
	case !hasExecution && hasBlockHash:
		if len(header.ExecutionBlockHash) != 32 {
			return errorsmod.Wrapf(ErrInvalidHeader,
				"execution block hash is %d bytes, expected 32", len(header.ExecutionBlockHash))
		}
		// Every Gloas-format header carries the Gloas-length branch, whatever
		// its own epoch: headers that predate Gloas arrive with leading zero
		// nodes, which isValidNormalizedMerkleBranch checks and skips.
		if err := ensureBranchLength("execution branch", header.ExecutionBranch, executionBlockHashGindexGloas); err != nil {
			return err
		}
		gindex := uint64(executionBlockHashGindexDeneb)
		if cs.isGloasEpoch(epoch) {
			gindex = executionBlockHashGindexGloas
		}
		branch, err := bytesTo32Array(header.ExecutionBranch)
		if err != nil {
			return err
		}
		return isValidNormalizedMerkleBranch(
			bytes32(header.ExecutionBlockHash),
			branch,
			gindex,
			bytes32(header.Beacon.BodyRoot),
		)
	default:
		return errorsmod.Wrap(ErrInvalidHeader, "light client header must set exactly one of execution and execution_block_hash")
	}
}

// validateLightClientUpdate verifies a light client update against the
// trusted consensus state, per the altair/electra light client sync
// protocol: participant counts (minimum and 2/3 supermajority), slot
// ordering, signature period, finality and next-sync-committee branches,
// and the aggregate BLS signature.
func (cs *ClientState) validateLightClientUpdate(
	consState *ConsensusState,
	committee *SyncCommittee,
	update *LightClientUpdate,
	currentSlot uint64,
	isNextCommittee bool,
) error {
	if err := validateUpdateBounds(update); err != nil {
		return err
	}

	// Bound the bitfield before counting participants, and require the
	// Bitvector[SYNC_COMMITTEE_SIZE] padding bits to be zero: set padding is a
	// malformed encoding, and counting it would let fewer real signers pass
	// the supermajority when the committee size is not a multiple of 8.
	bits := update.SyncAggregate.SyncCommitteeBits
	if expectedBitsLen := (cs.SyncCommitteeSize + 7) / 8; uint64(len(bits)) != expectedBitsLen {
		return errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"bitfield length %d does not match committee size %d", len(bits), cs.SyncCommitteeSize)
	}
	// The bitvector is LSB-first: the last byte holds SyncCommitteeSize % 8
	// valid bits in its low positions, and the padding above them must be
	// zero. Shifting the last byte right by the valid-bit count leaves exactly
	// the padding bits.
	if validBits := cs.SyncCommitteeSize % 8; validBits != 0 && bits[len(bits)-1]>>validBits != 0 {
		return errorsmod.Wrap(ErrInvalidSyncCommittee, "bitfield has set padding bits beyond the committee size")
	}

	if sa := update.SyncAggregate; sa.numParticipants(cs.SyncCommitteeSize) < cs.MinSyncCommitteeParticipants {
		return errorsmod.Wrapf(ErrInsufficientParticipants,
			"%d participants is below the minimum %d", sa.numParticipants(cs.SyncCommitteeSize), cs.MinSyncCommitteeParticipants)
	}

	// Require a 2/3 supermajority for every verified update, including
	// misbehaviour reports: two conflicting updates signed by disjoint small
	// subsets are normal beacon chain activity, not misbehaviour.
	if sa := update.SyncAggregate; !sa.hasSupermajority(cs.SyncCommitteeSize) {
		return errorsmod.Wrapf(ErrNotEnoughSignatures,
			"%d of %d participants is below the 2/3 supermajority",
			sa.numParticipants(cs.SyncCommitteeSize), cs.SyncCommitteeSize)
	}

	if err := cs.isValidLightClientHeader(&update.AttestedHeader); err != nil {
		return errorsmod.Wrap(err, "attested header is invalid")
	}

	updatePeriods, err := cs.validateUpdatePeriods(consState, update, currentSlot)
	if err != nil {
		return err
	}

	// The signing committee is bound to the signature period (altair/electra
	// light client sync protocol): a stored-period signature must come from the
	// current committee and a stored+1 signature from the next one. Without
	// this check a retiring committee could sign updates one period past its
	// tenure, or an incoming committee one period early, and install an
	// attacker-chosen committee as the stored next one.
	required, provided := "current", "current"
	if updatePeriods.signature == updatePeriods.stored+1 {
		required = "next"
	}
	if isNextCommittee {
		provided = "next"
	}
	if required != provided {
		return errorsmod.Wrapf(ErrInvalidSignaturePeriod,
			"signature period %d over stored period %d requires the %s committee; the update names the %s committee",
			updatePeriods.signature, updatePeriods.stored, required, provided)
	}

	if err := cs.validateUpdateBranches(consState, update, updatePeriods); err != nil {
		return err
	}

	return cs.verifyUpdateSignature(committee, update)
}

// updatePeriods records the period relations an update must satisfy.
type updatePeriods struct {
	stored              uint64
	signature           uint64
	attested            uint64
	nextKnown           bool
	nextCommitteeUpdate bool
}

// validateUpdatePeriods enforces the slot ordering and sync committee
// period rules of the light client sync protocol.
func (cs *ClientState) validateUpdatePeriods(
	consState *ConsensusState,
	update *LightClientUpdate,
	currentSlot uint64,
) (updatePeriods, error) {
	updateAttestedSlot := update.AttestedHeader.Beacon.Slot
	updateFinalizedSlot := update.FinalizedHeader.Beacon.Slot

	if updateFinalizedSlot == cs.GenesisSlot {
		return updatePeriods{}, errorsmod.Wrap(ErrInvalidHeader, "finalized slot is the genesis slot")
	}

	if currentSlot < update.SignatureSlot {
		return updatePeriods{}, errorsmod.Wrapf(ErrInvalidHeader,
			"current slot %d is behind signature slot %d", currentSlot, update.SignatureSlot)
	}

	if !(update.SignatureSlot > updateAttestedSlot && updateAttestedSlot >= updateFinalizedSlot) {
		return updatePeriods{}, errorsmod.Wrapf(ErrInvalidSlots,
			"invalid slot ordering: signature %d, attested %d, finalized %d",
			update.SignatureSlot, updateAttestedSlot, updateFinalizedSlot)
	}

	storedPeriod, err := cs.computeSyncCommitteePeriodAtSlot(consState.Slot)
	if err != nil {
		return updatePeriods{}, err
	}
	signaturePeriod, err := cs.computeSyncCommitteePeriodAtSlot(update.SignatureSlot)
	if err != nil {
		return updatePeriods{}, err
	}

	nextSyncCommitteeKnown := consState.NextSyncCommittee != nil
	if nextSyncCommitteeKnown {
		if signaturePeriod != storedPeriod && signaturePeriod != storedPeriod+1 {
			return updatePeriods{}, errorsmod.Wrapf(ErrInvalidSignaturePeriod,
				"signature period %d is not %d or %d+1", signaturePeriod, storedPeriod, storedPeriod)
		}
	} else if signaturePeriod != storedPeriod {
		return updatePeriods{}, errorsmod.Wrapf(ErrInvalidSignaturePeriod,
			"signature period %d is not %d while the next sync committee is unknown", signaturePeriod, storedPeriod)
	}

	updateAttestedPeriod, err := cs.computeSyncCommitteePeriodAtSlot(updateAttestedSlot)
	if err != nil {
		return updatePeriods{}, err
	}

	periods := updatePeriods{
		stored:              storedPeriod,
		signature:           signaturePeriod,
		attested:            updateAttestedPeriod,
		nextKnown:           nextSyncCommitteeKnown,
		nextCommitteeUpdate: len(update.NextSyncCommitteeBranch) > 0,
	}

	updateHasNextSyncCommittee := !nextSyncCommitteeKnown && periods.nextCommitteeUpdate && updateAttestedPeriod == storedPeriod
	if updateAttestedSlot <= consState.Slot && !updateHasNextSyncCommittee {
		return updatePeriods{}, errorsmod.Wrapf(ErrIrrelevantUpdate,
			"attested slot %d does not advance trusted slot %d and carries no next sync committee update",
			updateAttestedSlot, consState.Slot)
	}

	return periods, nil
}

// validateUpdateBranches proves the finalized header and, when present, the
// next sync committee from the attested header state root.
func (cs *ClientState) validateUpdateBranches(
	consState *ConsensusState,
	update *LightClientUpdate,
	periods updatePeriods,
) error {
	if err := cs.isValidLightClientHeader(&update.FinalizedHeader); err != nil {
		return errorsmod.Wrap(err, "finalized header is invalid")
	}

	// The attested header's state root anchors the finality and next
	// committee branches. It is a fixed 32-byte hash: a wrong-length input
	// must be rejected instead of zero-substituted into the zero root.
	if err := requireLen32("attested beacon state root", update.AttestedHeader.Beacon.StateRoot); err != nil {
		return err
	}

	// Prove the finalized header root from the attested header state root. The
	// index depends on the fork of the attested slot, and the branch must have
	// exactly its depth.
	finalizedGindex, err := cs.finalizedRootGindexAtSlot(update.AttestedHeader.Beacon.Slot)
	if err != nil {
		return err
	}
	if err := ensureBranchLength("finality branch", update.FinalityBranch, finalizedGindex); err != nil {
		return err
	}
	finalizedRoot := beaconBlockHeaderRoot(&update.FinalizedHeader.Beacon)
	finalityBranch, err := bytesTo32Array(update.FinalityBranch)
	if err != nil {
		return errorsmod.Wrap(err, "finality branch is invalid")
	}
	branch, err := normalizeMerkleBranch(finalityBranch, finalizedGindex)
	if err != nil {
		return errorsmod.Wrap(err, "finality branch is invalid")
	}
	if err := isValidNormalizedMerkleBranch(
		finalizedRoot,
		branch,
		finalizedGindex,
		bytes32(update.AttestedHeader.Beacon.StateRoot),
	); err != nil {
		return errorsmod.Wrap(err, "finality branch verification failed")
	}

	if !periods.nextCommitteeUpdate {
		if update.NextSyncCommittee != nil {
			return errorsmod.Wrap(ErrInvalidSyncCommittee, "unexpected next sync committee")
		}
		return nil
	}

	// Prove the next sync committee from the attested header state root and
	// cross-check it against the stored one.
	if update.NextSyncCommittee == nil {
		return errorsmod.Wrap(ErrInvalidSyncCommittee, "next sync committee branch present but committee missing")
	}
	// Bound the committee size before hashing its pubkeys.
	if uint64(len(update.NextSyncCommittee.Pubkeys)) != cs.SyncCommitteeSize {
		return errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"next sync committee has %d pubkeys, expected %d",
			len(update.NextSyncCommittee.Pubkeys), cs.SyncCommitteeSize)
	}
	// Keys are fixed 48-byte values. Wrong-length keys hash as zero-padded
	// encodings, which can collide with the honest encoding of a key ending
	// in zero bytes; reject them instead of storing such a committee.
	for i, pk := range update.NextSyncCommittee.Pubkeys {
		if len(pk) != blsPubkeySize {
			return errorsmod.Wrapf(ErrInvalidSyncCommittee,
				"next sync committee pubkey %d is %d bytes, expected %d", i, len(pk), blsPubkeySize)
		}
	}
	if len(update.NextSyncCommittee.AggregatePubkey) != blsPubkeySize {
		return errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"next sync committee aggregate pubkey is %d bytes, expected %d",
			len(update.NextSyncCommittee.AggregatePubkey), blsPubkeySize)
	}
	if periods.attested == periods.stored && periods.nextKnown {
		if !summarizeEqual(update.NextSyncCommittee.summarize(), consState.NextSyncCommittee) {
			return errorsmod.Wrap(ErrSyncCommitteeMismatch, "next sync committee does not match the stored one")
		}
	}
	nextGindex, err := cs.nextSyncCommitteeGindexAtSlot(update.AttestedHeader.Beacon.Slot)
	if err != nil {
		return err
	}
	if err := ensureBranchLength("next sync committee branch", update.NextSyncCommitteeBranch, nextGindex); err != nil {
		return err
	}
	nextRoot := syncCommitteeRoot(update.NextSyncCommittee)
	nextCommitteeBranch, err := bytesTo32Array(update.NextSyncCommitteeBranch)
	if err != nil {
		return errorsmod.Wrap(err, "next sync committee branch is invalid")
	}
	branch, err = normalizeMerkleBranch(nextCommitteeBranch, nextGindex)
	if err != nil {
		return errorsmod.Wrap(err, "next sync committee branch is invalid")
	}
	if err := isValidNormalizedMerkleBranch(
		nextRoot,
		branch,
		nextGindex,
		bytes32(update.AttestedHeader.Beacon.StateRoot),
	); err != nil {
		return errorsmod.Wrap(err, "next sync committee branch verification failed")
	}
	return nil
}

// verifyUpdateSignature verifies the sync committee aggregate signature over
// the attested beacon header.
func (cs *ClientState) verifyUpdateSignature(committee *SyncCommittee, update *LightClientUpdate) error {
	var participants [][]byte
	for i, pk := range committee.Pubkeys {
		if update.SyncAggregate.SyncCommitteeBits[i/8]&(1<<uint(i%8)) != 0 {
			participants = append(participants, pk)
		}
	}

	domain := cs.computeDomain(
		domainTypeSyncCommittee,
		cs.ForkParameters.computeForkVersion(cs.computeEpochAtSlot(forkVersionSlot(update.SignatureSlot))),
	)
	signingRoot := signingRoot(beaconBlockHeaderRoot(&update.AttestedHeader.Beacon), domain)

	return fastAggregateVerify(participants, signingRoot, update.SyncAggregate.SyncCommitteeSignature)
}

// domainTypeSyncCommittee is the consensus-spec domain type for sync
// committee signatures.
var domainTypeSyncCommittee = [4]byte{0x07, 0x00, 0x00, 0x00}

// verifyHeader verifies a Header client message against the stored
// consensus state at its trusted slot.
func (cs *ClientState) verifyHeader(consState *ConsensusState, currentTimestampSeconds uint64, header *Header) error {
	// Rejects a Gloas-format finalized header whose execution header is missing
	// or does not hash to the proven block hash, before any signature work.
	if _, err := finalizedExecutionSummary(header); err != nil {
		return err
	}

	if _, err := trustedSyncCommittee(cs, consState, header.ActiveSyncCommittee, header.IsNextCommittee); err != nil {
		return err
	}

	currentSlot, err := cs.computeSlotAtTimestamp(currentTimestampSeconds)
	if err != nil {
		return err
	}

	if err := cs.validateLightClientUpdate(consState, header.ActiveSyncCommittee, &header.ConsensusUpdate, currentSlot, header.IsNextCommittee); err != nil {
		return err
	}

	if header.ConsensusUpdate.FinalizedHeader.Beacon.Slot <= consState.Slot {
		return errorsmod.Wrapf(ErrInvalidHeader,
			"finalized slot %d does not advance trusted slot %d",
			header.ConsensusUpdate.FinalizedHeader.Beacon.Slot, consState.Slot)
	}

	updateFinalizedPeriod, err := cs.computeSyncCommitteePeriodAtSlot(header.ConsensusUpdate.FinalizedHeader.Beacon.Slot)
	if err != nil {
		return err
	}
	storedPeriod, err := cs.computeSyncCommitteePeriodAtSlot(consState.Slot)
	if err != nil {
		return err
	}
	if updateFinalizedPeriod > storedPeriod && len(header.ConsensusUpdate.NextSyncCommitteeBranch) == 0 {
		return errorsmod.Wrap(ErrInvalidHeader, "period change requires a next sync committee update")
	}

	return nil
}

// validateUpdateBounds enforces the byte lengths of every fixed-size field
// before any hashing or BLS work. The SSZ helpers pad or truncate their input
// to the chunk size, so a field one byte longer than its spec length hashes
// like the honest encoding and passes every branch check, while raw uses of
// the field (the stored state root, the misbehaviour conflict comparison)
// still see the longer value: that combination stores an unprovable
// consensus state and forges conflicts out of a single honest signature.
func validateUpdateBounds(update *LightClientUpdate) error {
	const maxExtraDataBytes = 32
	for _, header := range []*LightClientHeader{&update.AttestedHeader, &update.FinalizedHeader} {
		for _, root := range []struct {
			name  string
			value []byte
		}{
			{"beacon parent root", header.Beacon.ParentRoot},
			{"beacon state root", header.Beacon.StateRoot},
			{"beacon body root", header.Beacon.BodyRoot},
		} {
			if len(root.value) != 32 {
				return errorsmod.Wrapf(ErrInvalidHeader, "%s is %d bytes, expected 32", root.name, len(root.value))
			}
		}

		// A Gloas-format header carries no execution payload header, so there
		// are no execution fields to check.
		exec := header.Execution
		if exec == nil {
			continue
		}
		if len(exec.ExtraData) > maxExtraDataBytes {
			return errorsmod.Wrapf(ErrInvalidHeader,
				"execution payload extra_data is %d bytes, maximum %d", len(exec.ExtraData), maxExtraDataBytes)
		}
		for _, field := range []struct {
			name  string
			value []byte
			want  int
		}{
			{"parent_hash", exec.ParentHash, 32},
			{"fee_recipient", exec.FeeRecipient, 20},
			{"state_root", exec.StateRoot, 32},
			{"receipts_root", exec.ReceiptsRoot, 32},
			{"logs_bloom", exec.LogsBloom, 256},
			{"prev_randao", exec.PrevRandao, 32},
			{"base_fee_per_gas", exec.BaseFeePerGas, 32},
			{"block_hash", exec.BlockHash, 32},
			{"transactions_root", exec.TransactionsRoot, 32},
			{"withdrawals_root", exec.WithdrawalsRoot, 32},
		} {
			if len(field.value) != field.want {
				return errorsmod.Wrapf(ErrInvalidHeader,
					"execution payload %s is %d bytes, expected %d", field.name, len(field.value), field.want)
			}
		}
	}
	return nil
}

// verifyMisbehaviour verifies two conflicting light client updates: both
// must pass full verification while finalizing different beacon headers at
// the same finalized slot. Conflicting finality is what an honest chain
// never produces, so it is the one report that must freeze the client.
// Comparing attested headers instead would be too eager: an honest committee
// can sign sibling attested blocks at the same slot without any fault, and a
// freeze on them would stop the bridge until governance recovered it.
// Sync-committee signatures are not slashable, so nothing deters that shape.
//
// The consensus state the evidence is anchored at may be any the client still
// holds, because evidence has to be anchored below its attested slot and the
// client keeps advancing. What the client does not accept is evidence signed
// by a committee it has left: the signature period of both updates must be the
// period of the client's latest slot or a later one. The keys of an earlier
// committee can belong to validators that exited long ago and carry no stake.
// An anchor from the previous period is still enough for evidence signed by
// the committee that took over from it, which the anchor knows as its next
// sync committee.
func (cs *ClientState) verifyMisbehaviour(consState *ConsensusState, currentTimestampSeconds uint64, mis *Misbehaviour) error {
	// Bounds first, as in verifyHeader: nothing below hashes or aggregates
	// anything for an oversized update.
	for i, update := range []*LightClientUpdate{&mis.Update_1, &mis.Update_2} {
		if err := validateUpdateBounds(update); err != nil {
			return errorsmod.Wrapf(err, "update %d", i+1)
		}
	}

	committee, err := trustedSyncCommittee(cs, consState, mis.ActiveSyncCommittee, mis.IsNextCommittee)
	if err != nil {
		return err
	}

	// Same finalized slot, different finalized beacon headers: conflicting
	// finality, which an honest chain never produces. The beacon header root
	// covers any execution difference, in either the pre-Gloas or Gloas
	// format, since the execution commitment is proven against the beacon
	// body root.
	f1 := &mis.Update_1.FinalizedHeader.Beacon
	f2 := &mis.Update_2.FinalizedHeader.Beacon
	if f1.Slot != f2.Slot {
		return errorsmod.Wrapf(ErrInvalidMisbehaviour, "finalized slots differ: %d vs %d", f1.Slot, f2.Slot)
	}
	if beaconBlockHeaderRoot(f1) == beaconBlockHeaderRoot(f2) {
		return errorsmod.Wrap(ErrInvalidMisbehaviour, "finalized headers match: no conflict")
	}

	latestPeriod, err := cs.computeSyncCommitteePeriodAtSlot(cs.LatestSlot)
	if err != nil {
		return err
	}
	for i, update := range []*LightClientUpdate{&mis.Update_1, &mis.Update_2} {
		signaturePeriod, err := cs.computeSyncCommitteePeriodAtSlot(update.SignatureSlot)
		if err != nil {
			return err
		}
		if signaturePeriod < latestPeriod {
			return errorsmod.Wrapf(ErrSignedByPastCommittee,
				"update %d: signature period %d is before the latest period %d", i+1, signaturePeriod, latestPeriod)
		}
	}

	currentSlot, err := cs.computeSlotAtTimestamp(currentTimestampSeconds)
	if err != nil {
		return err
	}

	for i, update := range []*LightClientUpdate{&mis.Update_1, &mis.Update_2} {
		if err := cs.validateLightClientUpdate(consState, committee, update, currentSlot, mis.IsNextCommittee); err != nil {
			return errorsmod.Wrapf(err, "update %d failed verification", i+1)
		}
	}
	return nil
}

// updateConsensusState applies a verified header to the current consensus
// and client state. It returns the new consensus state and, when the update
// advanced the finalized slot, the new client state.
func (cs *ClientState) updateConsensusState(consState *ConsensusState, header *Header) (*ConsensusState, *ClientState, error) {
	if cs.LatestSlot != consState.Slot {
		return nil, nil, errorsmod.Wrapf(ErrClientAndConsensusMismatch,
			"client latest slot %d does not match consensus slot %d", cs.LatestSlot, consState.Slot)
	}

	storeSlot := consState.Slot
	storePeriod, err := cs.computeSyncCommitteePeriodAtSlot(storeSlot)
	if err != nil {
		return nil, nil, err
	}

	updateFinalizedSlot := header.ConsensusUpdate.FinalizedHeader.Beacon.Slot
	updateFinalizedPeriod, err := cs.computeSyncCommitteePeriodAtSlot(updateFinalizedSlot)
	if err != nil {
		return nil, nil, err
	}

	execution, err := finalizedExecutionSummary(header)
	if err != nil {
		return nil, nil, err
	}

	newConsState := &ConsensusState{
		Slot:                 consState.Slot,
		StateRoot:            consState.StateRoot,
		Timestamp:            consState.Timestamp,
		CurrentSyncCommittee: consState.CurrentSyncCommittee,
		NextSyncCommittee:    consState.NextSyncCommittee,
	}

	if consState.NextSyncCommittee != nil {
		// The sync committee only changes when the period changes.
		if updateFinalizedPeriod == storePeriod+1 {
			next := *consState.NextSyncCommittee
			newConsState.CurrentSyncCommittee = next
			if header.ConsensusUpdate.NextSyncCommittee != nil {
				newConsState.NextSyncCommittee = header.ConsensusUpdate.NextSyncCommittee.summarize()
			} else {
				newConsState.NextSyncCommittee = nil
			}
		}
	} else {
		// Without a known next committee, the finalized period cannot
		// advance; unreachable for verified headers (see updateState).
		if updateFinalizedPeriod != storePeriod {
			return nil, nil, errorsmod.Wrapf(ErrInvalidSignaturePeriod,
				"finalized period %d is beyond stored period %d but the next sync committee is unknown",
				updateFinalizedPeriod, storePeriod)
		}
		if header.ConsensusUpdate.NextSyncCommittee != nil {
			newConsState.NextSyncCommittee = header.ConsensusUpdate.NextSyncCommittee.summarize()
		}
	}

	newConsState.Slot = updateFinalizedSlot
	newConsState.StateRoot = execution.stateRoot
	newConsState.Timestamp = execution.timestamp

	var newClientState *ClientState
	if updateFinalizedSlot > consState.Slot {
		updated := cs.Clone()
		updated.LatestSlot = updateFinalizedSlot
		updated.LatestExecutionBlockNumber = execution.blockNumber
		newClientState = updated
	}
	return newConsState, newClientState, nil
}

// Clone returns a deep copy of the client state.
func (cs *ClientState) Clone() *ClientState {
	dup := *cs
	dup.ForkParameters = *cs.ForkParameters.Clone()
	dup.GenesisValidatorsRoot = append([]byte(nil), cs.GenesisValidatorsRoot...)
	dup.IbcContractAddress = append([]byte(nil), cs.IbcContractAddress...)
	dup.IbcCommitmentSlot = append([]byte(nil), cs.IbcCommitmentSlot...)
	return &dup
}

// Clone returns a deep copy of the fork parameters.
func (fp *ForkParameters) Clone() *ForkParameters {
	dup := *fp
	dup.GenesisForkVersion = append([]byte(nil), fp.GenesisForkVersion...)
	dup.Altair = *fp.Altair.Clone()
	dup.Bellatrix = *fp.Bellatrix.Clone()
	dup.Capella = *fp.Capella.Clone()
	dup.Deneb = *fp.Deneb.Clone()
	dup.Electra = *fp.Electra.Clone()
	dup.Fulu = *fp.Fulu.Clone()
	dup.Gloas = *fp.Gloas.Clone()
	return &dup
}

// Clone returns a deep copy of a fork entry.
func (f *Fork) Clone() *Fork {
	dup := *f
	dup.Version = append([]byte(nil), f.Version...)
	return &dup
}

// bytesTo32Array converts a slice of 32-byte values to a [][32]byte slice.
// An entry of any other length is malformed input, not a zero chunk: beacon
// tree branch nodes are fixed 32-byte hashes and nothing re-checks the
// length after conversion.
func bytesTo32Array(values [][]byte) ([][32]byte, error) {
	out := make([][32]byte, 0, len(values))
	for i, v := range values {
		if len(v) != 32 {
			return nil, errorsmod.Wrapf(ErrInvalidMerkleBranch, "node %d is %d bytes, expected 32", i, len(v))
		}
		var chunk [32]byte
		copy(chunk[:], v)
		out = append(out, chunk)
	}
	return out, nil
}
