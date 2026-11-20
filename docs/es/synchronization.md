<!-- docs-pair: synchronization; revision: 1 -->

[English](../en/synchronization.md) | [Español](synchronization.md) | [Índice](README.md)

# Sincronización de carpetas

El primer servicio de sincronización apunta a los dispositivos autorizados de
una misma persona. Cada equipo mantiene localmente la carpeta seleccionada. No
es un sistema de archivos virtual ni un editor colaborativo de documentos.

## Detectar y publicar cambios

Utilizar notificaciones del sistema de archivos y escaneos periódicos; las
notificaciones pueden perderse. Esperar a que el archivo se estabilice antes de
leerlo. Si cambia su tamaño o estado durante la captura, reintentar para no
publicar una instantánea mezclada.

Cada versión inmutable registra autor autorizado, vector de versión causal,
metadatos cifrados y referencias al contenido protegido. Un archivo modificado
genera una versión nueva; la primera implementación no exige deduplicación entre
versiones. Las versiones conservadas consumen cuota física.

Los equipos pueden editar sin conexión. Sus cambios quedan pendientes hasta
disponer de pares, capacidad y autorizaciones vigentes. Guardar localmente no
significa que la red ya haya protegido la nueva versión.

## Conflictos

- Una versión causalmente posterior sustituye a su antecesora.
- Las ediciones concurrentes conservan ambas versiones completas. Elegir la
  principal mediante un orden determinista de identificadores de versión firmados;
  conservar la otra con un sufijo de conflicto basado en su identificador, no en la hora.
- Borrado y edición concurrentes conservan el contenido editado como conflicto recuperable.
- Borrar un directorio no elimina silenciosamente ediciones concurrentes de sus
  hijos; conservarlas en una ubicación de conflicto recuperable.
- Renombrar se representa inicialmente como creación y marcador de borrado.
  Optimizarlo queda para después.
- Reconciliar anuncios repetidos de forma idempotente para no multiplicar conflictos.

El usuario resuelve los conflictos editando o eliminando las copias conservadas.
No hay fusión automática de texto. Los relojes no deciden qué edición concurrente
quería conservar el usuario.

## Aplicar y recuperar

Descargar a un temporal, verificar resúmenes y autenticación AEAD, persistir los
datos y reemplazar atómicamente el destino. Comprobar de nuevo la versión local
antes del reemplazo; una edición nueva debe convertirse en conflicto en lugar
de sobrescribirse. Guardar tareas y metadatos transaccionalmente para reanudar tras reinicios.

Conservar versiones anteriores y contenido eliminado durante 30 días por defecto,
sujeto a acuerdos y cuotas. Los marcadores de borrado permanecen en los metadatos
hasta que todos los dispositivos autorizados los confirmen. Un equipo desconectado
mucho tiempo no se olvida silenciosamente: el dueño debe revocarlo o reincorporarlo
explícitamente después de reconciliarlo. No interpretar un punto de montaje vacío
o inaccesible como orden de borrar una carpeta.

Restaurar una versión antigua la publica como edición nueva. El proyecto no
promete borrado seguro en proveedores ni retirar datos ya descargados. La
sincronización y el historial limitado no sustituyen una copia independiente.

## Reglas entre plataformas

Sincronizar archivos regulares y directorios. Excluir inicialmente enlaces
simbólicos, archivos especiales, dispositivos y replicación de permisos o propietarios
específicos. Nunca seguir enlaces fuera de la raíz al escanear o aplicar datos.

Rechazar rutas absolutas, escapes, nombres reservados y rutas no compatibles.
Informar colisiones por mayúsculas, normalización Unicode y longitud sin
sobrescribir silenciosamente. Los ejemplos portables deben funcionar en Windows,
Linux y macOS. Las reglas exactas de nombres pertenecen a la especificación del
protocolo de sincronización previa a su implementación.

Referencia: [comportamiento de sincronización de Syncthing](https://docs.syncthing.net/users/syncing).
