<!-- docs-pair: provider-persistence; revision: 5 -->

[English](../en/provider-persistence.md) | [Español](provider-persistence.md) | [Índice](README.md)

# ¿Qué conserva el proveedor después de un reinicio?

En la primera prueba en LAN, el recibo compromete al proveedor a conservar dos
archivos juntos. Si se corta la luz después de que Luis envíe el recibo, Ana
debe encontrar la misma subida cuando él reinicie. Si el disco pierde después
alguno de los archivos, Ana debe ver que su copia está inaccesible. Este diseño
especifica qué guardar, cuándo confirmarlo y cómo comprobar ambos resultados.
[Confirmación de subida y recibo de referencia](upload-confirmation.md)

## Almacenamiento de permisos en el prototipo Linux

El operador configura un almacén privado con un permiso inicial firmado.
Cuando cambia el permiso, el proveedor guarda el nuevo registro firmado en
SQLite, reemplaza un archivo de comprobación separado que contiene su
resumen y sincroniza el directorio. Tras reiniciar, contrasta esos registros
con la identidad del almacén y el acuerdo antes de autorizar una lectura.

Ante una revocación, el proveedor guarda primero un registro privado de la
operación pendiente. Contiene el resumen del permiso anterior y los bytes
exactos de la revocación firmada por la autoridad de la carpeta. El registro
debe quedar en disco antes de modificar SQLite. Desde ese momento, un
reinicio mantiene bloqueadas ambas lecturas con la credencial antigua. Si el
proveedor se detiene antes de guardar evidencia duradera de la solicitud,
el permiso anterior puede seguir vigente. Los bytes que ya se entregaron al
transporte aún pueden llegar a su destino.

El proveedor reconoce tres combinaciones cuando el registro pendiente es válido.

| Registro firmado en SQLite | Resumen en el archivo de comprobación | Al reiniciar |
| --- | --- | --- |
| Permiso anterior | Permiso anterior | Mantiene las lecturas bloqueadas a la espera de la misma revocación firmada. |
| Revocación | Permiso anterior | Mantiene las lecturas bloqueadas. Un reintento idéntico puede completar el archivo de comprobación. |
| Revocación | Revocación | Mantiene las lecturas bloqueadas. Un reintento idéntico puede completar la eliminación del registro pendiente. |

Al reintentar, el proveedor verifica la firma de la autoridad y su alcance,
y compara los bytes exactos con el registro pendiente. Completa las
escrituras necesarias en SQLite y en el archivo de comprobación, elimina el
registro pendiente, sincroniza el directorio y entonces devuelve `GRANT_ACK`
con el resumen de la revocación. Si se pierde la respuesta a partir de ese
momento, reenviar los mismos bytes firmados permite obtener la confirmación.
La falta de datos o un estado incoherente o desconocido deja el almacén
fuera de servicio hasta conciliarlo. Una base de datos anterior junto con un
archivo de comprobación más reciente mantiene las lecturas bloqueadas.
Una instantánea del volumen completo puede
hacer retroceder todas las comprobaciones locales del proveedor.

