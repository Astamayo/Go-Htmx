# RepuestosDirect

Plataforma de pedidos de repuestos para talleres mecánicos en Puerto Plata (República Dominicana). Construida con **Go + HTMX + PostgreSQL**.

Tres roles separados — **Admin**, **Taller (shop)** y **Repartidor (driver)** — cada uno con su propio login y panel. Los clientes públicos pueden buscar piezas y comprar sin cuenta.

---

## Inicio rápido (producción / Render)

1. Despliega desde este repo (`render.yaml` configura web + Postgres).
2. En Render → **Environment**, configura al menos:
   - `ADMIN_PASSWORD` — contraseña del administrador
3. Render inyecta `DATABASE_URL` automáticamente.
4. Abre la URL de tu app.

**Primer acceso admin**

| Campo | Valor |
|-------|-------|
| URL | `https://tu-app.onrender.com/admin/login` |
| Usuario | `admin` |
| Contraseña | valor de `ADMIN_PASSWORD` (si no existe admin en DB, se crea con esa contraseña; default dev: `admin123`) |

> El enlace **Admin** no aparece en la página pública. Guarda `/admin/login` como favorito.

Health check: `GET /healthz`

---

## Roles y accesos

| Rol | URL de login | Panel principal | Qué puede hacer |
|-----|--------------|-----------------|-----------------|
| **Público / invitado** | — | `/catalogo`, `/buscar` | Buscar piezas, carrito, checkout de contado sin cuenta |
| **Taller (empleado)** | `/login` | `/tienda/pedidos` | Ver pedidos, entregas, comprar desde catálogo con crédito |
| **Administrador** | `/admin/login` | `/admin` | Todo: talleres, inventario, pedidos, repartidores, reportes |
| **Repartidor** | `/driver/login` | `/driver` | Ver entregas asignadas, marcar En camino / Entregado |

Cada rol usa **usuario + contraseña** propios (sesión en cookie, no Basic Auth).

---

## Guía por rol

### Cliente público (sin cuenta)

1. **Catálogo** — `/catalogo` → Marca → Modelo → Año → piezas
2. **Búsqueda** — `/buscar` por nombre, número de parte, OEM, marca o ID
3. **Carrito** — `/carrito`
   - Cambiar cantidad o eliminar líneas (✕)
   - Al agregar una pieza ya en el carrito, pide confirmación
   - **Checkout invitado**: contado, sin registro
4. **Pagos aceptados** (info en búsqueda pública): tarjeta crédito/débito. Crédito comercial solo para talleres registrados.

### Taller — empleado del taller

**Login:** `/login` con el **usuario** que asignó el admin (no el nombre comercial del taller).

| Dónde se define el usuario | |
|-----------------------------|---|
| Admin crea taller | `/admin/shops/create` → campo **Usuario de acceso** |
| Auto-registro | `/signup` → campo **Usuario de acceso** |
| Talleres antiguos | Puede ser el ID del taller, ej. `S-0001` |

**Menú del taller (logueado):**

| Enlace | Ruta | Descripción |
|--------|------|-------------|
| Pedidos | `/tienda/pedidos` | Pedidos activos del taller |
| Historial | `/tienda/pedidos/historial` | Pedidos entregados o cerrados |
| Entregas | `/tienda/entregas` | Pedidos en camino o listos para recoger |
| Catálogo | `/catalogo` | Comprar más repuestos |
| Carrito | icono en header | Ver carrito |

**Crédito comercial**

- Disponible al confirmar pedido si el taller está aprobado
- **Pedido mínimo a crédito:** $50 (configurable con `MIN_CREDIT_ORDER`)
- Default: límite $300 · 30 días · 2 cuotas (50% día 15, 50% día 30)
- Admin ajusta límite y términos en `/admin/shops`

**Registro nuevo taller:** `/signup` → espera aprobación del admin en `/admin/shops`.

### Administrador

**Login:** `/admin/login` — usuario `admin` + `ADMIN_PASSWORD`.

**Menú admin:**

| Sección | Ruta | Uso |
|---------|------|-----|
| Panel | `/admin` | Resumen, cuotas por cobrar |
| Pedidos | `/admin/orders` | Activos + historial; cambiar estado |
| Entregas | `/admin/delivery` | Asignar repartidor a pedidos listos |
| Talleres | `/admin/shops` | Aprobar, crédito, editar, eliminar |
| Crear taller | `/admin/shops/create` | Nombre, usuario, contraseña |
| Repartidores | `/admin/drivers` | Crear/editar conductores |
| Inventario | `/admin/inventory` | Stock, P/N, OEM, precio B2B |
| Reportes | `/admin/reports` | Gráficos, ordenar, auditoría |

**Estados de pedido** (admin y taller ven historial):

