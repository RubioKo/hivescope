<!-- docs-pair: storage; revision: 2 -->

[English](../en/storage.md) | [Español](storage.md) | [Índice](README.md)

# Almacenamiento cifrado y mantenimiento

En el hito 1 un solo proveedor conservará un objeto cifrado completo. La
recuperación en un equipo de reemplazo deberá reproducir los bytes originales.
Esa copia verificada depende de que dicho proveedor siga disponible. La política
de redundancia y reparación descrita a continuación se aplica a partir del hito 3.
Sus parámetros necesitan validación.

## Protección de archivos

- Dividir una versión estable del archivo en bloques de hasta 4 MiB.
- En la implementación candidata con Go, usar AES-256-GCM de la biblioteca estándar. Utilizar
  una clave nueva por versión inmutable y nonces únicos para todos sus bloques.
  Vincular versión y posición como datos asociados autenticados.
- Aplicar Reed–Solomon al contenido cifrado, incluyendo lo necesario para
  descifrarlo y autenticarlo, mediante una biblioteca existente como `klauspost/reedsolomon`.
- El primer perfil de redundancia tiene dos fragmentos de datos y uno de paridad por bloque.
  Cualquier par válido reconstruye el bloque cifrado. Un fragmento corrupto
  cuenta como no disponible; la paridad por sí sola no autentica el contenido.
- Colocar los tres fragmentos en proveedores distintos seleccionados según la
  política de recursos del nodo. En ensayos controlados, confirmar que operadores
  y hogares son distintos. En una red abierta, un operador puede tener varias
  identidades. Indicar qué evidencia sustenta la independencia de las ubicaciones.

El consumo es aproximadamente 1,5 veces el contenido cifrado, más relleno,
metadatos y versiones conservadas. Las copias originales del propietario son
adicionales. Un perfil posterior 10 + 4 consumiría aproximadamente 1,4 veces el
contenido cifrado y necesitaría catorce ubicaciones independientes para tolerar
la pérdida de cuatro proveedores.

## Catálogos, recibos y estado

Los proveedores reciben IDs opacos y compromisos de integridad, no rutas legibles
ni claves de descifrado. Los catálogos firmados contienen información de
codificación y referencias cifradas para recuperar datos. Sus versiones y los
registros de autorización se replican en los tres proveedores de metadatos
seleccionados. Los lectores conservan el estado verificado más reciente para
detectar la repetición de estados antiguos.

Bajo el perfil de redundancia, una versión está `pending` hasta confirmar ubicaciones y metadatos. Está
`protected` con tres ubicaciones válidas y confirmaciones de metadatos, `degraded`
si quedan dos disponibles y `unavailable` si se pueden leer menos de dos.
El requisito se aplica por bloque: el archivo completo necesita todos los bloques
y su catálogo. No disponible no significa necesariamente pérdida permanente;
los proveedores pueden volver a conectarse.

Un recibo demuestra que el proveedor aceptó una obligación, no disponibilidad
perpetua. La falta de capacidad deja los cambios pendientes y visibles.

## Acuerdos y capacidad física

Cualquier persona puede solicitar espacio sin aportar hardware ni ganar créditos.
Los nodos compatibles aceptan automáticamente un acuerdo según el perfil local
del anfitrión. Consta de participantes, cuota física en bytes, operaciones
permitidas, vencimiento y límites de transferencia. El anuncio con vencimiento
en un directorio permite encontrar al anfitrión. Este comprueba espacio real y
compromisos existentes antes de hacer una reserva.

La vigencia inicial es de 30 días. Se intenta renovar diariamente cuando haya
conexión. Tras el vencimiento hay siete días de gracia; un cliente desconectado
no renueva automáticamente. Las escrituras nuevas requieren acuerdo vigente.
Los datos existentes siguen siendo legibles durante la gracia; después el proveedor
puede eliminarlos. Perder todos los equipos propios durante más tiempo puede
causar pérdida de datos incluso conservando una exportación de recuperación.

La cuota incluye datos, versiones retenidas, metadatos, reservas entrantes y
temporales. Contabilizar una sola vez cada solicitud aunque se reintente.
Reservar de forma atómica para que dos solicitudes no consuman los mismos bytes.
Mantener un margen configurable de espacio libre en el volumen ofrecido. Pausar
detiene las reservas nuevas. Reducir una cuota detiene las escrituras nuevas y
solicita migración. Los datos existentes conservan su acuerdo de retención
incluso si el anfitrión deja de anunciarse.

## Comprobaciones y reparación

- Comprobar conectividad cada cinco minutos. Tres fallos consecutivos marcan al
  proveedor como sospechoso y activan la evaluación de reparación si hay menos redundancia.
- Reparar sin esperar un plazo fijo de 48 horas cuando solo quedan dos fragmentos
  válidos. El destino debe cumplir la política local de asignación y confirmar un
  acuerdo con cuota disponible antes de recibir datos.
- El propietario puede delegar lectura, reconstrucción y escritura limitadas de
  fragmentos cifrados en un reparador con delegación expresa, sin entregar claves de lectura.
  Esto no permite editar carpetas, firmar acuerdos ni borrar sin límites.
- El reparador reconstruye contenido cifrado, verifica resúmenes firmados y
  obtiene recibos nuevos. Los dispositivos autorizados reconcilian sus anuncios.
- Utilizar IDs de tarea estables, reservas, espera progresiva y cancelación para
  que reparaciones duplicadas no consuman espacio ilimitado. Conservar copias
  válidas hasta confirmar el reemplazo y permitir su retirada por el acuerdo.
- Con menos de dos fragmentos, esperar e informar que no está disponible; no
  simular una reparación. Sin destino de reemplazo, informar degradación y reintentar.

Las comprobaciones de almacenamiento solicitan muestras impredecibles de bytes
reales con pruebas Merkle frente a compromisos firmados. No basta responder con
hashes de hojas almacenados. La frecuencia y el tamaño de las muestras necesitan
medición antes de cerrar el protocolo en el hito de reparación. Son controles
probabilísticos: no prueban almacenamiento físico independiente ni evitan
colusión o subcontratación.

## Fuentes para evaluar durante la implementación

- [Cifrado autenticado en Go](https://pkg.go.dev/crypto/cipher)
- [Biblioteca Reed–Solomon](https://github.com/klauspost/reedsolomon)
- [Arquitectura y reparación de Tahoe-LAFS](https://tahoe-lafs.readthedocs.io/en/stable/architecture.html)
