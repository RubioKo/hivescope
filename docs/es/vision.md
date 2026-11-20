<!-- docs-pair: vision; revision: 5 -->

[English](../en/vision.md) | [Español](vision.md) | [Índice](README.md)

# Visión y límites

HiveScope busca que cualquier persona con una computadora compatible pueda usarla como
un pequeño servidor comunitario. Podría prestar espacio para archivos cifrados, ejecutar
una tarea para otra persona o mantener en línea un servicio que use un grupo. El primer
servicio previsto es el almacenamiento privado. El dueño de cada carpeta conserva sus
claves y decide qué dispositivos pueden leerla.

## Cómo se prestan los recursos

Quien presta el equipo elegiría qué servicios acepta, a quién se los ofrece, en qué
horario y con qué límites de disco, CPU, memoria, GPU disponible y conexión. Por ejemplo,
Luis podría ofrecer solo disco durante el día y ayudar a transcribir grabaciones públicas por la noche.
Quien necesita recursos elegiría una oferta compatible entre sus contactos aprobados.
Podría solicitar un servicio aunque su propio equipo solo participe como cliente, si la
otra parte acepta el acuerdo. Cada tarea usaría únicamente los datos y recursos acordados.
El flujo propuesto muestra el acuerdo antes de enviarlo y las consecuencias de
apagar el equipo durante el servicio. [Ofrecer mi equipo](resource-offers.md)

## Qué problemas queremos resolver

- **Mi única copia está en una computadora que puede fallar.** Una familia podría
  proteger sus fotos, y una estudiante, su tesis o el código de un proyecto.
  La primera prueba será apagar el equipo original y recuperar un archivo idéntico en
  otro equipo con el paquete de recuperación, su contraseña y un proveedor accesible
  cuyo acuerdo permita la lectura.
- **Trabajo en dos equipos y ya no sé cuál tiene la última versión.** En un hito posterior,
  los equipos autorizados por el propietario podrán sincronizar una carpeta. Si se edita
  el mismo documento en ambos mientras están desconectados, la prueba será encontrar
  las dos versiones al reconectarlos y poder elegir cuál conservar.
- **Nuestro archivo necesita transcripciones de grabaciones públicas.** En el piloto
  propuesto, una voluntaria podría elegir una grabación de dominio público ya publicada
  en un catálogo revisado por la comunidad. Un anfitrión podría dedicar tiempo de su
  equipo a convertirla en texto. Una creadora que quiera transcribir una entrevista
  inédita usaría su propio dispositivo.
- **Quiero ayudar y sigo necesitando mi computadora.** Una persona podría prestar disco
  por la noche y reservar la capacidad de cómputo para su propio trabajo. La prueba será
  comprobar que HiveScope respeta su horario y su cuota de disco y que, para ejecutar
  una transcripción, hace falta que la persona acepte además prestar cómputo.
- **Nuestro mundo de juego se apaga cuando se va quien lo creó.** Un grupo de jugadores
  podría organizar turnos de alojamiento entre equipos voluntarios y seguir jugando
  en el mismo mundo sin depender del horario de una sola persona.

## Servidores de juego y otros servicios comunitarios

Para probar el alojamiento continuo proponemos un juego cuyo servidor dedicado permita
guardar y restaurar un mundo. La comunidad sugirió Project Zomboid como ejemplo. En el
piloto, un equipo autorizado mantendría la partida y otro estaría preparado para tomar
el relevo con el último guardado válido. Solo uno aceptaría
cambios en el mundo a la vez. Cada anfitrión aceptaría ese servicio por separado y
elegiría los recursos y horarios que ofrece. Quien ejecuta el servidor puede acceder
a los datos del mundo que procesa.

El objetivo sería que el grupo pudiera entrar a cualquier hora, con baja latencia
y sin pagar un alquiler obligatorio. Para evaluar esos tres objetivos mediríamos el
tiempo durante el cual se puede entrar y jugar, el tiempo de respuesta desde las
conexiones reales de los jugadores y los recursos que aporta cada anfitrión. También
probaríamos un cambio de turno y un corte inesperado para saber cuánto tarda el grupo
en reconectarse y cuánto progreso puede perder. La electricidad, la conexión y los
posibles relays forman parte del coste, incluso cuando los aportan voluntarios.

