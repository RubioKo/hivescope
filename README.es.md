<!-- docs-pair: home; revision: 2 -->

[English](README.md) | Español

![HiveScope — almacenamiento descentralizado, código abierto y comunidad. En fase de diseño; buscamos colaboradores.](assets/banner.es.svg)

# HiveScope

**¿Qué archivos querrías recuperar si mañana tu computadora no encendiera?**
Quizá sean tus fotos familiares, una tesis, tus grabaciones de música o el código de un proyecto.
HiveScope es un proyecto de almacenamiento comunitario de código abierto. Lo diseñamos para
que puedas elegir qué carpetas proteger, guardar copias cifradas en equipos de personas de
tu confianza y recuperar tus archivos en otro dispositivo.
Tú conservas las claves y decides cuáles de tus equipos pueden leer cada carpeta. Quien presta
espacio elige cuánto disco y ancho de banda comparte y en qué horario.

**Estado del proyecto — diseño y documentación.** El primer prototipo se usará desde la línea
de comandos en una computadora. Las plataformas previstas son Windows, Linux y macOS. La
primera aplicación instalable sigue pendiente. [Visión](docs/es/vision.md) · [Hoja de ruta](docs/es/roadmap.md)

## Los archivos de Ana y el espacio que presta Luis

[![Ana frente a su computadora con una foto familiar, el borrador de su tesis y una grabación. Ver la historia ilustrada.](assets/story/hivescope-story.jpg)](assets/story/hivescope-es.mp4)

[Ver o descargar el ejemplo ilustrado del diseño en español (MP4, 30 segundos)](assets/story/hivescope-es.mp4)

1. **Ana elige una carpeta.** En `Estudios` guarda una foto familiar, el borrador de su tesis
   y la grabación de una entrevista. Quiere abrir esos archivos tanto en el portátil como en
   su computadora de escritorio. Para `Documentos personales` solo autoriza el portátil.
   Cada elección se aplica a todos los archivos de esa carpeta. [Los archivos de Ana en la ilustración](assets/story/hivescope-story.jpg)
2. **Luis presta espacio.** Acepta alojar una copia cifrada, fija un límite de disco y
   acuerda recibir transferencias por la tarde. La computadora de Ana cifra los archivos
   y sus nombres antes de enviarlos. Luis puede ver el tamaño de una transferencia.
   Las claves que permiten abrir el PDF, ver la foto o escuchar la grabación se quedan
   con Ana. [Lo que ve Luis](assets/story/encrypted-storage.jpg) · [Modelo de seguridad](docs/es/security.md)
3. **Ana recupera su trabajo.** Con el portátil original apagado, Ana usa una computadora
   de reemplazo y su contraseña para abrir el paquete de recuperación que guardó aparte.
   Mientras el equipo de Luis esté accesible y el acuerdo permita la lectura, puede
   recuperar y descifrar `Estudios`. Allí vuelven a estar la misma foto, el borrador y la entrevista.
   [La escena de recuperación](assets/story/recovering-files.jpg)

El primer prototipo probará la recuperación con un archivo de ejemplo. En hitos posteriores
se probará la recuperación con un proveedor desconectado y la sincronización entre los equipos
del propietario. [Almacenamiento y mantenimiento](docs/es/storage.md) · [Sincronización de carpetas](docs/es/synchronization.md)

## Cómo puedes ayudar hoy

- **Revisa un caso de recuperación.** Si la transferencia se corta a mitad de camino,
  ¿qué debería ver Ana en el equipo nuevo? Describe el resultado esperado y cómo comprobarlo.
  [Arquitectura](docs/es/architecture.md) · [Proponer un caso](https://github.com/RubioKo/hivescope/issues/new?template=proposal.yml)
- **Diseña una prueba en casa.** ¿Cuánto disco podrías prestar, en qué horario y con qué
  límite de subida? Propón una prueba que respete esos límites cuando el equipo se suspenda.
  [Proponer una prueba](https://github.com/RubioKo/hivescope/issues/new?template=proposal.yml)
- **Mejora una explicación o traducción.** Elige un pasaje que cueste explicarle a otra
  persona. Sugiere una redacción más clara en inglés o español, o ayuda a incorporar otra
  lengua. [Guía de traducción](docs/es/translations.md)

Aceptamos preguntas y propuestas en inglés y español. Para comunicar un hallazgo de seguridad
sensible, utiliza el canal privado. [Issues de GitHub](https://github.com/RubioKo/hivescope/issues) ·
[Informes de seguridad](SECURITY.md)

[Guía de contribución](docs/es/contributing.md) · [Índice de documentación](docs/es/README.md) ·
[Gobernanza](docs/es/governance.md) · [Código de conducta](docs/es/code-of-conduct.md) ·
[Instrucciones para colaboradores](AGENTS.md) · [Guía opcional de IA y ECC](docs/es/ai-assisted-development.md)

## Licencia

El material nuevo del proyecto utiliza **AGPL-3.0-only**. Los banners animados adaptados conservan
su licencia **MIT** y su atribución. [Licencia del proyecto](LICENSE) ·
[Guía de licencias](docs/es/licensing.md) · [Avisos de terceros](THIRD_PARTY_NOTICES.md)
