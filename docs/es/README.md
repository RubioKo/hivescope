<!-- docs-pair: index; revision: 8 -->

[English](../en/README.md) | [Español](README.md) | [Inicio del proyecto](../../README.es.md)

# Documentación de HiveScope

HiveScope está en diseño. La [hoja de ruta](roadmap.md) presenta la primera
prueba de recuperación cifrada y los hitos de asignación automática de recursos,
cómputo privado y alojamiento. [Ofrecer mi equipo](resource-offers.md) sigue
una solicitud desde la persona que necesita recursos hasta la máquina que los presta.

## Diseño

- [Visión y límites](vision.md)
- [Ofrecer mi equipo](resource-offers.md)
- [Arquitectura e interfaces](architecture.md)
- [Diseño y criterios de aceptación del primer prototipo](readiness.md)
- [Identidad, permisos y custodia de claves](identity-custody.md)
- [Paquete de recuperación y lecturas firmadas](recovery-format.md)
- [Cómo se identifica y verifica el archivo al recuperarlo](file-catalog.md)
- [Cuándo puede el proveedor confirmar una subida](upload-confirmation.md)
- [Protocolo y límites de transferencia en la LAN](lan-protocol.md)
- [Persistencia del proveedor y casos de reinicio](provider-persistence.md)
- [Almacenamiento cifrado y mantenimiento](storage.md)
- [Sincronización de carpetas](synchronization.md)
- [Seguridad y modelo de amenazas](security.md)
- [Hoja de ruta y criterios de aceptación](roadmap.md)
- [Glosario](glossary.md)

## Participación

- [Contribuir](contributing.md)
- [Gobernanza y mantenimiento](governance.md)
- [Código de conducta](code-of-conduct.md)
- [Traducciones](translations.md)
- [Desarrollo asistido por IA y ECC opcional](ai-assisted-development.md)
- [Licencias](licensing.md)

Para comprobar la documentación localmente, utiliza Node.js 22 o posterior,
ejecuta `npm ci` en la raíz del repositorio y después `npm run check`. Estos
comandos comprueban únicamente la documentación.
