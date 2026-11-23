<!-- docs-pair: readiness; revision: 21 -->

[English](../en/readiness.md) | [Español](readiness.md) | [Índice](README.md)

# Diseño y criterios de aceptación del primer prototipo

La computadora de Ana ha dejado de encender. ¿Puede recuperar un archivo en
otro equipo con solo su paquete de recuperación, su contraseña y el proveedor
que guarda la copia cifrada? ¿Puede ese mismo proveedor rechazar el mismo
paquete después de confirmar una revocación y reiniciarse? Estos dos resultados
definen el primer prototipo de escritorio.

**Resultados actuales.** Las pruebas locales cubren los formatos de recuperación
y el guardado de archivos verificados. Las pruebas TCP en la interfaz de bucle
local comprueban la identidad del proveedor y las firmas de un solo uso para
leer por separado el catálogo y su objeto cifrado. En Linux también se comprueba
la identidad del proveedor en equipos virtuales aislados. Las pruebas de
reinicio cubren una revocación confirmada y otra interrumpida que el
propietario reenvía. Los ensayos de subida comprueban que perder la respuesta a
una reserva deja un solo cargo de espacio y que, tras un corte en el envío del
catálogo, el proveedor conserva el objeto ya recibido. Si se pierde la respuesta
de publicación, el ensayo reabre el proveedor, reintenta y comprueba el recibo
firmado original, el cargo único y los bytes recuperados. Recuperar el archivo en otra computadora
sigue siendo parte del ensayo en red local.

## Alcance y contratos

Se usará un perfil de propietario, un proveedor con dirección IPv4/TCP conocida
y un perfil de reemplazo limpio con identidad de dispositivo nueva. Aislar los
perfiles para que el reemplazo carezca de acceso a la base original, el almacén
de claves original, el archivo fuente, los datos cifrados en caché y el respaldo
administrativo. El proveedor guarda un objeto cifrado completo y su catálogo
cifrado y firmado. Un único proveedor puede desconectarse, perder su disco o
devolver una versión antigua. Ana necesita una copia independiente de todo
archivo valioso durante el ensayo.

| Decisión | Dónde consultar el contrato |
| --- | --- |
| ¿Quién puede escribir, recuperar o revocar? | La configuración de la LAN fija las identidades de los pares y la autoridad de la carpeta. El permiso de lectura tiene generación y alcance limitados. El propietario guarda por separado el respaldo administrativo. [Identidad y custodia de claves](identity-custody.md) |
| ¿Qué lleva consigo el equipo de reemplazo? | Un paquete de 255 bytes protegido por contraseña identifica la instantánea, el proveedor y los secretos de lectura. Cada lectura del catálogo o del objeto exige una prueba nueva. [Formato de recuperación y firmas de referencia](recovery-format.md) |
| ¿Qué bytes pertenecen al archivo? | El catálogo autenticado proporciona el localizador, la clave de versión, la longitud y el resumen esperados del objeto. Un archivo temporal privado conserva el contenido verificado hasta publicarlo en el destino. [Formato de archivo y catálogo](file-catalog.md) |
| ¿Qué intercambian los equipos? | El protocolo versionado para la LAN define mensajes acotados, plazos, autenticación, cambios de permisos y errores. [Protocolo de red local](lan-protocol.md) |
| ¿Cuándo se puede confirmar la subida? | Una sola reserva cubre ambos elementos cifrados. El cliente contrasta el recibo firmado por el proveedor con su manifiesto local. Repetir la solicitud devuelve el resultado registrado. [Confirmación de subida](upload-confirmation.md) |
| ¿Qué debe sobrevivir a un reinicio? | El proveedor de referencia ordena la sincronización de archivos y directorios, la transacción de SQLite, la confirmación y la limpieza. La reserva física cubre temporales y metadatos. [Persistencia del proveedor e interrupciones](provider-persistence.md) |

## Recorrido de una recuperación

1. **Ofrecer espacio.** El operador elige un volumen local, el límite de disco
   y la disponibilidad del proveedor. En la LAN controlada se configura el
   acuerdo y se comprueban por separado las huellas de las claves públicas.
   Se activa el permiso de recuperación del dueño. El futuro descubrimiento
   comunitario podrá mantener esta separación entre escritura y lectura.
   [Ofrecer mi equipo](resource-offers.md)
