<!-- docs-pair: governance; revision: 4 -->

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

`main` es la rama estable y predeterminada. Crea una rama breve a partir de
`development` y abre un pull request hacia `development` para integrar tu trabajo.
Los mantenedores revisan alcance, precisión, paridad de idiomas, licencias y
las comprobaciones pertinentes. Cuando el hito cumpla los
[criterios de la hoja de ruta](roadmap.md), abre un PR de `development` a `main`
con los resultados y usa **Create a merge commit** para conservar el historial común.

Ambas ramas permanentes exigen un PR, conversaciones de revisión resueltas y
la comprobación `Markdown, links and language parity` de GitHub Actions aprobada.
Ambas bloquean el borrado y los envíos forzados, sin excepciones. GitHub elimina
las ramas de trabajo de este repositorio tras integrarlas. Mientras haya un solo
mantenedor con escritura, el mínimo de aprobaciones es cero. Al conceder
escritura a un segundo mantenedor, se debe
exigir una aprobación independiente como parte de ese cambio.

Las actualizaciones de Dependabot y las correcciones urgentes pueden llegar a
`main` mediante un PR. Después de integrarlo, abre un PR de `main` a `development`
y usa un commit de fusión para incorporar la corrección al trabajo en curso.
Los mantenedores documentan los cambios administrativos que afecten a los
colaboradores. La comprobación documental revisa Markdown, enlaces locales
y pares de idiomas.

La configuración semanal de Dependabot cubre los módulos Go, npm y las acciones de GitHub.
Dependabot puede abrir PR de seguridad entre revisiones programadas. El repositorio
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
