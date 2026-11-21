<!-- docs-pair: recovery-format; revision: 7 -->

[English](../en/recovery-format.md) | [Español](recovery-format.md) | [Índice](README.md)

# Recuperar un archivo en un equipo de reemplazo

**Propuesta de formato v1 para la prueba de escritorio en red local.** La computadora
original de Ana está apagada. Ana conserva su paquete de recuperación y su contraseña,
y puede conectarse al proveedor que guardó su archivo cifrado. Con el paquete, el
equipo nuevo identifica al proveedor, demuestra su derecho a leer y descifra el resultado
localmente. El proveedor puede aceptar o rechazar cada lectura por separado.
Las pruebas en Go cubren el desbloqueo local y dos lecturas autorizadas mediante
TCP en la interfaz de bucle local. El equipo de reemplazo verifica el catálogo y
firma un reto nuevo para el objeto que este identifica. Reutilizar la firma del
catálogo para pedir cualquiera de los dos en una conexión nueva devuelve
`DENIED` y cero bytes del elemento solicitado. [Primera prueba de recuperación](readiness.md)

## Qué conserva Ana

Cuando Ana solicita la exportación, obtiene un archivo binario de 255 bytes para una
carpeta y un acuerdo. Solo la cabecera de 33 bytes queda en claro. Los campos se leen
de izquierda a derecha. Los enteros son sin signo y se codifican con el byte más
significativo primero (big-endian). Los tamaños indicados son exactos.

| Parte | Contenido en orden | Bytes |
| --- | --- | ---: |
| Cabecera | ASCII `HSRP`, byte de versión `01`, sal aleatoria (16), nonce aleatorio de GCM (12) | 33 |
| Cuerpo | Cifrado con AES-256-GCM de la carga útil descrita abajo | 206 |
| Etiqueta | Etiqueta de autenticación completa de GCM | 16 |

