<!-- docs-pair: contributing; revision: 1 -->

[English](../en/contributing.md) | [Español](contributing.md) | [Índice](README.md)

# Contribuir

HiveScope acepta documentación, análisis de protocolos, revisión de seguridad,
traducción y trabajo futuro de implementación. Se aceptan inglés y español. No
necesitas ECC, un asistente de IA, una suscripción ni una GPU para contribuir.

## Elegir una tarea acotada

Lee la [visión](vision.md), la [arquitectura](architecture.md), la [hoja de ruta](roadmap.md)
y el [código de conducta](code-of-conduct.md). Describe en un Issue el problema,
el resultado propuesto y los criterios de aceptación. Las correcciones pequeñas
pueden ir directamente a un pull request. Los cambios de criptografía, confianza,
protocolos, retención o licencias requieren discusión de diseño antes de implementar.

Buen trabajo inicial incluye un fallo reproducible, revisión del modelo de amenazas,
una explicación más clara, revisión de traducción o descripción del hardware de
pruebas sin información personal. No declares implementada una tarea porque exista
una propuesta o maqueta.

## Flujo local

1. Crea un fork y clónalo.
2. Crea una rama descriptiva desde `main` actualizado.
3. Realiza el cambio completo más pequeño, incluyendo textos en inglés y español.
4. Con Node.js 22 o posterior, ejecuta `npm ci` y `npm run check` en la raíz.
5. Abre un pull request hacia `RubioKo/hivescope:main` con la plantilla.
6. Atiende la revisión. Solo los mantenedores integran el trabajo aceptado.

Estos comandos npm ejecutan herramientas documentales, no un servicio HiveScope.
No añadas árboles vacíos, ejecutables de relleno ni instrucciones de compilación
para software no implementado.

## Calidad documental

- Explica resultados y límites con lenguaje claro; cita fuentes técnicas primarias.
- Actualiza los archivos emparejados y su revisión `docs-pair` si cambia el significado.
- Conserva enlaces relativos, descripciones accesibles de imágenes y atribuciones.
- Utiliza rutas y datos sintéticos. Nunca publiques archivos personales, claves,
  credenciales ni material de recuperación.
- Al cambiar banners, revisa tamaño completo y reducido, animación y movimiento reducido.
- Mantén identificadores y comandos en inglés; la prosa puede estar en inglés o español.
- Para otros idiomas, sigue la [guía de traducción](translations.md).

Utiliza prefijos Conventional Commits como `docs:`, `fix:`, `chore:` o, cuando
exista código, `feat:` y `test:`. Los resúmenes en inglés facilitan búsquedas
internacionales; las discusiones y explicaciones en español siguen siendo bienvenidas.

## Revisión y licencias

Un pull request debe explicar problema, resultado, comprobaciones realizadas y
dudas pendientes. Requiere revisión del mantenedor y comprobaciones documentales
correctas. El código futuro también requerirá pruebas significativas de
comportamiento y fallos apropiadas al componente.

Envía solo material que tengas derecho a aportar. La contribución utiliza la
licencia del archivo afectado; declara fuentes de terceros y conserva sus avisos.
El material original nuevo utiliza AGPL-3.0-only por defecto. Los banners animados
son una excepción MIT documentada explícitamente.

La asistencia de IA es opcional. Eres responsable de precisión, atribución,
confidencialidad y revisión sin importar la herramienta utilizada. Consulta
[desarrollo asistido por IA](ai-assisted-development.md) y [AGENTS.md](../../AGENTS.md).
