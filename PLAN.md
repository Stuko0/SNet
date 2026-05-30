# NetworkManager TUI — Plan de Desarrollo

## 1. Stack Tecnológico

| Componente        | Elección                                  | Razón                                                                  |
|-------------------|-------------------------------------------|------------------------------------------------------------------------|
| Lenguaje          | **Go 1.25+**                              | Binario estático, cross-platform, excelente ecosistema TUI             |
| TUI Framework     | **Bubble Tea (charmbracelet/bubbletea)**  | Modelo Elm-like (model/update/view), reactivo, más popular en Go TUI   |
| Componentes UI    | **Bubbles (charmbracelet/bubbles)**       | Widgets pre-hechos: tabla, input, spinner, paginación, help            |
| Estilos           | **Lipgloss (charmbracelet/lipgloss)**     | Colores, bordes, alineación, estilos responsive                        |
| Backend NM        | **nmcli** vía `os/exec`                   | Más simple y robusto que D-Bus directo; nmcli cubre 100% de lo necesario|
| Config persistente| **TOML** (archivo local ~/.config/nmtui/) | Sencillo, legible, tipado                                              |
| Barra de estado   | **Bubble Tea + Lipgloss**                 | Hecho a mano con los mismos bloques                                    |

## 2. Estructura del Proyecto

```
nmtui/
├── main.go                  # Punto de entrada, inicialización
├── go.mod
├── go.sum
│
├── tui/
│   ├── app.go               # Modelo principal, enrutamiento de vistas
│   ├── styles.go            # Estilos Lipgloss globales (colores, bordes)
│   ├── keys.go              # Definición de keybindings globales
│   └── views/               # Cada vista como un modelo Bubble Tea
│       ├── dashboard.go     # Panel principal: estado actual, redes
│       ├── wifilist.go      # Escaneo y listado de redes Wi-Fi
│       ├── saved.go         # Redes guardadas / gestionar conexiones
│       ├── vpnlist.go       # Lista de VPNs registradas
│       ├── hotspot.go       # Configuración y control de hotspot
│       ├── editor.go        # Editor genérico de conexiones (SSID, pwd, IP, etc)
│       └── help.go          # Pantalla de ayuda interactiva
│
├── nm/
│   ├── nmcli.go             # Wrapper para ejecutar comandos nmcli
│   ├── wifi.go              # Escaneo, conexión Wi-Fi
│   ├── connection.go        # CRUD de conexiones (add, modify, delete)
│   ├── vpn.go               # Gestión de VPNs
│   ├── hotspot.go           # Crear/controlar hotspot
│   └── types.go             # Tipos compartidos (Connection, WiFiNetwork, VPN, etc)
│
├── config/
│   └── config.go            # Carga/guarda configuración TOML
│
└── utils/
    ├── formatter.go         # Formateo de señal, velocidad, bytes, etc
    └── executor.go          # Ejecución de comandos con timeout y parsing
```

## 3. Flujo de Navegación

```
┌─────────────────────────────────────────────────────────────────┐
│  nmtui  (barra de título con versión)                           │
├─────────────────────────────────────────────────────────────────┤
│  [Dashboard] [Wi-Fi] [Guardadas] [VPN] [Hotspot]  (tabs)        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                        <CONTENIDO PRINCIPAL>                     │
│                                                                  │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  Ctrl+Q: Salir  ?: Ayuda  Tab/Ctrl+Tab: Navegar vistas          │
└─────────────────────────────────────────────────────────────────┘
```

### Mapa de navegación

```
Dashboard ──→ Wi-Fi List ──→ Saved Networks ──→ VPN Manager ──→ Hotspot
   │               │               │                │               │
   │               ├─ Conectar     ├─ Editar        ├─ Conectar     ├─ Iniciar
   │               ├─ Ver detalles ├─ Eliminar      ├─ Desconectar  ├─ Detener
   │               └─ Guardar      └─ Ver contraseña └─ Agregar VPN └─ Configurar
   │
   └── Reconectar / Refresh automático
```

### Keybindings globales

| Tecla              | Acción                           |
|--------------------|----------------------------------|
| `Tab` / `Shift+Tab`| Navegar entre vistas (tabs)      |
| `Ctrl+Q` / `q`     | Confirmar salida                 |
| `?`                | Pantalla de ayuda                |
| `r`                | Refresh / rescanear              |
| `↑`/`↓`            | Navegar listas                   |
| `Enter`            | Seleccionar / conectar           |
| `d`                | Eliminar conexión seleccionada   |
| `e`                | Editar conexión seleccionada     |
| `Ctrl+N`           | Nueva conexión / VPN / Hotspot   |
| `Esc`              | Volver / Cancelar                |

## 4. Funcionalidades por Vista

### 4.1 Dashboard
- Estado actual de conectividad (icono: conectado/desconectado/limitado)
- Nombre de red activa, SSID, tipo (Wi-Fi/Ethernet/VPN)
- IP local, gateway, DNS
- Velocidad de enlace (si aplica)
- Intensidad de señal (barras) para Wi-Fi activo
- Dispositivos de red disponibles
- Acción rápida: desconectar / reconectar

