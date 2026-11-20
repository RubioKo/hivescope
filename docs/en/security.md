<!-- docs-pair: security; revision: 3 -->

[English](security.md) | [Español](../es/security.md) | [Index](README.md)

# Security and threat model

HiveScope is in design. The security properties below are requirements for
future implementations. The design awaits cryptographic review. Keep an
independent copy of any file used in a future pilot.

## Trust boundaries

| Party | Access by design | Remaining risk |
| --- | --- | --- |
| Owner-authorized device | Folder plaintext, keys and permitted edits. | Compromise exposes those folders and can publish harmful changes. |
| Storage provider | Agreed operations on opaque objects, ciphertext and bounded maintenance metadata. | It can drop data, misreport capacity, collude or observe traffic. |
| Delegated repair worker | Limited reconstruction and replacement of ciphertext. | It can waste allowed resources or fail to repair a fragment. |
| Community directory | Expiring service advertisements with peer contact details. | It can omit offers, retain contact metadata or return stale candidates. |
| Relay | Connection metadata and transport of encrypted traffic. | It can observe peers, delay traffic or deny service. |
| Compute host | Physical machine, offered resources, timing and traffic volume. | It can pause or cut power. Protection of session contents depends on the verified environment. |
| Requester | Its own session, programs and results within the agreed resources. | A workload can exhaust its allocation. The host needs isolation from the guest and enforceable limits. |
| Maintainer | Repository review and release administration. | A compromised distribution channel can ship malicious software. |

Anyone can request capacity. A device identity authenticates one peer, and one
person may control several identities. Each host must cap total disk, bandwidth,
reservations, streams and concurrent work as well as per-requester use. Account
for temporary files and unfinished reservations. Test simultaneous requests,
repeated IDs and reconnect bursts. An advertisement grants no access to folders
or sessions. The node must authenticate the peer and authorize each operation.

## Required storage defenses

- Use established cryptographic and transport libraries. Keep folder keys,
  device identity keys, recovery grants and repair privileges separate.
- Encrypt names, paths and catalogs. Verify ciphertext commitments and
  authenticated encryption before publishing a recovered file.
- Bound parsing and queues. Reject unauthorized, replayed, malformed and
  oversized requests before allocating substantial resources.
- Keep writes inside the configured storage roots. Treat remote paths and
  temporary entries as untrusted. Clean up only entries owned by the interrupted operation.
- Protect local secrets with explicit unlock and encryption at rest. Recovery
  exports require a reviewed passphrase-based format and an explicit owner action.
- Persist authorization epochs and remembered version heads to limit rollback.
  A stale export and an outdated provider can still conceal a newer version.
  A provider that acknowledged revocation must retain that decision over restart.

## What storage can protect

File contents and names are encrypted on the owner's device. Providers can
observe IP addresses, sizes, timing, placement relationships and repair activity.
A signed receipt records an obligation at a point in time. Recovery still needs
valid data, catalog, keys and a reachable provider with a valid read grant.
Simultaneous household outages, expired agreements or lost keys can prevent
recovery. Plaintext already read on a compromised owner device may remain there.

## Proposed decision for private compute

The requester controls what to install and run inside its guest. The host must
isolate the guest from its own files, credentials, other guests and the hardware
outside the allocation. Resource limits govern consumption. Protecting private
data in use also requires a verifiable execution environment.

Milestone 5 will evaluate a confidential VM with hardware-backed attestation.
Before releasing a session key or private input, the requester verifies fresh,
signed evidence against its accepted measurements and platform policy. The
evidence must identify the intended guest and bind its channel key to this
session. Deliver secrets only through that channel. A GPU request must also
verify the accelerator and data path. Missing, stale, failed or mismatched
evidence leaves the request pending and the keys with the requester. Every new
host, including a game standby, must pass verification before receiving session
secrets. [Confidential VM isolation and key release](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-vm-overview) · [CPU and GPU trust boundaries](https://docs.nvidia.com/confidential-computing/virtual-machines/threat-model)

Only data the requester chooses for the guest enters the session. Folder keys
and recovery grants stay separate. Encrypt saved state and results before they
leave the guest for community storage. Logs, crash reports and directory entries
must exclude payloads, private paths and keys. Software inside the guest can use
or disclose the inputs given to it. An attested host may still disconnect or
withhold a result. Publish the supported hardware, trusted components and
remaining risks with the measurements before offering the service.

**Future acceptance tests.** Run a requester-supplied program with synthetic
input after successful verification. Check that an unsupported host, replayed
evidence, a different guest key or a missing required GPU proof yields zero
session keys and zero private input at that host, including after retries.
Pause an active session and verify that resource use stops as agreed. Report
only the last confirmed checkpoint. Try to resume on another host and require
fresh evidence before restoring state. Inspect logs and crash reports for the
synthetic marker, private paths and secrets.

## Reporting a concern

Use [GitHub private vulnerability reporting](https://github.com/RubioKo/hivescope/security/advisories/new)
for sensitive findings. Include the document or future version, a synthetic
reproducer, expected behavior and impact. Keep real files, credentials, recovery
exports and personal logs private. English and Spanish are accepted.

Use public [Issues](https://github.com/RubioKo/hivescope/issues) for non-sensitive
design questions. Maintainers will assess impact, coordinate fixes and document
verified outcomes. Response times and a bounty program have not been established.
