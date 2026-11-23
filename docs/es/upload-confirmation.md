<!-- docs-pair: upload-confirmation; revision: 3 -->

[English](../en/upload-confirmation.md) | [Español](upload-confirmation.md) | [Índice](README.md)

# ¿Cuándo puede Ana dar su copia por guardada?

**Contrato propuesto para un archivo y un proveedor en la red local.** Ana envía
el objeto cifrado y su catálogo, también cifrado y firmado. Aunque la barra de
progreso llegue al 100 %, la subida sigue **pendiente de confirmación** hasta que
su equipo verifica y guarda el recibo del proveedor por ambos elementos. El recibo
deja constancia de lo que el proveedor aceptó en ese momento. Recuperar el archivo
en un equipo con un perfil limpio es otra prueba del hito 1. [Ejemplo de archivo y catálogo](file-catalog.md)

## Una solicitud para los dos elementos

Antes del envío, el equipo de Ana guarda de forma duradera un identificador
aleatorio nuevo de 16 bytes para la solicitud, la identidad del proveedor
elegido, el manifiesto que se describe a continuación y la ubicación de los
bytes cifrados originales. Reutiliza esos datos si se agota el tiempo de espera
o se reinicia el equipo. El proveedor reconoce el identificador dentro del
ámbito de su propia identidad, el equipo autenticado que envía los datos y el
acuerdo. Solo ese equipo, con la autorización prevista en el acuerdo, puede
escribir o consultar el estado de su subida. La autorización de recuperación
permite efectuar lecturas independientes.

La huella del manifiesto es el SHA-256 de los siguientes campos en bruto,
concatenados en el orden indicado. Los literales de dominio se codifican en
ASCII, los enteros son sin signo y se codifican con el byte más significativo
primero (big-endian), y las huellas ocupan 32 bytes en bruto.

| Orden | Campos del manifiesto |
| --- | --- |
| 1 | `HiveScope/upload/v1`, un byte cero, la clave pública del proveedor y la del equipo que envía los datos (32 bytes cada una) |
| 2 | Identificador de carpeta, de acuerdo y de solicitud (16 bytes cada uno) |
| 3 | Localizador del objeto (32), tamaño cifrado (8), SHA-256 del objeto cifrado |
| 4 | Localizador del catálogo (32), tamaño cifrado y firmado (8), SHA-256 del catálogo firmado completo |

El proveedor comprueba ambas longitudes según los [límites de formato del ensayo](file-catalog.md)
y reserva ambos localizadores para ese acuerdo. Rechaza los que ya pertenecen a
otra solicitud activa o a un elemento almacenado. En una sola transacción duradera
de SQLite guarda el manifiesto, la fecha límite absoluta de la reserva, qué archivos
pertenecerán a la solicitud y cuánto espacio se compromete a guardar. La reserva
vence, como máximo, al expirar el acuerdo. Para admitir solicitudes concurrentes,
comprueba y asigna el espacio de forma serializada según las cuotas del anfitrión y
del acuerdo, los límites por solicitante, el espacio físico disponible y el margen
que el anfitrión desea mantener libre. El cálculo incluye el espacio asignado por el
sistema de archivos, los temporales y el coste del índice y del diario de transacciones.
Los bytes que ya se escribieron consumen parte de la reserva existente. El proveedor
vuelve a aplicar los límites durante la escritura. [Cuotas de almacenamiento](storage.md)

## Qué sobrevive a una interrupción

El proveedor asigna a los dos archivos ubicaciones exclusivas en su directorio
de almacenamiento. Puede mantenerlos allí desde la primera escritura hasta la
confirmación final de la transacción. Las lecturas autorizadas solo pueden
acceder a ellos cuando el índice ya está confirmado. Cada solicitud admite una
sola escritura a la vez. La cancelación, la limpieza y la confirmación se coordinan
con esa escritura para que se ejecuten en un orden definido. Si se confirma
primero, una cancelación posterior devuelve el recibo y ambos archivos siguen
sujetos a las condiciones de retención del acuerdo.

