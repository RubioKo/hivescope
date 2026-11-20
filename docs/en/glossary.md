<!-- docs-pair: glossary; revision: 1 -->

[English](glossary.md) | [Español](../es/glossary.md) | [Index](README.md)

# Glossary

| Term | Spanish term | Meaning in HiveScope |
| --- | --- | --- |
| Peer | Par | A device participating through an authenticated network identity. |
| Approved provider | Proveedor aprobado | A peer locally accepted to retain encrypted fragments under an agreement. |
| Authorized device | Dispositivo autorizado | An owner's device permitted to read and synchronize a folder. |
| Daemon | Daemon / servicio en segundo plano | The proposed long-running process implementing node operations. |
| Agreement | Acuerdo | Mutually accepted capacity, duration, operations and transfer limits. |
| Physical quota | Cuota física | Bytes actually accounted for, including redundancy, metadata and temporaries. |
| Block | Bloque | A portion of a file version, encrypted before redundant encoding. |
| Fragment / shard | Fragmento | One encoded piece of an encrypted block. |
| Erasure coding | Codificación de borrado | Redundancy that reconstructs data from a sufficient subset of fragments. |
| Catalog | Catálogo | Protected metadata needed to locate and reconstruct folder content. |
| AEAD | Cifrado autenticado con datos asociados | Encryption that authenticates ciphertext and associated context. |
| Nonce | Nonce | A value that must be unique per encryption key in the chosen AEAD scheme. |
| Version vector | Vector de versión | Causal history used to distinguish later and concurrent edits. |
| Tombstone | Marcador de borrado | Metadata recording deletion so an offline copy does not silently resurrect it. |
| Churn | Conexión y desconexión de nodos | Peers joining, leaving, sleeping or becoming temporarily unreachable. |
| Relay | Relay / retransmisor | A peer forwarding encrypted traffic when direct connectivity is unavailable. |
| Repair | Reparación | Reconstructing missing ciphertext fragments without needing plaintext keys. |
| Sybil attack | Ataque Sybil | Using many identities to manipulate assumptions about independent participants. |
| Inference | Inferencia | Executing a trained model to produce an output. |

Keep contract and field names in English in every translation. Use **MiB/GiB**
for binary capacity and **MB/GB** for decimal capacity; do not mix the units.
