# RepuestosDirect — Prototipo de Plataforma de Pedidos

Prototipo funcional de la plataforma descrita en el plan de negocio: catálogo
por vehículo, carrito, cuentas de taller con crédito, seguimiento de pedidos,
confirmaciones por WhatsApp (simuladas) y un panel de administración con
antigüedad de cuentas por cobrar.

Construido en **Go usando solo la librería estándar** — sin Node.js, sin
paso de compilación de frontend, sin dependencias externas que descargar.
Compila a un solo binario. Esto sigue la misma arquitectura recomendada
(Go + renderizado en servidor + HTMX), simplificada para que corra en
cualquier lado sin instalar nada más que Go.

## Cómo correrlo

Requiere Go 1.22 o más reciente (usa el enrutador nuevo de `net/http`).

```bash
cd repuestosdirect
go run .
```

Abre **http://localhost:8080** en el navegador.

Cuentas de taller de prueba (contraseña para todas: `1234`):
- Taller El Bravo — crédito con espacio disponible
- Auto Servicios Núñez — crédito casi sin usar
- Mecánica Los Hermanos — crédito casi agotado (para probar el rechazo por límite)

También puedes comprar de contado sin iniciar sesión.

Para compilar un binario:
```bash
go build -o repuestosdirect .
./repuestosdirect
```

## Qué incluye este prototipo

- **Catálogo en cascada** Marca → Modelo → Año → Piezas, vía fragmentos HTMX
  (el navegador nunca descarga una app de JavaScript completa, solo HTML).
- **Dos tipos de disponibilidad** por pieza: `local` (mismo día/siguiente) e
  `import` (3 a 4 semanas), como se definió en el plan.
- **Carrito** con actualización de cantidades en vivo.
- **Cuentas de taller con crédito**: límite, saldo usado, términos en días,
  y un **candado real en el servidor** que rechaza un pedido a crédito si
  excede el límite disponible (no es solo una validación visual).
- **Checkout de invitado** (solo contado, sin cuenta) para talleres o
  personas que aún no tienen una cuenta de crédito.
- **Confirmaciones por WhatsApp simuladas** — se imprimen en la consola del
  servidor exactamente como se enviarían; ver `notify.go`.
- **Panel del taller** con historial de pedidos y estado de crédito.
- **Panel de administración** con antigüedad de cuentas por cobrar,
  marcando en rojo lo vencido — el número que el plan de negocio identifica
  como el más importante del negocio.

## Estructura del proyecto

```
main.go       — rutas HTTP, sesiones, handlers de cada página
data.go       — modelo de datos y "base de datos" en memoria (reemplaza a Postgres en el demo)
notify.go     — capa de notificaciones por WhatsApp (stub, ver abajo)
templates/    — plantillas HTML (Go html/template) + HTMX
```

## Moviendo esto a producción

Este prototipo usa una base de datos en memoria y CDNs externos para que
corra sin ninguna instalación adicional. Antes de usarlo con talleres
reales, hay que cambiar tres cosas:

### 1. Base de datos: memoria → PostgreSQL

Todo el archivo `data.go` está escrito para que cada función mapee
directamente a una consulta SQL — por ejemplo, `Store.PartsFor(make, model,
year)` se convierte en:

```sql
SELECT * FROM parts WHERE make = $1 AND model = $2 AND year = $3;
```

Pasos:
1. Crear tablas: `shops`, `parts`, `orders`, `order_items` (los campos ya
   están definidos en los structs `Shop`, `Part`, `Order`, `OrderItem`).
2. Reemplazar el `*Store` en memoria por un `*sql.DB` o un pool de
   [pgx](https://github.com/jackc/pgx) (recomendado por rendimiento).
3. Los handlers en `main.go` no cambian — solo cambia la implementación
   interna de `Store`.
4. Usar transacciones para `PlaceOrder`, ya que actualiza tanto la orden
   como el saldo de crédito del taller atómicamente (el mutex en memoria
   cumple ese rol en el demo).

### 2. WhatsApp: stub → API real

`notify.go` tiene una sola función, `SendWhatsApp(toPhone, message)`. Para
producción, reemplaza su contenido por una llamada HTTP a la
[WhatsApp Business Cloud API](https://developers.facebook.com/docs/whatsapp)
de Meta, o a un proveedor como Twilio. Nada más en el código cambia, porque
todos los llamados ya pasan por esta única función.

### 3. Assets: CDN → self-hosted

El demo carga Tailwind y HTMX desde CDNs externos por velocidad de
desarrollo. Para producción, dado que el plan prioriza rendimiento en
redes 3G/4G lentas:
- Compilar Tailwind localmente (purga las clases no usadas — termina
  pesando pocos KB en vez de cargar el framework completo).
- Descargar `htmx.min.js` (~14 KB) y servirlo desde el propio servidor
  Go usando `http.FileServer`.
- Servir imágenes de piezas en WebP con lazy loading.

### 4. Otras mejoras para producción

- Autenticación real (hash de contraseñas con `bcrypt`, no comparación en
  texto plano como en este demo).
- Sesiones respaldadas por Redis en vez del mapa en memoria de `main.go`,
  para sobrevivir reinicios y escalar a varios servidores.
- Panel de administración protegido con su propio login (en el demo,
  `/admin` es público para poder probarlo fácilmente).
- Integración con el catálogo del yonker en EE.UU. y con la naviera,
  como se describe en la sección 3 del plan de negocio.

## Notas de diseño

- El límite de crédito se valida **en el servidor**, no solo deshabilitando
  el botón en el HTML — así un taller no puede saltarse el límite aunque
  manipule la página.
- Las plantillas usan nombres únicos por página (`page_catalog`,
  `page_cart`, etc.) en vez de un bloque `"body"` compartido, porque
  `html/template` de Go trata los nombres de plantilla como globales al
  conjunto completo — dos archivos con el mismo nombre de bloque se
  sobrescriben entre sí silenciosamente. Si agregas una página nueva,
  sigue el mismo patrón.
