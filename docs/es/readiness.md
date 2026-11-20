<!-- docs-pair: readiness; revision: 1 -->

[English](../en/readiness.md) | [Español](readiness.md) | [Índice](README.md)

# ¿Está HiveScope listo para su primer prototipo?

La [arquitectura](architecture.md) define el nodo, las funciones, los límites de
confianza, el flujo de datos y el modelo de almacenamiento. La [hoja de ruta](roadmap.md)
establece los hitos. Ya se puede empezar a especificar el núcleo seguro. Para
implementar cada parte hay que cerrar su contrato, descrito abajo. Las comprobaciones
actuales verifican documentación. La fiabilidad, la interoperabilidad y la
recuperación se comprobarán con software en funcionamiento.

## Primera carga y recuperación manual

**Alcance propuesto para el hito 1.** Utilizar un equipo del propietario, un proveedor
de almacenamiento aprobado y una conexión directa en LAN. Un tercer perfil de dispositivo,
limpio, representa al propietario tras perder su equipo. Un único proveedor conserva
un objeto cifrado completo y un catálogo mínimo autenticado que permite localizarlo.
Si se pierde ese proveedor, los datos de la demostración quedan inaccesibles. La
distribución entre tres proveedores y el estado `protected` corresponden al hito 2.

1. El propietario verifica la huella del proveedor y ambos aceptan un acuerdo.
2. El propietario cifra un archivo sintético y sube el contenido y el catálogo.
   El proveedor reserva espacio, persiste los bytes y confirma la operación.
3. El propietario exporta un paquete de recuperación protegido con contraseña.
   Se apaga el equipo original y se empieza con una base de datos vacía y una
   identidad de dispositivo nueva.
4. Con solo el paquete, su contraseña y el proveedor accesible, se recupera el
   archivo. Se compara cada byte con una copia de prueba independiente. Se registra
   qué credencial autorizó la lectura y qué información local hizo falta.

Recuperar las claves de carpeta permite descifrar. El proveedor también debe
reconocer el derecho del equipo nuevo a leer los datos bajo el acuerdo. Conviene
resolver esto primero porque afecta a la incorporación de dispositivos, al acuerdo,
a la búsqueda del catálogo y al formato de exportación.

## Cuatro decisiones por cerrar

Cada fila está abierta. Los colaboradores pueden proponer el documento indicado
en inglés y español. Antes de implementar el componente, el mantenedor registra
en un PR revisado el comportamiento elegido, su motivo, un caso aceptado y un
caso de fallo.

| Orden | Pregunta de diseño | Qué hace falta para empezar ese componente |
| --- | --- | --- |
| 1. Identidad y recuperación | ¿Quién custodia la autoridad de la carpeta? ¿Qué puede hacer cada dispositivo o proveedor? ¿Cómo acredita su acceso el equipo recuperado, encuentra el catálogo y trata una exportación antigua o credenciales revocadas? | Una tabla de funciones y operaciones y un recorrido de recuperación que identifique quién firma, quién verifica, secretos necesarios, método de desbloqueo local en cada sistema objetivo y límites para verificar si los datos son recientes. Incluir una operación denegada. |
| 2. Formato de archivos y catálogos | ¿Cómo se generan, protegen y recuperan las claves por versión? ¿Qué bytes vinculan cada bloque con su versión, posición y longitud? ¿Qué ocurre con el nonce y el contenido cifrado tras un reinicio? | Un formato binario versionado para el objeto cifrado, el catálogo y el paquete de recuperación, con ejemplos sintéticos de entrada y salida para un archivo vacío y otro de varios bloques. Incluir un bloque que no corresponde y una contraseña incorrecta. |
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
| Recuperar desde un equipo limpio con el original apagado | El archivo coincide con la copia de prueba. El operador puede identificar el acuerdo, catálogo y material de recuperación utilizados. |

## Qué puede esperar

En el hito 2 se evaluarán el coste de las comprobaciones, los umbrales de reparación
ante desconexiones reales y el comportamiento al perder proveedores con el perfil
2 + 1. El hito 3 probará conflictos sin conexión y nombres de archivo entre plataformas.
La alfa cerrada medirá conectividad doméstica, tráfico de relay, uso de disco e
impacto en batería en Windows, Linux y macOS. Acotar el ensayo inicial a una LAN
permite conservar esos objetivos de plataforma. La futura IA tiene su propio
diseño de cómputo y privacidad.

**Siguiente contribución.** Describir la primera lectura desde un dispositivo de
reemplazo en la decisión 1, indicando por qué el proveedor la acepta o la rechaza.
Una vez acordado el recorrido, utilizarlo como base para especificar los formatos
y la transferencia.
