<!-- docs-pair: ai-development; revision: 1 -->

[English](../en/ai-assisted-development.md) | [Español](ai-assisted-development.md) | [Índice](README.md)

# Desarrollo asistido por IA y ECC opcional

Puedes contribuir con un editor de texto y las comprobaciones documentadas.
Las herramientas de IA son opcionales. Cada colaborador responde de la precisión,
licencias y confidencialidad de todo lo que envía.

## Qué aporta ECC

[ECC](https://github.com/affaan-m/ECC) es un proyecto externo con flujos, reglas y
habilidades para asistentes de programación. Puede ayudar a organizar investigación,
planificación, implementación, revisión y verificación. No es la red P2P, el motor
de almacenamiento, el servicio de cómputo ni la auditoría de seguridad de HiveScope.

Este repositorio incluye instrucciones propias en [AGENTS.md](../../AGENTS.md),
no una instalación de ECC. No incluye ni exige código, hooks, habilidades,
memorias ni configuraciones MCP de ECC. No afirmamos mejoras de productividad medidas.

## Configuración opcional

1. Lee el [README oficial de ECC](https://github.com/affaan-m/ECC#readme) y su
   información actual de soporte para tu asistente. Las capacidades varían por herramienta.
2. Elige un único método documentado, revisa los permisos y archivos que modifica
   y sigue las instrucciones oficiales.
3. Mantén credenciales, hooks, memoria privada y ajustes personales fuera del
   repositorio. No publiques configuraciones globales de asistentes.
4. Comienza con ayuda de planificación, documentación o revisión. Evalúa sus
   guías de Go cuando comience el hito de implementación.
5. Ejecuta las comprobaciones de HiveScope independientemente de lo que afirme el asistente.

El proyecto externo puede cambiar comandos y funciones compatibles. Esta guía
enlaza sus instrucciones mantenidas en lugar de fijar un instalador no probado.
Instalar ECC no autoriza publicar en el repositorio ni cambia las reglas de
contribución y revisión de HiveScope.

## Aplicar las ideas a HiveScope

Investiga primero el diseño. Propón cambios pequeños con criterios claros. Revisa
desde otra perspectiva y verifica resultados reales. En documentación esto
significa paridad de idiomas, enlaces y estado veraz. En el código futuro también
incluye pruebas relevantes de comportamiento y fallos.

Adapta las recomendaciones al componente: las listas de seguridad web no cubren
confianza P2P, abuso de cuotas, recuperación de secretos, corrupción o desconexiones.
Los patrones generales de Go son sugerencias por evaluar, no mandatos de arquitectura.
No introduzcas delegación automática ni una suscripción obligatoria a modelos.

## Datos y atribución

No envíes a un asistente archivos personales, claves, credenciales ni exportaciones
de recuperación. Utiliza ejemplos sintéticos y revisa los cambios generados.
Cita fuentes e identifica ayuda sustancial cuando permita comprender cómo se
verificó el trabajo. No presentes una salida generada como auditada independientemente.
ECC mantiene su propia [licencia MIT](https://github.com/affaan-m/ECC/blob/main/LICENSE).