| Último estado duradero | Qué hace el proveedor | Qué indica un reintento |
| --- | --- | --- |
| Sin reserva | Consultar el estado no modifica nada. Solo una solicitud de reserva bajo un acuerdo vigente puede asignar espacio. | La solicitud es desconocida. Ana puede repetir la reserva original con el mismo identificador y manifiesto. |
| Con reserva, posiblemente recibiendo datos | Recibe los datos sin superar el tamaño declarado. Para cada elemento comprueba la longitud exacta y la huella, sincroniza el contenido del archivo con el almacenamiento y hace duradera su entrada de directorio. Solo entonces registra de forma duradera que el elemento está preparado. | Se conservan la reserva y su vencimiento. El proveedor devuelve la lista de elementos completos y preparados. Si un elemento llegó parcialmente, el proveedor descarta esa parte y el cliente lo reenvía desde el byte cero. Todavía no se puede leer ninguno de los dos. |
| Ambos elementos preparados | Bajo el mismo bloqueo de la solicitud, vuelve a comprobar el permiso de escritura, el acuerdo y el vencimiento de la reserva. En una sola transacción duradera de SQLite publica las dos entradas de los localizadores y su pertenencia al acuerdo, convierte la cuota reservada en espacio almacenado y registra el recibo firmado y el estado confirmado. | Mientras la transacción no se confirme, Ana sigue viendo la subida como pendiente. Si los dos elementos siguen preparados y cumplen las condiciones, puede reintentar la confirmación sin retransmitirlos. |
| Subida confirmada | Solo envía el recibo una vez que la base de datos confirma la persistencia de la transacción. Conserva los bytes originales del recibo. | La misma solicitud devuelve exactamente ese recibo. Las reconexiones y las confirmaciones duplicadas no crean archivos ni consumen más cuota. |
| Reserva vencida o solicitud cancelada antes de la confirmación | Registra el estado como definitivo, detiene las escrituras, elimina únicamente los archivos de la solicitud y hace duradera la limpieza. Después libera el espacio ocupado. | Devuelve el estado definitivo. Una nueva subida requiere otro identificador de solicitud y una nueva admisión. |

Al arrancar, el proveedor recupera SQLite y concilia las ubicaciones registradas
con los archivos antes de atender las lecturas afectadas, informar del estado o
admitir nuevas solicitudes. Si sobrevive un elemento sin la marca de preparado,
verifica sus bytes exactos y completa la sincronización con el almacenamiento
antes de marcarlo como preparado, o lo descarta y pide que se retransmita.
También contrasta las marcas de preparado con sus archivos. Si falla la
sincronización o es incierto el resultado de la transacción, el proveedor vuelve
a abrir el estado duradero y sigue la fila que corresponda, sin responder que la
subida se completó. Si falta un elemento preparado o está dañado, borra su marca
y comunica que lo necesita hasta que se sustituya y verifique. Si falta un archivo
previamente confirmado o está dañado, indica **no disponible** y conserva el
recibo histórico. Un índice ausente o incoherente exige una conciliación antes
de considerar desconocido un identificador o reutilizar el espacio.

