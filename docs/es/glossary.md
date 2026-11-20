<!-- docs-pair: glossary; revision: 1 -->

[English](../en/glossary.md) | [Español](glossary.md) | [Índice](README.md)

# Glosario

| Término inglés | Término español | Significado en HiveScope |
| --- | --- | --- |
| Peer | Par | Dispositivo que participa mediante una identidad de red autenticada. |
| Approved provider | Proveedor aprobado | Par aceptado localmente para conservar fragmentos cifrados mediante un acuerdo. |
| Authorized device | Dispositivo autorizado | Equipo del dueño con permiso para leer y sincronizar una carpeta. |
| Daemon | Daemon / servicio en segundo plano | Proceso continuo propuesto para las operaciones del nodo. |
| Agreement | Acuerdo | Capacidad, duración, operaciones y límites aceptados por ambas partes. |
| Physical quota | Cuota física | Bytes contabilizados, incluyendo redundancia, metadatos y temporales. |
| Block | Bloque | Parte de una versión, cifrada antes de codificar la redundancia. |
| Fragment / shard | Fragmento | Parte codificada de un bloque cifrado. |
| Erasure coding | Codificación de borrado | Redundancia que reconstruye datos con un subconjunto suficiente de fragmentos. |
| Catalog | Catálogo | Metadatos protegidos necesarios para localizar y reconstruir contenido. |
| AEAD | Cifrado autenticado con datos asociados | Cifrado que autentica contenido cifrado y su contexto asociado. |
| Nonce | Nonce | Valor que debe ser único por clave en el esquema AEAD elegido. |
| Version vector | Vector de versión | Historia causal para distinguir ediciones posteriores y concurrentes. |
| Tombstone | Marcador de borrado | Registro que evita que una copia desconectada resucite silenciosamente un borrado. |
| Churn | Conexión y desconexión de nodos | Pares que entran, salen, suspenden o pierden conexión temporalmente. |
| Relay | Relay / retransmisor | Par que reenvía tráfico cifrado si no existe conexión directa. |
| Repair | Reparación | Reconstrucción de fragmentos cifrados sin necesitar claves de lectura. |
| Sybil attack | Ataque Sybil | Uso de muchas identidades para manipular supuestos de participantes independientes. |
| Inference | Inferencia | Ejecución de un modelo entrenado para producir una salida. |

Conserva los nombres de contratos y campos en inglés. Utiliza **MiB/GiB** para
capacidad binaria y **MB/GB** para capacidad decimal; no mezcles las unidades.
