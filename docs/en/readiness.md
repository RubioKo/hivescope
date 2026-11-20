<!-- docs-pair: readiness; revision: 10 -->

[English](readiness.md) | [Español](../es/readiness.md) | [Index](README.md)

# First prototype design and acceptance record

Ana's computer has stopped working. Can she recover one file on another
computer using only her recovery package, its passphrase and the provider
that holds the encrypted copy? Can that same provider refuse the same package
after it has acknowledged revocation and restarted? These two outcomes define
the first desktop prototype.

**Documentary status.** The LAN trial has a bilingual design, public synthetic
reference values and an acceptance procedure. The project has yet to run a
HiveScope client or provider. A documentation check covers the documents and
links. Only observations from a running build can establish a recovery, a
revocation or durability on a particular filesystem.

## Scope and contracts

Use one owner profile, one provider at a known IPv4/TCP address and one clean
replacement profile with its own new device identity. Isolate the profiles so
the replacement cannot access the original database, OS key store, source
file, cached ciphertext or administrative backup. The provider stores one
complete encrypted object and its signed, encrypted catalog. A single provider
can disconnect, lose its disk or return an older snapshot. Ana needs an
independent copy of anything valuable throughout the trial.

| Decision | Where the implementer finds the contract |
| --- | --- |
| Who can write, recover or revoke? | Trusted LAN setup pins the peers and folder authority. The read grant has a generation and a narrow scope. The owner keeps administrative recovery separate from file recovery. [Identity and key custody](identity-custody.md) |
| What does the replacement bring? | One 255-byte password-protected package identifies a snapshot, its provider and the secrets needed for a read. Each catalog or object read needs a fresh proof. [Recovery format and reference signatures](recovery-format.md) |
| Which bytes belong to the file? | The authenticated catalog supplies the object locator, version key and expected length and digest. A private temporary destination holds verified plaintext until publication. [File and catalog format](file-catalog.md) |
| What do the peers exchange? | The versioned direct-LAN protocol specifies bounded messages, deadlines, authentication, grant updates and error handling. [LAN protocol](lan-protocol.md) |
| When may the client say the upload is confirmed? | One reservation covers both sealed items. The client checks the provider's signed receipt against its saved manifest. Repeating the same request returns the recorded outcome. [Upload confirmation](upload-confirmation.md) |
| Which state must survive a restart? | The reference provider orders file and directory flushes, the SQLite commit, acknowledgment and cleanup. Physical admission includes temporary and metadata space. [Provider persistence and interruption cases](provider-persistence.md) |

## Walk through one recovery

1. **Offer space.** The operator selects a local volume, a disk budget and
   availability on the provider. On the controlled LAN, provision the agreement
   and check the public fingerprints separately. Enable the owner's recovery
   grant. Future community discovery can use the same separation of write and
   read authority. [Resource offers](resource-offers.md)
2. **Store a snapshot.** Start with the synthetic file `hive.txt`. The owner
   seals the object and catalog, records the request locally and reserves space
   for both. If the reply disappears, the owner sees **awaiting confirmation**
   and queries the same request. Mark **upload confirmed** only after verifying
   and saving the receipt. The five input bytes produce 21 object bytes and
   250 catalog bytes, plus separate disk overhead.
3. **Prepare for loss.** Explicitly export the snapshot's recovery package.
   Keep the package and passphrase available separately. Turn off the original
   profile. Keep the independent reference bytes with the test observer, outside
   the replacement profile.
4. **Recover.** Select the package and a local destination on the clean device.
   Unlock locally, authenticate the pinned provider, then prove the active
   grant separately for the catalog and object. Verify the catalog, every block,
   length and digest before publishing to the selected destination. Display
   **recovered** only after successful publication. Compare the output with the
   independent reference.
5. **Exercise revocation.** In a separate run from the same starting state,
   sign the revocation with the folder authority, wait for the provider's
   durable acknowledgment, restart it and unlock its profile. Reconnect to the
   pinned provider. For each locator obtain a new challenge and submit a valid
   signature with the old key. Both must return `DENIED` with zero payload bytes. Bytes
   delivered before revocation remain in their recipient's possession.

## Acceptance record for the first implementation

Use synthetic inputs throughout. The identifiers below let a contributor record
each result against a build. At commit `990e836`, M1-01 passed in memory
with the five-byte fixture and Go 1.27.4 on Windows amd64. M1-02 through
M1-10 await a running node.

