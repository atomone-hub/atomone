# CHANGELOG

## Unreleased

*Release date*

### API BREAKING

### BUG FIXES

### DEPENDENCIES

### FEATURES

### STATE BREAKING

### IMPROVEMENTS

## v2.0.0

*Release date*

### BUG FIXES

- Fix swagger generation [#38](https://github.com/atomone-hub/atomone/pull/38)
- Fix vuln GO-2024-3279 [#60](https://github.com/atomone-hub/atomone/pull/60)
- Fix vuln GO-2024-3112 and GO-2024-2951 [#62](https://github.com/atomone-hub/atomone/pull/62)
- Fix vuln GHSA-8wcc-m6j2-qxvm [#67](https://github.com/atomone-hub/atomone/pull/67)

### FEATURES

- Add the photon module and use photon as the only fee denom [#57](https://github.com/atomone-hub/atomone/pull/57)

### DEPENDENCIES

- Upgrade CometBFT to v0.37.15 to fix securities issues (ASA-2025-001, ASA-2025-002) [#78](https://github.com/atomone-hub/atomone/pull/78)
- Upgrade ibc-go to v7.9.2 to fix ASA-2025-004 [#84](https://github.com/atomone-hub/atomone/pull/84) [#85](https://github.com/atomone-hub/atomone/pull/85)
- Remove x/crisis [#93](https://github.com/atomone-hub/atomone/pull/93)

### IMPROVEMENTS

- (x/gov): override MinVotingPeriod with ldflags [#63](https://github.com/atomone-hub/atomone/pull/63)

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
