package env

import (
	"bufio"
	"os"
	"strings"
)

// Load lee un archivo .env si existe e inyecta las variables en el entorno sin sobrescribir las ya definidas.
func Load(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Es opcional; en producción o CI/CD se inyectan directamente al entorno
		}
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			// Remover comillas si las tuviera
			val = strings.Trim(val, `"'`)

			// No sobreescribir si ya está fijada en el entorno (ej. en CI/CD o contenedor)
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}
	}

	return scanner.Err()
}
