<!-- docs-pair: security; revision: 2 -->

[English](../en/security.md) | [Español](security.md) | [Índice](README.md)

# Seguridad y modelo de amenazas

HiveScope no tiene versiones de software con soporte. Este diseño no ha recibido
una auditoría criptográfica. Nunca utilices una versión experimental como única
copia de un archivo.

## Límites de confianza

| Participante | Acceso autorizado | Riesgo restante |
| --- | --- | --- |
| Dispositivo autorizado | Texto legible, claves y ediciones permitidas. | Su compromiso expone carpetas y puede publicar cambios dañinos. |
| Proveedor aprobado | Fragmentos cifrados y metadatos de mantenimiento limitados. | Puede borrar datos, mentir, coludirse u observar tráfico. |
| Reparador delegado | Reconstrucción y reemplazo limitados de contenido cifrado. | Puede desperdiciar recursos permitidos o no reparar. |
| Relay | Metadatos de conexión y tráfico; transporte de contenido cifrado. | Puede observar pares, retrasar o denegar tráfico. |
| Mantenedor | Revisión y administración de publicaciones. | Un canal de distribución comprometido puede publicar software malicioso. |

La aprobación limita el ingreso; no demuestra honestidad. La alfa no resuelve
la resistencia Sybil sin permisos ni impone reciprocidad global.

## Defensas requeridas

- Utilizar bibliotecas criptográficas y de transporte existentes; no inventar primitivas.
- Autenticar pares y autorizar independientemente cada operación y función de carpeta.
- Separar claves de carpeta, identidades de dispositivo y permisos de reparación.
- Cifrar nombres, rutas y catálogos; verificar compromisos y AEAD antes de mostrar
  datos. Un resumen firmado prueba integridad, no disponibilidad.
- Rechazar solicitudes malformadas, repetidas, excesivas o no autorizadas con
  límites de análisis, colas, streams y concurrencia por par.
- Tratar las rutas como no confiables y escribir dentro de las raíces configuradas.
- Proteger secretos locales con desbloqueo explícito y cifrado en reposo; los
  permisos del sistema no protegen frente a un equipo desbloqueado y comprometido.
- Cifrar las exportaciones de recuperación con un formato revisado basado en
  contraseña. Exigir una acción explícita para exportar o revelar secretos.
- Usar épocas, estados firmados y versiones recordadas para limitar retrocesos.
  Recuperar desde una exportación antigua sin un par honesto actualizado puede
  omitir cambios recientes; una firma no demuestra actualidad por sí sola.

## Límites de privacidad

La confidencialidad buscada impide que proveedores sin claves lean los archivos.
No significa ocultar todos los metadatos ni implementar un sistema de pruebas
de conocimiento cero. Los pares correspondientes pueden observar direcciones IP,
tamaños, tiempos, relaciones de ubicación y actividad de reparación.

El cifrado no recupera datos borrados, deshace ediciones de un dueño comprometido,
retira texto descargado ni garantiza el servicio de voluntarios. Fallos simultáneos
de hogares, acuerdos vencidos y pérdida de claves pueden impedir la recuperación.
La inferencia distribuida tiene otras propiedades de privacidad y no debe heredar
las afirmaciones del almacenamiento.

## Decisión propuesta sobre la transcripción

Tratar las grabaciones de una persona como privadas por defecto. Transcribirlas
únicamente en el equipo del propietario. Durante el procesamiento, el audio,
los extractos, los datos derivados, las instrucciones y la transcripción
permanecen allí. Los registros, informes de fallos y datos de telemetría deben
excluir ese contenido, las rutas privadas y las claves. Si falta el modelo,
se agota la memoria o falla la tarea, informar del problema en el propio equipo.
Los reintentos siguen la misma regla. Una copia de seguridad solicitada por
separado puede enviar fragmentos cifrados bajo un acuerdo de almacenamiento.

Para el cómputo voluntario se propone usar material ya publicado en un
catálogo público aprobado. Una solicitud remota elige el identificador de una
obra del catálogo y una operación aprobada. El operador puede leer la entrada
pública y el resultado. Aprobar a un anfitrión o marcar una casilla de envío
no convierte un archivo privado en material del catálogo. El procesamiento
remoto de material privado sigue siendo una cuestión de investigación.
Quien comprometa el equipo del propietario aún podría leer los datos locales.

**Prueba de aceptación futura.** Dar a una grabación privada sintética una frase
hablada y un nombre de archivo distintivos. Con el modelo instalado, transcribirla
sin conexión. Después, ofrecer un equipo remoto, agotar los recursos locales y
reintentar tras recuperar la conexión. Inspeccionar las solicitudes salientes en
el equipo de origen y la cola del proveedor. La primera transcripción debe quedar
en el equipo y ninguno de los intentos debe solicitar trabajo remoto. Buscar la
frase y la ruta privada en los registros, informes de fallos y datos de telemetría.
Repetir con una grabación sin clasificar.

## Informar un problema

Utiliza los [informes privados de vulnerabilidad de GitHub](https://github.com/RubioKo/hivescope/security/advisories/new)
para hallazgos sensibles. Incluye documento o versión futura afectada, reproducción
sintética, comportamiento esperado e impacto. No adjuntes archivos reales,
credenciales, exportaciones de recuperación ni registros personales. Se aceptan
inglés y español.

Utiliza [Issues públicos](https://github.com/RubioKo/hivescope/issues) para preguntas
de diseño no sensibles. No hay plazo de respuesta garantizado ni programa de
recompensas. Los mantenedores evaluarán el impacto, coordinarán correcciones cuando
corresponda y documentarán resultados verificados sin atribuir auditorías inexistentes.
