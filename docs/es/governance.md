<!-- docs-pair: governance; revision: 2 -->

[English](../en/governance.md) | [Español](governance.md) | [Índice](README.md)

# Gobernanza y mantenimiento

El mantenedor inicial es [RubioKo](https://github.com/RubioKo). No se presupone
comité, fundación, patrocinio empresarial ni otros mantenedores.

## Decisiones

Utiliza Issues públicos para propuestas no sensibles y pull requests para cambios
revisables. Describe problema, alternativas, comportamiento elegido, riesgos y
criterios de aceptación. Las decisiones importantes deben actualizar el diseño
en inglés y español. Los hallazgos sensibles siguen la [política de seguridad](security.md).

Buscar acuerdo técnico mediante evidencia y revisión. Mientras exista un único
mantenedor, este tiene la decisión final sobre integración y alcance. Se puede
documentar un desacuerdo razonado sin ataques personales. Incorporar futuros
mantenedores requiere un cambio público revisado de gobernanza y su consentimiento.

## Ramas y revisión

`main` es la rama de referencia. El trabajo se realiza en ramas breves o forks y
llega mediante pull requests. El mantenimiento revisa alcance, precisión, paridad
de idiomas, licencias y comprobaciones. Una comprobación documental correcta no
es una auditoría de seguridad. Los cambios administrativos directos se reservan
al mantenimiento del repositorio y se describen si afectan a los colaboradores.

Antes de integrar un PR en `main`, hay que resolver las conversaciones de revisión
y superar la comprobación `Markdown, links and language parity` de GitHub Actions.
La regla activa bloquea el borrado de la rama y los envíos forzados, sin
excepciones. Mientras haya un solo mantenedor con escritura, el mínimo de
aprobaciones es cero. Al conceder escritura a un segundo mantenedor, se debe
exigir una aprobación independiente como parte de ese cambio.

Dependabot revisa las dependencias de npm y las acciones de GitHub cada semana
y puede abrir PR de seguridad entre revisiones programadas. El repositorio
exige fijar las acciones al SHA completo de un commit y permite dos acciones
externas, `actions/checkout` y `actions/setup-node`. Para añadir otra, explica
en un PR la necesidad y los permisos que requiere para que el mantenedor pueda
revisar el cambio en la lista de acciones permitidas.

## Publicaciones futuras

Todavía no hay versiones de software con soporte. Una publicación debe identificar
alcance, licencia, plataformas, comprobaciones y límites conocidos. Antes de
publicar ejecutables se documentarán verificación y confianza en actualizaciones.
No crear etiquetas que sugieran un servicio de almacenamiento utilizable cuando
solo se publique documentación.

Administrar el repositorio es independiente de operar P2P: ser mantenedor no
concede autoridad sobre archivos o acuerdos de otros participantes.
