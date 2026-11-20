<!-- docs-pair: identity-custody; revision: 1 -->

[English](../en/identity-custody.md) | [Español](identity-custody.md) | [Índice](README.md)

# Quién puede guardar una copia, recuperarla y revocar el acceso

Si Ana pierde su portátil, su paquete de recuperación y su contraseña permiten
que un equipo nuevo lea una carpeta del proveedor acordado. Una copia separada
le permite reemplazar o revocar esa credencial de lectura. Estas decisiones
definen el ensayo de escritorio en LAN. El desbloqueo nativo y el reinicio aún
requieren pruebas en equipos reales.

## Establecer el acuerdo una sola vez

En el ensayo controlado, el operador traslada los datos **públicos** de alta
entre los dos perfiles locales por un canal de confianza fuera de línea. Compara
la huella SHA-256 completa de cada clave pública de 32 bytes en bruto con la que
muestra el equipo de origen. El proveedor fija las claves de la autoridad de la
carpeta y del equipo que sube los datos. El propietario fija esa autoridad y el
proveedor. Ambas partes obtienen estas claves antes de la primera conexión. Un
proveedor distinto en la misma IP falla la autenticación.

Ambos perfiles registran el mismo ID aleatorio de carpeta de 16 bytes, un nuevo
ID aleatorio de acuerdo de 16 bytes, las claves públicas fijadas, el inicio y
vencimiento en segundos Unix UTC, la cuota física y los límites de transferencia.
El perfil local del proveedor debe admitir esas condiciones. Un acuerdo nuevo
empieza con un registro de autorización explícitamente vacío y deniega toda
lectura. Reabrir un acuerdo existente requiere su estado persistido. Una
importación contradictoria o un ID cerrado impiden inicializarlo de nuevo. La
dirección sirve para conectar. El descubrimiento público y la negociación
automática corresponden al hito 2. [Ofrecer mi equipo](resource-offers.md) · [Condiciones de almacenamiento](storage.md)

| Credencial | Titular y operación permitida |
| --- | --- |
| Identidad del dispositivo proveedor | El proveedor autentica su conexión y firma recibos de almacenamiento. |
| Identidad del dispositivo que sube los datos | El equipo registrado reserva espacio, escribe y consulta sus solicitudes bajo el acuerdo. Un equipo de reemplazo limpio genera su propia identidad. |
| Autoridad de la carpeta | El propietario firma los cambios de autorización de recuperación. El proveedor conserva solo su clave pública. La clave de autoridad por sí sola no descifra la carpeta. |
| Clave de firma de recuperación | El propietario y un dispositivo que desbloquee el paquete pueden demostrar el derecho registrado a obtener los datos cifrados. |
| Clave de lectura de la carpeta y claves del catálogo | El propietario genera el secreto de lectura y la clave de firma del catálogo por separado. El paquete contiene el secreto de lectura y la clave de **verificación** del catálogo. Las claves de contenido de cada versión están dentro del catálogo cifrado y autenticado. |

Generar las semillas Ed25519 y las claves de cifrado de forma independiente con
el generador aleatorio criptográfico del sistema operativo. Guardar las claves
privadas solo en el dispositivo que cumple su función o en la exportación
prevista. Las identidades de proveedor, carga, recuperación y administración
cumplen funciones distintas. Cambiar de autoridad o de equipo de carga requiere
un nuevo acuerdo en este ensayo. [Paquete de recuperación](recovery-format.md) · [Catálogo de archivos](file-catalog.md)

## Emitir, reemplazar o revocar la autorización de recuperación

El propietario firma exactamente 134 bytes con Ed25519 puro y añade la firma
de 64 bytes en bruto. Los 198 bytes resultantes forman un registro `GRANT_UPDATE`.
Concatenar los siguientes campos sin separadores, relleno ni prefijos de
longitud. Las posiciones empiezan en cero. [Formato de firma Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-5.1) · [Mensajes en LAN](lan-protocol.md)

| Posición | Campo y codificación |
| ---: | --- |
| 0 | Los 27 bytes ASCII `HiveScope/recovery-grant/v1`, seguidos de un byte cero. |
| 28 | Clave pública Ed25519 del proveedor (32 bytes en bruto). |
| 60 | ID opaco de carpeta (16), seguido del ID de acuerdo (16). |
| 92 | Generación positiva de la autorización (8, sin signo, big-endian). |
| 100 | Estado (1), `01` activa o `02` revocada. Alcance (1), siempre `01` para leer los objetos y el catálogo confirmados en el acuerdo. |
| 102 | Clave pública de recuperación activa (32). Un registro de revocación exige 32 bytes cero en este campo. |
| 134 | Firma de la autoridad de la carpeta sobre los bytes 0 a 133 (64). |

