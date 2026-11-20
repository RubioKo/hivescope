<!-- docs-pair: home; revision: 7 -->

[English](README.md) | Español

![HiveScope — almacenamiento comunitario de código abierto. En fase de diseño. Buscamos colaboradores.](assets/banner.es.svg)

# HiveScope

HiveScope es un proyecto de código abierto para que las personas puedan usar sus propias
computadoras como **pequeños servidores comunitarios**. Quien tenga espacio libre podría
guardar una copia cifrada del trabajo de una amiga. Alguien con un equipo más potente
podría ayudar a subtitular grabaciones de un archivo público. Un grupo de jugadores podría turnarse
para mantener en línea su mundo compartido. Cada persona decide qué presta, a quién y por cuánto tiempo.

**Estado del proyecto — diseño y documentación.** El primer prototipo previsto es una
herramienta de línea de comandos para almacenar y recuperar un archivo. En este repositorio
se describen el diseño, el modelo de seguridad y las pruebas que queremos realizar.
[Visión del proyecto](docs/es/vision.md) · [Hoja de ruta](docs/es/roadmap.md)

## Para qué podría usarlo la comunidad

- **Proteger archivos y trabajar en varios equipos.** Cada persona podría elegir carpetas con
  fotos familiares, trabajos de clase, música, código fuente u otros archivos. Su equipo
  cifraría el contenido antes de enviarlo a los proveedores de almacenamiento aprobados.
  La primera meta es recuperar los archivos. Más adelante, los equipos autorizados
  sincronizarían los cambios y conservarían ambas versiones si hubiera ediciones incompatibles sin conexión.
- **Transcribir una grabación privada.** HiveScope trataría toda grabación personal
  como privada por defecto. Una creadora de pódcast podría transcribir su entrevista
  en su propia computadora. Durante la tarea, el audio y el texto se quedarían allí.
  Si faltaran recursos, HiveScope le avisaría y le permitiría reintentar en ese mismo equipo.
- **Prestar cómputo para trabajos públicos.** En el primer piloto comunitario de IA,
  un voluntario podría ejecutar un modelo abierto aprobado sobre una grabación ya
  publicada, como un cuento que una biblioteca ofrece para su transcripción. La solicitud identificaría
  una entrada de un catálogo público aprobado. El anfitrión podría acceder a esa
  grabación y al texto resultante.
- **Mantener un servicio en línea entre varias personas.** En la prueba de alojamiento de
  juegos, el grupo elegiría un anfitrión activo y otro de relevo. Si el primero se desconecta,
  el segundo podría restaurar el último guardado válido del mundo y los jugadores podrían
  volver a entrar. El anfitrión puede leer los datos del mundo que procesa. Project Zomboid
  es uno de los ejemplos propuestos. Otro posible uso sería el calendario de una asociación,
  que necesitaría sus propias pruebas de alojamiento y recuperación.

