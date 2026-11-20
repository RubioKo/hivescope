<!-- docs-pair: readiness; revision: 10 -->

[English](../en/readiness.md) | [Español](readiness.md) | [Índice](README.md)

# Diseño y criterios de aceptación del primer prototipo

La computadora de Ana ha dejado de encender. ¿Puede recuperar un archivo en
otro equipo con solo su paquete de recuperación, su contraseña y el proveedor
que guarda la copia cifrada? ¿Puede ese mismo proveedor rechazar el mismo
paquete después de confirmar una revocación y reiniciarse? Estos dos resultados
definen el primer prototipo de escritorio.

**Estado de la fase documental.** El ensayo en red local tiene un diseño
bilingüe, valores de referencia sintéticos y un procedimiento de aceptación.
Aún queda por ejecutar el primer cliente y proveedor de HiveScope. La
comprobación documental revisa los documentos y sus enlaces. Solo una ejecución
del prototipo permitirá acreditar una recuperación, una revocación o la
persistencia en un sistema de archivos concreto.

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
amd64. Los casos M1-02 a M1-10 quedan pendientes de ejecutar con un nodo en
funcionamiento.

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

Desde la raíz del repositorio, con Go 1.26 o posterior, ejecutar una vez
`go mod download` y después `go test -mod=readonly -count=1 ./...`. Las
pruebas usan datos sintéticos. Tras preparar las dependencias,
también pasan con el proxy de módulos desactivado. Reproducen el paquete
de recuperación de 255 bytes, los cuatro ejemplos de lectura firmada, el
objeto de 21 bytes, el catálogo firmado de 250 bytes y el recibo de 132
bytes. El contenido recuperado es `48 69 76 65 0a`. La contraseña incorrecta,
el bloque sustituido, la firma de catálogo alterada y el recibo contrastado con
un manifiesto modificado fallan sus respectivas comprobaciones. [Objeto y catálogo](../../internal/conformance/snapshot_test.go) · [Recuperación y lecturas](../../internal/conformance/recovery_test.go) · [Recibo](../../internal/conformance/receipt_test.go)

El ejemplo incluye una firma válida hecha con una clave antigua. Para
demostrar que esa clave pierde el acceso tras la revocación hace falta ejecutar
al proveedor.

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

Si la sincronización del directorio falla después del reemplazo,
la API devuelve `ErrPublicationUncertain`. El nuevo
archivo completo puede estar ya visible. El siguiente paso es resolver esa
incertidumbre antes de reintentar. Recuperar por separado la misma versión
autenticada y comparar el destino byte por byte. Si coincide, sincronizar el
archivo y su directorio antes de confirmar. Si difiere, conservar el destino
y pedir otra ruta. [Recuperación tras un guardado incierto](file-catalog.md)

El descubrimiento y la asignación automáticos corresponden al hito 2. La
redundancia y reparación con tres proveedores, la interfaz de escritorio y el
acceso móvil, el cómputo privado y los servicios persistentes tienen alcance y
criterios propios. [Hoja de ruta](roadmap.md)
