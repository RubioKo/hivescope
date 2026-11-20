<!-- docs-pair: identity-custody; revision: 1 -->

[English](identity-custody.md) | [Español](../es/identity-custody.md) | [Index](README.md)

# Who can store, recover and revoke a copy

If Ana loses her laptop, her recovery package and its passphrase let a new device
read one folder from the agreed provider. A separate backup lets her replace or
revoke that reading credential. The following decisions specify the desktop LAN
trial. The native unlock and restart behaviors still need to be exercised on hardware.

## Establish the agreement once

For the controlled trial, the operator transfers the **public** enrollment data
between the two local profiles using a trusted offline channel. Compare the full
SHA-256 fingerprint of each raw 32-byte public key against the display on the
originating device. The provider pins the folder authority and uploader. The
owner pins that authority and the provider. Neither party learns a pin from the
first network connection. A different provider at the old IP fails authentication.

Both profiles record the same random 16-byte folder ID, new random 16-byte
agreement ID, public pins, start and expiry (UTC Unix seconds), physical quota
and transfer limits. The provider's local profile must admit the terms. A new
agreement starts with an explicit empty grant and denies all reads. Reopening
an existing agreement requires its durable state. A conflicting import or a
previously closed ID cannot initialize it again. The address is a connection
hint. Public discovery and automatic negotiation arrive in milestone 2.
[Resource offers](resource-offers.md) · [Storage terms](storage.md)

| Credential | Holder and permitted operation |
| --- | --- |
| Provider device identity | The provider authenticates its connection and signs storage receipts. |
| Uploader device identity | The enrolled uploader reserves space, writes and queries its own requests under the agreement. A clean replacement creates its own device identity. |
| Folder authority | The owner signs recovery grant changes. The provider stores only its public key. The authority alone cannot decrypt a folder. |
| Recovery signing key | The owner and a device that unlocks the recovery package can prove the registered right to fetch ciphertext. |
| Folder read key and catalog keys | The owner generates the folder read secret and the catalog signing key separately. The recovery package carries the read secret and catalog **verification** key. Per-version content keys are inside the authenticated, encrypted catalog. |

Generate independent Ed25519 seeds and encryption keys with the OS cryptographic
random source. Store private keys only on the device performing their role or in
their designated export. Provider, uploader, recovery and authority identities
are never interchangeable. A change of authority or uploader requires a new
agreement in this trial. [Recovery package](recovery-format.md) · [File catalog](file-catalog.md)

## Issue, replace or revoke the recovery grant

