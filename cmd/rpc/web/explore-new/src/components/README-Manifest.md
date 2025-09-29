# Table Manifest System

Este sistema permite configurar tablas dinámicamente usando archivos JSON sin tocar el código. Mantiene toda la UI actual intacta.

## Archivos Creados

### 1. `src/data/table-manifests.json`
Contiene todas las configuraciones de tablas para:
- `blocks` - Tabla de bloques
- `transactions` - Tabla de transacciones  
- `validators` - Tabla de validadores
- `accounts` - Tabla de cuentas
- `home-validators` - Tabla de validadores en Home
- `home-transactions` - Tabla de transacciones en Home

### 2. `src/hooks/useTableManifest.ts`
Hook principal que maneja:
- Carga de manifests
- Formateo de celdas basado en configuración
- Tipos TypeScript para todas las configuraciones

### 3. `src/components/TableFilters.tsx`
Componente de filtros dinámico que se genera automáticamente desde el manifest.

### 4. `src/components/TableWithManifest.tsx`
Componente wrapper que combina filtros y tabla usando manifests.

## Cómo Usar

### Opción 1: Usar TableCard con manifestKey
```tsx
import TableCard from './Home/TableCard';

<TableCard
  title="Blocks"
  manifestKey="blocks"
  data={blocksData}
  loading={loading}
  paginate={true}
  currentPage={currentPage}
  totalCount={totalCount}
  onPageChange={onPageChange}
/>
```

### Opción 2: Usar TableWithManifest (incluye filtros)
```tsx
import TableWithManifest from './TableWithManifest';

<TableWithManifest
  manifestKey="blocks"
  data={blocksData}
  loading={loading}
  currentPage={currentPage}
  totalCount={totalCount}
  onPageChange={onPageChange}
  showFilters={true}
/>
```

### Opción 3: Usar componentes específicos con manifest
```tsx
import BlocksTableWithManifest from './block/BlocksTableWithManifest';
import TransactionsTableWithManifest from './transaction/TransactionsTableWithManifest';
import ValidatorsTableWithManifest from './validator/ValidatorsTableWithManifest';
```

## Configuración de Columnas

Cada columna en el manifest puede tener:

```json
{
  "key": "height",
  "label": "Block Height", 
  "type": "number",
  "format": "animated",
  "link": "/block/{{value}}",
  "icon": "cube",
  "color": "primary"
}
```

### Tipos de Columna
- `text` - Texto simple
- `number` - Números
- `address` - Direcciones (se truncan automáticamente)
- `hash` - Hashes (se truncan automáticamente)
- `datetime` - Fechas y horas
- `relative-time` - Tiempo relativo (ej: "2h ago")

### Formatos
- `truncate` - Trunca texto largo
- `badge` - Muestra como badge con colores
- `currency` - Formato de moneda
- `percentage` - Formato de porcentaje
- `duration` - Formato de duración
- `animated` - Números animados

### Colores
- `primary` - Color primario
- `green` - Verde
- `red` - Rojo
- `blue` - Azul
- `yellow` - Amarillo
- `orange` - Naranja
- `gray` - Gris

## Configuración de Filtros

```json
{
  "filters": {
    "enabled": true,
    "fields": [
      {
        "key": "timeRange",
        "type": "select",
        "label": "Time Range",
        "options": [
          { "value": "all", "label": "All Blocks" },
          { "value": "hour", "label": "Last Hour" }
        ]
      }
    ]
  }
}
```

### Tipos de Filtro
- `select` - Dropdown
- `text` - Campo de texto
- `range` - Slider de rango

## Configuración de Paginación

```json
{
  "pagination": {
    "enabled": true,
    "pageSize": 10,
    "showTotal": true
  }
}
```

## Agregar Nueva Tabla

1. Agregar configuración en `table-manifests.json`:
```json
{
  "tables": {
    "mi-tabla": {
      "title": "Mi Tabla",
      "columns": [...],
      "filters": {...},
      "pagination": {...}
    }
  }
}
```

2. Usar en componente:
```tsx
<TableCard manifestKey="mi-tabla" data={miData} />
```

## Ventajas

✅ **Sin tocar UI existente** - Todo el diseño actual se mantiene
✅ **Configuración dinámica** - Cambiar columnas sin código
✅ **Filtros automáticos** - Se generan desde el manifest
✅ **Formateo consistente** - Mismo estilo en todas las tablas
✅ **TypeScript** - Tipado completo y seguro
✅ **Reutilizable** - Un manifest para múltiples tablas
✅ **Mantenible** - Cambios centralizados en JSON

## Compatibilidad

- ✅ Mantiene toda la funcionalidad existente de TableCard
- ✅ Compatible con paginación externa e interna
- ✅ Soporte para loading states
- ✅ Mensajes de "no data" automáticos
- ✅ Export a CSV (si está habilitado)
- ✅ Links a páginas de detalle
- ✅ Animaciones y transiciones