### 4.2 Wi-Fi List
- Escaneo en tiempo real (con spinner de carga)
- Tabla con columnas: SSID | Seguridad | Señal | Frecuencia | BSSID
- Paginación si hay muchas redes
- Indicador de redes ya conocidas (★)
- Al seleccionar Enter:
  - Si es abierta → conecta directamente
  - Si tiene contraseña → muestra input de password (oculto) + opción de guardar
  - Si ya está guardada → conecta directamente
- Feedback visual: "Conectando...", "Conectado ✓", "Error ✗"

### 4.3 Saved Networks (Conexiones guardadas)
- Lista de conexiones conocidas (nmcli connection show)
- Iconos por tipo: 📶 Wi-Fi, 🔒 VPN, 🔌 Ethernet
- Indicador de conexión activa
- Acciones por item:
  - Conectar
  - Editar configuración (SSID, password, IP estática, DNS, etc)
  - Eliminar (con confirmación)
  - Ver contraseña (si aplica)
  - Establecer como prioritaria (autoconnect priority)

### 4.4 VPN Manager
- Lista de conexiones VPN registradas
- Soporte para: OpenVPN, WireGuard, L2TP, SSTP (lo que nmcli soporte)
- Indicador de conexión activa
- Acciones:
  - Conectar / Desconectar
  - Configurar (servidor, usuario, certificados)
  - Agregar nueva VPN (asistente por tipo)
  - Eliminar
- Estado: tiempo conectado, bytes transmitidos (si activa)

### 4.5 Hotspot
- Estado actual (apagado / activo con N clientes)
- Configuración:
  - SSID
  - Contraseña (mín. 8 caracteres)
  - Banda (2.4GHz / 5GHz)
  - Interfaz Wi-Fi (si hay múltiples)
- Botones:
  - Iniciar hotspot
  - Detener hotspot
  - Guardar configuración como perfil
- Trampa conocida: si el Wi-Fi está conectado a una red, el hotspot puede no funcionar por limitación del hardware

### 4.6 Editor de Conexiones (genérico)
Se reutiliza para editar cualquier tipo de conexión:
- Modo formulario con inputs
- Soporta: SSID, password, IPv4/IPv6, DNS, rutas, MTU, prioridad autoconnect
- Validación en tiempo real de campos (IP válida, password mínima, etc)
- Guardar / Cancelar / Aplicar

## 5. UX / Diseño Visual

### Paleta de colores (Lipgloss)
- Adaptable a terminal claro/oscuro (usar `lipgloss.AdaptiveColor`)
- Primario: azul (tono interfaz)
- Secundario: verde para acciones exitosas
- Rojo para errores y desconexión
- Amarillo para advertencias
- Fondo: heredado del terminal

### Layout de cada vista
1. **Header**: Título de la vista con icono
2. **Body**: Contenido principal (tablas, formularios, estado)
3. **Footer**: Atajos de teclado relevantes para la vista actual

### Componentes visuales
- **Tabla estilizada**: Bubbles `table.Model` con colores alternados y scroll
- **Spinner**: Bubbles `spinner.Model` para operaciones async (escaneo, conexión)
- **Input**: Bubbles `textinput.Model` con soporte de password mode
- **Confirm dialog**: Modal de confirmación para acciones destructivas (eliminar)
- **Notification**: Toast-style para feedback rápido ("Conectado a Red" / "Error: ...")
- **Help overlay**: Pantalla completa con todos los keybindings

### Estados de carga y error
- Cada operación larga (scan, connect, disconnect) muestra un spinner
- Errores de nmcli se muestran en un banner rojo con el mensaje real
- Timeout de escaneo: 15s máximo, con mensaje de error si falla
- Si nmcli no está instalado, mensaje claro al iniciar

## 6. Implementación por Fases

### Fase 1: Esqueleto base (día 1-2)
- [ ] `go mod init` + dependencias (bubbletea, bubbles, lipgloss, toml)
- [ ] `main.go` con inicialización básica
- [ ] `tui/app.go` con el modelo principal y enrutamiento de vistas (tabs)
- [ ] `tui/styles.go` con la paleta y estilos globales
- [ ] Dashboard básico: estado de conectividad, IP, dispositivo activo
- [ ] Barra de estado inferior con keybindings esenciales

### Fase 2: Wi-Fi scanning y conexión (día 3-4)
- [ ] `nm/nmcli.go` wrapper base de comandos
- [ ] `nm/wifi.go` escaneo (nmcli device wifi list) y conexión
- [ ] `tui/views/wifilist.go` tabla de redes con señal y seguridad
- [ ] Input de contraseña con toggle de visibilidad
- [ ] Feedback de conexión (spinner + resultado)
- [ ] Refresh manual y automático

