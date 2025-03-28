<!--
parent:
  order: false
layout: home
-->

# AtomOne Documentation

Welcome to the documentation of the **AtomOne application: `atomone`**.

## Testing Chain Upgrade

Chain upgrade is an important procedure that should be tested carefully. This
section aims to provide a guide for testing chain upgrades in AtomOne using a
localnet. 

1. Git checkout the version of AtomOne you want to upgrade from.
2. Update `contrib/localnet/upgrade_proposal.json` with the correct plan name,
   which means the exact `UpgradeName` used to qualify the upgrade in the
   next version. For instance for the v2 upgrade, the plan name is `v2` (see
   the `app/upgrade` folder).
3. Run `make localnet-start` to start a new localnet.
4. Run `make localnet-submit-upgrade-proposal` to submit the upgrade proposal
   and give it enough yes votes for passing the tally.
5. Wait for 5 minutes and run `atomoned --home ~/.atomone-localnet q gov proposals`
   to check that the proposal has passed.
6. Wait for the block height that was registered in the upgrade proposal. Once
   reached the localnet should stop producing blocks, and return an error like:
   ```
   ERR UPGRADE "v2" NEEDED (...)
   ERR CONSENSUS FAILURE!!!
   ```
   This means it is time to upgrade the binary.
7. Stop the `make localnet-start`
8. Git checkout the version of AtomOne you want to upgrade to.
9. Run `make localnet-restart` (/!\ not `localnet-start` which would delete all
   the chain data). Block production should restart.
10. Check that the upgrade procedure has been executed properly.