2. **Guardar una versión.** Empezar con el archivo sintético `hive.txt`. El
   equipo del dueño cifra el objeto y el catálogo, registra la solicitud y
   reserva espacio para ambos. Si la respuesta se pierde, Ana ve **pendiente
   de confirmación** y su equipo consulta la misma solicitud. Solo muestra
   **subida confirmada** tras verificar y guardar el recibo. Los cinco bytes
   originales dan 21 bytes de objeto y 250 de catálogo, más el espacio que
   necesite el disco para guardarlos.
3. **Prepararse para perder el equipo.** Exportar explícitamente el paquete
   de recuperación de esa versión. Guardar el paquete y la contraseña por
   separado. Apagar el perfil original. Quien ejecute el ensayo conservará
   los bytes de referencia fuera del perfil de reemplazo.
4. **Recuperar.** Elegir en el equipo limpio el paquete y un destino local.
   Desbloquearlo localmente, autenticar al proveedor fijado y demostrar el
   permiso vigente en dos lecturas, una del catálogo y otra del objeto. Verificar
   catálogo, bloques, longitud y resumen antes de publicar el archivo en el
   destino elegido. Mostrar **recuperado** tras completar la publicación.
   Comparar el resultado con la referencia independiente.
5. **Ejercer la revocación.** En otro ensayo con el mismo estado inicial, firmar
   la revocación con la autoridad de la carpeta, esperar la confirmación
   persistente del proveedor, reiniciarlo y desbloquear su perfil. Reconectar
   con el proveedor autenticado. Para cada localizador, obtener un reto nuevo
   y firmarlo correctamente con la clave antigua. Ambas lecturas deben devolver
   `DENIED` y cero bytes de contenido. Quien recibió bytes antes de la revocación
   conserva su copia.

## Registro de aceptación de la primera implementación

Todos los ensayos usan datos sintéticos. Los identificadores permiten asociar
cada resultado con la versión que lo produjo. En el commit `990e836`,
M1-01 pasó en memoria con la muestra de cinco bytes y Go 1.27.4 sobre Windows
amd64. Las pruebas de M1-04 descritas más adelante cubren el rechazo del
paquete y la discrepancia de identidad del proveedor en la misma dirección y
puerto. El procedimiento completo requiere todavía un equipo de reemplazo y
un proveedor de almacenamiento en funcionamiento.

| ID | Ensayo | Evidencia que registrar |
| --- | --- | --- |
| M1-01 | Antes de usar la red, comparar con una implementación local el paquete, los mensajes firmados de lectura, el catálogo, el objeto y el recibo publicados. | Las longitudes, los resúmenes y la verificación de firmas coinciden con las referencias. Un bloque sustituido falla la verificación. |
| M1-02 | Usar el paquete activo de generación 7 tras perder el perfil original. Repetir con un archivo vacío y uno de 4 MiB + 1 byte. | El reemplazo recibe los dos elementos y recupera exactamente los bytes previstos. El archivo vacío también autentica su único bloque. |
| M1-03 | Confirmar la revocación de generación 8, reiniciar al proveedor y desbloquearlo. Probar ambos localizadores por separado con retos nuevos correctamente firmados. Por separado, repetir un reto ya consumido. | El proveedor autenticado devuelve expresamente `DENIED` y cero bytes para ambas lecturas con el permiso antiguo. Repetir un reto no autoriza otra lectura. El archivo de cinco bytes que ya existía en destino permanece intacto. |
| M1-04 | Probar una contraseña errónea, un paquete dañado y un equipo que presenta otra clave de proveedor. | El fallo de autenticación del paquete termina localmente. Una identidad distinta detiene la operación antes del reto de lectura. Se conserva el destino. |
| M1-05 | Cortar la conexión a mitad de un elemento. En otro ensayo, perder la respuesta posterior a la confirmación en disco. Reintentar con la solicitud original y con el manifiesto cambiado. | Un elemento incompleto se reenvía desde el byte cero. La solicitud confirmada devuelve el mismo recibo, dos localizadores y un único cargo. El cambio de manifiesto devuelve conflicto. |
| M1-06 | Enviar dos solicitudes distintas del equipo autorizado a subir, hacerlas competir por el último espacio, pausar nuevas ofertas, cancelar una reserva y dejar vencer otra. | Como máximo una reserva obtiene ese espacio. El estado se conserva al reintentar. El espacio ocupado sigue contabilizado hasta que la limpieza sea persistente. |
| M1-07 | Interrumpir antes y después de cada punto de publicación definido en el contrato de persistencia. En una copia desechable, eliminar o dañar un archivo confirmado. | Tras cada reinicio coinciden visibilidad, cuota y recibo con lo previsto. Una copia confirmada que sufrió daños figura como inaccesible. Registrar por separado las interrupciones del proceso y la pérdida de alimentación del almacenamiento. |
| M1-08 | Probar una lectura fuera de alcance, una trama excesiva, un objeto truncado y un volumen de destino lleno. | El par aplica sus límites. Si la verificación falla, el contenido parcial queda fuera del destino. Registrar el resumen del archivo de destino conservado. |
| M1-09 | Bloquear el almacén de claves del sistema, reiniciar al proveedor y restaurar el respaldo administrativo independiente en un perfil del dueño aislado. | El bloqueo impide usar las claves de las que depende cada operación. Tras el desbloqueo se conservan identidad y generación máxima aceptada. La autoridad restaurada puede firmar un cambio. El paquete de recuperación de archivos por sí solo carece de ese permiso. |
| M1-10 | Probar un acuerdo de escritura vencido dentro de su periodo de gracia de lectura y después del cierre. Desconectar el único proveedor. | Las escrituras nuevas cesan al vencer el acuerdo. Las lecturas autorizadas funcionan durante la gracia y cesan al cierre. El cliente informa que el proveedor está inaccesible sin declarar el archivo perdido ni recuperado. |

