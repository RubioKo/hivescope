<!-- docs-pair: file-catalog; revision: 2 -->

[English](../en/file-catalog.md) | [Español](file-catalog.md) | [Índice](README.md)

# Cómo recupera Ana el archivo correcto

**Formato propuesto para la prueba con un solo archivo en la red local.** Ana
abre el paquete de recuperación en otra computadora. El paquete apunta a un
catálogo cifrado. Tras verificar su firma, el equipo puede abrirlo y averiguar
qué objeto debe solicitar, qué clave descifra esa versión y qué bytes debe recibir.
El proveedor guarda el catálogo y el objeto con localizadores opacos. Puede
observar sus tamaños y cuándo se leen. El nombre del archivo y la clave de la
versión quedan dentro del catálogo cifrado. [Paquete de recuperación y autorización de lectura](recovery-format.md)

## Cómo se guarda el catálogo

La versión 1 describe un archivo inmutable de entre 0 y 64 MiB. Este límite se
propone para el primer ensayo. Al exportar el paquete, el propietario fija el
localizador del catálogo. Cada nueva instantánea recibe localizadores aleatorios
nuevos y requiere otra exportación que apunte a ella. La exportación anterior
sigue apuntando a su instantánea original. La autenticación permite saber qué
instantánea se firmó. Un equipo de reemplazo que solo cuenta con una exportación
antigua y un único proveedor no puede determinar si esa instantánea es la más reciente.

Las definiciones siguientes concatenan los bytes en bruto en el orden indicado.
Cada literal de texto usa ASCII y solo lleva un byte cero al final cuando se
indica por separado. Todos los enteros son sin signo y se codifican con el byte más
significativo primero (big-endian). Todas las etiquetas de autenticación ocupan 16 bytes.

| Partes en disco, en orden | Codificación |
| --- | --- |
| Cabecera | Los cuatro bytes ASCII `HSC1`, una sal aleatoria nueva de 32 bytes y después `u32(entry_bytes)`. La cabecera ocupa 40 bytes. |
| Entrada cifrada | Exactamente `entry_bytes` bytes de texto cifrado con AES-256-GCM, seguidos de su etiqueta. |
| Firma | Firma de 64 bytes con Ed25519 puro. Ana fijó en el paquete la clave pública de quien firma el catálogo. |

