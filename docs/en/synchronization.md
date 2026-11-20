<!-- docs-pair: synchronization; revision: 1 -->

[English](synchronization.md) | [Español](../es/synchronization.md) | [Index](README.md)

# Folder synchronization

The first synchronization service targets a person's own authorized devices.
Every device materializes the selected folder locally. This is not a virtual
filesystem or a collaborative document editor.

## Detect and publish changes

Use filesystem notifications plus periodic scans; notifications alone can be lost.
Wait for a file to stabilize before reading it. If its size or modification state
changes during capture, retry rather than publishing a mixed snapshot.

Each immutable version records an authorized writer, a causal version vector,
encrypted metadata and references to protected content. A changed file produces
a new version; the first implementation need not provide cross-version block
deduplication. Retained versions consume physical quota.

Devices can edit offline. Their changes stay pending and synchronize when peers,
capacity and valid authorizations become available. A local save is not a claim
that the network has protected the new version.

## Conflicts

- A causally later version supersedes its ancestor.
- Concurrent edits preserve both complete versions. Choose the primary copy by
  a deterministic ordering of signed version identifiers; keep the other as a
  conflict copy with an identifier-based suffix, independent of wall-clock time.
- Concurrent delete and edit preserve the edited content as a recoverable conflict.
- Directory deletion never silently discards concurrent child edits; keep those
  edits in a recoverable conflict location.
- Rename is initially a create plus a deletion marker. Optimizing it is later work.
- Reconcile repeated announcements idempotently so conflicts do not multiply.

The user resolves conflicts by editing or deleting the preserved copies. There
is no automatic text merge. Device clocks do not decide which concurrent edit
the user intended to keep.

## Apply and recover

Download to a temporary file, verify digests and AEAD authentication, flush the
data and atomically replace the destination. Re-check the local version before
replacement; a new local edit must become a conflict instead of being overwritten.
Persist work and metadata transactionally so restart resumes safely.

Keep superseded versions and deleted content for 30 days by default, within
agreements and quotas. Deletion markers remain in the metadata until all currently
authorized devices acknowledge them. A long-offline device cannot be forgotten
silently: the owner must revoke or explicitly re-enroll it after reconciliation.
Do not treat an empty or inaccessible mount as an instruction to delete a folder.

Restoring an old version publishes it as a new edit. The project does not promise
secure erasure from providers or the ability to revoke data already downloaded.
Synchronization and finite history are not a substitute for an independent backup.

## Cross-platform rules

Synchronize regular files and directories. Exclude symbolic links, special files,
device files and platform-specific ownership or permission replication initially.
Never follow a symlink outside the selected root while scanning or applying data.

Reject absolute paths, traversal, reserved names and unsupported paths. Report
case-only collisions, Unicode normalization collisions and path-length limits
without silently overwriting data. Portable examples must work on Windows,
Linux and macOS. Exact filename rules belong in the synchronization protocol
specification before implementation.

Reference: [Syncthing's synchronization behavior](https://docs.syncthing.net/users/syncing).
