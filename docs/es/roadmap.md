<!-- docs-pair: roadmap; revision: 4 -->

[English](../en/roadmap.md) | [Español](roadmap.md) | [Índice](README.md)

# Hoja de ruta y criterios de aceptación

HiveScope está en diseño. Cada hito describe una entrega propuesta y la
evidencia que deberán publicar los colaboradores. Las fechas de lanzamiento
dependen de la revisión, del hardware disponible y de los voluntarios.

## Hito 0 — Diseño y documentación pública

Publicar en inglés y español la visión, los perfiles de recursos, la arquitectura,
el modelo de amenazas y la hoja de ruta. Conservar en ambas portadas el selector
de idioma, el relato ilustrado, las vías de contribución y el canal privado de
seguridad.

**Aceptación.** Preguntar a alguien que descubre HiveScope qué puede solicitar o
prestar, quién controla el software de una sesión, quién fija los límites de la
máquina física y qué sucede si falta un anfitrión con protección adecuada.
Registrar las dudas y revisar ambas versiones. [Ofrecer mi equipo](resource-offers.md)

## Hito 1 — Primera copia recuperable

Empezar con una CLI de escritorio, un archivo sintético, un proveedor de
almacenamiento en una LAN y un dispositivo de reemplazo con un perfil limpio.
Antes de implementar, especificar los mensajes de protocolo, los límites de
recursos, el manejo de claves y nonces, la protección local de secretos y el
paquete de recuperación. El borrador de diseño explica cómo acredita el equipo
nuevo su derecho a leer y cómo el proveedor recuerda una revocación.
[PR de diseño de recuperación](https://github.com/RubioKo/hivescope/pull/1)

**Aceptación.** Apagar el equipo original. Con el paquete de recuperación, su
contraseña y un proveedor accesible, recuperar un archivo idéntico byte por byte.
Revocar la credencial, esperar la confirmación del proveedor, reiniciarlo y
reintentar con el paquete antiguo. Tanto la lectura del catálogo como la del
objeto deben devolver cero bytes. Probar también transferencias interrumpidas,
una contraseña incorrecta y datos corruptos. Una recuperación fallida conserva
cualquier archivo ya existente en el destino. Un único proveedor permite
verificar una copia que todavía depende de ese equipo.

## Hito 2 — Asignación automática en una red abierta

Usar equipos de prueba controlados y datos sintéticos con el mismo flujo sin
invitaciones previsto para la red pública. Incorporar anuncios con vencimiento,
directorios reemplazables, autenticación entre pares, perfiles locales,
reservas con plazo y presupuestos totales por anfitrión que incluyan los
archivos temporales.

**Aceptación.** Una persona que solo solicita obtiene una copia en el equipo de
un desconocido sin pedirle aprobación personal. Descartar a un anfitrión fuera
de horario. Lanzar dos solicitudes para el último espacio disponible, cortar
la conexión después de guardar los datos y antes de recibir el comprobante, y
reintentar. Solo se podrá consumir ese espacio una vez. Pausar el anfitrión y
desconectar el primer directorio. El solicitante debe poder usar otro directorio
o ver la causa de la espera y cancelar. Las peticiones repetidas y las que usan
múltiples identidades deben respetar la capacidad total del anfitrión.

## Hito 3 — Almacenamiento resistente y sincronización de carpetas

Distribuir cada bloque cifrado entre tres proveedores con dos fragmentos de
datos y uno de paridad. Replicar los metadatos del catálogo, emitir recibos y
medir el coste y la cobertura de las comprobaciones mediante muestras de bytes.
Añadir reparación con recursos acotados, versiones firmadas, copias en conflicto,
marcadores de borrado e historial recuperable de 30 días.
[Almacenamiento](storage.md) · [Sincronización](synchronization.md)

**Aceptación.** Indicar protección redundante solo tras confirmar las tres
ubicaciones de cada bloque y el catálogo. Retirar un proveedor y recuperar el
archivo con dos fragmentos válidos de cada bloque. Indicar protección degradada
hasta confirmar la reparación en un reemplazo compatible. Si algún bloque tiene
menos de dos fragmentos válidos, o falta el catálogo necesario, informar de que
el archivo no está disponible. La reparación trabaja con datos cifrados y
respeta la cuota tras reintentos y reinicios. Resolver ediciones sin conexión y
conflictos entre borrado y edición conservando las ediciones que compiten y los
cambios locales más recientes. Comprobar que un volumen sin montar o un nombre
incompatible genera un error visible sin borrado silencioso.

## Hito 4 — Alfa de escritorio

Probar la CLI y el servicio en segundo plano en Windows, Linux y macOS sobre
conexiones domésticas. Participarán dos dispositivos autorizados por el dueño,
tres proveedores de hogares distintos y un proveedor de reserva. Desarrollar la
interfaz gráfica sobre las mismas operaciones ya medidas. Los celulares y las
tabletas se evaluarán primero como clientes de acceso y recuperación.

**Aceptación.** Publicar resultados de conexiones directas, con relay y fallidas,
y probar reinicios, falta de disco, pausa por batería y recuperación tras perder
el equipo original. Acordar los presupuestos de recursos antes de medir. En un PC
de cuatro núcleos y 8 GB de RAM, registrar memoria en reposo y en transferencia,
CPU en reposo y en uso, velocidad de recuperación y tiempo de respuesta a la
pausa. Go es el primer candidato para el nodo. Reevaluar Rust si una prueba
equivalente demuestra una ventaja relevante cuando Go exceda un presupuesto.
Para la red, publicar por separado renovaciones de anuncios, reconexiones masivas,
uso de relays y tráfico de reparación. El PC de referencia sirve para medir el
nodo de escritorio. Las sesiones privadas requieren hardware compatible que se
probará por separado.

## Hito 5 — Cómputo privado de propósito general

Ofrecer CPU y RAM en una máquina virtual confidencial. Quien solicita elige
el software, carga su propio programa y recupera el resultado. Evaluar por
separado la GPU y la comunicación protegida entre CPU y GPU.
[Modelo de seguridad](security.md)

**Aceptación.** Ejecutar un programa elegido por el solicitante respetando el perfil
del anfitrión. Si las pruebas de protección faltan, han caducado o son incompatibles,
comprobar que las claves y entradas privadas quedan en el equipo solicitante y
que la petición puede esperar o cancelarse. Probar una GPU sin verificar y un
reintento en otro anfitrión. Detener una sesión activa, liberar los recursos
asignados e indicar qué resultados se conservaron y desde qué punto se puede
reanudar el trabajo, si existe un guardado válido.

## Hito 6 — Servicios persistentes y juegos

La comunidad elige un servicio, como una aplicación o un juego, y los recursos
que necesita. Los nodos buscan un anfitrión activo y un relevo compatible
de acuerdo con sus respectivos perfiles.

### Prueba de un servidor de juego comunitario

Elegir un juego cuya licencia permita alojar servidores comunitarios. Project
Zomboid es uno de los candidatos. Registrar qué grupos y regiones participan
y quién cubre los costes de electricidad, conexión y relays.

**Aceptación.** Desconectar el anfitrión activo durante la partida, verificar
el entorno de reemplazo y restaurar el último guardado válido. Reconectar a
los jugadores y volver a encender el anfitrión anterior. Solo una instancia
podrá aceptar cambios en cada momento. Publicar la latencia por región, la
duración de la interrupción, el tiempo de reconexión y el progreso perdido
antes de establecer expectativas de disponibilidad.
