<!-- docs-pair: vision; revision: 6 -->

[English](../en/vision.md) | Español | [Índice](README.md)

# Visión y límites

HiveScope está diseñando una red abierta donde cualquier persona pueda solicitar
almacenamiento o cómputo. Se podrá participar solo para solicitar recursos. Quien
presta un equipo decide qué recursos ofrece y cuánta capacidad necesita reservar
para su propio uso. Las máquinas se encontrarían y acordarían el trabajo automáticamente.

## Qué podría hacer una persona

**Conservar una copia recuperable.** Una estudiante elige la carpeta de su tesis,
con apuntes y fotografías. Su computadora cifra los nombres y el contenido antes de
transferirlos. La estudiante decide cuáles de sus dispositivos pueden leer esa carpeta.
Tras un fallo del portátil, el primer prototipo deberá recuperar un archivo de prueba
desde un proveedor accesible en un equipo de reemplazo limpio con una credencial de
recuperación válida. Un hito posterior probará la reparación entre anfitriones
independientes y conservará ambas versiones cuando haya ediciones incompatibles sin
conexión. [Almacenamiento y reparación](storage.md) · [Sincronización de carpetas](synchronization.md)

**Utilizar una sesión privada de cómputo.** Una creadora solicita CPU y RAM e instala
las herramientas que elija dentro de su entorno. Podría compilar un proyecto, renderizar
una escena o ejecutar su propio modelo. La máquina virtual confidencial propuesta
protegería el trabajo durante su ejecución. Antes de entregar las claves de sesión,
el dispositivo solicitante deberá verificar el entorno y vincular las claves a esa
sesión. Un trabajo que utilice GPU necesitará además pruebas sobre la GPU y su conexión
con el entorno protegido. Si la protección no se puede verificar, la solicitud seguirá
en espera o la persona podrá cancelarla. Las entradas privadas y las claves
se quedarán con quien solicita. El anfitrión aún podrá observar información
operativa, como la duración de la sesión o el volumen de tráfico, y podrá desconectarse.
[Modelo de seguridad](security.md)

**Alojar un servicio con la comunidad.** Un grupo de jugadores podría solicitar
alojamiento para su mundo aprovechando los horarios de los voluntarios. Project Zomboid
es un ejemplo propuesto. Un único anfitrión activo acepta cambios. Tras un relevo, un
equipo compatible podría iniciar el mundo desde el último guardado verificado. El ensayo
mediría cuánto tardan los jugadores en volver a conectarse, qué progreso pierden y la
latencia desde sus conexiones reales. El calendario de una asociación u otro servicio
persistente necesitarían su propia prueba de recuperación de estado. [Hoja de ruta](roadmap.md)

## Qué ocurre entre las máquinas

Quien presta el equipo configura un perfil una sola vez, elige disco, CPU, RAM, GPU o
ancho de banda y fija horario y límites. Por ejemplo, Luis podría prestar 20 GB de disco
por las tardes y reservar toda la capacidad de cómputo para sí mismo. Ana podría pedir
una copia sin agregar a Luis como contacto. Los directorios comunitarios reemplazables
distribuirían anuncios de servicios con vencimiento. Los nombres de archivos, el
contenido de las sesiones y las claves quedarían fuera de esos anuncios.

El equipo solicitante encuentra candidatos y se conecta a un anfitrión. Este comprueba
su perfil local, la capacidad física y los compromisos vigentes antes de aceptar una
reserva con vencimiento. Si varias solicitudes compiten por el último espacio, solo una
reserva puede obtenerlo. Un reintento devuelve el estado de la misma solicitud. Una
pausa detiene las reservas nuevas. Los datos almacenados siguen su acuerdo de retención
y migración. Si se interrumpe una sesión de cómputo, se deberá indicar el último
guardado confirmado para que su titular sepa qué trabajo se conservó.
[Ofrecer mi equipo](resource-offers.md) · [Arquitectura](architecture.md)

## Participación y costes

El objetivo es acceder sin una cuota de alquiler obligatoria. Cada persona podrá pedir
ayuda con independencia del hardware que aporte. El piloto dependerá de la capacidad
voluntaria y de cuotas locales, sin créditos, saldos ni criptomonedas obligatorios.
Los participantes o donantes asumirán los costes del equipo, la electricidad, el
mantenimiento, la conexión y el tráfico de los relays. Si faltan recursos, quien los
solicita verá el motivo y podrá esperar o cancelar.

## Cómo se comprobará lo propuesto

1. Empezar con un archivo cifrado, un solo proveedor y la recuperación en un equipo
   nuevo. Probar una credencial válida y volver a probarla tras confirmar su revocación
   y reiniciar el proveedor. Registrar este resultado por separado de la redundancia.
2. Ensayar las reservas automáticas entre equipos que se encuentran por primera vez
   antes de ampliar el almacenamiento a dos fragmentos de datos y uno de paridad,
   reparación y sincronización. Probar la caída de un directorio, la pausa de un
   anfitrión y la pérdida de una respuesta.
3. Probar el nodo de escritorio en Windows, Linux y macOS. Un PC de cuatro núcleos
   y 8 GB de RAM será la referencia para medir el consumo durante el almacenamiento
   y el tiempo de respuesta al pausar. Medir por separado la demanda de los directorios
   y relays. Los celulares y tabletas se evaluarán primero como clientes de acceso y
   recuperación.
4. Probar sesiones de CPU y RAM de propósito general con hardware que pueda demostrar
   la protección requerida. Los ensayos posteriores cubrirán sesiones que puedan
   verificar la GPU y después el alojamiento persistente con un relevo compatible y
   un estado guardado recuperable.

El repositorio publica el diseño, sus supuestos y los criterios de aceptación.
La implementación y las mediciones corresponden a hitos posteriores. [Hoja de ruta](roadmap.md)
