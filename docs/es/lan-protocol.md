<!-- docs-pair: lan-protocol; revision: 6 -->

[English](../en/lan-protocol.md) | [Español](lan-protocol.md) | [Índice](README.md)

# El primer intercambio de almacenamiento en una red local

Este es el diseño v1 para guardar un archivo cifrado en un proveedor. El
equipo original sube el objeto y su catálogo bajo una misma reserva. El
equipo de reemplazo presenta una prueba distinta para leer cada elemento.
La reserva, el recibo y la prueba de recuperación conservan su significado
al reconectar. En la prueba en Go con TCP de bucle local, el proveedor parte
del catálogo y del objeto cifrados ya cargados. El cliente verifica el catálogo
antes de firmar otro reto para leer el objeto. En Linux, el proveedor con un
almacén de permisos SQLite confirma los cambios de permiso firmados y vuelve
a cargar el último permiso tras reiniciar el proceso. Las subidas quedan
para una etapa posterior.
[Prueba de recuperación](readiness.md)

## Conexión y tramas

La prueba parte de un acuerdo configurado localmente en ambos extremos y
una dirección IPv4/TCP directa. Usa TLS 1.3 de libp2p con identidades
Ed25519 y yamux. Compara la clave de identidad autenticada del proveedor,
en bytes, con la clave fijada. Negocia el protocolo exacto de flujo
`/hivescope/storage/1.0.0`. Una identidad distinta o una versión
incompatible termina el intento antes de enviar datos de aplicación. Cada
flujo bidireccional lleva un intercambio. Varios flujos pueden compartir la
conexión autenticada. [TLS de libp2p](https://github.com/libp2p/specs/blob/master/tls/tls.md) · [yamux](https://github.com/libp2p/specs/blob/master/yamux/README.md)

Para leer el catálogo, el equipo de reemplazo se conecta a la dirección del
paquete desbloqueado con su propia identidad de dispositivo. Entrega a libp2p
el identificador de par derivado de la clave Ed25519 fijada y comprueba la
clave remota autenticada en esa conexión. Solo entonces abre el flujo de
almacenamiento y envía `READ_BEGIN` para el catálogo. Si otro equipo ocupa la
misma dirección, puede recibir el intento de conexión TCP. Si presenta otra
clave, la negociación termina antes de enviarle el localizador del catálogo.
El proveedor que acredita la clave responde con un `CHALLENGE` de 32 bytes en
ese flujo. Después debe comprobar `PROOF` para autorizar la lectura.

Cada trama empieza con cuatro bytes enteros sin signo en orden de red que
indican la longitud del mensaje Protobuf que sigue (2–65.542 bytes). Se
comprueba antes de asignar memoria. El campo 1 contiene el número de
operación obligatorio (VARINT). El campo 2 lleva los datos (LEN) y se omite
si están vacíos. Se envía cada campo una vez, por orden numérico y con la
representación varint más corta. El perfil v1 rechaza cualquier otro campo,
duplicados, tipos de codificación incorrectos, campos de datos explícitamente
vacíos y bytes sobrantes. Los límites de cada trama son independientes de
las lecturas TCP. Los datos admiten hasta 4.096 bytes, salvo `DATA`. Se
divide cada elemento en bloques consecutivos de 65.536 bytes y uno final
más corto cuando haga falta. Cada `DATA` lleva al menos un byte, incluso
para el ejemplo cifrado de archivo vacío. Las firmas y huellas cubren los registros binarios
de los formatos enlazados. [Codificación Protobuf](https://protobuf.dev/programming-guides/encoding/)

Las tablas indican los números de operación en hexadecimal. Dentro de cada
mensaje se concatenan los campos binarios en el orden indicado. Los enteros
son sin signo y en orden de red. `U` incluye ID de carpeta (16), ID de
acuerdo (16), ID de solicitud (16) y huella del manifiesto (32). Las
identidades autenticadas del dispositivo que sube y del proveedor también
delimitan la referencia. `K` ocupa un byte, `01` para el catálogo y `02`
para el objeto. El objeto cifrado mide de 16 a 67.109.120 bytes y el
catálogo firmado de 243 a 497. El cliente comprueba además las longitudes
exactas de su manifiesto o del catálogo verificado.
[Formatos de los elementos cifrados](file-catalog.md)

## Subida y confirmación

| Solicitud inicial | Contenido | Respuesta y efecto en el proveedor |
| --- | --- | --- |
| `01 RESERVE` | Manifiesto completo de 276 bytes | Comprueba el dominio, las identidades, el alcance, los tamaños y que los localizadores sean distintos. Devuelve `STATE` tras guardar la reserva de forma duradera, o el estado existente ante un reintento idéntico. Otro manifiesto con el mismo ID en ese ámbito devuelve `CONFLICT`. |
| `02 STATUS` | `U` | Devuelve `STATE` sin asignar espacio. Comprueba la autorización del dispositivo que sube antes de revelar el estado. |
| `03 PUT` | `U, K` | Para un elemento pendiente con reserva vigente, obtiene el bloqueo de escritura de la solicitud y envía `20 READY` (vacío). El dispositivo envía tramas consecutivas `21 DATA` desde el byte cero y una sola `22 END` (vacía). Devuelve `STATE` con el bit del elemento preparado tras comprobar longitud, huella y persistencia. Si el elemento ya está preparado o la solicitud terminó, devuelve `STATE` de inmediato. |
| `04 FINALIZE` | `U` | Bajo el bloqueo de la solicitud, confirma la subida si ambos elementos están preparados, la autorización sigue vigente y se cumplen los plazos. Si falta un elemento, devuelve el `STATE` pendiente. Si ya se confirmó, devuelve el recibo original. |
| `05 CANCEL` | `U` | Serializa la cancelación con la escritura y la confirmación. Devuelve el `STATE` resultante. Si se confirmó primero, conserva ambos elementos y el recibo. |

En `02`–`05`, un ID conocido con huella distinta devuelve `CONFLICT` al
dispositivo autorizado que sube. Un ID desconocido devuelve el estado
desconocido. Los reintentos reutilizan los datos cifrados, el manifiesto y el
ID originales. Si se corta `PUT`, se descarta el avance parcial de ese
elemento. Tras reconectar, se consulta `STATUS` antes de reenviarlo. Si se
pierde la respuesta a `FINALIZE`, la confirmación sigue pendiente hasta que
el cliente obtiene y verifica el recibo original.
[Manifiesto y recibo](upload-confirmation.md) · [Persistencia del proveedor](provider-persistence.md)

En esta prueba se admiten hasta 256 ID de solicitud conservados por acuerdo
y 4.096 en todo el proveedor. Se cuentan las solicitudes pendientes,
confirmadas, inaccesibles, canceladas y vencidas hasta el cierre del acuerdo
tras la gracia. Se respetan también los límites inferiores del anfitrión.
Al alcanzar el máximo, un ID adicional recibe `CAPACITY`. Un ID existente
todavía puede consultar su estado. Los recibos vigentes y los ID terminales
se conservan aunque admitir una nueva solicitud requiera más espacio.

`23 STATE` empieza con un byte de estado. Los campos restantes son exactos.

| Byte de estado | Campos restantes | Qué muestra el cliente |
| --- | --- | --- |
| `00` | Vacío | Solicitud desconocida. |
| `01` | Máscara de elementos preparados (1), vencimiento absoluto de la reserva (8, segundos Unix UTC) | Pendiente. El bit 0 indica catálogo y el bit 1, objeto. Los demás valen cero. La máscara 3 permite confirmar. |
| `02` | Recibo histórico firmado (132) | Confirmada. Verifica y guarda el recibo de forma duradera antes de marcar la copia como confirmada. |
| `03`, `04` | Vacío | Vencida o cancelada, respectivamente. Una nueva subida requiere un ID nuevo. |
| `05` | Recibo histórico firmado (132) | Se confirmó en el pasado y ahora está inaccesible. Muestra que la copia no está disponible. |

## Recuperación y actualización de la autorización

| Solicitud inicial | Contenido | Intercambio en el mismo flujo |
| --- | --- | --- |
| `07 READ_BEGIN` | `K`, ID de carpeta (16), ID de acuerdo (16), generación (8), localizador (32) | Para una petición bien formada que respeta los límites, devuelve `25 CHALLENGE` con 32 bytes aleatorios nuevos. Este paso no revela si existe el elemento en ese acuerdo. El equipo de reemplazo envía `08 PROOF` con la firma de 64 bytes de esa lectura. |
| `06 GRANT_UPDATE` | Registro administrativo firmado completo (198) | Un par autenticado puede retransmitirlo. Comprueba la firma del propietario, el alcance y la generación. Devuelve `24 GRANT_ACK` con SHA-256 (32) de los bytes exactos del registro tras aceptarlo de forma duradera e invalidar la autorización anterior. Un registro idéntico que sigue vigente recibe el mismo acuse. |

Cada flujo admite un solo reto a la vez. Queda vinculado al flujo, a la
conexión, a ambas identidades y a todos los campos solicitados. La primera
prueba lo consume atómicamente, incluso si es incorrecta. Caduca a los
60 segundos según el reloj monótono del proveedor. Una desconexión o un
reinicio descarta los retos afectados. La admisión de lecturas se serializa
con los cambios de autorización. Una credencial obsoleta, un elemento que no
pertenece al acuerdo, la falta de estado de autorización, una firma inválida
o un período de lectura ya cerrado devuelven `DENIED` y cero bytes del
elemento. Una lectura autorizada de un elemento dañado conocido devuelve
`UNAVAILABLE`. [Lectura firmada y ejemplos](recovery-format.md) · [Identidad y autorizaciones firmadas](identity-custody.md)

Al sustituir una autorización, el proveedor deja de poner datos en cola
para las lecturas activas afectadas e invalida sus retos pendientes antes
de confirmar el cambio. Los bytes ya entregados al transporte todavía
pueden llegar después.

Si la conexión termina antes de verificar `GRANT_ACK`, el propietario
mantiene la revocación pendiente y reenvía los mismos 198 bytes firmados en
un flujo autenticado nuevo. Si al reiniciar el proveedor encuentra una
revocación pendiente válida, deniega la lectura de ambos elementos y solo
acepta ese mismo registro firmado para completar la operación. El propietario
la da por confirmada tras recibir el resumen esperado y el cierre ordenado
del flujo. [Recuperación de una revocación interrumpida](provider-persistence.md)

En una lectura autorizada, el proveedor envía `26 READ_SIZE` con la longitud
cifrada (8), tramas consecutivas `DATA` que suman exactamente esa longitud,
`END` y un cierre ordenado del flujo. Un flujo truncado, una trama sobrante,
una longitud incorrecta o un fallo de integridad abortan la recuperación.
Se verifica el catálogo firmado antes de pedir el objeto en otra lectura.
El archivo local se publica solo tras superar todas las comprobaciones de
formato e integridad.

## Límites, fallos y reintentos

El solicitante cierra su lado de escritura tras su última trama de petición
(`END` para subir datos, `PROOF` para leer). Espera a `READY` o `CHALLENGE`
antes de enviar la siguiente parte. Un `STATE` anticipado para un `PUT` ya
preparado o confirmado termina el intercambio. El dispositivo omite los datos,
cierra su lado de escritura y lee hasta el cierre del proveedor. Si cierra
antes de un `END` o `PROOF` obligatorio, se devuelve `INVALID` cuando sea
posible, se descarta el elemento parcial o el reto y, si hay reserva, se mantiene
hasta su vencimiento. Un `FINALIZE` prematuro devuelve la máscara pendiente
y el vencimiento. Ambos localizadores siguen privados. El proveedor comprueba
el fin ordenado de la petición antes de cambiar el estado registrado de la
solicitud o autorización, o de enviar bytes en una lectura. Cierra su lado de escritura tras la respuesta
final. Un intercambio incompleto deja el éxito sin confirmar.

Se admiten hasta 16 flujos de aplicación activos por proveedor, dos por par
contando todas sus conexiones y un escritor por solicitud. Como máximo 15
plazas pueden atender `PUT` o `READ_BEGIN`. Cuando solo queda la plaza
reservada, un flujo nuevo dispone de dos segundos para presentar su primera
trama completa (hasta 281 bytes tras el prefijo de longitud). Allí solo
se admiten `STATUS`, `CANCEL` y `GRANT_UPDATE`. Las demás operaciones
reconocidas reciben `BUSY`. Una trama sin clasificar o demasiado grande
provoca el cierre abrupto del flujo. La plaza limita la competencia de las
transferencias de datos. Las peticiones de control la comparten y siguen
sujetas al límite por par. Aplica los límites de transferencia del anfitrión en ambas
direcciones. Cada flujo puede anticipar como máximo una trama de datos
mientras espera al disco o a la red. La negociación vence a los 10 segundos.
El flujo vence a los 30 segundos de inactividad o a los 5 minutos en total,
lo que ocurra primero. La espera de la prueba de recuperación usa el plazo
de 60 segundos del reto en lugar del límite de inactividad. Se mide el tiempo
transcurrido con un reloj monótono. El proveedor persiste el vencimiento de
la reserva, como máximo 15 minutos tras la admisión y siempre dentro de la
vigencia del acuerdo. Los reintentos conservan ese plazo. Estos límites de
la prueba están pendientes de medición. El proveedor usa una hora UTC
fiable para los vencimientos absolutos. Si el reloj es incierto o continúa
la conciliación tras el reinicio, las decisiones que dependen de la hora
devuelven `UNAVAILABLE`. [Vencimientos y reloj](provider-persistence.md)

`7f ERROR` contiene exactamente un byte de código y termina el flujo. Una
trama ilegible o demasiado grande provoca el cierre abrupto del flujo en
cuanto se detecta. En la red solo se usan estos códigos. El diagnóstico
local puede añadir detalles. Un registro administrativo auténtico con una
generación anterior o con bytes distintos para la generación actual devuelve
`CONFLICT`. Una autoridad inválida devuelve `DENIED`.

| Código | Significado y acción |
| --- | --- |
| `01 INVALID` | Campos, orden, longitud o huella incorrectos. Corrige la petición o compara los datos cifrados de origen con el manifiesto guardado. |
| `02 DENIED` | Falló la autorización, la prueba o el período de lectura. Detén la operación y comprueba la credencial local, la identidad fijada o el acuerdo. Un lector sin autorización jamás recibe confirmación de que exista un localizador. |
| `03 CONFLICT` | Se reutilizó de forma incompatible un ID, localizador o generación. Conserva el estado aceptado. Consulta el manifiesto o el registro de autoridad guardados. |
| `04 BUSY` | Hay un bloqueo, o se alcanzó un límite de frecuencia o de flujos. Espera antes de reintentar. |
| `05 CAPACITY`, `06 PAUSED` | Un límite de capacidad, incluida una cuota reducida por el anfitrión, bloquea nuevas asignaciones e impide publicar datos parciales. Una pausa cierra la admisión de reservas nuevas y conserva los plazos existentes. Espera a que cambie la situación o cancela la espera local. |
| `07 UNAVAILABLE` | La conciliación, un fallo del almacenamiento o datos dañados impiden atender la petición. Consulta el estado cuando se restablezca el servicio. Conserva el origen local y cualquier destino anterior. |

Tras un vencimiento con resultado incierto, usa un flujo nuevo. Para una
subida, empieza con `STATUS`. Antes de los tres reintentos automáticos,
espera 1, 2 y 4 segundos respectivamente más una demora aleatoria de 0–1
segundo cada vez. Después muestra la operación como pendiente y ofrece
reintentar o cancelar. Un vencimiento o una espera local no acreditan la
cancelación en el proveedor. Una nueva prueba de lectura requiere un reto
reciente.

## El ejemplo de cinco bytes en la red

Para `hive.txt`, una sola trama `DATA` con los 21 bytes del objeto cifrado
empieza por `00 00 00 19 08 21 12 15` en hexadecimal. La trama vacía `END`
es `00 00 00 02 08 22`. Tras preparar ese objeto, `STATE` figura pendiente
con máscara `02`. Preparar el catálogo de 250 bytes cambia la máscara a
`03`. Si se descarta la respuesta final de confirmación y se consulta de
nuevo, debe devolverse el estado `02` con el mismo recibo de 132 bytes. Con
un paquete válido de generación 7, las longitudes de lectura esperadas son
250 y luego 21. Una vez confirmada la revocación de generación 8, los
intentos independientes para ambos localizadores conocidos deben devolver
`02 DENIED` con cero bytes de cada elemento. Son resultados de
interoperabilidad esperados para el futuro prototipo.
[Bytes de referencia](recovery-format.md) · [Huella del recibo](upload-confirmation.md)