The owner signs exactly 134 bytes using pure Ed25519 and appends the raw 64-byte
signature. The resulting 198-byte `GRANT_UPDATE` is one record. Concatenate the
fields below without separators, padding or lengths. Offsets are zero-based.
[Ed25519 signature format](https://www.rfc-editor.org/rfc/rfc8032#section-5.1) · [LAN messages](lan-protocol.md)

| Offset | Field and encoding |
| ---: | --- |
| 0 | The 27 ASCII bytes `HiveScope/recovery-grant/v1`, followed by one zero byte. |
| 28 | Provider's raw Ed25519 public key (32 bytes). |
| 60 | Opaque folder ID (16), then agreement ID (16). |
| 92 | Positive grant generation (8, unsigned big-endian). |
| 100 | State (1), `01` active or `02` revoked. Scope (1), always `01` for reading the committed catalog and objects in this agreement. |
| 102 | Active recovery public key (32). A revoked record requires 32 zero bytes here. |
| 134 | Folder authority's signature on bytes 0 through 133 (64). |

Any authenticated peer may deliver a signed update. The provider checks its own
identity, both IDs, exact layout, scope, state, valid key encoding for an active
grant and signature against the enrolled authority before any change. It accepts
updates during the agreement and its seven-day read grace. Grant validity ends
with that read window. A grant never extends retention, adds a writer or creates
an agreement. The LAN trial provisions a fixed term locally.

| Compared with the provider's highest accepted generation | Provider decision |
| --- | --- |
| First positive generation, or a higher one | Persist the entire signed record, generation and state atomically. A higher signed revocation also applies if an earlier update never arrived. |
| Same generation and byte-identical record | Return the acknowledgement for this record. A lost reply is safe to retry. |
| Lower generation, or same generation with different bytes | Reject it and preserve the recorded state. Even a correct signature cannot reverse a newer revocation. |

Use one authority writer per agreement. Durably save the exact signed update
before sending it. A replacement active grant uses a fresh recovery key and a
new export. Reserve the maximum unsigned generation for a final revocation.
Never wrap. Serialize grant changes with read authorization. Invalidate pending
challenges for superseded grants and stop queuing their active reads before
acknowledging a change. Bytes already handed to the transport may still arrive.
Acknowledgement
on the pinned connection echoes SHA-256 of all 198 bytes after durable commit.
The owner compares it with the saved record. If confirmation is missing, report
the change as pending and retry. Previously delivered plaintext and keys remain
with their holder. [Read authorization and its challenge](recovery-format.md)

After restart, discard all challenges. Load the persisted highest generation,
recheck its signature and scope and deny access on missing or inconsistent
state. Restoring an older provider database must take the agreement offline for
reconciliation. A dishonest host can roll back an entire disk image. A single
provider cannot attest historical freshness. [Provider persistence](provider-persistence.md)

## Unlock a local profile

The CLI starts locked. A person explicitly opens the profile in their OS login
session. On app lock, session lock, logout or process exit, stop secret-dependent
operations, close authenticated streams and release in-memory keys. Reopening
requires explicit unlock. A provider that restarts must unlock its identity and
reconcile its authorization state before serving. These are design requirements.

| Desktop | Proposed custody of secrets at rest |
| --- | --- |
| Windows | Wrap secrets using current-user DPAPI. Store the protected blobs under a private profile ACL. Exclude machine-wide protection. [Microsoft DPAPI](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata) |
| macOS | Use the user's data-protection Keychain with `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`, an app-specific access group and synchronization disabled. [Apple Keychain selection](https://developer.apple.com/documentation/security/ksecusedataprotectionkeychain) · [When-unlocked access](https://developer.apple.com/documentation/security/ksecattraccessiblewhenunlockedthisdeviceonly) |
| Linux | Use an encrypted persistent Secret Service collection in the desktop session. Prompt via the service if locked. Use opaque lookup attributes and labels. Qualify the chosen backend and its lock behavior on the reference system. [Secret Service](https://specifications.freedesktop.org/secret-service/latest/description.html) |

A missing vault, refused prompt or unsupported session leaves the profile
locked. The trial requires a local interactive session. The database holds
opaque vault references, public keys, receipts and protocol state. Keep private
file names and original paths out of its cleartext rows, vault labels and logs.
Spool pending sealed bytes under opaque names in a private directory. Accept
export passphrases through a hidden local prompt. Exclude secrets from command
arguments, environment variables, crash reports and logs. A process that already
controls an unlocked owner account is outside the trial's protection.

## Keep administration recoverable

At enrollment, the owner exports an independent, offline **authority backup**
under a different strong passphrase. Store it separately from the read package
and keep each passphrase separate from its export. The owner also retains the
public enrollment fingerprints and latest acknowledged signed grant. Those
public records identify the authority and its last known generation.

The authority backup is 145 bytes. Its 33-byte header is ASCII `HSAB`, version
`01`, a fresh random 16-byte salt and a fresh random 12-byte nonce. Use the
exact passphrase encoding, Argon2id parameters, AES-256-GCM and 16-byte tag of
the [recovery format](recovery-format.md). Authenticate the entire `HSAB`
header as associated data. The 96-byte encrypted payload contains, in order,
the folder ID (16), agreement ID (16), provider public key (32) and authority
seed (32). An importer requires the exact size, magic and version for the
selected operation. A bad tag stops before parsing, contacting a provider or
changing the vault. Offline password guessing remains possible for either export.

1. Before retiring the original device, open the authority backup in an isolated
   local profile. Derive its public key and compare it and the scope with the
   independently saved enrollment record. Discard the temporary profile after
   the check. The ordinary file restore uses only the read package.
2. To resume administration after loss, restore that backup in a trusted vault
   and keep one authority writer. Obtain the latest signed grant from the
   provider's trusted local operator and compare it with the owner's last known
   signed record. Verify both against the pinned authority and scope. If the
   provider is behind, reconcile using the owner's newer signed record. Resolve
   conflicting records of equal generation before issuing anything. Generate a
   change above the highest verified generation. Activation of a fresh read
   credential also needs the owner's folder read material to make a usable export.
3. With the read package alone, file recovery may work and administrative changes
   are unavailable. With the authority backup alone, the owner can revoke a
   reader and cannot decrypt the file. Losing both copies of an authority seed
   prevents further signed changes on that agreement. A compromised authority
   requires closing the agreement locally and enrolling a fresh one.

## Acceptance cases for the first implementation

| Trial | Expected evidence |
| --- | --- |
| Issue the generation 7 active grant for the [recovery example](recovery-format.md), resend it and submit different bytes at generation 7. | The identical retry receives the same digest. The conflicting record leaves the active key and generation unchanged. |
| Acknowledge generation 8 revoked, restart, unlock and replay the signed generation 7 record. | The old update is rejected. Both independent reads with the old package return zero payload bytes. |
| Change the provider, scope or agreement bytes of a signed record, or sign it with the uploader or recovery key. | No update or read privilege is created. No previously stored grant is lost. |
| Lock the OS session mid-operation, deny vault access at startup, or use a wrong passphrase for either export. | The profile requires local unlock. A failed import contacts no provider and leaves the destination and vault intact. |
| Take the original device offline. Use each backup independently and try importing it as the other type. | The read package can authorize a valid read, the administrative backup restores the pinned authority, and both wrong-type imports stop locally. |
