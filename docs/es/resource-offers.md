<!-- docs-pair: resource-offers; revision: 2 -->

[English](../en/resource-offers.md) | Español | [Índice](README.md)

# Ofrecer mi equipo

Propuesta de diseño para escritorio. Al pulsar **Ofrecer mi equipo**, Luis abre
un borrador local. Las tres actividades empiezan sin seleccionar. **Continuar**
se habilita al elegir una. Luis puede salir sin compartir recursos. La primera implementación
prevista cubre almacenamiento por línea de comandos. La transcripción de grabaciones
públicas de un catálogo y el alojamiento de juegos ilustran servicios posteriores.

## Qué elige quien presta el equipo

1. **Servicio y equipo.** Luis elige su computadora y una actividad concreta.
   Puede crear ofertas independientes para guardar copias cifradas, transcribir
   grabaciones de un catálogo público con una herramienta aprobada o alojar un servidor de juego aprobado.
2. **Quién puede usarlo.** Elige un contacto aprobado para ese acuerdo. Puede crear
   otra oferta para otra persona dentro de la capacidad total de su equipo. En un juego,
   el resumen identifica al responsable del mundo. Entrar como jugador solo
   concede los permisos de la partida.
3. **Cuánto y cuándo.** Elige los límites de la actividad, el horario con su
   zona horaria, la vigencia de la oferta y si requiere conexión a la corriente.
   Ve la capacidad ya comprometida y la que quedará para su propio uso. Si dos
   ofertas solapadas superan un límite local, debe ajustar una antes de enviarla.

| Servicio | Límites que revisa Luis | Datos a los que tendrá acceso su equipo |
| --- | --- | --- |
| Copia cifrada | Directorio de destino, cuota de disco, reserva libre y topes de subida y bajada. La cuota incluye los temporales. | Fragmentos cifrados y tamaño de las transferencias. Las claves de lectura permanecen con los dispositivos autorizados por el dueño. |
| Transcripción del catálogo público | Herramienta admitida, CPU, memoria, uso opcional de GPU y tareas simultáneas. | Luis recibe un identificador del catálogo público. Puede escuchar la grabación publicada a la que corresponde y leer la transcripción. |
| Servidor de juego | Juego y versión, CPU, memoria, disco, conexión, horario y posible anfitrión de relevo. | Los datos del mundo que ejecuta el servidor. El relevo requiere autorización y un guardado válido. |

## Qué autoriza el botón de confirmación

El resumen reúne el contacto, el servicio, sus límites, los datos que recibirá
el equipo y lo que ocurrirá si Luis interrumpe el servicio. Puede **Volver y
editar** o **Enviar oferta a Ana**. Por ejemplo, podría ofrecer transcribir para
Ana una grabación del catálogo público a la vez, con un máximo de 2 núcleos y 4 GB de memoria, GPU
desactivada, de 18:00 a 21:00 en su hora local y solo con corriente. Esos valores
son un ejemplo que Luis eligió para esta oferta.

Enviar la oferta la deja **pendiente de aceptación**. Luis puede retirarla mientras
está pendiente. Si vence antes de que Ana la acepte, hay que hacer una oferta nueva.
Cuando Ana acepta las mismas condiciones, el acuerdo autoriza las solicitudes
de Ana que coincidan con el servicio, el horario y los límites. En el ejemplo,
Ana elige una entrada del catálogo público y confirma la tarea. El único dato que
puede aportar como entrada es el identificador. Luis obtiene la grabación publicada
directamente. Cambiar el proveedor de esa tarea requiere la aprobación de Ana.

Si Ana abre su propia entrevista para transcribirla, HiveScope ejecutaría la tarea
en su dispositivo. La grabación y la transcripción permanecen allí, incluso si los
recursos son insuficientes o falla la tarea. Aceptar la oferta de Luis no permite
reenviar una grabación privada a su equipo.

Ampliar destinatarios, recursos o permisos requiere revisar y aceptar las nuevas
condiciones. Luis puede pausar nuevas solicitudes de inmediato. Fuera del horario
acordado, el equipo deja de admitir trabajos. Una tarea solo podrá seguir usando
recursos fuera de ese horario si Luis acepta la ampliación.

## Si Luis necesita apagar la computadora

**Preparar apagado** pausa las nuevas solicitudes y muestra qué sigue en uso.
Luis puede revisar las consecuencias y elegir si espera o detiene el trabajo.
**Volver** permite seguir observando la tarea. Para admitir nuevas solicitudes
otra vez, Luis elige **Reanudar** dentro del horario acordado.
Conserva el control del apagado desde su sistema operativo incluso si falta un relevo.

| Qué sigue en uso | Qué debe explicar la pantalla |
| --- | --- |
| Copias cifradas | Mientras esté apagado, este equipo no podrá servir sus fragmentos. Los datos locales se conservarán conforme al acuerdo. Para retirarse, Luis podrá ver qué copias tienen migración confirmada y cuáles siguen pendientes. |
| Una transcripción del catálogo público | Luis puede esperar a que termine o elegir **Detener la tarea**. Si la detiene, Ana verá la interrupción cuando su equipo reciba la confirmación. Solo se mostrará progreso recuperable cuando la herramienta lo haya guardado y verificado. |
| Un mundo de juego | Antes de detenerlo, Luis ve el último guardado válido y si hay relevo confirmado. Los jugadores pueden necesitar reconectarse. Si falta relevo, el mundo quedará fuera de línea. Solo un anfitrión podrá aceptar cambios al mundo a la vez. |

Si se corta la corriente, los demás equipos solo pueden indicar que han perdido
contacto. Al reconectar, deben comprobar el último resultado, guardado o recibo
válido antes de anunciar una recuperación. Retirar una oferta de disco mantiene
los compromisos de retención, migración y vencimiento del acuerdo de
[almacenamiento](storage.md). La decisión de cómo conservar el progreso de una
transcripción sigue abierta.

## Cómo probar la decisión

- Salir del borrador y comprobar que ese borrador no habilitó ningún servicio.
- Aceptar solo disco para Ana e intentar solicitar una transcripción con ese acuerdo.
  La solicitud debe quedar sin autorizar.
- Con un acuerdo de cómputo vigente, intentar enviar una grabación local,
  instrucciones privadas o un identificador desconocido. Deben rechazarse antes de que
  los datos de entrada salgan del equipo. Con una entrada válida del catálogo público
  la tarea sí debe poder ejecutarse.
- Interrumpir una transcripción privada en el equipo de Ana y reintentarla con un
  anfitrión voluntario disponible. Comprobar que el audio y el texto siguen en su equipo.
- Mostrar el resumen a quien presta el equipo por primera vez. Antes de enviar,
  pedirle que señale quién puede usarlo, los límites y el efecto de apagarlo.
- Ensayar un apagado avisado y un corte inesperado durante una tarea. Comprobar
  lo que cada persona ve cuando recibe confirmación y cuando pierde el contacto.
