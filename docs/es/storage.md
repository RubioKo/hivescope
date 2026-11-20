<!-- docs-pair: storage; revision: 1 -->

[English](../en/storage.md) | [Español](storage.md) | [Índice](README.md)

# Almacenamiento cifrado y mantenimiento

Esta es la política propuesta para la alfa. Los valores son parámetros de diseño
por validar, no mediciones ni garantías de una red desplegada.

## Protección de archivos

- Dividir una versión estable del archivo en bloques de hasta 4 MiB.
- Cifrar cada bloque con AES-256-GCM de la biblioteca estándar de Go. Utilizar
  una clave nueva por versión inmutable y nonces únicos para todos sus bloques.
  Vincular versión y posición como datos asociados autenticados.
- Aplicar Reed–Solomon al contenido cifrado, incluyendo lo necesario para
  descifrarlo y autenticarlo, mediante una biblioteca existente como `klauspost/reedsolomon`.
- El perfil inicial tiene dos fragmentos de datos y uno de paridad por bloque.
  Cualquier par válido reconstruye el bloque cifrado. Un fragmento corrupto
  cuenta como no disponible; la paridad por sí sola no autentica el contenido.
- Guardar los tres fragmentos en tres proveedores aprobados distintos. Separar
  operadores y hogares mediante información explícita; tres identidades no
  demuestran independencia física.

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

Una versión está `pending` hasta confirmar ubicaciones y metadatos. Está
`protected` con tres ubicaciones válidas y confirmaciones de metadatos, `degraded`
si quedan dos disponibles y `unavailable` si se pueden leer menos de dos.
El requisito se aplica por bloque: el archivo completo necesita todos los bloques
y su catálogo. No disponible no significa necesariamente pérdida permanente;
los proveedores pueden volver a conectarse.

Un recibo demuestra que el proveedor aceptó una obligación, no disponibilidad
perpetua. La falta de capacidad deja los cambios pendientes y visibles.

## Acuerdos y capacidad física

Cada acuerdo aceptado por ambas partes define participantes, cuota física en
bytes, operaciones, vencimiento y límites de subida y bajada. No se generan
créditos automáticamente por anunciar capacidad libre. La reciprocidad se negocia;
no se impone globalmente.

La vigencia inicial es de 30 días. Se intenta renovar diariamente cuando haya
conexión. Tras el vencimiento hay siete días de gracia; un cliente desconectado
no renueva automáticamente. Las escrituras nuevas requieren acuerdo vigente.
Los datos existentes siguen siendo legibles durante la gracia; después el proveedor
puede eliminarlos. Perder todos los equipos propios durante más tiempo puede
causar pérdida de datos incluso conservando una exportación de recuperación.

La cuota incluye datos, versiones retenidas, metadatos, reservas entrantes y
temporales. Repetir una operación no debe contabilizarla dos veces. Conservar una
reserva configurable de disco libre. Reducir una cuota detiene nuevas escrituras
y solicita migración; no borra inmediatamente obligaciones vigentes.

## Comprobaciones y reparación

- Comprobar conectividad cada cinco minutos. Tres fallos consecutivos marcan al
  proveedor como sospechoso y activan la evaluación de reparación si hay menos redundancia.
- Reparar sin esperar un plazo fijo de 48 horas cuando solo quedan dos fragmentos
  válidos. El destino debe estar aprobado y disponer de acuerdo y cuota libre.
- El propietario puede delegar lectura, reconstrucción y escritura limitadas de
  fragmentos cifrados en un reparador aprobado, sin entregar claves de lectura.
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
