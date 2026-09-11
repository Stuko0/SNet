package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config es la configuración persistente de SNet.
//
// Este paquete existía desde el principio con su test, pero nadie lo llamaba:
// la pestaña activa se perdía al cerrar. Ahora se carga al arrancar y se guarda
// al salir.
type Config struct {
	// LastTab es la pestaña activa al cerrar (0-4).
	LastTab int `toml:"last_tab"`
	// RefreshSec es cada cuántos segundos se refresca el dashboard. 0 = nunca.
	RefreshSec int `toml:"refresh_sec"`
	// ConfirmDelete pide confirmación antes de borrar una conexión.
	ConfirmDelete bool `toml:"confirm_delete"`
}

// NumTabs es la cantidad de pestañas; se usa para validar LastTab y evitar
// arrancar en una pestaña inexistente si el config viene de una versión con más
// vistas.
const NumTabs = 5

// Default devuelve la configuración por defecto.
func Default() *Config {
	return &Config{
		LastTab:       0,
		RefreshSec:    0, // sin auto-refresh: el usuario decide con 'r'
		ConfirmDelete: true,
	}
}

func GetConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME") + "/.config"
	}
	return filepath.Join(configDir, "nmtui", "config.toml")
}

// LoadConfig lee el archivo de configuración. Si no existe devuelve los valores
// por defecto; si está corrupto también, para que una config rota no impida
// arrancar.
func LoadConfig() (*Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	cfg := Default()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		// Archivo ilegible: seguir con defaults en vez de fallar al arrancar.
		return Default(), nil
	}
	cfg.normalize()
	return cfg, nil
}

// normalize corrige valores fuera de rango. Una config vieja o editada a mano
// no debe dejar la app en un estado inválido.
func (c *Config) normalize() {
	if c.LastTab < 0 || c.LastTab >= NumTabs {
		c.LastTab = 0
	}
	if c.RefreshSec < 0 {
		c.RefreshSec = 0
	}
	// Un refresh de 1-2s satura de llamadas a nmcli sin aportar nada.
	if c.RefreshSec > 0 && c.RefreshSec < 5 {
		c.RefreshSec = 5
	}
	if c.RefreshSec > 3600 {
		c.RefreshSec = 3600
	}
}

func SaveConfig(cfg *Config) error {
	path := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	// Escritura atómica: si el proceso muere a mitad, no queda un config
	// truncado que rompa el próximo arranque.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
