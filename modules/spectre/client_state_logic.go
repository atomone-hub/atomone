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
func (sa *SyncAggregate) numParticipants() uint64 {
	var count uint64
	for _, b := range sa.SyncCommitteeBits {
		for i := 0; i < 8; i++ {
			if b&(1<<uint(i)) != 0 {
				count++
			}
		}
	}
	return count
}

// syncCommitteeSize returns the committee size implied by the bitfield
// length.
func (sa *SyncAggregate) syncCommitteeSize() uint64 {
	return uint64(len(sa.SyncCommitteeBits)) * 8
}

// hasSupermajority reports whether at least 2/3 of the sync committee signed.
func (sa *SyncAggregate) hasSupermajority() bool {
	return sa.numParticipants()*3 >= sa.syncCommitteeSize()*2
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

	summary := committee.summarize()
	if isNext {
		if consState.NextSyncCommittee == nil {
			return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "next sync committee is unknown")
		}
		if !summarizeEqual(summary, consState.NextSyncCommittee) {
			return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "provided committee does not match stored next sync committee")
		}
	} else {
		if !summarizeEqual(summary, &consState.CurrentSyncCommittee) {
			return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "provided committee does not match stored current sync committee")
		}
	}

	agg, err := aggregatePubkeys(committee.Pubkeys)
	if err != nil {
		return nil, err
	}
	var claimed [48]byte
	copy(claimed[:], committee.AggregatePubkey)
	if agg != claimed {
		return nil, errorsmod.Wrap(ErrSyncCommitteeMismatch, "aggregate pubkey does not match the sum of pubkeys")
	}

	return committee, nil
}

// isValidLightClientHeader validates a light client header: the execution
// payload header must be proven from the beacon body root via the execution
// branch, and both headers must be in a supported fork.
func (cs *ClientState) isValidLightClientHeader(header *LightClientHeader) error {
	if err := cs.verifySupportedForkAtEpoch(cs.computeEpochAtSlot(header.Beacon.Slot)); err != nil {
		return err
	}
	if len(header.ExecutionBranch) != floorLog2(executionPayloadGindex) {
		return errorsmod.Wrapf(ErrInvalidMerkleBranch,
			"execution branch has %d nodes, expected %d", len(header.ExecutionBranch), floorLog2(executionPayloadGindex))
	}
	leaf := executionPayloadHeaderRoot(&header.Execution)
	return validateMerkleBranch(
		leaf,
		bytesTo32Array(header.ExecutionBranch),
		executionPayloadGindex,
		bytes32(header.Beacon.BodyRoot),
	)
}

// validateLightClientUpdate verifies a light client update against the
// trusted consensus state, per the altair/electra light client sync
// protocol: participant counts, slot ordering, signature period, finality
// and next-sync-committee branches, and the aggregate BLS signature.
func (cs *ClientState) validateLightClientUpdate(
	consState *ConsensusState,
	committee *SyncCommittee,
	update *LightClientUpdate,
	currentSlot uint64,
) error {
	// Bound the bitfield before counting participants.
	if expectedBitsLen := (cs.SyncCommitteeSize + 7) / 8; uint64(len(update.SyncAggregate.SyncCommitteeBits)) != expectedBitsLen {
		return errorsmod.Wrapf(ErrInvalidSyncCommittee,
			"bitfield length %d implies committee size %d, expected %d",
			len(update.SyncAggregate.SyncCommitteeBits), update.SyncAggregate.syncCommitteeSize(), cs.SyncCommitteeSize)
	}

	if sa := update.SyncAggregate; sa.numParticipants() < cs.MinSyncCommitteeParticipants {
		return errorsmod.Wrapf(ErrInsufficientParticipants,
			"%d participants is below the minimum %d", sa.numParticipants(), cs.MinSyncCommitteeParticipants)
	}

	if err := cs.isValidLightClientHeader(&update.AttestedHeader); err != nil {
		return errorsmod.Wrap(err, "attested header is invalid")
	}

	updatePeriods, err := cs.validateUpdatePeriods(consState, update, currentSlot)
	if err != nil {
		return err
	}

	if err := cs.validateUpdateBranches(consState, update, updatePeriods); err != nil {
		return err
	}

	return cs.verifyUpdateSignature(committee, update)
}