Para elegir un anfitrión proponemos comparar el tiempo de ida y vuelta desde los
jugadores hasta cada candidato que tenga capacidad. Anotar los resultados por grupo
permitirá ver cuándo una buena media esconde una conexión lenta.
[Ejemplo de selección por latencia de los jugadores](https://docs.aws.amazon.com/gameliftservers/latest/developerguide/queues-design-latency.html)

Si el piloto funciona, el mismo planteamiento podría evaluarse para otros servicios,
como el calendario de actividades de una asociación. Cada servicio necesitaría un
programa aprobado, responsables de su mantenimiento y una prueba de recuperación de
sus propios datos. El piloto se describe en la [hoja de ruta](roadmap.md).

## Primer servicio útil

El primer prototipo probará la carga y recuperación manual de un archivo de ejemplo.
La sincronización de carpetas privadas entre los dispositivos autorizados de su dueño
vendrá después, como detalla la [hoja de ruta](roadmap.md).
Los proveedores comunitarios aprobados conservan fragmentos cifrados y ayudan a
recuperar disponibilidad. Aprobar a un proveedor no lo convierte automáticamente
en lector de la carpeta.

La red inicial utiliza acuerdos bilaterales. Cada proveedor decide a quién presta
servicio, cuánta capacidad física concede, durante cuánto tiempo y con qué límites
de transferencia. No hay saldo global, token, blockchain ni tipo de cambio universal.
La aprobación es una decisión local, no una cuenta en un registro central.

Se espera que algunos nodos comunitarios estén conectados habitualmente. Es posible
editar sin conexión, pero la recuperación remota necesita suficientes fragmentos
y metadatos accesibles. La alfa no anunciará un SLA de durabilidad o disponibilidad.

## Control del participante

Compartir requiere aceptación explícita. Cada participante configura el directorio
de almacenamiento, la cuota, los límites de subida y bajada y el horario antes de
aceptar datos. El diseño incluye pausa manual y pausa por batería. Detectar
videojuegos o videollamadas queda para después.
Los proveedores que se retiren deberían permitir la migración durante la vigencia
restante del acuerdo; los fallos abruptos también forman parte del modelo de amenazas
y fiabilidad.

## Qué significa gratuito

El objetivo es no exigir una tarifa de servicio. El hardware, la electricidad,
la conexión, el mantenimiento y el tráfico de relay siguen teniendo costes
asumidos por participantes o donantes. Ofrecer disco no garantiza acceso a una
GPU escasa. El proyecto no promete almacenamiento, tráfico ni inferencia ilimitados.

## IA comunitaria posterior

Las grabaciones personales son privadas por defecto. La herramienta propuesta las
procesaría en el equipo de su dueño y guardaría allí la transcripción. Si se desconoce
si un audio es público, se aplica la misma regla. Si faltan recursos o falla un intento,
la tarea quedaría en ese equipo para reintentarla. La copia de seguridad de una carpeta
es una operación aparte que envía fragmentos cifrados a los proveedores de disco.

Para el piloto comunitario, quien solicita una tarea elegiría un anfitrión aprobado
y una grabación pública del catálogo revisado. El anfitrión recibiría el identificador
del catálogo, obtendría la grabación publicada y ejecutaría un modelo abierto
completo en un entorno aprobado.
Podría escuchar ese audio y leer el texto resultante. La solicitud solo admite
identificadores del catálogo. No admite archivos locales, notas ni instrucciones
privadas. Elegir un anfitrión de confianza o marcar un archivo para compartirlo
no lo incorpora al catálogo. Hay que revisar la licencia de cada modelo para
determinar cómo se pueden usar y redistribuir sus pesos.

Cada nuevo tipo de trabajo requiere una revisión de diseño y seguridad antes de
incorporarlo. Una vez medido el primer servicio, la comunidad podrá evaluar si
conviene repartir un modelo entre varios proveedores. El entrenamiento de modelos
requiere un plan propio de datos, licencias, evaluación y cómputo.

## Fuera de la primera implementación

Los enlaces públicos, la edición multiusuario, los clientes móviles, los archivos
transparentes solo en la nube, los créditos globales, el ingreso sin aprobación,
el entrenamiento de modelos y la navegación anónima quedan fuera de la primera
implementación. La arquitectura apunta a Windows, Linux y macOS con daemon Go y CLI;
la interfaz gráfica llegará después.