### Fase 3: Gestión de conexiones guardadas (día 5-6)
- [ ] `nm/connection.go` CRUD de conexiones vía nmcli
- [ ] `tui/views/saved.go` lista con acciones (conectar, editar, eliminar)
- [ ] `tui/views/editor.go` formulario de edición de conexión
- [ ] Confirmación antes de eliminar
- [ ] Ver/ocultar contraseña guardada

### Fase 4: Hotspot (día 7)
- [ ] `nm/hotspot.go` crear/iniciar/detener hotspot
- [ ] `tui/views/hotspot.go` configuración y control
- [ ] Validación de SSID y password (mín 8 chars)
- [ ] Manejo de error: Wi-Fi ocupado

### Fase 5: VPN Manager (día 8-9)
- [ ] `nm/vpn.go` listar, agregar, conectar VPNs
- [ ] `tui/views/vpnlist.go` interfaz de VPNs
- [ ] Asistente de creación por tipo (OpenVPN, WireGuard)

### Fase 6: Pulido y extras (día 10-11)
- [ ] Help overlay completo
- [ ] Confirmación al salir (Ctrl+Q)
- [ ] Auto-refresh de estado en dashboard
- [ ] Manejo de errores robusto (timeouts, nmcli no instalado)
- [ ] Pruebas manuales de todos los flujos
- [ ] README.md con instrucciones de uso
- [ ] System tray notification opcional (vía `notify-send`)

## 7. Dependencias Go

```
require (
    github.com/charmbracelet/bubbletea    v1.x    # Framework TUI
    github.com/charmbracelet/bubbles      v0.x    # Componentes (table, spinner, textinput, help)
    github.com/charmbracelet/lipgloss     v1.x    # Estilos
    github.com/pelletier/go-toml/v2       v2.x    # Config TOML
)
```

## 8. Casos Borde y Trampas Conocidas

### nmcli
- `nmcli device wifi list` puede fallar si el Wi-Fi está desactivado → mostrar mensaje claro
- Algunas redes pueden no mostrar BSSID → mostrar placeholder
- nmcli a veces devuelve "unknown" para seguridad → mostrar como "Unknown"
- Conexión a redes con WPA3 puede requerir versiones recientes de NM

### Hotspot
- No se puede tener hotspot y Wi-Fi cliente en el mismo adaptador (depende del hardware)
- Algunos adaptadores no soportan 5GHz en modo AP
- La contraseña debe tener mínimo 8 caracteres (nmcli lo exige)

### General
- El usuario podría estar en entorno sin Wi-Fi (solo ethernet) → ocultar opciones Wi-Fi
- nmcli usa locale del sistema para algunos mensajes → forzar LC_ALL=C al ejecutar comandos
- Colores: respetar si el terminal es oscuro o claro (Lipgloss AdaptiveColor)
- Tamaño de terminal mínimo: 80x24

## 9. Arquitectura de Estado (Bubble Tea Model)

```go
type model struct {
    activeTab  int           // 0-5
    width      int
    height     int
    ready      bool          // todo inicializado
    err        error         // error global (si hay)
    
    // Estado de red (refresh periódico)
    state      nm.NetworkState
    
    // Sub-modelos por vista
    dashboard   dashboardModel
    wifiList    wifiListModel
    saved       savedModel
    vpnList     vpnListModel
    hotspot     hotspotModel
    editor      *editorModel  // nil si no activo
    
    // Overlays
    showHelp   bool
    quitConfirm bool
    toast       *toast
}
```

Cada sub-modelo implementa Bubble Tea (Init/Update/View) pero se comunica con el modelo padre a través de mensajes (cmd personalizados).

## 10. Mockup de la vista principal (Dashboard)

```
┌─────────────────────────────────────────────────────────────────────┐
│  📡 nmtui v0.1.0                                    NetworkManager  │
├────────┬──────────┬──────────┬──────────┬──────────┬────────────────┤
│ Status │  Wi-Fi   │ Saved    │   VPN    │ Hotspot  │                │
├────────┴──────────┴──────────┴──────────┴──────────┴────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  🌐 Estado: Conectado                             ● full     │    │
│  │  ─────────────────────────────────────────────────────────── │    │
│  │  Red activa:  Stukos House INV     📶 ████████░░  (85%)      │    │
│  │  Tipo:        Wi-Fi (wlp0s20f3)                              │    │
│  │  Velocidad:   866.7 MBit/s (TX/RX)                           │    │
│  │  IP local:    192.168.1.10/24                                │    │
│  │  Gateway:     192.168.1.1                                   │    │
│  │  DNS:         1.1.1.1, 8.8.8.8                              │    │
│  │  ─────────────────────────────────────────────────────────── │    │
│  │  📶 Redes disponibles: 12                                    │    │
│  │  🔒 VPN activa: ❌ Ninguna                                  │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                      │
├─────────────────────────────────────────────────────────────────────┤
│  r: Refresh  Enter: Detalles  Tab/Ctrl+Tab: Navegar  ?: Ayuda       │
│  Ctrl+Q: Salir                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

¿Te parece bien este plan? Si quieres empezar, te propongo instalar Go, crear el proyecto base e implementar la **Fase 1** para tener un esqueleto funcional corriendo.