El almacén Linux publica el registro pendiente a partir de un archivo sin
nombre. Esto requiere un sistema de archivos compatible con `O_TMPFILE` y
acceso a `/proc/self/fd`. [Archivos temporales sin nombre en Linux](https://man7.org/linux/man-pages/man2/open.2.html)

La prueba de bucle local presupone un reloj de sistema confiable y utiliza
elementos cifrados precargados. El reinicio del proceso comprueba la recuperación del permiso.
Los recibos de subida, el comportamiento del volumen ante cortes eléctricos
y la conciliación del reloj requieren ensayos por separado.
[Prueba de reinicio y requisitos de compilación](readiness.md)

## Volumen de la prueba

La **primera configuración de prueba del proveedor** usará Linux con un volumen
ext4 local, datos en modo ordenado, barreras de escritura y un dispositivo que
respete las órdenes de sincronización. El índice SQLite, su diario de reversión
y ambos archivos cifrados residirán en el mismo volumen. SQLite usará el modo
de diario `DELETE` y el modo de sincronización `EXTRA`. Cada conexión verificará
estos ajustes antes de habilitar el servicio. Así, SQLite sincroniza el directorio
del diario tras eliminarlo para confirmar la transacción. Los archivos externos
requieren además sincronizar correctamente su contenido y sus entradas de directorio.
[Modos de sincronización de SQLite](https://www.sqlite.org/pragma.html#pragma_synchronous) ·
[Sincronización de archivos y directorios en Linux](https://man7.org/linux/man-pages/man2/fsync.2.html) ·
[Modos de datos y barreras en ext4](https://www.kernel.org/doc/html/latest/admin-guide/ext4.html)

Registrar el sistema de archivos, las opciones de montaje, la versión de
SQLite y las capas de almacenamiento con los resultados. Reiniciar el proceso
comprueba la recuperación de la aplicación. Un apagado brusco sobre un volumen
de prueba desechable, con un modelo documentado de pérdida de alimentación,
comprueba la persistencia. Repetir ambas pruebas por sistema operativo y sistema
de archivos antes de habilitar esa configuración de proveedor. Los clientes de
Windows y macOS pueden evaluarse con independencia de su capacidad para emitir
recibos duraderos como proveedores.

## Registros que mantiene el proveedor

| Registro | Qué conserva el proveedor |
| --- | --- |
| Almacén | Directorio privado asociado al volumen y al identificador de almacén configurados, versión del esquema del índice, límites de capacidad y una sola instancia activa del proveedor. Al inicializarlo, hacer persistentes las entradas de directorio creadas hasta llegar al directorio padre preexistente. |
| Acuerdo | Claves públicas del proveedor, del dispositivo que sube y de la autoridad de la carpeta, identificadores opacos de acuerdo y carpeta, cuota física, vencimiento, período de gracia, autorización de escritura, generación de recuperación más alta aceptada y estado firmado del permiso. Conservar el cierre definitivo en disco. |
| Subida | Una fila única por proveedor, dispositivo autenticado que sube, acuerdo e identificador de solicitud. Conservar manifiesto exacto y huella, vencimiento de reserva, compromiso físico, preparación por elemento, estado terminal, estado de los archivos y bytes originales del recibo tras la confirmación. |
| Elemento | Dos espacios distintos por subida, uno para el objeto y otro para el catálogo. Conservar función, localizador opaco, longitud cifrada y resumen esperados, nombre generado por el proveedor y solicitud propietaria. Un localizador solo puede pertenecer a una reserva o elemento confirmado dentro del acuerdo. |

Restringir el índice privado y sus archivos a la cuenta del servicio. Resolver
los nombres generados desde el directorio privado ya abierto. Crear cada archivo
de forma exclusiva y, antes de reutilizarlo, verificar que sea un archivo
regular, con un único enlace físico y en el volumen previsto. Un localizador
remoto sirve para consultar el índice. Los nombres del catálogo de Ana permanecen
cifrados. El estado o los archivos de otro volumen no pueden inicializar un
almacén existente. [Custodia local de identidades y permisos](identity-custody.md)

## Cómo reservar el último espacio disponible

El primer ensayo de admisión parte de un almacén Linux vacío y dedicado, un
acuerdo registrado localmente y un dispositivo autorizado para subir archivos.
`RESERVE` y `STATUS` comprueban la identidad autenticada del dispositivo.
Una transacción SQLite asigna ambos elementos, registra el manifiesto exacto
y contabiliza el espacio antes de responder con el estado pendiente. Al volver
a conectarse, el cliente puede consultar el mismo ID y resumen o reenviar el
manifiesto original. El plazo y el cargo se fijan en la primera admisión.
La reserva por sí sola no publica datos. El ensayo utiliza un reloj de
laboratorio confiable y conserva los ID vencidos. Cada elemento preparado
utiliza la reserva inicial. La emisión del recibo firmado requiere todavía
el paso final de publicación.

Este almacén conserva hasta 64 identificadores de solicitud, sujetos a un
límite local menor. Redondea por separado el tamaño de cada elemento y
presupuesta bloques adicionales para los segmentos de archivo y las entradas
del directorio. Con bloques de 4 KiB, el ejemplo de cinco bytes compromete
16 KiB para el par. SQLite y su diario de transacciones tienen un presupuesto
separado. La reserva contabiliza espacio para escrituras futuras. La asignación
de los archivos empieza cuando el dispositivo envía un elemento.

1. Definir el límite del anfitrión y el margen libre en bytes físicos. Limitar
   las solicitudes activas, los identificadores terminales conservados y los
   escritores simultáneos. Reservar a nivel del anfitrión espacio para el tamaño
   máximo permitido del índice y del diario, el crecimiento de directorios y las
   transacciones de limpieza. Limitar las páginas de la base de datos y prever
   por separado el crecimiento del diario. Asignar a cada acuerdo una parte del
   presupuesto de metadatos para sus subidas y estados terminales. Si estos
   límites no caben, rechazar la reserva antes de aceptar datos del archivo.
   [Límites de SQLite](https://www.sqlite.org/limits.html) · [Acuerdos de almacenamiento](storage.md)
2. Estimar por separado la asignación física máxima de cada elemento cifrado.
   Redondear a la unidad de asignación del volumen y prever metadatos de
   extensiones y preasignación. Un elemento parcial ocupa su espacio reservado.
   Reiniciarlo en ese mismo espacio después de hacer persistente el descarte.
   Si se presupuestaron dos archivos, no crear una tercera copia. Los bytes
   asignados cuentan hasta que la limpieza sea duradera.
3. Con un único bloqueo de admisión para el anfitrión, conciliar los bytes
   físicos ya asignados y la **parte todavía sin asignar** de cada compromiso,
   incluidos los metadatos. La suma más el nuevo compromiso debe caber en el
   límite del anfitrión. Repetir para los límites del acuerdo y del solicitante.
   El espacio que el volumen declara disponible para esta cuenta debe cubrir
   todos los compromisos aún sin asignar, incluido el nuevo, y el margen libre.
   Comprobar también los inodos disponibles para los archivos previstos.
   Persistir la reserva antes de liberar el bloqueo.
   [Capacidad del sistema de archivos en Linux](https://man7.org/linux/man-pages/man3/statvfs.3.html)
4. Serializar la contabilidad de asignaciones entre escritores. Antes de
   autorizar otra escritura acotada o asignación de metadatos, comprobar el
   espacio, el margen y los presupuestos. Contabilizar las escrituras en curso.
   Si falta espacio o falla la escritura o sincronización, detener la escritura
   afectada. Consultar el estado recuperado antes de reintentar. Otras aplicaciones
   pueden agotar el volumen después de admitir una reserva. Una subida incompleta
   sigue pendiente aunque se haya reservado espacio.

Para el ejemplo de cinco bytes, una unidad de asignación de 4 KiB implica dos
previsiones redondeadas de 4 KiB cada una para el objeto de 21 bytes y el
catálogo de 250 bytes. El sistema de archivos y el índice requieren presupuesto
adicional. Contabilizar solo 271 bytes físicos permitiría admitir más subidas
de las que caben en el equipo.

Pausar las ofertas bloquea las reservas nuevas. Si Luis **reduce un límite**
por debajo de un compromiso existente, se conservan los archivos confirmados
y se informa del exceso. Se suspenden las nuevas escrituras y finalizaciones
que excedan el presupuesto revisado, incluso en una reserva pendiente. Quien
sube ve que falta capacidad y puede esperar o cancelar. La reserva conserva
su vencimiento y los bytes ocupados siguen contando. Elevar el límite puede
permitir reanudar la misma subida si aún está vigente. Reducirlo no permite
retirar una copia confirmada durante su acuerdo de retención.

## ¿Qué reloj decide el vencimiento?

Los plazos persistidos se rigen por la hora UTC confiable del proveedor.
La ventana de escritura incluye el instante inicial y excluye el vencimiento
del acuerdo. Una reserva solo puede finalizar antes de su propio vencimiento
y del vencimiento del acuerdo. La gracia de lectura empieza en el instante
exacto del vencimiento del acuerdo y termina exactamente siete días después,
sin incluir ese instante final. Durante la gracia, los datos almacenados siguen
en modo de solo lectura y los cambios de permiso firmados aún pueden restringir
quién los lee. Registrar el vencimiento observado o el cierre
definitivo antes de recuperar espacio. Un cambio de hora posterior no puede
reabrir una reserva terminal o un acuerdo cerrado. La prueba usa un plazo
fijo sin renovación. [Plazos y estados en LAN](lan-protocol.md) · [Política de acuerdos](storage.md)

Guardar en el índice el mayor valor UTC confiable usado para decisiones de
vigencia, en la misma transacción que cualquier cambio de estado que produzca.
Verificar la hora UTC al iniciar y al salir de suspensión. En esta prueba
basta con un reloj controlado de laboratorio o una referencia horaria confiable
independiente. Durante la ejecución comparar el cambio del reloj civil con el
tiempo transcurrido según el reloj monotónico. Si el reloj retrocede por debajo
del valor persistido o se detecta un salto inexplicado, suspender lecturas,
escrituras y limpiezas que dependan de la hora hasta volver a verificarla.
Una solicitud autenticada que dependa de esta decisión recibe `UNAVAILABLE`.
Entretanto se mantienen contabilizadas las reservas y los bytes ocupados.
Una fecha futura inverosímil no puede activar la recolección de datos. Volver
a verificar la hora antes de eliminar archivos por vencimiento. Los plazos
de streams y retos usan tiempo transcurrido y se descartan al reiniciar.

## De la reserva a la publicación

En `PUT`, cada solicitud admite un solo escritor para ambos elementos. El
proveedor Linux escribe en un archivo sin nombre en su volumen privado.
Comprueba el tamaño declarado, el SHA-256, la trama final `END` y el cierre
ordenado de la solicitud. Luego sincroniza el archivo, lo enlaza en el espacio
reservado, sincroniza el directorio y registra el elemento antes de responder.
Al cerrar una transferencia interrumpida se descarta su archivo sin nombre.
El otro elemento conserva su espacio y la reserva inicial sigue vigente.
Tras reiniciar, el proveedor coteja cada archivo enlazado con el manifiesto
antes de comunicar su estado.

| Etapa | Orden requerido |
| --- | --- |
| Admitir | Detectar repeticiones de la misma solicitud dentro de su ámbito, asignar ambos espacios y confirmar capacidad y propiedad en SQLite. Un manifiesto en conflicto conserva intacto el original. Después, crear los archivos en sus ubicaciones privadas definitivas. |
| Preparar | Permitir un escritor por solicitud. Verificar longitud y resumen de cada archivo completo, sincronizar primero el archivo y después su directorio. Confirmar su marca de preparado. La autorización de lectura consulta las asociaciones ya publicadas. |
| Publicar | Serializar la finalización con cancelaciones, vencimientos y cambios de autorización. Comprobar de nuevo que ambos elementos están preparados y que la escritura sigue permitida. En una transacción SQLite publicar ambos localizadores, convertir el uso reservado en uso almacenado y guardar el recibo firmado exacto. Responder después de la confirmación duradera. Ante un resultado de confirmación incierto, reabrir SQLite antes de comunicar el estado. |
| Cancelar o vencer | Persistir el estado terminal de la solicitud antes de recuperar el espacio. Detener y cerrar sus escritores, eliminar únicamente sus archivos, sincronizar el directorio y después actualizar el espacio en SQLite. Reanudar la limpieza pendiente al reiniciar. Tras la confirmación, el acuerdo rige la retención. |
| Cerrar | Tras el vencimiento del acuerdo y del período de gracia de lectura, denegar lecturas y escrituras antes de limpiar. Conservar un registro de cierre que impida reutilizar el identificador del acuerdo. Contabilizar los archivos abiertos hasta que el último lector los cierre y termine la limpieza. |

## Reapertura del almacén

Antes de atender el estado afectado, adquirir el bloqueo de instancia, verificar
el volumen y el identificador de almacén previstos, abrir el índice existente,
dejar que SQLite recupere su diario, comprobar la integridad del índice y
conciliar propiedad, capacidad y permisos. Si falta el índice de un directorio
configurado o es ilegible, el almacén entra en mantenimiento. Inicializarlo
vacío perdería revocaciones confirmadas y permitiría prometer dos veces el
mismo espacio. Restaurar una instantánea antigua del volumen entero también
puede retroceder un índice que aparenta ser válido. La prueba presupone un
proveedor y un sistema de almacenamiento honestos. No acredita resistencia a
un anfitrión que restaure tal instantánea. [Recuperación y supuestos de SQLite](https://www.sqlite.org/atomiccommit.html)

Volver a comprobar longitudes y resúmenes SHA-256 por partes acotadas y de un
archivo a la vez en esta prueba. Si los bytes de un elemento pendiente coinciden,
completar ambas sincronizaciones y confirmar la marca de preparado. Si el elemento
propio está incompleto o difiere, descartarlo, retirar la marca y volver a pedir
ese elemento mientras siga vigente la reserva. Si falta un archivo confirmado
o está dañado, marcar la copia como inaccesible. Conservar el recibo, bloquear
la lectura de ambos elementos y mantener su propiedad y los cargos aplicables.
Una nueva consulta de estado no repone datos perdidos ni firma otro recibo.

Durante la prueba, volver a comprobar la pareja confirmada antes de comunicar
que está íntegra o iniciar una lectura. Autenticar primero la solicitud,
respetar los límites de E/S del anfitrión y verificar los mismos archivos
abiertos que se van a servir. Un error de E/S durante la transferencia
interrumpe esa lectura. Ana verifica igualmente el catálogo y el objeto
completos antes de publicar el archivo recuperado en claro.

Registrar las entradas inesperadas como incidencia de mantenimiento local.
Dejarlas en su sitio, contabilizar el espacio ocupado y suspender la admisión
hasta conciliar propiedad y presupuesto. Un volumen desmontado, enlace
inesperado, fallo de integridad o versión de esquema desconocida requiere
atención local. La conciliación usará una cola acotada y respetará el límite
de ancho de banda de disco del anfitrión. Al reiniciar, se descartan todos los
retos de lectura pendientes y se carga la generación registrada del permiso
antes de emitir otro. [Operaciones y límites en LAN](lan-protocol.md)

## Evidencia que deberá aportar el primer proveedor

| ID | Interrupción o cambio | Resultado esperado al reabrir |
| --- | --- | --- |
| D1 | Detener tras sincronizar el archivo o el directorio y antes de marcarlo preparado. Repetir con un elemento truncado. | Los bytes coincidentes quedan preparados tras verificarse. Solo el elemento incompleto vuelve a empezar desde el byte cero. Ambos localizadores siguen privados hasta publicar. |
| D2 | Detener entre la preparación de ambos elementos y la publicación final. En otra prueba, perder la respuesta después de confirmar. | En la primera ejecución hay cero localizadores publicados. En la segunda, las consultas devuelven el mismo recibo, un cargo y dos elementos legibles con un permiso válido. |
| D3 | Hacer competir dos solicitudes por el último espacio. Agotar después el disco durante la sincronización de la ganadora. | Se admite como máximo una reserva. La escritura incompleta queda sin confirmar. Un reinicio o una limpieza interrumpida sigue contabilizando cada asignación propia hasta liberarla. |
| D4 | Dañar un catálogo confirmado, desmontar el volumen de prueba o introducir un archivo sin propietario conocido. | La copia dañada figura como inaccesible y conserva el recibo. Un volumen ausente nunca aparece como almacén nuevo. Los archivos desconocidos quedan intactos y se contabilizan. |
| D5 | Reiniciar justo después de confirmar una revocación. Por separado, interrumpir el borrado de una pareja vencida antes de confirmar la liberación del espacio. | La clave de recuperación vieja recibe cero bytes del catálogo y del objeto. Repetir la limpieza elimina solo la pareja propia y libera el espacio una vez. |
| D6 | Reducir el límite por debajo de una reserva vigente. Por separado, reiniciar con el reloj atrasado respecto a su última hora registrada y después indicar una fecha sin verificar posterior a la gracia. | El límite reducido impide finalizar y conserva el cargo reservado. Si la hora es incierta, el proveedor indica que no está disponible para las solicitudes que dependan de ella y conserva archivos y cargos. Restablecer la hora confiable en el instante de vencimiento para comprobar la gracia de solo lectura y en su fin para verificar el cierre. |

Las seis filas describen observaciones futuras. Registrar estados, recuentos
de bytes, cuota antes y después del reinicio y resumen del recibo sintético.
Excluir del informe rutas privadas, claves, paquetes de recuperación y contenido.
