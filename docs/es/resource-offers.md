<!-- docs-pair: resource-offers; revision: 4 -->

[English](../en/resource-offers.md) | Español | [Índice](README.md)

# Ofrecer mi equipo

El flujo previsto para escritorio empieza con dos decisiones independientes.
Ana puede solicitar una copia o una sesión de cómputo sin prestar hardware.
Luis puede ofrecer su equipo a la red abierta activando una sola vez un perfil
local de recursos. A partir de ahí, sus dispositivos pueden llegar a un acuerdo
sin que Ana y Luis tengan que conocerse o ponerse en contacto.

## Qué ofrece Luis

Luis abre **Ofrecer mi equipo**, elige un servicio y revisa los límites. Por
ejemplo, podría prestar hasta 20 GB para copias cifradas por las tardes y reservar
la GPU para sí mismo. Antes de activar el perfil ve el horario y la zona horaria,
si el equipo debe estar conectado a la corriente, el espacio ya comprometido y
los recursos que conservará. Puede salir del borrador sin compartir nada.

| Servicio | Qué elige Luis | Qué cubre el acuerdo |
| --- | --- | --- |
| Almacenamiento cifrado | Directorio, cuota total de disco, espacio libre que quiere conservar, límites de transferencia y horario. | El anfitrión conserva datos cifrados. Las reservas, los metadatos y los temporales cuentan para la cuota física. El dueño de la carpeta elige qué dispositivos pueden leer los archivos. |
| Cómputo privado | CPU, RAM, GPU opcional, ancho de banda, número de sesiones y horario. | Quien solicita elige y ejecuta el software dentro de un entorno privado verificado. Luis controla su máquina física y hace cumplir los límites. |
| Alojamiento de un servicio o juego | Capacidad para la sesión, disco persistente, ancho de banda y disponibilidad. | Quien solicita controla el servicio y sus datos. El relevo debe cumplir las condiciones de protección de la sesión y contar con un guardado verificado para asumir el servicio. |

El anuncio del directorio incluye el servicio, los datos de contacto del nodo y
un vencimiento. Los nombres de archivos, el contenido de las sesiones y el horario
personal de Luis quedan fuera del directorio. Los participantes pueden observar
detalles operativos como los horarios de conexión y el tamaño de las transferencias.
[Modelo de seguridad](security.md)

## Cómo llegan las máquinas a un acuerdo

1. Ana elige una carpeta que proteger o indica qué recursos necesitará su sesión.
   Su nodo consulta un directorio comunitario. Si está fuera de servicio, puede
   probar otro. El anuncio permite localizar a un candidato.
2. Los dispositivos se autentican. El nodo de Luis comprueba su perfil, el
   horario, la capacidad libre en el dispositivo ofrecido y sus compromisos
   vigentes. Solo concede una reserva con vencimiento si cabe en el presupuesto
   total del anfitrión y en los límites del solicitante. Si dos peticiones
   compiten por el último espacio disponible, como máximo una lo reserva.
3. El equipo de Ana utiliza la reserva para la operación acordada. Si reintenta
   una solicitud con el mismo identificador, recupera su estado y la cuota se
   contabiliza una sola vez. Una reserva sin utilizar vence y libera capacidad.
   Un acuerdo de almacenamiento aceptado tiene sus propios plazos de retención
   y recuperación.
4. Si falta un anfitrión adecuado, Ana ve el motivo de la espera y puede cancelar.
   Su nodo puede probar otro candidato con las mismas condiciones. Tras una
   desconexión, cualquier resultado pendiente de confirmación permanece en ese
   estado hasta que los equipos concilien la solicitud.

Para el cómputo privado, el dispositivo de Ana comprueba pruebas recientes que
identifican la máquina virtual confidencial y la clave de esa sesión antes de
enviar secretos o un trabajo privado. Si su programa usa una GPU, la verificación
también debe cubrir la GPU y la comunicación protegida con ella. Si un candidato
falla la comprobación, Ana conserva las claves y la solicitud espera un anfitrión
compatible. Tras verificarlo, puede cargar sus propias herramientas, código y
datos en la sesión. Una sesión posterior en otro anfitrión requiere una nueva
verificación. [Arquitectura](architecture.md)

## Pausa y apagado

**Pausar** detiene de inmediato las nuevas reservas. Luis puede ver qué trabajos
admitió y cuáles interrumpiría el próximo apagado. Solo él puede ampliar su
perfil. Conserva el control de su equipo aunque falle un relevo.

| Trabajo en curso | Qué necesitan ver los participantes |
| --- | --- |
| Copias almacenadas | Mientras esté apagado, el anfitrión no podrá entregar sus fragmentos. La retención, la migración y el vencimiento siguen el acuerdo de almacenamiento. El dueño ve qué reemplazos se han confirmado. |
| Sesión de cómputo | Al terminar el horario acordado o si Luis detiene la sesión, Ana ve el último resultado o punto de recuperación verificado, si su programa lo guardó. El anfitrión libera la CPU, la RAM y la GPU de esa sesión. |
| Servicio o juego | Mostrar si un relevo compatible puede restaurar el último estado verificado. Solo un anfitrión puede aceptar cambios. Si el relevo no se completa, se informa de la interrupción para que los clientes puedan volver a conectarse cuando se reanude el servicio. |

Tras un corte inesperado, lo primero que puede observarse es la pérdida de
conexión. El solicitante comprueba el último recibo, resultado o guardado antes
de anunciar una recuperación. [Almacenamiento y reparación](storage.md)

## Cómo comprobar el diseño

- Contrastar las opciones de solicitar y ofrecer con el perfil local. Quien solicita
  elige el software, el anfitrión fija los límites físicos y las claves permanecen
  con el solicitante mientras falta hardware compatible.
- Empezar con los recursos desactivados. Activar un perfil que solo ofrece disco,
  solicitar CPU y después solicitar disco fuera de horario. Comprobar que ninguna
  petición ocupa capacidad.
- Interrumpir una transferencia tras guardarla en el anfitrión y antes de
  que el solicitante vea el recibo. Comprobar que el reintento deja un solo
  objeto y un solo cargo de cuota. Pausar el trabajo nuevo y comprobar que se
  mantienen los compromisos de retención.
- Probar una sesión privada con pruebas de protección ausentes o caducadas y
  otra con una GPU sin verificar. Comprobar que las claves de sesión y las entradas
  privadas permanecen en el equipo solicitante, incluso al reintentar. En hardware
  compatible, ejecutar un programa elegido por el solicitante y recuperar el resultado.