`Pedido` → `Confirmado` → `Enviado` → `En aduana` → `Listo para recoger` → `En camino` → `Entregado`

También: `No se pudo entregar` (cierra el pedido).

Los pedidos **Entregado** salen de la lista activa y van al historial.

**Gestión de crédito** (`/admin/shops`, por taller):

- Límite (USD), días total, # pagos, días de recordatorio
- Marcar cuotas pagadas desde el panel o detalle del pedido

**Eliminar taller:** requiere saldo cero y sin pedidos a crédito impagos.

### Repartidor

**Login:** `/driver/login` — credenciales creadas en `/admin/drivers`.

**Pantalla única:** `/driver` (Mis entregas)

Por cada pedido:

| Botón | Efecto |
|-------|--------|
| **Mapa** | Abre dirección en mapas |
| **En camino** | Marca en ruta; el pedido **permanece** en la lista |
| **Entregado** | Cierra pedido; WhatsApp al taller; desaparece de la lista |
| **No entregado** | Cierra como fallido; desaparece de la lista |

**Asignación de pedidos**

1. Admin marca pedido como **Listo para recoger**
2. Admin asigna repartidor en `/admin/delivery`
3. Si el repartidor toca **En camino** sin asignación previa, el pedido se asigna automáticamente a ese conductor

---

## Variables de entorno

| Variable | Requerida | Descripción |
|----------|-----------|-------------|
| `DATABASE_URL` | Sí | PostgreSQL connection string |
| `ADMIN_PASSWORD` | Sí* | Contraseña inicial del usuario `admin` |
| `MIN_CREDIT_ORDER` | No | Mínimo USD para compra a crédito (default `50`) |
| `AZUL_MERCHANT_ID` | Sí (prod) | ID comercio Azul |
| `AZUL_AUTH1` / `AZUL_AUTH2` | Sí (prod) | Credenciales Auth de Azul |
| `AZUL_ENVIRONMENT` | No | `sandbox`, `production`, o `mock` (dev) |
| `AZUL_ECOMMERCE_URL` | No | URL de tu sitio para Azul |
| `PAYMENT_GATEWAY` | No | `azul` (default) o `cardnet` |
| `WHATSAPP_TOKEN` | Sí (prod) | Meta WhatsApp Cloud API token permanente |
| `WHATSAPP_PHONE_ID` | Sí (prod) | WhatsApp Business phone ID |
| `WHATSAPP_VERIFY_TOKEN` | No | Token verificación webhook Meta |
| `WHATSAPP_APP_SECRET` | No | Firma webhook `X-Hub-Signature-256` |
| `WHATSAPP_USE_TEMPLATES` | No | `true` para plantillas pre-aprobadas |
| `WHATSAPP_TEMPLATE_ORDER_CONFIRM` | No | Nombre plantilla confirmación |
| `WHATSAPP_TEMPLATE_ORDER_STATUS` | No | Nombre plantilla cambio estado |
| `WHATSAPP_TEMPLATE_PAYMENT_REMINDER` | No | Nombre plantilla recordatorio pago |
| `DRIVER_PHONE` | No | WhatsApp para alertas de pedidos listos |
| `CRON_SECRET` | No | Secreto para cron de recordatorios de pago |
| `PORT` | No | Puerto HTTP (default `8080`) |
| `RENDER` | Auto | Render lo setea; activa cookies Secure |

\* Si no hay filas en tabla `admins`, se crea `admin` con esta contraseña al arrancar.

Sin credenciales de pago/WhatsApp, el sistema usa **modo mock** (tarjetas de prueba, WhatsApp en consola).

### Tarjetas de prueba (modo mock / sandbox)

| Tarjeta | Resultado |
|---------|-----------|
| `4111111111111111` | Aprobada |
| `4000000000000002` | Rechazada |

---

## Pre-lanzamiento (checklist)

### Pagos Azul — listo en código
1. Abrir cuenta comercio con Azul (Banco Popular) — 1-3 semanas
2. Configurar `AZUL_MERCHANT_ID`, `AZUL_AUTH1`, `AZUL_AUTH2`, `AZUL_ENVIRONMENT=production`
3. Probar en sandbox: cargo exitoso, rechazo, y que un rechazo **no** crea pedido
4. Webhook Azul (opcional v2) para conciliación

### WhatsApp — listo en código
1. Meta Business + app WhatsApp Cloud API
2. Token permanente + `WHATSAPP_PHONE_ID`
3. Crear y aprobar plantillas Meta; set `WHATSAPP_USE_TEMPLATES=true`
4. Webhook: `GET/POST /webhooks/whatsapp` con `WHATSAPP_VERIFY_TOKEN`

### Infraestructura (pendiente operaciones)
- **Fotos de piezas:** migrar `photo_url` a R2/S3 (Render filesystem es efímero)
- **Backups Postgres:** confirmar plan Render o cron `pg_dump`
- **Dominio propio:** agregar en Render → SSL automático
- **Monitoreo:** Sentry u otro (no integrado aún)

