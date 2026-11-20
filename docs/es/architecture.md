<!-- docs-pair: architecture; revision: 1 -->

[English](../en/architecture.md) | [Español](architecture.md) | [Índice](README.md)

# Arquitectura e interfaces

Todos los componentes de este documento son propuestas. La primera implementación
utilizará un ejecutable Go con modos daemon y CLI. Esta base documental no incluye
un módulo Go ni un ejecutable.

## Componentes

| Componente | Responsabilidad |
| --- | --- |
| Identidad y autorización | Identidades por dispositivo, aprobación de pares y permisos de carpetas. |
| Transporte P2P | Conexiones autenticadas, actualización de direcciones, NAT y relays limitados. |
| Sincronización | Vigilancia de archivos, escaneos, versiones firmadas, conflictos y escritura local. |
| Protección de datos | Cifrado autenticado, catálogos cifrados y recuperación de claves. |
| Almacenamiento | Fragmentos inmutables, integridad, reservas y cuotas físicas. |
| Mantenimiento | Disponibilidad, comprobaciones, reparación y renovación de acuerdos. |
| Control de recursos | Contabilidad de disco, tasas, horarios, concurrencia y pausa. |

Un dispositivo puede asumir varias funciones. Los lectores de carpetas, los
proveedores de almacenamiento cifrado y los relays públicos tienen permisos
independientes. La conectividad o reputación de un nodo no lo autoriza a descifrar.

## Flujo de datos

1. Un dispositivo autorizado detecta un cambio local estable y crea una versión inmutable.
2. Divide el archivo en bloques, los cifra y produce fragmentos redundantes.
3. Los proveedores aprobados reservan capacidad, verifican los bytes y emiten recibos.
4. El dispositivo publica metadatos de versión firmados y cifrados entre sus pares aprobados.
5. Otro dispositivo autorizado reconcilia versiones, recupera suficientes
   fragmentos verificados, descifra y reemplaza atómicamente el archivo local.

Las transferencias utilizan streaming con memoria acotada. Los cambios locales
permanecen pendientes hasta confirmar sus metadatos y los fragmentos requeridos.

## Red y descubrimiento

Utilizar go-libp2p con QUIC y TCP, canales seguros autenticados, AutoNAT, DCUtR y
Circuit Relay v2. La red inicial utiliza contactos intercambiados y descubrimiento
en LAN. Solo los pares aprobados pueden negociar almacenamiento o acceder a las
operaciones de carpetas. Los registros de direcciones firmados y con vencimiento
permiten intercambiar actualizaciones entre contactos conocidos.

Los contactos de relay públicos son configurables y sustituibles. Los relays
reenvían tráfico cifrado, aplican presupuestos de transferencia y no controlan
cuentas ni créditos. Se prefieren las conexiones directas; un NAT restrictivo
puede exigir relay. Si fallan ambas rutas, el trabajo queda pendiente con un
estado explícito de conectividad. La primera implementación no tendrá DHT pública
ni índice global de archivos.

## Persistencia e interfaces

SQLite guarda el índice local, los acuerdos y la cola reanudable de cada nodo.
Los fragmentos se guardan en un directorio independiente y configurable. Esta
base no es un ledger compartido. Los catálogos cifrados y registros de versión
se replican para no depender del SQLite de un único dispositivo.

La CLI cubrirá identidad, aprobación de pares, autorización de carpetas, acuerdos,
estado, pausa y recuperación. La futura interfaz gráfica utilizará las mismas
operaciones del daemon. Los mensajes P2P utilizarán protocolos versionados,
Protobuf y streaming con tramas acotadas. Las versiones desconocidas y los
mensajes excesivos deben rechazarse antes de reservar recursos.

Los contratos iniciales de diseño son:

| Contrato | Significado requerido |
| --- | --- |
| `DeviceIdentity` | Identificador de par y clave pública de verificación; nunca un secreto de recuperación. |
| `FolderAuthorization` | Membresía, función y época de autorización firmadas por el propietario. |
| `StorageAgreement` | Partes, cuota física, vigencia, límites y operaciones permitidas. |
| `FileVersion` | Autor autorizado, versión causal, metadatos cifrados y referencias de bloques. |
| `ShardReceipt` | Confirmación firmada del proveedor con ID, resumen, tamaño y acuerdo del fragmento. |

Son límites semánticos, no APIs estables ni esquemas definitivos. Cada hito de
implementación debe definir codificación, límites, tratamiento de repeticiones
y compatibilidad antes de añadir código de protocolo.

## Propiedad y recuperación

Cada carpeta tiene autoridad y claves de cifrado independientes controladas por
su dueño. Cada dispositivo tiene identidad propia; copiar una identidad entre
equipos no es el mecanismo de incorporación. Se verifica una huella y se
transfieren los secretos mediante un canal autenticado. Revocar un equipo inicia
una época nueva y rota las claves para versiones futuras; no permite retirar
texto o claves que ese equipo ya conoció.

La recuperación requiere una exportación cifrada con secretos de carpeta,
material de autoridad, localizadores de catálogo y contactos aprobados. Importarla
incorpora una nueva identidad de dispositivo. No hay restablecimiento central de
contraseñas. La protección local y el formato de exportación deben especificarse
y revisarse en el hito del núcleo seguro.

## Servicio posterior de IA

Reutilizar identidad, transporte, límites y conceptos de acuerdo, aislando el
runtime de cómputo del almacenamiento. No permitir programas arbitrarios enviados
por pares. Comenzar con runtimes aprobados y modelos completos en proveedores
capaces. Las cuotas de cómputo, la integridad de resultados y la privacidad de
consultas necesitan su propia revisión de diseño.

## Referencias

- [Perforación de NAT y relays de libp2p](https://github.com/libp2p/specs/blob/master/connections/hole-punching.md)
- [Arquitectura de Tahoe-LAFS](https://tahoe-lafs.readthedocs.io/en/stable/architecture.html)
- [Inferencia colaborativa con Petals](https://github.com/bigscience-workshop/petals)