La contraseña se codifica en UTF-8 exactamente como se introdujo, conservando los
espacios y los puntos de código Unicode. Se usa Argon2id versión 0x13 con 65 536 KiB
de memoria, tres pasadas, cuatro vías de paralelismo y la sal de la cabecera. El
secreto opcional y los datos asociados de Argon2id quedan vacíos. Su salida de 32
bytes se usa una sola vez como clave AES. GCM autentica la cabecera completa como
datos asociados. Cada exportación, incluso en un reintento, genera una sal y un
nonce nuevos con un generador aleatorio criptográfico. Antes de derivar la clave,
el equipo comprueba el tamaño, los bytes de identificación del formato y la versión.
Si la etiqueta es inválida o la contraseña es incorrecta, la recuperación se detiene
localmente antes de interpretar la carga útil, contactar al proveedor o escribir en
el archivo de destino. Quien obtenga una copia de la exportación puede intentar
adivinar la contraseña sin conectarse. Conviene que el propietario elija una
contraseña larga y única y la guarde por separado. [Perfil de Argon2id](https://www.rfc-editor.org/rfc/rfc9106#section-7.4) · [Parámetros de AES-GCM](https://www.rfc-editor.org/rfc/rfc5116#section-5.2)

| Posición inicial en la carga útil | Campo | Finalidad |
| ---: | --- | --- |
| 0 | `folder_id` (16) | Identificador opaco de la carpeta. |
| 16 | `agreement_id` (16) | Identificador opaco del acuerdo de almacenamiento. |
| 32 | `generation` (8) | Número positivo de generación de la autorización de recuperación registrada por el propietario. |
| 40 | `provider_key` (32) | Bytes de la clave pública Ed25519 fijada para la identidad libp2p del proveedor. |
| 72 | `ipv4` (4), `tcp_port` (2) | Dirección y puerto para la conexión directa en la LAN, en orden de bytes de red. El puerto está entre 1 y 65535. |
| 78 | `recovery_seed` (32) | Semilla Ed25519 para demostrar la posesión de la clave asociada a la autorización de lectura registrada. |
| 110 | `folder_read_key` (32) | Secreto de lectura de la carpeta que permite abrir el catálogo cifrado y recuperar las claves de cada versión. |
| 142 | `catalog_verify_key` (32) | Clave pública Ed25519 fijada para verificar las firmas de quien está autorizado a firmar el catálogo. |
| 174 | `catalog_locator` (32) | Identificador opaco del catálogo cifrado que debe solicitarse primero. |

El propietario guarda por separado la clave de administración de la carpeta.
El equipo de reemplazo genera su propia clave de dispositivo. Antes de pedir un
reto, autentica al proveedor mediante libp2p y compara la clave de identidad
remota con los bytes fijados. La dirección IPv4 orienta la conexión. Si cambia
o deja de ser accesible, sigue siendo obligatorio comprobar la misma identidad.
Este primer perfil usa identidades Ed25519 y conexiones directas TCP/IPv4. Si
la generación o el puerto valen cero o una clave tiene un formato inválido,
el equipo rechaza el paquete antes de conectarse. La clave de esa versión del
archivo queda cifrada dentro del catálogo firmado. [Catálogo y verificación de bloques](file-catalog.md) · [Autenticación entre pares](https://github.com/libp2p/specs/blob/master/tls/tls.md)

Tras autenticar la carga útil, se validan ambas claves públicas Ed25519 fijadas.
Cada una debe representar un punto de la curva Edwards25519 cuya codificación
canónica coincida con los 32 bytes originales. Se rechaza todo punto que dé el
elemento neutro al multiplicarlo por el cofactor ocho. La semilla de recuperación
y la clave simétrica de lectura de la carpeta son dos campos opacos de 32 bytes
cada uno. Para comprobar las claves públicas, el lector en Go usa `SetBytes`,
`Bytes` y `MultByCofactor`. [Decodificación de puntos Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-5.1.3) · [Operaciones sobre puntos en Go](https://pkg.go.dev/filippo.io/edwards25519@v1.2.0#Point)

## Qué firma el equipo de reemplazo

Para cada lectura, el equipo concatena los campos de la tabla y firma el resultado con
Ed25519 puro. El literal se representa con sus bytes ASCII exactos. Cada par
hexadecimal representa un byte y las claves se incluyen como bytes en bruto.
Los números entre paréntesis indican tamaños en bytes. La concatenación se hace
sin separadores, relleno, versiones textuales de las claves ni prefijos de longitud.
El proveedor verifica la firma de 64 bytes con la clave de recuperación activa que
tiene registrada. [Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-5.1)

| Orden | Campos firmados |
| --- | --- |
| 1 | Literal ASCII `HiveScope/recovery-read/v1`, un byte cero y la operación (1). `01` solicita leer un catálogo y `02`, un objeto. |
| 2 | Clave pública del proveedor (32) seguida de la clave pública del equipo de reemplazo (32), ambas obtenidas de la conexión autenticada. |
| 3 | Identificador de carpeta (16), identificador del acuerdo (16) y generación de la autorización (8). |
| 4 | Localizador solicitado (32) y reto aleatorio recién generado por el proveedor (32). |

El equipo de reemplazo construye localmente los bytes que firma a partir de
la lectura que quiere hacer y las dos identidades autenticadas. Solo acepta el
reto de esa lectura pendiente. El proveedor vincula el reto a esa conexión y a esos campos exactos. El reto
caduca 60 segundos después de emitirse según el reloj monotónico del proveedor.
La primera respuesta consume el reto de forma atómica. Cada reintento o reconexión
requiere uno nuevo. Antes de autorizar la lectura, el proveedor verifica la firma,
la autorización vigente en su almacenamiento persistente, su alcance, la pertenencia
del elemento al acuerdo y el periodo de lectura de ese acuerdo. La decisión de lectura
y la revocación se procesan en un orden definido para impedir que una lectura nueva
use la autorización antigua tras confirmarse la revocación. Los bytes ya entregados
permanecen con quien los recibió. Tras reiniciar, el proveedor descarta los retos,
carga la generación más alta que guardó y deniega las lecturas si ese estado no
está disponible. Ante una denegación, el proveedor devuelve cero bytes del catálogo
o del objeto y el equipo de reemplazo conserva intacto el archivo de destino
existente. [Acuerdo y periodo de gracia](storage.md)

## Dos ejemplos con la misma exportación

Todos los datos de prueba son públicos. La notación `a1 × 16` significa dieciséis
bytes con valor 0xa1. Las claves de recuperación, del proveedor y del equipo de
reemplazo corresponden respectivamente a TEST 1, TEST 2 y TEST 3 de la sección 7.1
del RFC 8032. La clave pública de TEST 1024 sirve para verificar el catálogo. Estas
claves están reservadas para ejemplos. [Vectores de prueba públicos de Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-7.1)

La exportación contiene la carpeta `a1 × 16`, el acuerdo `b2 × 16`, la generación
`7`, la dirección `192.0.2.10:4001`, la clave de lectura de la carpeta `e5 × 32`
y el localizador del catálogo `c3 × 32`. El localizador del objeto en el catálogo
de prueba es `d4 × 32`. Solo en este ejemplo se usa la contraseña `HiveScope fixture 1`,
la secuencia ascendente de bytes `00` a `0f` para la sal y la de `10` a `1b` para
el nonce. Los retos usan bytes repetidos para que los ejemplos sean reproducibles.

| Estado en el proveedor | Lectura del catálogo | Lectura del objeto | Resultado esperado |
| --- | --- | --- | --- |
| **A. Autorización activa en la generación 7.** Equipo original apagado, perfil limpio en el equipo de reemplazo, proveedor accesible y acuerdo dentro del periodo de lectura. | Firmar `01` para `c3 × 32` con el reto `11 × 32`. | Tras autenticar el catálogo, firmar `02` para `d4 × 32` con un reto nuevo `22 × 32`. | Ambas lecturas se completan. Tras verificar y descifrar el archivo localmente, el resultado coincide byte por byte con la muestra independiente `48 69 76 65 0a`. |
| **B. Autorización revocada en la generación 8.** El proveedor confirmó la revocación firmada por el propietario y se reinició. El equipo de reemplazo conserva la exportación de la generación 7. | Intentar leer con la autorización antigua y un reto nuevo `33 × 32`. | Por separado, intentar leer el objeto con el localizador conocido y un reto nuevo `44 × 32`. | Incluso con firmas correctas, las solicitudes de la generación 7 devuelven cero bytes de cada elemento. Si el archivo de destino contiene `6b 65 65 70 0a`, conserva exactamente esos bytes. |

La firma identifica la clave de recuperación. La autorización que guarda el proveedor
determina si esa clave sigue dando derecho a leer. Si el proveedor deja de estar
disponible o Ana pierde la contraseña, este ensayo con un único proveedor no permite
restaurar el archivo. Con una exportación antigua y un solo proveedor tampoco se
puede demostrar que el catálogo recuperado sea el más reciente.

## Valores de referencia en bytes

El SHA-256 del paquete de ejemplo es
`943561b2a66368c9f780b44fab56aa89a1d46c57b20a7d3b73b39dc090d548b4`
y su etiqueta final es `954c1dd1d641d7aab3ecdf664d56ef7d`.

Entre corchetes aparece el byte que se repite en el reto. La primera línea
contiene el SHA-256 de los 196 bytes firmados y las dos siguientes contienen
las dos mitades de la firma de 64 bytes. Se firman directamente los 196 bytes.
El hash permite comprobar su codificación. Las cuatro firmas son matemáticamente
válidas, incluidas las dos que el proveedor debe rechazar en el caso B.
Los valores se comprobaron con las claves públicas de prueba del RFC y sus
correspondientes vectores. La prueba de bucle local en Linux comprueba el caso B
con retos aleatorios nuevos tras reiniciar el proceso del proveedor. El ensayo
completo en red local aún requiere un equipo de reemplazo con un perfil limpio.
[Resultados de la prueba](readiness.md)

```text
[11]
051413d2b0f8963776b0757fa880d42dbb719b00ec1cb1e59c26aee6471eec43
fdf3cebfcb16055487868992ba1a08ac944f10bfefba8214f7ce539051d98025
5625bfcd48e01b8919bf78a080478dddafa69f8d0727a6592dd920470436b60a

[22]
81ed1365abcf3e1199a232837f940022f5ad3b1b866bf688199ae385a1e456d4
60c3faf2269e13e4c1dd5f3b2c4869d82213e4fc88cad0db08aab237780b689a
e33e4534314dbe532c9d47e1fc7405f247d7ac313ef3eb755bd18535ddc06806

[33]
5092bbf2018d71746727ae0b1936d4153ca93fb3e790cba191b40397b2083326
93ef07ba8fcb1d785d4904c41a8bae9cb59d7c932be1e402ae8e4e11271b8d7d
08a8923305a86132df74305abe70cba7cf7fed87a79507f2932fcea8d2ce9f06

[44]
76a16f0b3aa8a8934557d0a9c698e4cee327f932d0b2475e2efa62159c0dc403
ce65e6e3d879f9a2e1e384193c5767474a0f974afeae75b522a18dae04dc1c99
25677b6f04d6e6428c8149f5ebb63b0b074edc2f446d01e9be184b001e3bfe07
```