El objetivo es que se puedan utilizar estos servicios sin pagar un alquiler obligatorio.
Los participantes o donantes asumirían los costes del equipo, la electricidad, la conexión
y el tráfico que pase por relays. En juegos, el piloto mediría cuándo pueden entrar los
jugadores, qué latencia experimentan desde sus conexiones, cuánto tardan en reconectarse
y cuánto progreso se pierde durante un relevo. Así, cada grupo podría evaluar hasta qué
punto puede mantener su mundo disponible las 24 horas. [Prueba de alojamiento comunitario](docs/es/roadmap.md#prueba-de-un-servidor-de-juego-comunitario)

## Qué significa ofrecer tu equipo

Luis podría prestar a Ana 20 GB de disco por las tardes y reservar su GPU para su propio
trabajo. En el flujo propuesto **Ofrecer mi equipo**, elige un servicio y un contacto
aprobado, fija el horario y la fecha de vencimiento, y revisa los límites. Para disco
establece una cuota y límites de transferencia. Para cómputo puede limitar la CPU, la
memoria, el uso de la GPU y las tareas simultáneas. La oferta queda pendiente hasta que
Ana acepta las condiciones. Ana también puede participar solo para solicitar recursos si
alguien acepta prestárselos.

Cuando Luis necesita su equipo, puede pausar las solicitudes nuevas. Detener una
transcripción del catálogo público en curso puede hacer que se pierda el progreso sin guardar. Si apaga un
equipo que presta disco, sus fragmentos quedan temporalmente inaccesibles. Para ceder
un mundo de juego hacen falta otro anfitrión autorizado y un guardado válido. La pantalla
de apagado propuesta le mostraría a Luis cómo se vería afectado el servicio antes de
decidir qué hacer. [Ofertas de recursos](docs/es/resource-offers.md)

## Cómo funcionaría una carpeta privada

### Ana elige qué equipos pueden leer sus archivos

![Ana señala una foto familiar que aparece en su portátil junto a un documento y la onda de una grabación.](assets/story/hivescope-story.jpg)

Ana quiere abrir el borrador de su tesis, una foto y una entrevista grabada tanto en su
portátil como en su computadora de escritorio. Autoriza ambos equipos para `Estudios`.
Para `Documentos personales` solo autoriza el portátil. Cada elección se aplica a todos
los archivos de esa carpeta.

### Luis guarda la copia cifrada

![Luis ajusta un control debajo de dos filas de bloques de colores que representan datos cifrados en su pantalla.](assets/story/encrypted-storage.jpg)

La computadora de Ana cifra el contenido y los nombres de los archivos antes de enviarlos.
Luis guarda fragmentos cifrados y puede observar cuánto se transfiere y en qué momento.
Las claves que abren los archivos permanecen en los equipos autorizados por Ana. Si Ana
transcribe su entrevista privada, el modelo se ejecutaría en su propia computadora.
Luis seguiría alojando únicamente los fragmentos cifrados de la copia.
[Modelo de seguridad](docs/es/security.md)

### Ana recupera su trabajo en otra computadora

![Ana sostiene una memoria USB junto al portátil cerrado. La foto, el documento y la grabación vuelven a aparecer en un monitor de escritorio.](assets/story/recovering-files.jpg)

Con el portátil original apagado, Ana usa su contraseña para abrir el paquete de
recuperación que guardó aparte. Su nuevo equipo necesita acceso al catálogo y suficientes
fragmentos válidos de proveedores cuyos acuerdos aún permitan recuperar los datos.
El primer prototipo deberá recuperar un archivo de prueba idéntico al original, byte por
byte. [Diseño de recuperación](docs/es/architecture.md#propiedad-y-recuperación) · [Almacenamiento y reparación](docs/es/storage.md)

## Desarrollo y plataformas

1. **Núcleo seguro.** Crear un prototipo por línea de comandos en Go para identificar
   equipos, aprobar pares, transferir archivos cifrados y recuperarlos en otra computadora.
2. **Almacenamiento resistente y sincronización de carpetas.** Distribuir cada bloque
   cifrado entre tres proveedores aprobados. Cualquier par de fragmentos válidos
   permitirá reconstruirlo. Probar la reparación cuando uno de los proveedores
   se desconecte y, después, las ediciones sin conexión y los conflictos entre los
   equipos del propietario. [Sincronización de carpetas](docs/es/synchronization.md)
3. **Alfa cerrada para computadoras.** Validar la herramienta de línea de comandos en
   Windows, macOS y Linux con conexiones domésticas reales. La interfaz gráfica está
   prevista para cuando el servicio en segundo plano sea utilizable. Los clientes para
   celulares y tabletas siguen en fase exploratoria.

El cómputo comunitario y el alojamiento de servicios persistentes son propuestas para
etapas posteriores. Para quienes quieran participar en el desarrollo técnico, el diseño
describe un servicio en segundo plano en Go, conexiones autenticadas entre pares con
libp2p y un índice SQLite local en cada nodo. [Arquitectura](docs/es/architecture.md) · [Índice de documentación](docs/es/README.md)

## Cómo contribuir

- **Revisa un caso de fallo.** ¿Qué debería ver una persona si un proveedor de almacenamiento
  se desconecta a mitad de la recuperación? Describe el comportamiento esperado con archivos
  de ejemplo. [Proponer un caso](https://github.com/RubioKo/hivescope/issues/new?template=proposal.yml)
- **Diseña una prueba de alojamiento en casa.** Describe el equipo, el horario y el ancho
  de banda que podría prestar una persona, o cómo un grupo comprobaría que puede seguir
  jugando en su mundo después de un relevo. [Abrir una propuesta en Issues](https://github.com/RubioKo/hivescope/issues)
- **Mejora la documentación.** Mantenemos las versiones en inglés y español en paralelo.
  Puedes aclarar una explicación, revisar una traducción o ayudar a incorporar otro idioma.
  [Guía de traducción](docs/es/translations.md)

Crea una rama de trabajo a partir de `development` y dirige tu pull request a
`development`. Los hallazgos de seguridad sensibles cuentan con un canal privado de
reporte. [Guía de contribución](docs/es/contributing.md) · [Informes de seguridad](SECURITY.md) · [Gobernanza](docs/es/governance.md) · [Código de conducta](docs/es/code-of-conduct.md)

Las instrucciones para colaboradores y las herramientas opcionales de asistencia tienen
su propia documentación. [Instrucciones para colaboradores](AGENTS.md) · [Guía opcional de IA y ECC](docs/es/ai-assisted-development.md)

## Licencia y atribución

El material nuevo de HiveScope utiliza la licencia **AGPL-3.0-only**. Los banners
animados, adaptados del diseño original de RubioKo, conservan su licencia **MIT** y su
atribución. [Licencia del proyecto](LICENSE) · [Guía de licencias](docs/es/licensing.md) · [Avisos de terceros](THIRD_PARTY_NOTICES.md)