| ID | Trial | Evidence to retain |
| --- | --- | --- |
| M1-01 | Compare the published package, read transcripts, catalog, object and receipt with a local implementation before networking. | Exact lengths, digests and signature checks agree with the reference values. A substituted block fails verification. |
| M1-02 | Use the active generation 7 package after loss of the original profile. Repeat with an empty file and with a file of 4 MiB + 1 byte. | The replacement receives both items and restores the exact expected bytes. The empty file still authenticates its one block. |
| M1-03 | Acknowledge generation 8 revocation, restart and unlock the provider. Try both old locators independently with fresh, correctly signed challenges. Separately replay a consumed challenge. | The authenticated provider explicitly returns `DENIED` on each old-grant read with zero payload bytes. A replay authorizes no read. The existing five-byte destination stays intact. |
| M1-04 | Try a wrong passphrase, a damaged export and an endpoint presenting a different provider key. | A package authentication failure ends locally. An identity mismatch stops before a read challenge. The destination stays intact. |
| M1-05 | Cut the connection mid-item, then in a separate run lose the final reply after commit. Retry with the same request and with a changed manifest. | An incomplete item restarts at byte zero. The committed request returns the identical receipt, two locator mappings and one charge. The changed manifest gets a conflict. |
| M1-06 | Race two distinct requests by the enrolled uploader for the last space, pause new offers, cancel a reservation and let another expire. | At most one reservation wins the final space. Status survives retry. Occupied space is accounted for until cleanup is durable. |
| M1-07 | Interrupt before and after each publication boundary in the persistence contract. Remove or corrupt a committed file in a disposable copy. | Each restart returns the specified visibility, accounting and receipt. A damaged committed copy reports unavailable. Record process interruptions and storage power-loss trials separately. |
| M1-08 | Use a wrong-scope read, an over-limit frame, a truncated object and a full destination volume. | The peer enforces its limits. No failed verification publishes partial plaintext. Record the preserved destination digest. |
| M1-09 | Lock the OS key store, restart the provider and restore the separate authority backup in an isolated owner profile. | Locked keys prevent the operations that need them. After unlock, identity and highest accepted generation persist. The restored authority can sign an update. The file recovery package alone cannot. |
| M1-10 | Try an elapsed write agreement within its read grace period, then after grace. Disconnect the only provider. | New writes stop at expiry. Eligible reads succeed during grace and stop at closure. The client reports an unreachable provider without claiming the copy is lost or recovered. |

For each run record the case ID, build commit, OS, filesystem, fixture name,
initial state, interruption point, expected and observed outcome, and pass,
fail or unrun. Count payload bytes and committed reservations when those are
the claim. Use synthetic identifiers in reports. Keep passphrases, keys, export
contents, private paths and identifying logs out of the evidence. One successful
read exercises one snapshot. Repeat on each platform before claiming support.

## Reproduce the offline results

From the repository root with Go 1.26 or later, run `go mod download` once,
then `go test -mod=readonly -count=1 ./...`. The tests use synthetic data.
The test run also passes with the module proxy disabled after setup.
It reproduces the 255-byte recovery package, all four signed read examples,
the 21-byte object, the 250-byte signed catalog and the 132-byte receipt.
The restored bytes are `48 69 76 65 0a`. A wrong passphrase, a substituted
block, a changed catalog signature and a receipt checked against an altered
manifest all fail their respective checks. [Object and catalog](../../internal/conformance/snapshot_test.go) · [Recovery and reads](../../internal/conformance/recovery_test.go) · [Receipt](../../internal/conformance/receipt_test.go)

A mathematically valid signature from an old key is part of the reference
example. Rejecting its read after revocation requires a running provider.

The Go reader verifies the signed catalog, authenticates each encrypted block
and checks the object's total length and SHA-256 digest. It stages plaintext
in a temporary file in the caller's private directory. Only a complete,
verified object becomes readable. Closing the reader removes its temporary
file. [Catalog reader](../../internal/storageformat/catalog.go) · [Object reader](../../internal/storageformat/object.go)

The three [published fixtures](file-catalog.md) recover byte for byte offline.
`empty.bin` authenticates one empty block. `hive.txt` yields five bytes.
`edge.bin` yields 4 MiB plus one byte from the second block. If that block is
altered or truncated, the caller receives an error before it can read any
plaintext. M1-02 still requires recovery over the LAN on a clean replacement.

### Save at a chosen destination

The local Go API `SaveObject` targets private directories on Linux filesystems
that support the required rename and flush operations and on local NTFS on Windows.
The caller chooses the destination independently of the catalog's display name.
Create-only mode leaves an existing file intact, including one created during
verification. Replacement requires an explicit choice. The reader verifies
the object in a temporary file in the destination directory, flushes the file,
then publishes it with a single rename. [Local save tests](../../internal/conformance/publication_test.go)

The interruption test stops a child process after it has flushed the verified
temporary file and immediately before the rename. Reopening the destination
returns the exact original bytes. A process killed at this point can leave
the private temporary plaintext file behind. The test exercises process
termination. Physical power loss still requires separate filesystem trials.
[Process interruption and directory flush tests](../../internal/storageformat/publish_test.go)

Successful publication requires a final directory flush. The local Windows
NTFS trial completed that flush. [Linux directory synchronization](https://man7.org/linux/man-pages/man2/fsync.2.html) · [Windows filesystem flush semantics](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-fsa/0de7dc40-9627-437e-a4df-c4696cdc3d02)

If the flush fails after the rename, the
API returns `ErrPublicationUncertain`. The complete new file
may already be visible. The next step is to reconcile that outcome before a
retry. Recover the same authenticated snapshot independently and compare the
destination byte for byte. If they match, flush the file and directory before
confirming. If they differ, keep the destination and request a new path.
[Recovery after an uncertain publication](file-catalog.md)

Automatic discovery and matching belong to milestone 2. Three-provider
redundancy and repair, desktop GUI and mobile access, private compute and
persistent services each have their own scope and evidence. [Roadmap](roadmap.md)
