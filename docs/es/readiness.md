<!-- docs-pair: readiness; revision: 4 -->

[English](../en/readiness.md) | [Español](readiness.md) | [Índice](README.md)

# ¿Está HiveScope listo para su primer prototipo?

La [arquitectura](architecture.md) define el nodo, las funciones, los límites de
confianza, el flujo de datos y el modelo de almacenamiento. La [hoja de ruta](roadmap.md)
establece los hitos. Ya se puede empezar a especificar el núcleo seguro. Para
implementar cada parte hay que cerrar su contrato, descrito abajo. Las comprobaciones
actuales verifican documentación. La fiabilidad, la interoperabilidad y la
recuperación se comprobarán con software en funcionamiento.

## Primera recuperación desde un proveedor de almacenamiento

**Alcance propuesto para el hito 1.** Utilizar un equipo del propietario, un proveedor
de almacenamiento con dirección conocida en la LAN y una conexión directa. Un tercer perfil de dispositivo,
limpio, representa al propietario tras perder su equipo. Un único proveedor conserva
un objeto cifrado completo y un catálogo mínimo autenticado que permite localizarlo.
Si se pierde ese proveedor, los datos de la demostración quedan inaccesibles. La
distribución entre tres proveedores y el estado `protected` corresponden al hito 3.

1. Los pares se autentican. El proveedor contrasta la solicitud con su perfil
   local y reserva capacidad mediante un acuerdo de almacenamiento. El cliente
   registra la identidad del proveedor para recuperar la copia. Para el ensayo
   se proporciona directamente la dirección del proveedor.
2. El propietario cifra un archivo sintético y sube el contenido y el catálogo.
   El proveedor utiliza la reserva, persiste los bytes y confirma la operación.
3. El propietario exporta un paquete de recuperación protegido con contraseña.
   Se apaga el equipo original y se empieza con una base de datos vacía y una
   identidad de dispositivo nueva.
4. Con solo el paquete, su contraseña y el proveedor accesible, se recupera el
   archivo. Se compara cada byte con una copia de prueba independiente. Se registra
   qué credencial autorizó la lectura y qué información local hizo falta.

## Cómo acredita el equipo nuevo su primera lectura

**Decisión de diseño para revisión.** El proveedor acepta una prueba de posesión
de la clave de firma de recuperación que el propietario registró para una carpeta
y un acuerdo de almacenamiento. Su único permiso es obtener el catálogo cifrado
de esa carpeta y los objetos cifrados cubiertos por el acuerdo. Así el propietario
puede recuperar una copia tras perder su equipo y custodiar por separado la
autoridad administrativa.

| Quién | Qué conserva y qué hace |
| --- | --- |
| Propietario | Al establecer el acuerdo, ambas partes fijan la clave pública de la autoridad de carpeta. El propietario usa la privada para firmar la autorización inicial de recuperación y sus posteriores sustituciones o revocaciones. Guarda esa clave administrativa fuera del paquete de recuperación. |
| Proveedor | Guarda la autorización firmada con los identificadores opacos de carpeta y acuerdo, su propia identidad, la clave pública de recuperación, el alcance de lectura y la generación. Conserva de forma persistente la generación más alta aceptada y su estado activo o revocado. |
| Equipo de reemplazo | Crea su propia identidad de dispositivo. La contraseña desbloquea el paquete localmente. El paquete contiene la clave de firma de recuperación, la referencia de la autorización con su generación, el material para descifrar la carpeta y verificar el catálogo, el localizador opaco del catálogo, los datos de conexión del proveedor y la huella fijada en el acuerdo. |