### Legal / negocio (parcial)
- Páginas legales: `/legal/terms`, `/legal/privacy`, `/legal/refund` ✅
- NCF: generación básica en pedidos con tarjeta (secuencial; integrar DGII formal v2)
- RNC/DGA/SIGA: trámite comercial, no código

### Post-lanzamiento (v2)
- Devoluciones/garantía con tracking
- Tarifas por zona de entrega
- CardNet gateway completo (`PAYMENT_GATEWAY=cardnet`)

---

## Desarrollo local

```powershell
cd repuestosdirect
$env:DATABASE_URL = "postgres://user:pass@localhost:5432/repuestosdirect?sslmode=disable"
$env:ADMIN_PASSWORD = "admin123"
go run .
```

Abre http://localhost:8080

**Descargar HTMX** (si falta en `static/js/`):

```powershell
Invoke-WebRequest -Uri "https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js" -OutFile static/js/htmx.min.js
```

---

## Tests

```powershell
# Unit tests (sin DB)
go test ./... -v

# Integration tests (requiere PostgreSQL)
$env:TEST_DATABASE_URL = "postgres://..."
go test -tags=integration ./... -v
```

---

## Cron — recordatorios de pago

Programa una llamada diaria (Render Cron, etc.):

```
GET https://tu-app.onrender.com/cron/reminders?secret=TU_CRON_SECRET
```

Configura `CRON_SECRET` en el entorno. Envía WhatsApp a talleres con cuotas próximas a vencer o vencidas.

---

## Mapa de rutas

### Público
| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/catalogo` | Catálogo por vehículo |
| GET | `/buscar` | Búsqueda libre |
| GET | `/carrito` | Carrito |
| POST | `/cart/add`, `/cart/remove`, `/cart/update` | Carrito |
| POST | `/order/place` | Confirmar pedido |
| GET | `/signup` | Registro de taller |
| GET | `/login` | Login taller |

### Taller (requiere sesión shop)
| GET | `/tienda/pedidos` | Pedidos activos |
| GET | `/tienda/pedidos/historial` | Historial |
| GET | `/tienda/pedidos/{id}` | Detalle + timeline |
| GET | `/tienda/entregas` | Entregas pendientes |

### Admin (requiere sesión admin)
| GET | `/admin` | Panel |
| GET | `/admin/orders` | Pedidos (`?view=completed` para historial) |
| GET | `/admin/shops`, `/admin/shops/create`, `/admin/shops/edit/{id}` | Talleres |
| GET | `/admin/drivers` | Repartidores |
| GET | `/admin/inventory`, `/admin/inventory/add`, `/admin/inventory/edit/{id}` | Inventario |
| GET | `/admin/delivery` | Asignar repartidores |
| GET | `/admin/reports` | Reportes (`?sort=date|revenue|shop`) |

### Repartidor (requiere sesión driver)
| GET | `/driver` | Lista de entregas |
| POST | `/driver/status` | En camino |
| POST | `/driver/deliver` | Entregado / No entregado |

---

## Inventario y catálogo

Campos por pieza: marca, modelo, año, nombre, **número de parte**, **OEM**, condición (nuevo/usado/reacondicionado), precio, precio B2B, stock, punto de reorden, descripción, foto.

El stock baja automáticamente al confirmar un pedido. Alertas de stock bajo en admin y reportes.

---

## Seguridad

- Contraseñas con **bcrypt**
- **CSRF** en todos los formularios POST
- Sesiones en PostgreSQL (rol: guest / shop / admin / driver)
- Cookies `HttpOnly`, `SameSite=Lax`, `Secure` en producción
- Admin no puede entrar por `/login` de taller; roles aislados

---

## Estructura del código

```
repuestosdirect/
  main.go              — rutas y handlers principales
  auth_handlers.go     — login admin, talleres, repartidores
  portal_handlers.go   — panel taller y acciones repartidor
  roles.go             — middleware por rol
  users.go             — admins, drivers, auditoría, migraciones auth
  data.go              — PostgreSQL store
  orders.go            — consultas de pedidos y reportes
  payments.go          — cuotas de crédito y recordatorios
  session.go           — sesiones y carrito
  auth.go              — bcrypt, CSRF
  validate.go          — validación
  notify.go            — WhatsApp
  templates/           — HTML + HTMX
  static/              — HTMX + CSS
```

---

## Notas

- `seedInitialData()` en `data.go` es solo para pruebas locales; no corre en producción.
- Contraseñas legacy en texto plano se migran a bcrypt al iniciar sesión.
- Pedidos v2 pendientes: NCF fiscal, devoluciones/garantía, tarifas por zona, notificaciones SMS al cliente.