// updatePeriods records the period relations an update must satisfy.
type updatePeriods struct {
	stored              uint64
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

	// Prove the finalized header root from the attested header state root.
	finalizedRoot := beaconBlockHeaderRoot(&update.FinalizedHeader.Beacon)
	branch, err := normalizeMerkleBranch(bytesTo32Array(update.FinalityBranch), finalizedRootGindex)
	if err != nil {
		return errorsmod.Wrap(err, "finality branch is invalid")
	}
	if err := isValidNormalizedMerkleBranch(
		finalizedRoot,
		branch,
		finalizedRootGindex,
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
	if periods.attested == periods.stored && periods.nextKnown {
		if !summarizeEqual(update.NextSyncCommittee.summarize(), consState.NextSyncCommittee) {
			return errorsmod.Wrap(ErrSyncCommitteeMismatch, "next sync committee does not match the stored one")
		}
	}
	nextRoot := syncCommitteeRoot(update.NextSyncCommittee)
	branch, err = normalizeMerkleBranch(bytesTo32Array(update.NextSyncCommitteeBranch), nextSyncCommitteeGindex)
	if err != nil {
		return errorsmod.Wrap(err, "next sync committee branch is invalid")
	}
	if err := isValidNormalizedMerkleBranch(
		nextRoot,
		branch,
		nextSyncCommitteeGindex,
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
	if err := validateUpdateBounds(&header.ConsensusUpdate); err != nil {
		return err
	}

	if _, err := trustedSyncCommittee(cs, consState, header.ActiveSyncCommittee, header.IsNextCommittee); err != nil {
		return err
	}

	currentSlot, err := cs.computeSlotAtTimestamp(currentTimestampSeconds)
	if err != nil {
		return err
	}

	if err := cs.validateLightClientUpdate(consState, header.ActiveSyncCommittee, &header.ConsensusUpdate, currentSlot); err != nil {
		return err
	}

	if !header.ConsensusUpdate.SyncAggregate.hasSupermajority() {
		return errorsmod.Wrapf(ErrNotEnoughSignatures,
			"%d of %d participants is below the 2/3 supermajority",
			header.ConsensusUpdate.SyncAggregate.numParticipants(),
			header.ConsensusUpdate.SyncAggregate.syncCommitteeSize())
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

// validateUpdateBounds enforces variable-size limits before any hashing or
// BLS work.
func validateUpdateBounds(update *LightClientUpdate) error {
	const maxExtraDataBytes = 32
	for _, extraData := range [][]byte{
		update.AttestedHeader.Execution.ExtraData,
		update.FinalizedHeader.Execution.ExtraData,
	} {
		if len(extraData) > maxExtraDataBytes {
			return errorsmod.Wrapf(ErrInvalidHeader,
				"execution payload extra_data is %d bytes, maximum %d", len(extraData), maxExtraDataBytes)
		}
	}
	return nil
}

// verifyMisbehaviour verifies two conflicting light client updates for the
// same attested slot. Both must pass full verification while attesting
// different execution state roots.
func (cs *ClientState) verifyMisbehaviour(consState *ConsensusState, currentTimestampSeconds uint64, mis *Misbehaviour) error {
	committee, err := trustedSyncCommittee(cs, consState, mis.ActiveSyncCommittee, mis.IsNextCommittee)
	if err != nil {
		return err
	}

	slot1 := mis.Update_1.AttestedHeader.Beacon.Slot
	slot2 := mis.Update_2.AttestedHeader.Beacon.Slot
	if slot1 != slot2 {
		return errorsmod.Wrapf(ErrInvalidMisbehaviour, "attested slots differ: %d vs %d", slot1, slot2)
	}

	root1 := bytes32(mis.Update_1.AttestedHeader.Execution.StateRoot)
	root2 := bytes32(mis.Update_2.AttestedHeader.Execution.StateRoot)
	if root1 == root2 {
		return errorsmod.Wrap(ErrInvalidMisbehaviour, "attested state roots match: no conflict")
	}

	currentSlot, err := cs.computeSlotAtTimestamp(currentTimestampSeconds)
	if err != nil {
		return err
	}

	for i, update := range []*LightClientUpdate{&mis.Update_1, &mis.Update_2} {
		if err := validateUpdateBounds(update); err != nil {
			return errorsmod.Wrapf(err, "update %d", i+1)
		}
		if err := cs.validateLightClientUpdate(consState, committee, update, currentSlot); err != nil {
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
		// advance.
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
	newConsState.StateRoot = header.ConsensusUpdate.FinalizedHeader.Execution.StateRoot
	newConsState.Timestamp = header.ConsensusUpdate.FinalizedHeader.Execution.Timestamp

	var newClientState *ClientState
	if updateFinalizedSlot > consState.Slot {
		updated := cs.Clone()
		updated.LatestSlot = updateFinalizedSlot
		updated.LatestExecutionBlockNumber = header.ConsensusUpdate.FinalizedHeader.Execution.BlockNumber
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
	return &dup
}

// Clone returns a deep copy of a fork entry.
func (f *Fork) Clone() *Fork {
	dup := *f
	dup.Version = append([]byte(nil), f.Version...)
	return &dup
}

// bytesTo32Array converts a slice of 32-byte values to a [][32]byte slice,
// skipping entries of the wrong length.
func bytesTo32Array(values [][]byte) [][32]byte {
	out := make([][32]byte, 0, len(values))
	for _, v := range values {
		var chunk [32]byte
		if len(v) == 32 {
			copy(chunk[:], v)
		}
		out = append(out, chunk)
	}
	return out
}