Por ejecución, registrar ID del caso, commit, sistema operativo, sistema de
archivos, muestra, estado inicial, punto de interrupción, resultado esperado,
resultado observado y estado (aprobado, fallido o sin ejecutar). Contar bytes
de contenido y reservas confirmadas cuando sean el criterio que se comprueba.
Usar identificadores sintéticos en los informes. Las contraseñas, claves,
exportaciones, rutas privadas y registros identificativos quedan fuera de la
evidencia. Una lectura correcta comprueba una sola versión. Repetir en cada
plataforma antes de anunciar compatibilidad.

## Cómo reproducir los resultados locales

En Linux, con cgo y la biblioteca SQLite del sistema, ejecuta
`go test -mod=readonly -count=1 -run '^TestLostReservationReplyKeepsOneCharge$' ./internal/conformance`.
El equipo original solicita espacio para un objeto cifrado de 21 bytes y su
catálogo firmado de 250 bytes. La prueba corta la conexión una vez registrada
la reserva. Tras reabrir el almacén, el dispositivo autorizado consulta el
estado y reenvía el mismo manifiesto. El proveedor devuelve la reserva
pendiente con su plazo original y un solo cargo de espacio. El cliente debe
conservar el manifiesto, el ID de solicitud y los datos cifrados de origen
mientras espera la confirmación. [Prueba de reserva](../../internal/conformance/reserve_linux_test.go)

En la misma configuración Linux, ejecuta
`go test -mod=readonly -count=1 -run '^TestInterruptedCatalogPutKeepsStagedObject$' ./internal/conformance`.
El equipo de origen envía el objeto cifrado de 21 bytes, transmite 125 de los
250 bytes del catálogo y cierra el flujo. Al reabrir el almacén, `STATUS`
indica que solo el objeto está preparado (máscara `02`). Reintentar ese objeto
consume cero bytes del archivo de origen. Enviar el catálogo completo cambia
la máscara a `03` con la reserva y el vencimiento originales. En ese punto,
ambos localizadores siguen privados. [Prueba de PUT interrumpido](../../internal/conformance/put_linux_test.go)

En Linux con cgo, ejecuta
`go test -mod=readonly -count=1 -run '^TestLostFinalizeReplyReturnsOriginalReceipt$' ./internal/conformance`.
La prueba corta la respuesta a `FINALIZE` después de convertir la reserva en
espacio almacenado. El propietario aún carece del recibo local. Tras reabrir el
proveedor, dos reintentos devuelven los mismos 132 bytes firmados y el propietario
los guarda. Una identidad de reemplazo con un permiso vigente lee ambos elementos
y recupera los cinco bytes originales. El ensayo cubre la pérdida de una respuesta
y la reapertura del almacén en un solo equipo. El corte eléctrico y la recuperación
en equipos físicos distintos quedan para el ensayo en red local. [Prueba de pérdida de la respuesta](../../internal/conformance/finalize_linux_test.go)

