# Security Policy

All in Bits strives to contribute toward the security of our ecosystem through
internal security practices, and by working with external security researchers
from the community.

## Reporting a Vulnerability

If you've identified a vulnerability, please report it through one of the
following venues:

* Submit an advisory through GitHub: [https://github.com/atomone-hub/atomone/security/advisories/new](https://github.com/atomone-hub/atomone/security/advisories/new)
* Email security [at-symbol] tendermint [dot] com. If you are concerned about
  confidentiality e.g. because of a high-severity issue, you may email us for
  PGP or Signal contact details.
* We provide bug bounty rewards through our program at
  [HackenProof](https://hackenproof.com/all-in-bits). You must report via
  HackenProof in order to be eligible for rewards.

We will respond within 3 business days to all received reports.
Valid submissions from any of the above channels might be eligible for a reward,
so it is up to you to choose the one that best fits your needs.

Thank you for helping to keep our ecosystem safe!

## Security Audits

* March 2025: The security firm Zellic conducted a source code audit of the
  AtomOne daemon and published a
  [report](docs/v2%20-%20Zellic%20Audit%20Report.pdf) on March 11, 2025.
  Zellic has independently published this report
  [here](https://github.com/Zellic/publications/blob/master/AtomOne%20-%20Zellic%20Audit%20Report.pdf)
  with a SHA-256 hash of 60625f148263829921f7b8cc4a065290b197ddb869ba821f7dc4cfe4a4f96ff1.
  The audit scope was the whole codebase with a specific focus on:
  
  * the new `x/photon` module
  * the [dynamic deposit proposal](https://github.com/atomone-hub/atomone/pull/69)
  from the `x/gov` module

  The audit has been carried out in anticipation of and as a prerequisite for the
  [v2.0.0](https://github.com/atomone-hub/atomone/releases/tag/v2.0.0) upgrade.

* July 2025: The security firm Zellic conducted an audit of the AtomOne code
  scheduled to be included in the v3 release and published a
  [report](docs/v3%20-%20Zellic%20Audit%20Report.pdf) on July 4, 2025.
  Zellic has independently published this report
  [here](https://github.com/Zellic/publications/blob/master/All%20in%20Bits%20-%20Zellic%20Audit%20Report.pdf)
  with a SHA-256 hash of 0e35ec86cd73e1a63bb348dd6d3e98eee0fa86bdbfd632ed31b9bbb6606fd67d.
  The audit was mostly focused on:

  * the new `x/dynamicfee` module (renamed from `x/feemarket` as referred to in
    the audit)
  * the addition of the
    [dynamic quorum](https://github.com/atomone-hub/atomone/pull/135) feature
    to the `x/gov` module
  * the addition of the
    [deposit burn with enough *No* votes](https://github.com/atomone-hub/atomone/pull/90)
    to the `x/gov` module
  * the [revision](https://github.com/atomone-hub/atomone/pull/105) of the dynamic
    deposit for proposals that was ultimately not included in the v2 release to
    address some issues discovered in the previous design.

  The audit has been carried out in anticipation of and as a prerequisite for the
  [v3.0.0](https://github.com/atomone-hub/atomone/releases/tag/v3.0.0) upgrade.
