<!-- docs-pair: architecture; revision: 5 -->

[English](../en/architecture.md) | [Español](architecture.md) | [Índice](README.md)

# Arquitectura e interfaces

El nodo de escritorio propuesto tiene un servicio en segundo plano y una CLI.
Go es el primer lenguaje que se evaluará. El repositorio contiene el diseño.

## Componentes

| Componente | Responsabilidad |
| --- | --- |
| Identidad y autorización | Autenticar cada dispositivo y comprobar por separado el perfil de recursos del anfitrión y las autorizaciones de carpeta que concede el dueño. |
| Descubrimiento y transporte | Buscar candidatos en directorios comunitarios reemplazables y establecer conexiones autenticadas entre pares. |
| Protección de datos y almacenamiento | Cifrar versiones y catálogos, reservar bytes físicos, guardar objetos inmutables y verificar recibos. |
| Mantenimiento y sincronización | Comprobar conectividad, renovar acuerdos, reparar datos cifrados, reconciliar versiones firmadas y conservar ediciones en conflicto. |
| Control local de recursos | Aplicar el horario del anfitrión, las cuotas de disco y transferencia, los límites de cómputo y concurrencia y la pausa. |
| Sesión privada | En un hito posterior, verificar el entorno de ejecución remoto, entregar las claves de sesión y ejecutar el software del solicitante. |

Se pueden solicitar recursos sin aportar hardware. Cada anfitrión elige en un
perfil local persistente qué servicios presta. Las máquinas pueden acordar el
almacenamiento sin que las personas intercambien datos de contacto. Solo el
dueño de la carpeta puede conceder a otros dispositivos la facultad de leerla.

## Flujo de almacenamiento

1. Un dispositivo autorizado obtiene una versión local estable y la cifra por
   bloques. En el hito 1 un solo proveedor en LAN guarda el objeto cifrado
   completo y un catálogo autenticado. El descubrimiento público automático llega en el hito 2.
2. El anfitrión reserva espacio y confirma la escritura tras guardar de forma
   duradera y verificar los bytes. Si se interrumpe la escritura o se pierde la
   respuesta, se reintenta con el mismo identificador. Una carga por confirmar queda pendiente.
3. El hito 3 añade fragmentos redundantes, metadatos cifrados y firmados y
   catálogos replicados. Cada proveedor acepta un acuerdo acotado. Los lectores
   verifican los metadatos y todos los bloques necesarios antes de publicar el archivo local.
4. Un dispositivo autorizado o uno de reemplazo con autorización de recuperación
   vigente obtiene y verifica los datos, los descifra localmente y publica la
   versión de forma atómica. Los búferes acotados permiten transferir un archivo
   sin que la memoria necesaria crezca con su tamaño.

## Red y descubrimiento

La propuesta utiliza go-libp2p con QUIC y TCP, canales autenticados, AutoNAT,
DCUtR y Circuit Relay v2. A partir del hito 2, el nodo anuncia su servicio con
un registro firmado y con vencimiento en un directorio reemplazable. Si un
directorio no responde, el solicitante puede probar otro. El anuncio contiene
información del servicio y de contacto. Los nombres de archivos, catálogos de
carpetas y contenidos de sesión quedan fuera del directorio. El nodo autentica
directamente al candidato que encuentra.

El anfitrión comprueba horario, espacio físico libre en el volumen ofrecido,
compromisos vigentes y límites locales antes de reservar. Si dos solicitudes
compiten por el último espacio, como máximo una obtiene la reserva. Al vencer
se libera una reserva sin utilizar. La pausa detiene reservas nuevas y conserva
las obligaciones de retención vigentes. Si falta capacidad, el solicitante puede
esperar o cancelar. [Ofertas de recursos](resource-offers.md) · [Política de almacenamiento](storage.md)

Se utilizará conexión directa cuando sea posible. Los relays configurables
reenvían tráfico cifrado con presupuestos de transferencia. Si ambas rutas
fallan, el trabajo queda pendiente con un motivo de conectividad visible.
Las renovaciones de anuncios, las reconexiones simultáneas, la saturación de
relays y el tráfico de reparación se medirán por separado del consumo del PC.

## Persistencia e interfaces

SQLite conserva el índice, las reservas, los acuerdos y la cola reanudable de
cada nodo. Los objetos se guardan en un directorio configurado por separado.
En la primera subida, el proveedor confirma juntos el objeto cifrado y el catálogo.
Ambos archivos, sus entradas en el índice y el recibo deben sobrevivir a un
reinicio. El hito de redundancia replica los catálogos cifrados para poder
recuperar datos sin la base original. [Confirmación de subida y respuesta perdida](upload-confirmation.md)

La CLI y la futura interfaz gráfica compartirán operaciones para perfiles,
solicitudes, pausa, estado, permisos de carpeta y recuperación. Los protocolos
P2P llevarán versión, mensajes Protobuf y tramas de transferencia acotadas. Se
rechazarán las versiones desconocidas y los mensajes excesivos antes de reservar
recursos. Los contratos describen el comportamiento que habrá que especificar y
probar antes de elegir la codificación.

