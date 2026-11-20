<!-- docs-pair: roadmap; revision: 2 -->

[English](../en/roadmap.md) | [Español](roadmap.md) | [Índice](README.md)

# Hoja de ruta y criterios de aceptación

No hay fechas de publicación fijadas. El avance depende de colaboradores,
hardware y revisión confirmados. Un hito se completa con evidencia revisable.

## Base actual: repositorio y diseño

Entregar documentación en inglés y español, reglas de contribución, guía opcional
de ECC, banners animados y comprobaciones documentales. Esto no entrega un
ejecutable. Las primeras contribuciones útiles incluyen revisar fallos, aclarar
especificaciones, documentar hardware de prueba y mejorar traducciones.

## Hito 1: núcleo seguro

Consultar el [alcance del prototipo, decisiones abiertas y casos de fallo previstos](readiness.md).

Especificar mensajes, límites, derivación de claves y nonces, protección local,
incorporación y formato de recuperación antes de implementarlos. Desarrollar
identidad, transferencia autenticada, acuerdos y carga/recuperación cifrada manual
mediante una CLI Go.

Se acepta cuando el contenido sintético se recupera exactamente; se rechazan pares
no autorizados y corrupción; las transferencias se reanudan; las cuotas incluyen
temporales; y una exportación permite recuperar acceso con una identidad nueva.

## Hito 2: almacenamiento resistente

Añadir codificación 2 + 1, catálogos replicados, recibos, vencimiento de acuerdos,
comprobaciones de bytes y reparación delegada. Especificar tamaños de muestras,
costes y defensas contra repeticiones mediante experimentos antes de cerrar el protocolo.

Se acepta cuando perder uno de tres proveedores permite recuperar; los fragmentos
faltantes o corruptos activan reparación limitada hacia un reemplazo aprobado;
menos de dos fragmentos válidos producen estado no disponible; y tareas duplicadas,
vencimientos y reinicios no provocan borrado inseguro ni exceden cuotas. La
reparación debe funcionar sin entregar claves de descifrado al reparador.

## Hito 3: sincronización de carpetas

Añadir captura estable, versiones causales firmadas, notificaciones y escaneos,
aplicación atómica, copias de conflicto, marcadores de borrado e historial de 30 días.

Se acepta cuando las ediciones concurrentes desconectadas conservan ambas versiones;
los conflictos de borrado/edición retienen el contenido; volver a conectar no
resucita borrados confirmados; se conservan cambios locales durante descargas;
y los nombres incompatibles se informan sin pérdida silenciosa. También probar
carreras en directorios y puntos de montaje ausentes.

## Hito 4: alfa cerrada multiplataforma

Entregar binarios CLI reales y operación documentada para Windows, Linux y macOS.
Utilizar dos equipos autorizados y tres proveedores de hogares distintos; añadir
un proveedor de reserva para probar reemplazos. Incluir equipos estables e
intermitentes, NAT restrictivo y transferencias por internet real.

Se acepta tras probar conexión directa, relay y fallo total de conexión; superar
reinicios y falta de disco; observar límites, horarios y pausa por batería; y
recuperar sin el equipo original ni su base de datos. Publicar mediciones y
problemas pendientes, no un SLA.

## Trabajo posterior

La interfaz gráfica seguirá a un daemon utilizable. El ingreso público y los
créditos globales necesitan diseños adicionales de abuso y contabilidad. La IA
comunitaria comenzará con modelos existentes completos en proveedores capaces y
acuerdos de cómputo independientes; después se evaluará inferencia por partes.
Entrenar un modelo es otra iniciativa. Ninguna de estas capacidades forma parte
de la entrega documental actual.
