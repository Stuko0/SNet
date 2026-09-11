package config

import (
	"os"
	"path/filepath"
	"testing"
)

// conHomeTemporal aísla la config en un HOME temporal y devuelve la ruta del
// archivo que se crearía.
func conHomeTemporal(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "") // que UserConfigDir use HOME
	return filepath.Join(tmpDir, ".config", "nmtui", "config.toml")
}

func TestLoadSinArchivoDevuelveDefaults(t *testing.T) {
	conHomeTemporal(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("no debería fallar sin archivo: %v", err)
	}
	def := Default()
	if cfg.LastTab != def.LastTab {
		t.Errorf("LastTab = %d, se esperaba %d", cfg.LastTab, def.LastTab)
	}
	if cfg.ConfirmDelete != def.ConfirmDelete {
		t.Errorf("ConfirmDelete = %v, se esperaba %v", cfg.ConfirmDelete, def.ConfirmDelete)
	}
}

func TestSaveYLoadRoundtrip(t *testing.T) {
	path := conHomeTemporal(t)

	cfg := Default()
	cfg.LastTab = 3
	cfg.RefreshSec = 30
	cfg.ConfirmDelete = false
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig falló: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no se creó el archivo: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig falló: %v", err)
	}
	if got.LastTab != 3 || got.RefreshSec != 30 || got.ConfirmDelete {
		t.Errorf("roundtrip incorrecto: %+v", got)
	}
}

// TestLoadConfigCorruptaNoRompe: un TOML inválido no debe impedir arrancar.
func TestLoadConfigCorruptaNoRompe(t *testing.T) {
	path := conHomeTemporal(t)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("esto no es = = toml [[["), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("debería caer a defaults en vez de fallar: %v", err)
	}
	if cfg.LastTab != 0 {
		t.Errorf("LastTab = %d, se esperaba 0 (default)", cfg.LastTab)
	}
}

// TestNormalizeCorrigeValoresInvalidos: una pestaña fuera de rango no debe
// dejar la app en una vista inexistente.
func TestNormalizeCorrigeValoresInvalidos(t *testing.T) {
	casos := []struct {
		desc string
		in   Config
		want Config
	}{
		{"tab negativa", Config{LastTab: -1, ConfirmDelete: true}, Config{LastTab: 0, ConfirmDelete: true}},
		{"tab fuera de rango", Config{LastTab: 99, ConfirmDelete: true}, Config{LastTab: 0, ConfirmDelete: true}},
		{"tab válida se respeta", Config{LastTab: 4, ConfirmDelete: true}, Config{LastTab: 4, ConfirmDelete: true}},
		{"refresh negativo", Config{RefreshSec: -5}, Config{RefreshSec: 0}},
		{"refresh demasiado bajo se sube", Config{RefreshSec: 1}, Config{RefreshSec: 5}},
		{"refresh válido se respeta", Config{RefreshSec: 30}, Config{RefreshSec: 30}},
		{"refresh enorme se acota", Config{RefreshSec: 99999}, Config{RefreshSec: 3600}},
	}
	for _, c := range casos {
		got := c.in
		got.normalize()
		if got.LastTab != c.want.LastTab {
			t.Errorf("%s: LastTab = %d, se esperaba %d", c.desc, got.LastTab, c.want.LastTab)
		}
		if got.RefreshSec != c.want.RefreshSec {
			t.Errorf("%s: RefreshSec = %d, se esperaba %d", c.desc, got.RefreshSec, c.want.RefreshSec)
		}
	}
}

// TestSaveEsAtomico: no debe quedar un .tmp colgado tras guardar.
func TestSaveEsAtomico(t *testing.T) {
	path := conHomeTemporal(t)
	if err := SaveConfig(Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("quedó el archivo temporal %s.tmp", path)
	}
}

// TestNumTabsCoincideConLasVistas: si se agrega una pestaña hay que actualizar
// NumTabs, o LastTab se normalizaría siempre a 0.
func TestNumTabsEsPositivo(t *testing.T) {
	if NumTabs <= 0 {
		t.Fatal("NumTabs debe ser positivo")
	}
	// El índice máximo guardable debe ser válido tras normalize.
	cfg := Config{LastTab: NumTabs - 1}
	cfg.normalize()
	if cfg.LastTab != NumTabs-1 {
		t.Errorf("el último índice válido (%d) se normalizó a %d", NumTabs-1, cfg.LastTab)
	}
}