| Contrato | Significado requerido |
| --- | --- |
| `DeviceIdentity` | Identificador del par y clave pública de verificación. Los secretos de recuperación se custodian aparte. |
| `FolderAuthorization` | Pertenencia, función y época de autorización del dispositivo firmadas por el dueño. |
| Anuncio de servicio | Capacidades e información de contacto firmadas y con vencimiento para el descubrimiento. |
| Reserva de recursos | Solicitud única asociada a solicitante y servicio, cantidad reservada, vencimiento y estado conservado al reintentar. |
| `StorageAgreement` | Partes, cuota física, vigencia, límites de transferencia y operaciones acordadas por los dos nodos. |
| `FileVersion` | Autor autorizado, versión causal, metadatos cifrados y referencias de bloques. |
| `ShardReceipt` | Confirmación firmada por el proveedor con ID, resumen, tamaño y acuerdo del fragmento. |

## Propiedad y recuperación

Cada carpeta tiene su propia autoridad y claves de cifrado bajo el control del
dueño. Cada dispositivo tiene identidad propia. La incorporación verifica su huella
y transfiere los secretos de carpeta que le corresponden por un canal autenticado.
Revocar un equipo inicia una época de autorización y rota las claves para
versiones futuras. El contenido ya leído puede seguir en poder de ese equipo.

El diseño del hito 1 en revisión concede al equipo de reemplazo un permiso limitado
para recuperar objetos cifrados y el catálogo. El paquete cifrado contiene la clave
de firma de recuperación, material de descifrado, localizador y huella del proveedor.
El dueño guarda por separado la clave de firma administrativa. El proveedor verifica
una prueba reciente de posesión y la autorización vigente. Una vez confirmada una
revocación, tiene que seguir rechazando la autorización antigua tras un reinicio.
El formato del paquete y la protección local de los secretos requieren revisión
antes de implementar. Incorporar el equipo nuevo como lector permanente requiere
una autorización aparte. [Primera lectura, revocación y pruebas de recuperación](readiness.md)

## Cómputo privado y servicios persistentes

El solicitante elige el software de su entorno privado. El anfitrión aporta una
cantidad limitada de CPU, RAM, ancho de banda y, más adelante, GPU. En el hito 5
se evaluará una máquina virtual confidencial. Antes de entregar secretos, el
solicitante debe verificar evidencia reciente vinculada a la clave de sesión
del entorno invitado. Si se necesita GPU, la verificación debe abarcar también
el acelerador y el recorrido de los datos hasta él. Ante un anfitrión incompatible,
la solicitud espera y las claves siguen con el solicitante. El acceso a la sesión
es independiente de la autoridad para descifrar carpetas. [Modelo de seguridad y pruebas propuestas](security.md)

Un servicio persistente necesita conservar el estado y limitar la escritura a
una sola instancia. Antes de aceptar tráfico, el relevo autorizado debe verificar
el último guardado confirmado y adquirir la autoridad exclusiva de escritura.
El anfitrión que vuelve debe comprobar el relevo antes de reanudar escrituras.
La sesión privada en otro anfitrión requiere una nueva verificación allí. Se
medirán la interrupción, la latencia de reconexión y el progreso perdido.

## Evaluación tecnológica

Medir el nodo Go en un PC con cuatro núcleos y 8 GB de RAM. Registrar memoria
residente en reposo y máxima, CPU en reposo, velocidad de recuperación y tiempo
para hacer efectiva la pausa. Probar transferencia directa y por relay en
condiciones documentadas. Si Go incumple un presupuesto acordado, identificar
la causa y comparar Rust con el mismo hardware, archivo, cifrado, tamaño de
bloque y concurrencia. Evaluar por separado el alojamiento de sesiones
confidenciales en hardware compatible.

La reconciliación de capacidades, la selección de anfitriones y el manejo de
reintentos de OpenClaw sirven de referencia. Su SDK de TypeScript permitiría
ofrecer HiveScope como destino opcional de almacenamiento para OpenClaw.
El manejo de temporales y los límites de procesos en Windows de REA aportan
casos de prueba para recuperación y cuotas locales. Para cualquier código
reutilizado, registrar revisión, procedencia, licencia y avisos de terceros
aplicables. [Selector de OpenClaw](https://github.com/openclaw/openclaw/blob/3b63f9ea1f552930f8d20215a1cb89a95a261fd0/src/gateway/worker-environments/device-placement-selector.ts#L74-L110) · [SDK de almacenamiento de OpenClaw](https://github.com/openclaw/openclaw/blob/3b63f9ea1f552930f8d20215a1cb89a95a261fd0/docs/plugins/sdk-overview/capabilities.md#L59-L72) · [Archivos temporales de REA](https://github.com/morluto/rea/blob/8bb01b514ac766c0c3ecc8f362bd38186eaec9c2/src/filesystem/OwnedStagedOutput.ts) · [Control de procesos de REA](https://github.com/morluto/rea/blob/8bb01b514ac766c0c3ecc8f362bd38186eaec9c2/native/windows/README.md)

Lecturas sobre transporte y recuperación. [Conexión a través de NAT con libp2p](https://github.com/libp2p/specs/blob/master/connections/hole-punching.md) · [Arquitectura de Tahoe-LAFS](https://tahoe-lafs.readthedocs.io/en/stable/architecture.html)
