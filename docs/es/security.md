<!-- docs-pair: security; revision: 3 -->

[English](../en/security.md) | [Español](security.md) | [Índice](README.md)

# Seguridad y modelo de amenazas

HiveScope está en diseño. Las propiedades que se describen son requisitos para
las futuras implementaciones. El diseño está pendiente de revisión criptográfica.
Conserva una copia independiente de cualquier archivo que uses en un futuro piloto.

## Límites de confianza

| Participante | Acceso previsto | Riesgo restante |
| --- | --- | --- |
| Dispositivo autorizado por el dueño | Texto legible, claves de la carpeta y ediciones permitidas. | Un equipo comprometido expone esas carpetas y puede publicar cambios dañinos. |
| Proveedor de almacenamiento | Operaciones acordadas sobre objetos opacos, datos cifrados y metadatos de mantenimiento limitados. | Puede perder datos, declarar una capacidad falsa, coludirse u observar tráfico. |
| Reparador delegado | Reconstrucción y reemplazo limitados de datos cifrados. | Puede desperdiciar los recursos permitidos o no reparar un fragmento. |
| Directorio comunitario | Anuncios de servicio con vencimiento y datos de contacto del nodo. | Puede omitir ofertas, conservar metadatos de contacto o devolver candidatos desactualizados. |
| Relay | Metadatos de conexión y transporte de tráfico cifrado. | Puede observar pares, retrasar el tráfico o denegar el servicio. |
| Anfitrión de cómputo | Máquina física, recursos ofrecidos, horarios y volumen de tráfico. | Puede pausar o cortar la corriente. La protección del contenido de la sesión depende del entorno verificado. |
| Solicitante | Su propia sesión, programas y resultados dentro de los recursos acordados. | Una tarea puede agotar su asignación. El anfitrión necesita aislamiento respecto al invitado y límites que pueda hacer cumplir. |
| Mantenedor | Revisión del repositorio y administración de publicaciones. | Un canal de distribución comprometido puede publicar software malicioso. |

Cualquiera puede solicitar capacidad. Una identidad de dispositivo autentica a un
par y una persona puede controlar varias identidades. Cada anfitrión debe limitar
el total de disco, ancho de banda, reservas, flujos y trabajos simultáneos, además
del uso por solicitante. Contabilizar temporales y reservas incompletas. Probar
solicitudes simultáneas, identificadores repetidos y reconexiones masivas. Un
anuncio no concede acceso a carpetas ni sesiones. El nodo debe autenticar al par
y autorizar cada operación.

## Defensas requeridas para el almacenamiento

- Usar bibliotecas criptográficas y de transporte establecidas. Separar las
  claves de carpeta y dispositivo, las autorizaciones de recuperación y los permisos de reparación.
- Cifrar nombres, rutas y catálogos. Verificar los compromisos sobre el contenido
  cifrado y la autenticación de los bloques antes de publicar el archivo recuperado.
- Acotar el análisis de mensajes y las colas. Rechazar solicitudes sin autorización,
  repetidas de forma indebida, mal formadas o excesivas antes de asignar recursos significativos.
- Limitar las escrituras a las raíces configuradas. Tratar las rutas remotas y
  las entradas temporales como datos sin verificar. Limpiar solo las entradas de la operación interrumpida.
- Proteger los secretos locales con desbloqueo explícito y cifrado en reposo. Las
  exportaciones de recuperación requieren un formato revisado con contraseña y una acción expresa del dueño.
- Conservar las épocas de autorización y las últimas versiones verificadas para
  limitar retrocesos. Un paquete y un proveedor desactualizados pueden ocultar
  una versión más reciente. Una revocación confirmada debe sobrevivir al reinicio del proveedor.

## Qué puede proteger el almacenamiento

El contenido y los nombres se cifran en el dispositivo del dueño. Los proveedores
pueden observar direcciones IP, tamaños, horarios, ubicaciones de copias y tareas
de reparación. Un recibo firmado registra una obligación en un momento concreto.
La recuperación sigue necesitando datos válidos, catálogo, claves y un proveedor
accesible con permiso de lectura vigente. Los cortes simultáneos en varios hogares,
el vencimiento de acuerdos o la pérdida de claves pueden impedir la recuperación.
El texto ya leído en un equipo del dueño que haya sido comprometido puede permanecer allí.

## Decisión propuesta para el cómputo privado

El solicitante controla qué instala y ejecuta en su entorno invitado. El anfitrión
debe aislar a ese invitado de sus archivos, credenciales, otras sesiones y el
hardware ajeno a la asignación. Las cuotas controlan el consumo. La protección
de los datos privados durante el procesamiento exige además un entorno verificable.

El hito 5 evaluará una máquina virtual confidencial con atestación respaldada
por hardware. Antes de entregar una clave de sesión o una entrada privada, el
solicitante verifica evidencia reciente y firmada contra las mediciones aceptadas
y su política de plataforma. La evidencia debe identificar al invitado previsto
y vincular su clave de canal a esa sesión. Solo por ese canal se entregan los
secretos. Una solicitud de GPU debe verificar también el acelerador y el recorrido
de los datos hasta él. Ante evidencia ausente, caducada, inválida o que no coincida,
la solicitud espera y el solicitante conserva las claves. Cada nuevo anfitrión,
incluido el de relevo de un juego, debe superar la verificación antes de recibir
secretos de sesión. [Aislamiento y entrega de claves en una VM confidencial](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-vm-overview) · [Límites de confianza de CPU y GPU](https://docs.nvidia.com/confidential-computing/virtual-machines/threat-model)

Solo entran en la sesión los datos que el solicitante elige para el invitado.
Las claves de carpeta y los permisos de recuperación se conservan por separado.
Cifrar los guardados y resultados antes de que salgan del invitado hacia el
almacenamiento comunitario. Excluir contenidos, rutas privadas y claves de
registros, informes de fallos y entradas de directorio. El software invitado
puede utilizar o divulgar las entradas que recibe. Un anfitrión verificado aún
puede desconectarse o retener un resultado. Publicar hardware compatible,
componentes de confianza y riesgos restantes junto con las mediciones antes de ofrecer el servicio.

**Pruebas de aceptación futuras.** Tras una verificación válida, ejecutar un programa
elegido por el solicitante con datos sintéticos. Comprobar que un anfitrión
incompatible, una evidencia reutilizada, otra clave de invitado o la falta de la
prueba de GPU requerida producen cero claves de sesión y cero datos privados en
ese anfitrión, también al reintentar. Pausar una sesión activa y comprobar que el
consumo cesa según lo acordado. Informar únicamente del último guardado confirmado.
Intentar reanudar en otro anfitrión y exigir evidencia reciente antes de restaurar
el estado. Buscar el marcador sintético, las rutas privadas y los secretos en los
registros e informes de fallos.

## Informar un problema

Utiliza los [informes privados de vulnerabilidad de GitHub](https://github.com/RubioKo/hivescope/security/advisories/new)
para hallazgos sensibles. Incluye el documento o versión futura, cómo reproducir el
caso con datos sintéticos, el comportamiento esperado y el impacto. Conserva en
privado los archivos reales, credenciales, exportaciones y registros personales.
Se aceptan inglés y español.

Utiliza los [Issues públicos](https://github.com/RubioKo/hivescope/issues) para
preguntas de diseño no sensibles. Los mantenedores evaluarán el impacto, coordinarán
correcciones y documentarán los resultados verificados. Quedan sin establecer un
plazo de respuesta y un programa de recompensas.