En Linux con cgo, ejecuta
`go test -mod=readonly -count=1 -run '^TestCancelAndFinalizeKeepOneOutcome$' ./internal/conformance`.
El dispositivo prepara ambos elementos y prueba los dos órdenes de `CANCEL` y
`FINALIZE`. Si gana la cancelación, el proveedor conserva el ID, libera el espacio
y deniega la lectura de ambos elementos. Si gana la publicación, el dispositivo
obtiene el recibo original y el equipo de reemplazo recupera los cinco bytes.
Tras reabrir el almacén, cada reintento debe conservar el resultado que ganó.
[Prueba de cancelación](../../internal/conformance/cancel_linux_test.go)

Desde la raíz del repositorio, con Go 1.26 o posterior, ejecutar una vez
`go mod download` y después `go test -mod=readonly -count=1 ./...`. Las
pruebas usan datos sintéticos. Tras preparar las dependencias,
también pasan con el proxy de módulos desactivado. Reproducen el paquete
de recuperación de 255 bytes, los cuatro ejemplos de lectura firmada, el
objeto de 21 bytes, el catálogo firmado de 250 bytes y el recibo de 132
bytes. El contenido recuperado es `48 69 76 65 0a`. La contraseña incorrecta,
el bloque sustituido, la firma de catálogo alterada y el recibo contrastado con
un manifiesto modificado fallan sus respectivas comprobaciones. [Objeto y catálogo](../../internal/conformance/snapshot_test.go) · [Recuperación y lecturas](../../internal/conformance/recovery_test.go) · [Recibo](../../internal/conformance/receipt_test.go)