Cualquier par autenticado puede entregar una actualización firmada. Antes de
cambiar nada, el proveedor comprueba su propia identidad, ambos ID, el formato
exacto, el alcance, el estado, la codificación válida de la clave activa y la
firma contra la autoridad registrada. Acepta actualizaciones durante el acuerdo
y sus siete días de gracia de lectura. La autorización caduca al terminar ese
plazo. El acuerdo local determina la retención y la identidad del escritor.
Emitir una autorización de lectura exige que ese acuerdo ya exista. El plazo se
establece localmente en el ensayo LAN.

| Comparación con la mayor generación que aceptó el proveedor | Decisión del proveedor |
| --- | --- |
| Primera generación positiva o cualquier otra superior | Persistir atómicamente el registro firmado completo, la generación y el estado. Aplicar también una revocación superior firmada aunque se haya perdido una actualización anterior. |
| Misma generación y registro idéntico byte por byte | Devolver la confirmación de ese registro. Se puede reintentar si se perdió la respuesta. |
| Generación inferior o misma generación con bytes distintos | Rechazarlo y conservar el estado registrado. Una firma correcta tampoco invalida una revocación posterior. |

Un único dispositivo emite los cambios administrativos de cada acuerdo. Guarda
de forma durable los bytes firmados antes de enviarlos. Una autorización activa
de reemplazo exige una nueva clave de recuperación y otra exportación. Reservar
la generación máxima sin signo para una revocación final. Nunca volver a cero.
Serializar los cambios de autorización con la decisión de lectura. Invalidar los
retos pendientes de autorizaciones sustituidas y dejar de encolar los bytes de
sus lecturas activas antes de confirmar el cambio. Los bytes ya entregados al
transporte aún pueden llegar. Tras
el commit durable, la respuesta en la conexión fijada devuelve el SHA-256 de los
198 bytes. El propietario lo compara con el registro guardado. Si falta la
confirmación, el cambio queda pendiente y se reintenta. Quien ya recibió datos
en claro o claves los conserva. [Autorización de lectura y reto](recovery-format.md)

Tras reiniciar, descartar todos los retos. Cargar la mayor generación persistida,
comprobar de nuevo su firma y alcance, y denegar el acceso si el estado falta o
presenta incoherencias. Si se restaura una base anterior del proveedor, el
acuerdo debe quedar fuera de servicio hasta reconciliarlo. Un anfitrión desleal
puede restaurar una imagen de disco completa. Un único proveedor no demuestra
que su historial sea el más reciente. [Persistencia del proveedor](provider-persistence.md)

## Desbloquear un perfil local

La CLI arranca bloqueada. La persona abre explícitamente el perfil en su sesión
del sistema operativo. Al bloquear la aplicación o la sesión, cerrar la sesión o
terminar el proceso, detener operaciones que requieran secretos, cerrar flujos
autenticados y liberar las claves en memoria. Volver a abrir el perfil requiere
un desbloqueo explícito. Después de reiniciar, el proveedor debe desbloquear
su identidad y reconciliar los permisos antes de atender solicitudes. Son
requisitos del diseño.

