# RepuestosDirect — Plataforma de Pedidos

Plataforma de pedidos de repuestos para talleres mecánicos, construida con **Go + HTMX + PostgreSQL**. Catálogo por vehículo, carrito, crédito comercial, panel de administración, entregas y notificaciones WhatsApp.

## Requisitos

- Go 1.22+
- PostgreSQL (Render Postgres en producción)

## Variables de entorno

| Variable | Requerida | Descripción |
|---|---|---|
| `DATABASE_URL` | Sí | Connection string de PostgreSQL |
| `ADMIN_PASSWORD` | Sí | Basic Auth para rutas `/admin/*` |
| `DRIVER_PASSWORD` | No | Basic Auth para `/driver` (usa admin si no está) |
| `WHATSAPP_TOKEN` | No | Token Meta WhatsApp Cloud API |
| `WHATSAPP_PHONE_ID` | No | Phone ID de WhatsApp Business |
| `DRIVER_PHONE` | No | WhatsApp del repartidor para alertas |
| `CRON_SECRET` | No | Secreto para `GET /cron/reminders?secret=...` (recordatorios de pago) |
| `PORT` | No | Puerto HTTP (default 8080) |

## Desarrollo local

```bash
cd repuestosdirect
export DATABASE_URL="postgres://user:pass@localhost:5432/repuestosdirect?sslmode=disable"
export ADMIN_PASSWORD="admin123"
go run .
```

Abre http://localhost:8080

## Tests

```bash
# Unit tests (sin base de datos)
go test ./... -v

# Integration tests (requiere PostgreSQL)
export TEST_DATABASE_URL="postgres://..."
go test -tags=integration ./... -v
```

## Despliegue en Render

El archivo `render.yaml` en la raíz configura el servicio web y la base de datos Postgres.

1. Conecta el repo en Render
2. Configura `ADMIN_PASSWORD` (y opcionalmente `DRIVER_PASSWORD`, WhatsApp)
3. Render inyecta `DATABASE_URL` automáticamente

Health check: `GET /healthz`

### Recordatorios de pago (cron)

Configura un cron job (Render Cron o similar) que llame diariamente:

```
GET https://tu-app.onrender.com/cron/reminders?secret=TU_CRON_SECRET
```

Set `CRON_SECRET` en las variables de entorno de Render.

### Gestión de crédito por taller

En **Admin → Talleres** (`/admin/shops`) puedes por cada cliente:

- **Límite de crédito** — aumentar o reducir (no puede bajar del saldo usado)
- **Días total** — plazo completo (default 30)
- **# Pagos** — cuotas (default 2 → pago día 15 y día 30)
- **Recordar** — días antes del vencimiento para enviar WhatsApp (default 3)

Los pedidos a crédito generan cuotas automáticamente. Marca cada cuota como pagada desde el panel admin o la vista del pedido.

## Funcionalidades

- **Catálogo en cascada** Marca → Modelo → Año → Piezas (HTMX)
- **Búsqueda** por nombre, categoría, marca o ID
- **Carrito** con sesiones persistentes en PostgreSQL
- **Crédito comercial** con validación en servidor y bloqueo por límite
- **Checkout invitado** (contado, sin cuenta)
- **Cuentas de taller** con registro, aprobación admin, **ajuste de crédito/términos**, y eliminación de clientes inactivos
- **Crédito en cuotas** — default 30 días en 2 pagos (50% día 15, 50% día 30), configurable por taller
- **Recordatorios WhatsApp** — cron diario avisa cuotas próximas a vencer o vencidas
- **Inventario** con stock, reorden y decremento automático al comprar
- **Panel admin** con cuentas por cobrar, marcar pagado, reportes
- **Entregas** asignación de mensajeros y vista repartidor
- **WhatsApp** confirmaciones y actualizaciones de estado (mock en consola sin API keys)
- **Seguridad** bcrypt, CSRF, cookies Secure/SameSite en producción

## Estructura

```
repuestosdirect/
  main.go          — rutas HTTP y handlers
  data.go          — PostgreSQL store
  session.go       — sesiones en DB
  auth.go          — bcrypt, CSRF, roles admin/driver
  validate.go      — validación de inputs
  notify.go        — WhatsApp Cloud API
  logger.go        — logging JSON en producción
  templates/       — HTML + HTMX
  static/          — HTMX self-hosted + CSS
```

## Notas

- `seedInitialData()` en `data.go` es solo para pruebas locales; no se ejecuta en producción.
- Las contraseñas legacy en texto plano se migran automáticamente a bcrypt al iniciar sesión.
- Eliminar un taller requiere saldo de crédito en cero y sin pedidos impagos.
