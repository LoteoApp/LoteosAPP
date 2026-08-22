# Documentación técnica

Esta carpeta contiene la documentación que debe mantenerse junto con el código.

## Guías

- [Dominio](domain.md): entidades, roles y permisos, reglas de negocio,
  ciclo de vida del lote y estructura requerida del archivo DXF.
- [Diagrama de entidades v3](diagrama-entidades-v3.drawio): modelo relacional
  ([vista HTML](diagrama-entidades-v3.html)).
- [Desarrollo y Docker Compose](development.md): requisitos, arranque, puertos,
  variables de entorno, logs y resolución de problemas.
- [PostgreSQL y migraciones](database.md): conexión desde Go, Goose, creación,
  aplicación, rollback y reglas para modificar el esquema.
- [Arquitectura y estructura](architecture.md): arquitectura modular por
  funcionalidad, límites de dependencia, estructura objetivo del backend y del
  frontend, pruebas y servicios de Compose.
- [Pruebas y cobertura](testing.md): herramientas elegidas, estrategia, umbrales
  mínimos, ubicación de tests y comandos para ejecutar las suites.
- [Integración continua](ci.md): jobs del workflow de GitHub Actions, cuándo
  corre cada uno y su relación con la auditoría de dependencias.
- [Secrets con Doppler](secrets.md): setup del CLI, cómo correr comandos con
  secrets inyectados y cómo administrar valores por config.

## Regla de mantenimiento

Cuando una decisión técnica afecte la forma de levantar, probar o desplegar el
proyecto, actualizar la documentación correspondiente en el mismo cambio.
