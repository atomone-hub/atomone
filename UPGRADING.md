# Upgrading AtomOne

This guide provides instructions for upgrading AtomOne from v3.x to v4.x.

This document describes the steps for validators and full node operators, to
upgrade successfully for the AtomOne v4 release.

For more details on the release, please see the [release notes][v4].

> [!IMPORTANT]
> This release upgrades to Cosmos SDK v0.50 and IBC v10. It also introduces the
> `x/coredaos` module and migrates internal governance state to use the
> Cosmos SDK gov keeper. The `atomone.gov.v1` proto paths are preserved via the
> `x/gov` wrapper module, so external clients and queries are unaffected.

## Release Binary

Please use the correct release binary: `v4.0.0`.

## Go version

AtomOne v4 build requires Go compiler version 1.24.5. If you already have go
installed but with another version, you can install go1.24.5 with the following
command:

```sh
$ go install golang.org/dl/go1.24.5@latest
$ go1.24.5 download
```

Then you need to update some env variables to invoke the makefile commands of
AtomOne. For example, to run `make build` :
```
$ GOROOT=$(go1.24.5 env GOROOT) PATH=$GOROOT/bin:$PATH make build
```

## Instructions

- [Upgrading AtomOne](#upgrading-atomone)
  - [Release Binary](#release-binary)
  - [Go version](#go-version)
  - [Instructions](#instructions)
  - [On-chain governance proposal attains consensus](#on-chain-governance-proposal-attains-consensus)
  - [Upgrade date](#upgrade-date)
  - [Preparing for the upgrade](#preparing-for-the-upgrade)
    - [Backups](#backups)
    - [Current runtime](#current-runtime)
    - [Target runtime](#target-runtime)
  - [Upgrade steps](#upgrade-steps)
    - [Method I: Manual Upgrade](#method-i-manual-upgrade)
    - [Method II: Upgrade using Cosmovisor](#method-ii-upgrade-using-cosmovisor)
      - [Manually preparing the binary](#manually-preparing-the-binary)
        - [Preparation](#preparation)
      - [Auto-Downloading the AtomOne binary](#auto-downloading-the-atomone-binary)
  - [Expected upgrade result](#expected-upgrade-result)
  - [Upgrade duration](#upgrade-duration)
  - [Rollback plan](#rollback-plan)
  - [Communications](#communications)
  - [Risks](#risks)

## On-chain governance proposal attains consensus

Once a software upgrade governance proposal is submitted to the AtomOne chain,
both a reference to this proposal and an `UPGRADE_HEIGHT` are added to the
[release notes][v4].
If and when this proposal reaches consensus, the upgrade height will be used to
halt the "old" chain binaries. You can check the proposal on one of the block
explorers or using the `atomoned` CLI tool.
Neither core developers nor core funding entities control the governance.

## Upgrade date

The date/time of the upgrade is subject to change as blocks are not generated
at a constant interval. You can stay up-to-date by checking the estimated
time until the block is produced on one of the block explorers.

## Preparing for the upgrade

### Backups

Prior to the upgrade, validators are encouraged to take a full data snapshot.
Snapshotting depends heavily on infrastructure, but generally this can be done
by backing up the `.atomone` directory.
If you use Cosmovisor to upgrade, by default, Cosmovisor will backup your data
upon upgrade. See below [upgrade using cosmovisor](#method-ii-upgrade-using-cosmovisor)
section.

It is critically important for validator operators to back-up the
`.atomone/data/priv_validator_state.json` file after stopping the atomoned
process. This file is updated every block as your validator participates in
consensus rounds. It is a critical file needed to prevent double-signing, in
case the upgrade fails and the previous chain needs to be restarted.

### Current runtime

The AtomOne mainnet network, `atomone-1`, is currently running [AtomOne
v3.3.0][v3]. We anticipate that operators who are running on v3.3.0, will be
able to upgrade successfully. Validators are expected to ensure that their
systems are up to date and capable of performing the upgrade. This includes
running the correct binary and if building from source, building with the
appropriate `go` version.

### Target runtime

The AtomOne mainnet network, `atomone-1`, will run **[AtomOne v4.0.0][v4]**.
Operators _**MUST**_ use this version post-upgrade to remain connected to the
network. The new version requires `go v1.24.5` to build successfully.

## Upgrade steps

There are 2 major ways to upgrade a node:

- Manual upgrade
- Upgrade using [Cosmovisor](https://pkg.go.dev/cosmossdk.io/tools/cosmovisor)
    - Either by manually preparing the new binary
    - Or by using the auto-download functionality (this is not yet recommended)

If you prefer to use Cosmovisor to upgrade, some preparation work is needed
before upgrade.

### Method I: Manual Upgrade

Make sure **AtomOne v3.3.0** is installed by either downloading a [compatible
binary][v3], or building from source. Check the required version to build this
binary in the `Makefile`.

Run AtomOne v3.3.0 till upgrade height, the node will panic:

```shell
ERR UPGRADE "v4" NEEDED at height: <UPGRADE_HEIGHT>: upgrade to v4 and applying upgrade "v4" at height:<UPGRADE_HEIGHT>
```

Stop the node, and switch the binary to **AtomOne v4.0.0** and re-start by
`atomoned start`.

It may take several minutes to a few hours until validators with a total sum
voting power > 2/3 to complete their node upgrades. After that, the chain can
continue to produce blocks.

### Method II: Upgrade using Cosmovisor

#### Manually preparing the binary

##### Preparation

- Install the latest version of Cosmovisor:

```shell
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest
```

- Create a `cosmovisor` folder inside `$ATOMONE_HOME` and move AtomOne `v3.3.0`
into `$ATOMONE_HOME/cosmovisor/genesis/bin`:

```shell
mkdir -p $ATOMONE_HOME/cosmovisor/genesis/bin
cp $(which atomoned) $ATOMONE_HOME/cosmovisor/genesis/bin
```

- Build AtomOne `v4.0.0`, and move atomoned `v4.0.0` to
  `$ATOMONE_HOME/cosmovisor/upgrades/v4/bin`

```shell
mkdir -p  $ATOMONE_HOME/cosmovisor/upgrades/v4/bin
cp $(which atomoned) $ATOMONE_HOME/cosmovisor/upgrades/v4/bin
```

At this moment, you should have the following structure:

```shell
.
├── current -> genesis or upgrades/<name>
├── genesis
│   └── bin
│       └── atomoned  # old: v3.3.0
└── upgrades
    └── v4
        └── bin
            └── atomoned  # new: v4.0.0
```

- Export the environmental variables:

```shell
export DAEMON_NAME=atomoned
# please change to your own atomone home dir
# please note `DAEMON_HOME` has to be absolute path
export DAEMON_HOME=$ATOMONE_HOME
export DAEMON_RESTART_AFTER_UPGRADE=true
```

- Start the node:

```shell
cosmovisor run start --home $DAEMON_HOME
```

#### Auto-Downloading the AtomOne binary

**This method is not recommended!**

## Expected upgrade result

When the upgrade block height is reached, AtomOne will panic and stop:

This may take a few minutes to a few hours.
After upgrade, the chain will continue to produce blocks when validators with a
total sum voting power > 2/3 complete their node upgrades.

## Upgrade duration

The upgrade may take a few minutes to several hours to complete because
atomone-1 participants operate globally with differing operating hours and it
may take some time for operators to upgrade their binaries and connect to the
network.

## Rollback plan

During the network upgrade, core AtomOne teams will be keeping an ever vigilant
eye and communicating with operators on the status of their upgrades. During
this time, the core teams will listen to operator needs to determine if the
upgrade is experiencing unintended challenges. In the event of unexpected
challenges, the core teams, after conferring with operators and attaining
social consensus, may choose to declare that the upgrade will be skipped.

Steps to skip this upgrade proposal are simply to resume the `atomone-1`
network with the (downgraded) v3.3.0 binary using the following command:

```shell
atomoned start --unsafe-skip-upgrade <UPGRADE_HEIGHT>
```

Note: There is no particular need to restore a state snapshot prior to the
upgrade height, unless specifically directed by core AtomOne teams.

Important: A social consensus decision to skip the upgrade will be based solely
on technical merits, thereby respecting and maintaining the decentralized
governance process of the upgrade proposal's successful YES vote.

## Communications

Operators are encouraged to join the `#validate-private` channel
of the AtomOne Community Discord. This channel is the primary communication
tool for operators to ask questions, report upgrade status, report technical
issues, and to build social consensus should the need arise. This channel is
restricted to known operators and requires verification beforehand. Requests to
join the `#validator-private` channel can be sent to the `#support` channel.

## Risks

As a validator performing the upgrade procedure on your consensus nodes carries
a heightened risk of double-signing and being slashed. The most important piece
of this procedure is verifying your software version and genesis file hash
before starting your validator and signing.

The riskiest thing a validator can do is discover that they made a mistake and
repeat the upgrade procedure again during the network startup. If you discover
a mistake in the process, the best thing to do is wait for the network to start
before correcting it.

[v3]: https://github.com/atomone-hub/atomone/releases/tag/v3.3.0
[v4]: https://github.com/atomone-hub/atomone/releases/tag/v4.0.0