Del paquete desbloqueado se toman, en este orden, el identificador de carpeta (16 bytes),
el identificador del acuerdo (16) y el localizador del catálogo (32). Juntos forman
`context`. Para derivar la clave de catálogo de 32 bytes con HKDF-SHA-256, se usan
`folder_read_key` como material de clave de entrada, la sal de la cabecera y la cadena
exacta `HiveScope/catalog-key/v1` como parámetro de información. Se usa un nonce de 12
bytes a cero **una sola vez por clave derivada**. Cada nuevo cifrado del catálogo
requiere una sal nueva generada con aleatoriedad criptográfica. Si se reintenta el
envío, se retransmiten los bytes cifrados originales. [HKDF](https://www.rfc-editor.org/rfc/rfc5869#section-2) · [AES-GCM](https://www.rfc-editor.org/rfc/rfc5116#section-5.2)

Los datos asociados de GCM son `HiveScope/catalog/v1`, un byte cero, `context` y la
cabecera completa. Se firman `HiveScope/catalog-sign/v1`, un byte cero, `context`,
la cabecera completa, el texto cifrado y la etiqueta. Antes de reservar memoria,
el equipo comprueba los bytes que identifican el formato, la longitud declarada de
la entrada (123–377 bytes) y el tamaño total exacto (120 más la longitud de la entrada).
Rechaza los datos truncados y los bytes sobrantes. Verifica la firma con
`catalog_verify_key` y la autenticidad de la etiqueta GCM antes de interpretar un
campo de la entrada o solicitar un objeto. La clave que firma el catálogo es distinta
de la clave de firma que el cliente usa para la recuperación. [Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-5.1)

| Campos de la entrada, en orden | Bytes | Para qué los usa Ana |
| --- | ---: | --- |
| Identificador de versión | 16 | Identifica esta versión inmutable del archivo. |
| Localizador de objeto | 32 | Permite solicitar ese objeto opaco concreto bajo el mismo acuerdo. |
| Clave de versión | 32 | Descifra los bloques. Al incluirla en la entrada cifrada y firmada, la clave de lectura de la carpeta protege la clave de versión. |
| Longitud del archivo | 8 | Acota la recuperación y determina dónde termina cada bloque. Se rechazan los valores que superen el límite del ensayo. |
| Longitud del nombre y nombre para mostrar | 2 + 1–255 | La longitud cuenta bytes UTF-8. Se exige UTF-8 válido y que, tras el nombre, queden exactamente los bytes de la huella. Ana elige el destino local por separado. El nombre solo se muestra como texto y los caracteres de control se representan con secuencias de escape. |
| Huella del objeto | 32 | SHA-256 de la secuencia completa de bloques cifrados, cada uno seguido de su etiqueta. |

## Cómo se lee y se comprueba el objeto

1. Quien escribe el archivo toma una instantánea estable, genera un identificador
   de versión aleatorio nuevo y una clave aleatoria nueva de 256 bits. Cifra y
   autentica bloques consecutivos de hasta 4 MiB. Todos salvo el último ocupan
   exactamente 4 MiB. Un archivo vacío tiene un bloque autenticado de longitud
   cero. Si la longitud es un múltiplo exacto y positivo de 4 MiB, el último
   bloque queda completo.
2. Los índices de bloque empiezan en cero. Para el bloque `i`, el nonce de GCM
   se compone de ocho bytes cero seguidos de `u32(i)`. Sus datos asociados son
   `HiveScope/block/v1`, un byte cero, el identificador de carpeta, el identificador
   de versión, `u64(file_bytes)`, `u32(i)` y `u32(block_plaintext_bytes)`. El
   objeto concatena en orden el texto cifrado y la etiqueta de cada bloque. Tras
   una interrupción se retransmiten los bytes cifrados guardados. Si no se pueden
   recuperar esos bytes, el cifrado se reinicia con un nuevo identificador de versión,
   una clave nueva y un localizador de objeto nuevo. Un bloque modificado nunca se
   cifra con una combinación de clave e índice ya utilizada.
3. Ana solicita el catálogo y luego el objeto mediante dos lecturas que se autorizan
   por separado. A partir de la longitud firmada del archivo, el cliente espera
   recibir `file_bytes + 16 × block_count` bytes de objeto. Calcula el SHA-256 a
   medida que llegan los registros cifrados y autentica cada bloque con el índice
   y el tamaño esperados. Guarda el contenido descifrado y verificado en un archivo
   temporal creado de forma exclusiva, accesible solo para el propietario, en el
   sistema de archivos de destino. Si faltan o sobran bytes, una etiqueta es inválida
   o la huella difiere, la recuperación falla. Solo cuando se han verificado la
   longitud final, todas las etiquetas y la huella se publica el archivo de forma
   atómica en el destino elegido. Un fallo de escritura o verificación anterior
   a la publicación conserva el destino previo. Limpiar únicamente el temporal
   creado para esta operación.

En el ensayo de referencia, Ana elige un destino en un directorio local privado.
Si ya existe un archivo, se requiere su elección expresa para reemplazarlo.
Sincronizar el temporal verificado antes del cambio de nombre atómico en el
mismo sistema de archivos. Sincronizar después el directorio padre antes de
informar que la recuperación terminó. Si falla esa última sincronización,
indicar **publicación incierta**. El archivo completo y verificado puede estar
ya visible. Al reabrir, recuperar la misma versión autenticada en otro temporal
privado y compararlo byte por byte con el destino. Si coinciden, sincronizar el
destino y su directorio antes de confirmar. Si el proveedor está inaccesible,
mantener el resultado como incierto. Si los bytes difieren, conservar el
destino y pedir uno nuevo. Registrar por separado una
recuperación interrumpida y un fallo de autenticación. [Supuestos del proveedor y del sistema de archivos](provider-persistence.md)

## Tres entradas de catálogo reproducibles

La primera fila continúa el ejemplo de recuperación de Ana con la autorización de
la generación 7. Todas las filas usan la misma carpeta, acuerdo, clave de lectura
de la carpeta y clave de prueba para firmar el catálogo que los
[ejemplos de recuperación](recovery-format.md). Para firmar se usa el vector de
prueba público TEST 1024 del RFC 8032 §7.1. Cada par hexadecimal de las dos
últimas columnas representa un byte que se repite hasta llenar el campo
correspondiente, en el orden indicado. Estos valores fijos sirven únicamente
para comparar implementaciones.

| Nombre para mostrar | Bytes en claro | Localizadores de catálogo / objeto | Identificador de versión / clave / sal del catálogo |
| --- | --- | --- | --- |
| `hive.txt` | `48 69 76 65 0a` | `c3 / d4` | `71 / 81 / 91` |
| `empty.bin` | Cero bytes | `c4 / d5` | `72 / 82 / 92` |
| `edge.bin` | 4 194 304 bytes de valor hexadecimal `61`, seguidos de un byte `62` | `c5 / d6` | `73 / 83 / 93` |

El objeto cifrado de `hive.txt` es exactamente
`bac6d517956355ed4498a00c916ef93307ce8c69f8`. La entrada que apunta a él
ocupa 130 bytes. Su catálogo firmado ocupa 250 bytes. Los valores de referencia
siguientes se comprobaron verificando cada firma y descifrando todos los bloques
con bibliotecas criptográficas estándar. La recuperación en red local queda
pendiente de la prueba con el prototipo.

| Archivo | Bloques | Bytes del objeto | SHA-256 del objeto | SHA-256 del catálogo firmado |
| --- | ---: | ---: | --- | --- |
| `hive.txt` | 1 | 21 | `695479c3a0c4494d2aaa784c4786fd9bf5fc1838114df1c4d72cc50466574398` | `96ec886154982747c3560a392c77ab56d33125215d42c95aad695954ca7fa1cf` |
| `empty.bin` | 1 | 16 | `a543d20ea0e7176c0a2a0883c09f85a8d5df42e683334d48a349cd94679fbcbb` | `5a4bec97a22e1043f605df722f7b8f80142ffaa66642a4bcb5e03007cef72f40` |
| `edge.bin` | 2 | 4 194 337 | `6d1e257b5e9f01c9bb0cc90ce845c857ab4fe173d45dc2b0b46b13520f3b018b` | `aa5b005c0b98ffa622e41d4b6a773c81073ea6d2db9b3309dc03beb0d98a450b` |

**Bloque sustituido.** Se conserva el catálogo auténtico de `hive.txt` y se
sustituye su único registro cifrado por otro válido para los mismos cinco bytes
en claro, cifrado bajo el identificador de versión `74 × 16` y la clave `84 × 32`.
Los bytes que se introducen son `3a5fd4b146e8da1f9141eea7225f8b00b480c6ac5e`.
La longitud sigue coincidiendo. La clave y los datos asociados esperados fallan
al autenticarlo, y su SHA-256 difiere de la huella firmada. Si el archivo de
destino contiene `6b 65 65 70 0a`, debe conservar esos cinco bytes. Si se revocó
la autorización de lectura, el proveedor deniega el acceso antes de entregar
cualquier byte del catálogo o del objeto. Un catálogo válido no restablece
esa autorización. [Ejemplo de revocación](recovery-format.md)