La prueba de reinicio en Linux comprueba la parte de M1-03 relativa al
permiso. Compilar con cgo habilitado, un compilador de C y las cabeceras y
la biblioteca de la misma versión de SQLite del sistema, con soporte para
`SQLITE_OPEN_NOFOLLOW` (disponible desde SQLite 3.31.0).
[Notas de la versión de SQLite](https://www.sqlite.org/releaselog/3_31_0.html)

Para ejecutar solo este caso, usar
`go test -mod=readonly -count=1 -run '^TestAcknowledgedRevocationSurvivesRestart$' ./internal/conformance`.
Con el permiso de generación 7 activo, el cliente puede recuperar el archivo
de cinco bytes. El proveedor acepta la revocación de generación 8 firmada por
el propietario. Una vez que el cliente comprueba el resumen de la confirmación,
la prueba termina el proceso del proveedor y arranca otro con la misma identidad
y el mismo almacén. El equipo de reemplazo firma con la credencial antigua
un reto nuevo por cada elemento. Ambas solicitudes devuelven `DENIED` y cero
bytes del elemento. Reenviar la misma revocación devuelve el resumen original.
Reenviar el permiso anterior firmado devuelve `CONFLICT`. La prueba parte de
los elementos cifrados ya cargados. El ensayo completo en red local incluye
equipos físicos y el desbloqueo nativo del perfil. [Prueba de reinicio del proceso](../../internal/conformance/grant_restart_linux_test.go)

Para probar una revocación interrumpida, ejecutar
`go test -mod=readonly -count=1 -run '^TestInterruptedRevocationCanBeRetried$' ./internal/conformance`.
Este ensayo de bucle local interrumpe la escritura del archivo de
comprobación después de guardar la revocación en SQLite y termina el
proceso del proveedor antes de que el propietario reciba la confirmación.
Al reiniciar, el proveedor recibe primero pruebas de lectura nuevas firmadas
con la credencial antigua. Devuelve `DENIED` y cero bytes tanto para el
catálogo como para el objeto. El propietario reenvía la misma revocación
firmada y verifica el resumen de la confirmación. La prueba repite ambas
lecturas denegadas tras el reintento y después de otro reinicio.
[Persistencia de la revocación interrumpida](provider-persistence.md)

La función de Go `WithRecoveryPackage` autentica el paquete en el equipo y
comprueba sus campos antes de pasar el control al código de recuperación del
propietario. Con la contraseña correcta, la prueba local lee el catálogo firmado
y el objeto cifrado de un proveedor simulado en memoria y guarda los cinco bytes
de `hive.txt`. Una contraseña incorrecta y una etiqueta dañada producen el mismo
error de autenticación. En ambos casos la prueba registra cero lecturas del
proveedor y cero llamadas para guardar en el destino. Se conservan sus cinco
bytes originales, `keep\n`, y el directorio de estado sigue vacío. Esto comprueba
la autenticación local del paquete en M1-04. [Lector del paquete](../../internal/storageformat/recovery.go) · [Pruebas del inicio de la recuperación](../../internal/conformance/recovery_reader_test.go)

En la prueba de identidad de M1-04, dos proveedores con claves Ed25519
distintas se turnan en la misma dirección IPv4 de bucle local y el mismo
puerto TCP. El cliente desbloquea el mismo paquete de recuperación para
ambos intentos. El proveedor cuya clave figura en el paquete recibe una
solicitud `READ_BEGIN` para el catálogo y devuelve un reto en ese flujo.
El otro proveedor recibe un intento de negociación TLS y cero solicitudes
de catálogo. Con otro paquete que contiene su propia clave pública, este
segundo proveedor recibe una solicitud en esa misma dirección.
[Prueba de identidad del proveedor](../../internal/conformance/provider_identity_test.go)

El ensayo en Linux repite los tres casos con pilas de red independientes para
el cliente y los dos proveedores virtuales. Cada proveedor escucha en la dirección
de la muestra, `192.0.2.10:4001`. Al cambiar la ruta en el cliente, la siguiente
conexión con el paquete original llega al otro equipo. En el commit `5f41330`,
ambos proveedores registraron un intento TLS. A recibió una solicitud de catálogo
de 81 bytes. B recibió cero bytes de aplicación con el paquete de A y una solicitud
de 81 bytes con su propio paquete. El recuento incluye la cabecera de la trama. [Ejecución verificada](https://github.com/RubioKo/hivescope/actions/runs/44822400007/job/136586322550)

En Linux, con los módulos Go en caché, iproute2 y permiso para crear espacios de
nombres de red con sudo, ejecuta `bash scripts/test-provider-netns.sh` desde la raíz
del repositorio. [Prueba de red aislada](../../internal/conformance/provider_netns_linux_test.go) · [Script del ensayo](../../scripts/test-provider-netns.sh)

La prueba de autorización del catálogo usa las mismas identidades del proveedor
y del equipo de reemplazo en dos conexiones TCP de bucle local. El proveedor
conserva en memoria el permiso vigente y el catálogo cifrado y firmado. En la
primera conexión, el equipo de reemplazo firma el reto del proveedor. El permiso
autoriza la lectura y el equipo recibe los 250 bytes exactos del catálogo de
`hive.txt`. En la segunda conexión, el proveedor emite un reto nuevo para ese
mismo catálogo. Reenviar la primera firma de 64 bytes devuelve `DENIED` y
**cero bytes del catálogo**. La prueba cuenta por separado la trama de error
y los datos del catálogo. El cliente verifica el catálogo firmado en su equipo
para identificar el objeto cifrado que deberá solicitar después.
[Prueba de autorización del catálogo](../../internal/conformance/catalog_proof_test.go)

Para recuperar `hive.txt`, el cliente busca el localizador del objeto en el
catálogo verificado y vuelve a conectarse al proveedor cuya identidad figura
en el paquete. Reutilizar la firma del catálogo para leer el objeto devuelve
`DENIED` y **cero bytes del objeto**. Si firma un reto nuevo para el objeto
con el permiso vigente, recibe los 21 bytes cifrados. El lector local verifica
y entrega los cinco bytes originales, `48 69 76 65 0a`. La prueba de bucle
local también recupera `empty.bin` y `edge.bin` (4 MiB + 1 byte). El
proveedor parte de las muestras de prueba ya cargadas.
[Prueba de recuperación del objeto](../../internal/conformance/object_read_test.go)

El lector en Go verifica el catálogo firmado, autentica cada bloque cifrado
y comprueba la longitud total y el resumen SHA-256 del objeto. Guarda el
contenido descifrado en un archivo temporal dentro del directorio privado que
indica el cliente. Solo permite leer el objeto completo y verificado. Al cerrar
el lector se elimina su archivo temporal. [Lector de catálogo](../../internal/storageformat/catalog.go) · [Lector de objetos](../../internal/storageformat/object.go)

Las tres [muestras publicadas](file-catalog.md) se recuperan byte por byte sin
red. En `empty.bin` se autentica un bloque vacío. De `hive.txt` se recuperan
cinco bytes. De `edge.bin` se recuperan 4 MiB y un byte más en el segundo
bloque. Si ese bloque llega alterado o truncado, el cliente recibe un error
antes de poder leer el contenido. M1-02 aún requiere recuperar los archivos
por la LAN en un equipo de reemplazo con un perfil limpio.

### Guardar en el destino elegido

La API local de Go `SaveObject` guarda en directorios privados. En Linux
requiere un sistema de archivos local que admita el cambio de nombre y las
sincronizaciones indicadas. En Windows requiere NTFS local. El cliente elige
el destino independientemente del nombre que figura
en el catálogo. El modo de solo creación conserva cualquier archivo previo,
incluso uno creado mientras se verifica la copia. Reemplazarlo requiere una
elección expresa. El lector verifica el objeto en un temporal del directorio
de destino, sincroniza el archivo y lo publica mediante un único cambio de
nombre. [Pruebas del guardado local](../../internal/conformance/publication_test.go)

La prueba de interrupción detiene un proceso hijo después de sincronizar el
temporal verificado y justo antes de cambiarle el nombre. Al reabrir el destino
se recuperan exactamente los bytes del archivo anterior. Si el proceso termina
en ese punto, el temporal privado puede quedar en disco con el contenido en
claro. El ensayo comprueba la interrupción del proceso. Un corte de alimentación
requiere pruebas independientes en el sistema de archivos.
[Pruebas de interrupción y sincronización del directorio](../../internal/storageformat/publish_test.go)

Confirmar el guardado requiere sincronizar el directorio. La prueba local en
Windows con NTFS completó esa sincronización. [Sincronización de directorios en Linux](https://man7.org/linux/man-pages/man2/fsync.2.html) · [Sincronización del sistema de archivos en Windows](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-fsa/0de7dc40-9627-437e-a4df-c4696cdc3d02)

### Comprobar un guardado incierto tras reiniciar

Si la sincronización del directorio falla después del cambio de nombre, la API
devuelve `ErrPublicationUncertain`. El archivo nuevo puede estar ya visible y
completo. Para poder comprobarlo más adelante, llama a `SaveRecoverableObject`
con un directorio de estado privado dedicado a esa recuperación. Conserva ese
directorio y el acceso al paquete de recuperación original y al destino elegido.
Un guardado anterior sin registro carece del vínculo necesario para comprobarlo.
[Registro de la recuperación](file-catalog.md)

Tras reiniciar, abre el catálogo firmado original y llama a `ReopenPublication`
con su objeto cifrado, el destino elegido y el directorio de estado. El lector
vuelve a verificar el objeto en otro archivo temporal privado y lo compara byte
por byte con el destino existente. Si coinciden, confirma el guardado tras
sincronizar el archivo de destino y su directorio. Si los bytes difieren,
`ErrDestinationDiffers` indica al cliente que conserve el archivo y pida un
destino nuevo. Si el objeto está inaccesible o falla una sincronización, el
guardado sigue incierto. El cliente de escritorio deberá invocar esta API local
de Go al reabrir una recuperación.

En la prueba de cinco bytes ya terminó el proceso que guardó el archivo. Al
reabrirlo se confirma `Hive\n`. Si una edición local cambia solo el último byte
por `!`, la misma comprobación detecta la diferencia y conserva los cinco bytes
de `Hive!`. [Reinicio tras un fallo de sincronización](../../internal/storageformat/publish_test.go) · [Pruebas de reapertura](../../internal/conformance/reconciliation_test.go)

El descubrimiento y la asignación automáticos corresponden al hito 2. La
redundancia y reparación con tres proveedores, la interfaz de escritorio y el
acceso móvil, el cómputo privado y los servicios persistentes tienen alcance y
criterios propios. [Hoja de ruta](roadmap.md)