SQLite garantiza la atomicidad dentro de su propia base de datos. Los dos
archivos cifrados requieren operaciones de persistencia adicionales. En Linux,
también hay que sincronizar el directorio padre de un archivo nuevo para que su
nombre sobreviva a una caída. Antes de anunciar esta garantía, el prototipo
deberá verificar la persistencia del archivo, del directorio y de la base de
datos en cada sistema de archivos y sistema operativo que admita.
[Confirmación atómica en SQLite](https://www.sqlite.org/atomiccommit.html) · [Configuración de durabilidad de SQLite](https://www.sqlite.org/pragma.html#pragma_synchronous) · [fsync en Linux](https://man7.org/linux/man-pages/man2/fsync.2.html)

## Qué acredita el recibo en un reintento

En la primera implementación, ambos espacios ya pertenecen al mismo
manifiesto. Guardar el recibo firmado en la fila confirmada de SQLite hace
visibles ambos localizadores para quienes tienen permiso de recuperación y
convierte el único cargo de espacio reservado en espacio almacenado. Un
reintento recupera el recibo original. El solicitante comprueba la identidad
fijada del proveedor, el manifiesto y el vencimiento pactado. Guarda esos
132 bytes en su estado privado antes de confirmar la copia. Si se pierde
un elemento confirmado, ambas lecturas quedan indisponibles y se conservan
el recibo original y el cargo de esa solicitud.

El proveedor firma con su identidad Ed25519 fijada de antemano la concatenación
de `HiveScope/stored/v1`, un byte cero, la huella del manifiesto, el instante
de confirmación (8) y el vencimiento del acuerdo vigente en ese instante (8).
Ambos tiempos se expresan en segundos Unix en UTC. Añade la firma Ed25519 pura
de 64 bytes y guarda esos bytes exactos como parte de la transacción. Ana
verifica la firma, vuelve a calcular la huella a partir del manifiesto que guardó,
comprueba el vencimiento frente a las condiciones pactadas y guarda el recibo
de forma duradera. Marca la copia como confirmada cuando el proveedor autenticado
informa de que la subida está confirmada. Si indica que está indisponible, Ana
ve **copia no disponible** incluso si el recibo histórico es válido.

- Si coinciden el identificador dentro de su ámbito y el manifiesto, el proveedor
  devuelve el estado existente antes de intentar otra admisión, incluso si ha
  pausado las nuevas reservas. El mismo identificador con un manifiesto distinto
  produce un conflicto y conserva la solicitud original.
- Una pausa detiene las nuevas reservas y conserva el vencimiento de las ya
  admitidas. Tras vencer el acuerdo, el periodo de gracia para lecturas impide
  confirmar escrituras nuevas. El equipo autorizado que hizo la subida aún puede
  recuperar su recibo histórico durante ese periodo.
- El proveedor conserva los recibos y los identificadores de solicitudes ya
  finalizadas mientras dure el acuerdo, incluidas sus renovaciones y el periodo
  de gracia para lecturas. Limita el número y tamaño de los registros al admitir
  la solicitud. Una vez cerrado definitivamente el acuerdo, rechaza toda escritura
  posterior y nunca reutiliza su identificador. Si llega un reintento tardío después
  de la eliminación de los datos, ese reintento no puede recrear la subida.

## Recorrido y criterios de aceptación del prototipo

Para el [ejemplo de cinco bytes](file-catalog.md), Ana envía **21 bytes del objeto
cifrado y 250 bytes del catálogo firmado**. Esos 271 bytes de datos ocupan una
sola reserva. El proveedor contabiliza aparte el espacio adicional que realmente
ocupan en disco.

| Punto de interrupción del ensayo | Resultado esperado tras reiniciar |
| --- | --- |
| El objeto está preparado. Solo ha llegado la mitad del catálogo. | El proveedor indica que solo el objeto está preparado. Ana reenvía los 250 bytes del catálogo con la reserva original. |
| Los dos archivos ya están guardados de forma duradera. Se detiene el proveedor justo antes de confirmar la transacción de la base de datos. | Ninguno de los localizadores permite leer datos. Si siguen cumpliéndose las condiciones, se reintenta la confirmación y se publican ambos elementos una sola vez. |
| Se confirma la transacción. Se pierde la respuesta, se reinicia el proveedor y se reintenta dos veces. | Al principio Ana ve la subida pendiente de confirmación. Ambos reintentos devuelven el mismo recibo firmado. Hay una sola subida confirmada, dos entradas de localizadores, 271 bytes cifrados y una sola conversión de cuota reservada en ocupada. La recuperación con el paquete válido devuelve los cinco bytes originales. |
| Dos clientes compiten por el espacio necesario para un solo par de archivos. | Como máximo una reserva obtiene el último espacio disponible en todo el anfitrión. |
| Un reintento concurrente usa el mismo identificador dentro de su ámbito, con una huella distinta. | El proveedor devuelve un conflicto. Se conservan el manifiesto, los archivos y la cuota comprometida para la solicitud original. |
| La reserva vence durante la subida o el disco se llena antes de la última sincronización. | No se emite recibo. El vencimiento inicia la limpieza. Si falla la escritura, la copia figura como pendiente. El elemento afectado solo puede reintentarse bajo la misma reserva mientras esta siga vigente. Los bytes ocupados se siguen contabilizando hasta la limpieza. |

### Recibo de referencia cuando se pierde la respuesta

Se usa la clave TEST 2 para el proveedor y TEST SHA(abc) para el equipo que
sube los datos, ambas del RFC 8032 §7.1. La carpeta, el acuerdo, los localizadores,
los tamaños y las dos huellas son los del ejemplo anterior de `hive.txt`. El
identificador de la solicitud son dieciséis bytes de valor hexadecimal `15`.
La transacción se confirma en `1700000000` y el acuerdo vence en `1702592000`.
El manifiesto de 276 bytes tiene la huella
`5cef2907ccccffa568c547206a2a4324294367cb9d54695ba33671d48a149e03`.
Se firman los primeros 68 bytes del recibo. Al añadir la firma, el SHA-256
del recibo de 132 bytes es
`ff68ed2f38d36637e790a353415e3d69c92cdf6eabcfa5639250ae48b2175e2d`.
[Vectores públicos de prueba de Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-7.1) · [Identificadores del ejemplo de recuperación](recovery-format.md)

La firma de referencia se muestra en dos mitades consecutivas.

```text
d391894a4b1772edefcca619d2ffad9cc9d67e223a3f0f8f643bfc1a4db85377
8fc23139c2702f0713d2d5d9635ea5222935924ecfac6d115465a0b228510405
```

La firma del ejemplo se verificó con la clave pública del proveedor. Al cambiar
únicamente la longitud del objeto de 21 a 22 bytes, cambió la huella y la firma
original dejó de ser válida para ese manifiesto. Los ensayos TCP en Linux usan
el mismo objeto y catálogo para probar un envío interrumpido y la pérdida de
la respuesta de publicación. Al reintentar, el propietario guarda el recibo
firmado original y un equipo de reemplazo autorizado recupera los cinco bytes.
[Ensayos de subida y recuperación](readiness.md)
