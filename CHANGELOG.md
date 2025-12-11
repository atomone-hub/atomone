# CHANGELOG

## Unreleased

*Release date*

### API BREAKING

- Remove all unused code path in x/gov Atom One fork. The module wraps the x/gov Atom One SDK module instead [#248](https://github.com/atomone-hub/atomone/pull/248)

### BUG FIXES

- Use `TruncateInt` to compute `uphotonToMint` [250](https://github.com/atomone-hub/atomone/pull/250)

### DEPENDENCIES

- Upgrade CometBFT to v0.38.19 to fix security issue (ASA-2025-003) [#234](https://github.com/atomone-hub/atomone/pull/234)

### FEATURES

- Migrate x/gov fork from Atom One to Atom One SDK [#248](https://github.com/atomone-hub/atomone/pull/248)

### STATE BREAKING

### IMPROVEMENTS

## v3.0.3

*Oct 20th, 2025*

### DEPENDENCIES

- Upgrade CometBFT to v0.37.16 to fix security issue (ASA-2025-003) [#233](https://github.com/atomone-hub/atomone/pull/233)

## v3.0.2

*Oct 1st, 2025*

### API BREAKING

- CLI: the `photon mint` command now takes the `to_address` as first argument [#222](https://github.com/atomone-hub/atomone/pull/222)

## v3.0.1

*Aug 22th, 2025*

### BUG FIXES

- Fix wrong denom for minDeposit and minInitialDeposit [#205](https://github.com/atomone-hub/atomone/pull/205)

## v3.0.0

*Aug 4th, 2025*

### BUG FIXES

- Handle `maxBlockGas` in ConsensusParam set to 0 or -1 [#180](https://github.com/atomone-hub/atomone/pull/180)
- Remove dependency on `ConsensusParamKeeper` from `x/dynamicfee` [#179](https://github.com/atomone-hub/atomone/pull/179)
- Remove condition returning uninitialized `math.LegacyDec` in `x/gov` [#176](https://github.com/atomone-hub/atomone/pull/176)
- Return zero if max-min <= 0 for certain generated params in `x/gov` simulation [#168](https://github.com/atomone-hub/atomone/pull/168)
- Gracefully handle failure to unpack a `sdk.Msg` in `ProposalKinds` for `x/gov` [#167](https://github.com/atomone-hub/atomone/pull/167)
- Prevent setting `TargetBlockUtilization` to 0 in `x/dynamicfee` [#166](https://github.com/atomone-hub/atomone/pull/166)
- Ensure that quorum caps are consistent (max >= min) [#163](https://github.com/atomone-hub/atomone/pull/163)
- Change `getQuorumAndThreshold` in `x/gov` to return highest quorum and threshold [#161](https://github.com/atomone-hub/atomone/pull/161)
- Add add boundary checks for unified diffs for constitution amendments [#147](https://github.com/atomone-hub/atomone/pull/147)

### DEPENDENCIES

- Remove `statik` dependency in favor of `go:embed` for swagger UI assets [#193](https://github.com/atomone-hub/atomone/pull/193)

### FEATURES

- Add upgrade code to mint photon from 50% of bond denom funds of Community Pool and 90% of Treasury DAO address [#157](https://github.com/atomone-hub/atomone/pull/157) [#189](https://github.com/atomone-hub/atomone/pull/189)
- Make `x/gov` quorum dynamic [#135](https://github.com/atomone-hub/atomone/pull/135)
- Add the `x/dynamicfee` module and use the EIP-15559 AIMD algorithm [#114](https://github.com/atomone-hub/atomone/pull/114) [#170](https://github.com/atomone-hub/atomone/pull/170)
- Make `x/gov` proposals deposits dynamic [#69](https://github.com/atomone-hub/atomone/pull/69)
- Burn proposals deposit if percentage of no votes > `params.BurnDepositNoThreshold` when tallying [#90](https://github.com/atomone-hub/atomone/pull/90)

## v2.0.0

*May 1st, 2025*

### BUG FIXES

- Fix swagger generation [#38](https://github.com/atomone-hub/atomone/pull/38)
- Fix vuln GO-2024-3279 [#60](https://github.com/atomone-hub/atomone/pull/60)
- Fix vuln GO-2024-3112 and GO-2024-2951 [#62](https://github.com/atomone-hub/atomone/pull/62)
- Fix vuln GHSA-8wcc-m6j2-qxvm [#67](https://github.com/atomone-hub/atomone/pull/67)
- (x/gov): Fix proposal converter from v1 to v1beta1 when proposal has no
  messages [#102](https://github.com/atomone-hub/atomone/pull/102)

### FEATURES

- Add the photon module and use photon as the only fee denom [#57](https://github.com/atomone-hub/atomone/pull/57)

### DEPENDENCIES

- Upgrade CometBFT to v0.37.15 to fix securities issues (ASA-2025-001, ASA-2025-002) [#78](https://github.com/atomone-hub/atomone/pull/78)
- Remove `x/crisis` [#93](https://github.com/atomone-hub/atomone/pull/93)
- Upgrade ibc-go to v7.10.0 to fix ASA-2025-004 and ISA-2025-001 [#84](https://github.com/atomone-hub/atomone/pull/84) [#85](https://github.com/atomone-hub/atomone/pull/85) [#98](https://github.com/atomone-hub/atomone/pull/98)
- Upgrade Cosmos SDK to v0.47.17 [#98](https://github.com/atomone-hub/atomone/pull/98)
  
### IMPROVEMENTS

- (x/gov): override MinVotingPeriod with ldflags [#63](https://github.com/atomone-hub/atomone/pull/63)
- (CLI): backport `tx simulate` from cosmos-sdk v0.50.x and remove unused/misleading flags from `tx broadcast` [#109](https://github.com/atomone-hub/atomone/pull/109)

## v1.0.0

*Sep 26th, 2024*

### FEATURES

- Fork the gaia codebase and renamed it to AtomOne [d49b8634](https://github.com/atomone-hub/atomone/commit/d49b86344c3ee42f5182278601c6ce2bd1eff48e)
- Remove ICS [9e75d78b](https://github.com/atomone-hub/atomone/commit/9e75d78bd6adc490acee869ac98217a1623a9c6d) [#29](https://github.com/atomone-hub/atomone/pull/29)
- Remove x/metaprotocols module [8cc9a025](https://github.com/atomone-hub/atomone/commit/8cc9a02587c96f819d346673e40b4b683f3c0f5b)
- Add reproducible builds [707f1426](https://github.com/atomone-hub/atomone/commit/707f142613794e1fc8dc6371390d003f9245a457) [#24](https://github.com/atomone-hub/atomone/pull/24)
- Fork x/gov [ca0724f0](https://github.com/atomone-hub/atomone/commit/ca0724f036f077ffd3b2efc2a43db2ed98ad885e)
- Disable validator vote inheritance [#5](https://github.com/atomone-hub/atomone/pull/5)
- Remove usage of v1.ValidatorGovInfo [#7](https://github.com/atomone-hub/atomone/pull/7)
- Add x/gov/types [#8](https://github.com/atomone-hub/atomone/pull/8)
- Remove No With Veto, and set default pass threshold to 2/3 and quorum to 25% [#9](https://github.com/atomone-hub/atomone/pull/9)
- Change min voting period to 3 weeks [#10](https://github.com/atomone-hub/atomone/pull/10)
- Add constitution amendment and law proposals with specific quorum and pass thresholds [#11](https://github.com/atomone-hub/atomone/pull/11) [#20](https://github.com/atomone-hub/atomone/pull/20) [#25](https://github.com/atomone-hub/atomone/pull/25)
- Add late quorum voting period extension [#13](https://github.com/atomone-hub/atomone/pull/12)
- Add a minimum amount per deposit [#13](https://github.com/atomone-hub/atomone/pull/13)
- Add constitution to x/gov [#15](https://github.com/atomone-hub/atomone/pull/15)