| Escritorio | Custodia propuesta de los secretos en reposo |
| --- | --- |
| Windows | Proteger los secretos con DPAPI en el ámbito del usuario actual. Guardar los datos protegidos bajo una ACL privada del perfil. Excluir la protección a nivel de máquina. [DPAPI de Microsoft](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata) |
| macOS | Usar el llavero de protección de datos del usuario con `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`, un grupo de acceso propio de la aplicación y la sincronización desactivada. [Selección del llavero de Apple](https://developer.apple.com/documentation/security/ksecusedataprotectionkeychain) · [Acceso tras el desbloqueo](https://developer.apple.com/documentation/security/ksecattraccessiblewhenunlockedthisdeviceonly) |
| Linux | Usar una colección persistente y cifrada de Secret Service en la sesión de escritorio. Si está bloqueada, solicitar el desbloqueo mediante el servicio. Usar atributos de búsqueda y etiquetas opacos. Verificar el proveedor de secretos elegido y su bloqueo en el sistema de referencia. [Secret Service](https://specifications.freedesktop.org/secret-service/latest/description.html) |

Si falta el almacén de secretos, se rechaza el aviso o la sesión es incompatible,
el perfil sigue bloqueado. El ensayo requiere una sesión local interactiva. La
base conserva referencias opacas al almacén, claves públicas, recibos y estado
del protocolo. Excluir de sus campos en claro, de las etiquetas y de los registros
los nombres y rutas originales de los archivos. Guardar los bytes cifrados
pendientes con nombres opacos en un directorio privado. Recibir las contraseñas
de exportación mediante una entrada local oculta. Excluir secretos de los
argumentos, variables de entorno, informes de fallos y registros. El ensayo no
protege frente a un proceso que ya controle la cuenta desbloqueada del propietario.

## Recuperar la facultad de administrar

Durante el alta, el propietario exporta por separado una **copia de la autoridad**
fuera de línea con otra contraseña robusta. Guardarla separada del paquete de
lectura y mantener cada contraseña separada de su exportación. El propietario
conserva además las huellas públicas del alta y la última autorización firmada
que el proveedor confirmó. Esos registros públicos identifican la autoridad y
su última generación conocida.

La copia administrativa ocupa 145 bytes. Su cabecera de 33 bytes contiene ASCII
`HSAB`, versión `01`, una sal aleatoria nueva de 16 bytes y un nonce aleatorio
nuevo de 12 bytes. Usar la codificación de contraseña, los parámetros Argon2id,
AES-256-GCM y la etiqueta de 16 bytes del [formato de recuperación](recovery-format.md).
Autenticar toda la cabecera `HSAB` como datos asociados. La carga cifrada de
96 bytes contiene en orden el ID de carpeta (16), el ID de acuerdo (16), la
clave pública del proveedor (32) y la semilla de autoridad (32). La importación
exige el tamaño exacto, el identificador de formato y la versión de la operación
elegida. Una etiqueta inválida detiene la importación antes de interpretar la
carga, contactar a un proveedor o cambiar el almacén. Ambas exportaciones
permiten intentar adivinar su contraseña fuera de línea.

1. Antes de retirar el dispositivo original, abrir la copia de la autoridad en
   un perfil local aislado. Derivar su clave pública y comparar tanto la clave
   como el alcance con el registro del alta guardado por separado. Descartar el
   perfil temporal después. La recuperación ordinaria de archivos usa solo el
   paquete de lectura.
2. Para volver a administrar después de una pérdida, restaurar la copia en un
   almacén de confianza y mantener un solo dispositivo emisor. Obtener el último
   registro firmado del operador local de confianza del proveedor y compararlo
   con el último que conoce el propietario. Verificar ambos contra la autoridad y
   el alcance fijados. Si el proveedor está atrasado, reconciliarlo usando el
   registro más reciente firmado por el propietario. Resolver los registros
   contradictorios de igual generación antes de emitir otro. Emitir una
   generación mayor que la más alta verificada. Activar una nueva credencial de
   lectura también exige el material de lectura de la carpeta para poder generar
   una exportación utilizable.
3. Con solo el paquete de lectura, el dueño puede recuperar archivos si la
   autorización sigue vigente. Con solo la copia de la autoridad puede revocar
   un lector y carece de las claves para descifrar el archivo. Perder ambas
   copias de la semilla de autoridad impide firmar más cambios para ese acuerdo.
   Si se compromete la autoridad, cerrar el acuerdo localmente y dar de alta uno
   nuevo.

## Casos de aceptación de la primera implementación

| Ensayo | Evidencia esperada |
| --- | --- |
| Emitir la autorización activa de generación 7 del [ejemplo de recuperación](recovery-format.md), reenviarla y enviar otros bytes para la misma generación. | El reintento idéntico recibe el mismo resumen. El registro contradictorio deja intactas la clave activa y la generación. |
| Confirmar la revocación de generación 8, reiniciar, desbloquear y repetir el registro firmado de generación 7. | Se rechaza la actualización antigua. Las dos lecturas independientes con el paquete anterior devuelven cero bytes de contenido. |
| Cambiar el proveedor, el alcance o el acuerdo en un registro firmado, o firmarlo con la clave de carga o la de recuperación. | No se crea una autorización ni un privilegio de lectura. Se conserva toda autorización previa. |
| Bloquear la sesión durante una operación, denegar acceso al almacén al arrancar o usar una contraseña incorrecta en cualquiera de las exportaciones. | El perfil exige desbloqueo local. Una importación fallida no contacta al proveedor y conserva el destino y el almacén. |
| Apagar el dispositivo original. Usar cada copia de forma independiente e intentar importarla como si fuera del otro tipo. | El paquete de lectura puede autorizar una lectura válida, la copia administrativa recupera la autoridad fijada y las dos importaciones del tipo incorrecto se detienen localmente. |