1. El equipo nuevo verifica la huella del proveedor en una conexión libp2p
   autenticada. El proveedor puede emitir un reto a este nuevo par antes de
   autorizar una lectura. [Autenticación de pares en libp2p](https://github.com/libp2p/specs/blob/master/tls/tls.md)
2. Para cada lectura de catálogo u objeto, el proveedor emite en esa conexión
   un reto nuevo, de corta duración y de un solo uso. El equipo lo firma con la
   clave de recuperación y vincula la operación de lectura, las identidades de
   ambos pares, el acuerdo, la carpeta, la generación y el localizador opaco
   solicitado. La contraseña y las claves privadas permanecen en el equipo.
3. Antes de enviar datos, el proveedor consume el reto una sola vez, verifica la
   firma con la clave activa que tiene registrada, comprueba que la solicitud
   corresponde a la conexión y a la autorización, y verifica el elemento solicitado
   y el plazo de lectura. Los datos existentes se pueden leer durante el
   [periodo de gracia del acuerdo](storage.md). Si falta el estado de autorización,
   la generación está desactualizada o la petición excede el permiso, deniega la
   lectura. Leer no concede permiso para escribir, renovar o incorporar dispositivos.
4. El equipo verifica y descifra el catálogo y después solicita su objeto cifrado
   bajo los mismos controles. Autentica el contenido recuperado antes de reemplazar
   el archivo de destino. Una exportación antigua y un único proveedor no permiten
   determinar si el catálogo es el más reciente.

El proveedor verifica los registros con la clave de la autoridad de carpeta fijada
al establecer el acuerdo. Solo acepta cambios de generación superior para el mismo
alcance. Para revocar, el propietario firma un registro que marca la autorización
como revocada. El proveedor confirma la revocación tras persistirla e invalidar
los retos pendientes de la autorización antigua. Al reiniciarse, descarta los
retos pendientes y conserva la revocación. Una autorización de reemplazo usa
una clave nueva y requiere otra exportación. Quien conserve el paquete antiguo y
la contraseña puede usarlo hasta que el proveedor reciba la revocación. Revocar
solo la identidad de un dispositivo no invalida la clave de recuperación. Quien ya
recibió datos o claves conserva esas copias.

El [formato de recuperación v1 propuesto](recovery-format.md) detalla la estructura
del paquete, los bytes que se firman en cada lectura y dos ejemplos con la misma exportación.

## Decisiones pendientes para la implementación

La regla de primera lectura resuelve parte de la decisión 1. El trabajo pendiente
de cada fila requiere una especificación revisada en inglés y español.

| Orden | Pregunta de diseño | Qué hace falta para empezar ese componente |
| --- | --- | --- |
| 1. Identidad y recuperación | ¿Cómo se desbloquean los secretos locales y se mantiene recuperable la clave administrativa separada? | Especificar la custodia y el desbloqueo local en cada sistema objetivo respetando el permiso de primera lectura y los casos de revocación. |
| 2. Formato de archivos y catálogos | ¿Cómo se generan, protegen y recuperan las claves por versión? ¿Qué bytes vinculan cada bloque con su versión, posición y longitud? ¿Qué ocurre con el nonce y el contenido cifrado tras un reinicio? | Especificar el objeto cifrado y el catálogo que se abren con el paquete de recuperación. Dar ejemplos sintéticos de entrada y salida para un archivo vacío y otro de varios bloques, incluido un bloque que no corresponde. |
| 3. Protocolo de transferencia | ¿Cómo reservan espacio los pares, envían, reanudan, confirman y recuperan un objeto? ¿Qué se firma exactamente? ¿Cómo distingue el proveedor un reintento de una solicitud nueva? | Una tabla de mensajes versionados con campos obligatorios, bytes firmados, tamaños de trama, límites de concurrencia y tiempo, errores, reintentos y vigencia del acuerdo durante la transferencia. Incluir una carga cuya confirmación se pierde en la red. |
| 4. Disco y reinicios | ¿Cuándo puede confirmar el proveedor que el contenido quedó guardado de forma duradera? ¿Cómo compiten dos reservas por el último espacio libre? ¿Cómo se concilian el directorio de fragmentos y SQLite al reiniciar? | Una tabla de estados que cubra reserva, temporal, bytes verificados, índice persistido y confirmación. Documentar qué sobrevive a cada interrupción, cómo se contabiliza y cómo se liberan bytes o reservas huérfanos. |

Tres detalles técnicos orientan estas decisiones. La interfaz AEAD de Go exige
un nonce único por clave para cada cifrado. La serialización Protobuf no garantiza
una representación canónica para calcular una firma. La garantía transaccional
de SQLite cubre los cambios en su base de datos. HiveScope también escribe
fragmentos fuera de ella.
[AEAD en Go](https://pkg.go.dev/crypto/cipher#AEAD) ·
[Serialización Protobuf](https://protobuf.dev/programming-guides/serialization-not-canonical/) ·
[Transacciones atómicas en SQLite](https://sqlite.org/atomiccommit.html)

## Evidencia que deberá producir el prototipo

Estos ensayos están planificados. Registrar versión, sistema operativo, entradas
y resultado observado cuando exista una implementación que permita ejecutarlos.

| Escenario | Resultado que debe observarse |
| --- | --- |
| Subir un archivo vacío y otro de 4 MiB + 1 byte | La recuperación conserva longitud y contenido exactos al cruzar el límite entre bloques. |
| Cortar la conexión después de que el proveedor guarde el objeto y antes de que el cliente reciba la respuesta | Repetir la operación concluye con un objeto almacenado y un solo cargo de cuota. |
| Interrumpir entre la escritura del archivo y la actualización del índice | Al reiniciar se consolida o libera la reserva. Todo objeto ya confirmado sigue siendo legible. |
| Iniciar dos cargas que compiten por espacio suficiente para una | Como máximo se completa una. El uso físico, incluidos los temporales, respeta los límites acordados. |
| Entregar un bloque corrupto, intentar leer con una identidad sin aprobar o usar una contraseña de recuperación incorrecta | Cada caso indica fallo y conserva cualquier archivo que ya exista en el destino. |
| Autorización válida. Con el equipo original apagado y el perfil de reemplazo vacío, utilizar solo el paquete, la contraseña y el proveedor accesible. | La identidad nueva recibe catálogo y objeto tras demostrar posesión de la clave. El archivo recuperado coincide byte a byte con la copia de prueba independiente. |
| Autorización revocada. Partiendo de los mismos datos de prueba, hacer que el proveedor confirme la revocación del propietario, reiniciarlo y dejar apagado el equipo original. Reintentar desde un perfil limpio con el paquete y la contraseña antiguos. | Incluso una firma correcta de un reto nuevo con la clave antigua devuelve cero bytes de catálogo u objeto. Comprobar ambos localizadores por separado. El archivo existente en destino permanece intacto. |

## Qué puede esperar

El hito 2 incorpora asignación automática mediante directorios reemplazables y
prueba reservas concurrentes y reintentos entre equipos que se encuentran por primera vez.
El hito 3 prueba la reparación 2+1, el coste de las comprobaciones, los conflictos
sin conexión y la portabilidad de nombres. La alfa de escritorio medirá la conectividad
doméstica, el tráfico de relay, el uso de disco y el impacto en batería en Windows,
Linux y macOS. El cómputo privado de propósito general y el alojamiento persistente
tienen criterios propios en la [hoja de ruta](roadmap.md).

**Siguiente paso de diseño.** Especificar el catálogo autenticado del ensayo con un
solo archivo. Dar un ejemplo de cómo el equipo de reemplazo localiza el objeto,
obtiene la clave de su versión y detecta un bloque sustituido antes de escribir en el destino.
