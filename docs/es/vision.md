<!-- docs-pair: vision; revision: 1 -->

[English](../en/vision.md) | [Español](vision.md) | [Índice](README.md)

# Visión y límites

HiveScope busca permitir almacenamiento privado operado por la comunidad sin
depender de un proveedor central. Los participantes intercambian recursos con
límites explícitos y mantienen el control de sus archivos y claves.

## Primer servicio útil

Sincronizar una carpeta privada entre los dispositivos autorizados de su dueño.
Los proveedores comunitarios aprobados conservan fragmentos cifrados y ayudan a
recuperar disponibilidad. Aprobar a un proveedor no lo convierte automáticamente
en lector de la carpeta.

La red inicial utiliza acuerdos bilaterales. Cada proveedor decide a quién presta
servicio, cuánta capacidad física concede, durante cuánto tiempo y con qué límites
de transferencia. No hay saldo global, token, blockchain ni tipo de cambio universal.
La aprobación es una decisión local, no una cuenta en un registro central.

Se espera que algunos nodos comunitarios estén conectados habitualmente. Es posible
editar sin conexión, pero la recuperación remota necesita suficientes fragmentos
y metadatos accesibles. La alfa no anunciará un SLA de durabilidad o disponibilidad.

## Control del participante

Compartir requiere aceptación explícita. Cada participante configura el directorio
de almacenamiento, la cuota, los límites de subida y bajada y el horario antes de
aceptar datos. El diseño incluye pausa manual y pausa por batería. Detectar
videojuegos o videollamadas queda para después.
Los proveedores que se retiren deberían permitir la migración durante la vigencia
restante del acuerdo; los fallos abruptos también forman parte del modelo de amenazas
y fiabilidad.

## Qué significa gratuito

El objetivo es no exigir una tarifa de servicio. El hardware, la electricidad,
la conexión, el mantenimiento y el tráfico de relay siguen teniendo costes
asumidos por participantes o donantes. Ofrecer disco no garantiza acceso a una
GPU escasa. El proyecto no promete almacenamiento, tráfico ni inferencia ilimitados.

## IA comunitaria posterior

Comenzar con modelos abiertos existentes ejecutados completos por proveedores
capaces, repartiendo solicitudes mediante acuerdos de cómputo independientes.
Evaluar la inferencia repartida por partes del modelo después de medir ese servicio.
Entrenar modelos requiere un plan adicional de datos, licencias, evaluación y cómputo.

El cifrado de almacenamiento no hace privada la inferencia: un proveedor remoto
puede ver las entradas que procesa. Las licencias de los modelos deben evaluarse
individualmente; disponer de pesos abiertos no implica una licencia de código abierto.

## Fuera de la primera implementación

Los enlaces públicos, la edición multiusuario, los clientes móviles, los archivos
transparentes solo en la nube, los créditos globales, el ingreso sin aprobación,
el entrenamiento de modelos y la navegación anónima quedan fuera de la primera
implementación. La arquitectura apunta a Windows, Linux y macOS con daemon Go y CLI;
la interfaz gráfica llegará después.
