<!-- docs-pair: security; revision: 1 -->

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
