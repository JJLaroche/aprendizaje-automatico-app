package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	itemRegistry "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/registry"
	"github.com/marielarodriguez/aprendizaje-automatico-app/migrations"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/database"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/env"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/validator"
)

func main() {
	// Carga opcional de .env local (no falla si no existe, ideal para CI/CD y producción)
	_ = env.Load(".env")

	// Logger estructurado JSON nativo con log/slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Configuración de entorno requerida (sin fallback para no asumir ni exponer ubicaciones arbitrarias)
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		logger.Error("variable de entorno DB_PATH no configurada; debe ser provista explícitamente")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("inicializando base de datos SQLite")
	db, err := database.OpenSQLite(dbPath)
	if err != nil {
		logger.Error("fallo al abrir base de datos", "error", err)
		os.Exit(1)
	}
	defer func() {
		logger.Info("cerrando conexiones de base de datos")
		_ = db.Close()
	}()

	// Ejecutar migraciones automáticas al inicio desde embed.FS
	migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMigration()
	if err := database.MigrateFromFS(migrationCtx, db, migrations.Files, "000001_create_items_table.up.sql"); err != nil {
		logger.Error("fallo al ejecutar migraciones", "error", err)
		os.Exit(1)
	}
	logger.Info("migraciones de base de datos ejecutadas exitosamente")

	// Inicialización de componentes e inyección vía Registry (Singleton con sync.Once)
	customValidator := validator.New()
	itemReg := itemRegistry.GetItemRegistry(db, customValidator)

	// Enrutador Chi principal y Middlewares globales
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	// Endpoint de verificación de salud
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"data":{"status":"ok","timestamp":"%s"}}`, time.Now().UTC().Format(time.RFC3339))))
	})

	// Registro explícito y centralizado de rutas del dominio Item en main
	r.Route("/api/v1/items", func(r chi.Router) {
		r.Post("/", itemReg.Handler.Create)
		r.Get("/", itemReg.Handler.List)
		r.Get("/{id}", itemReg.Handler.GetByID)
		r.Patch("/{id}/status", itemReg.Handler.UpdateStatus)
		r.Delete("/{id}", itemReg.Handler.Delete)
	})

	// Servidor HTTP con configuración de Timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Canal para captura de señales de apagado (Graceful Shutdown)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("servidor HTTP escuchando", "puerto", port)
		serverErrors <- server.ListenAndServe()
	}()

	// Espera de señal de apagado o error fatal del servidor
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("error fatal en servidor HTTP", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdownSignal:
		logger.Info("señal de terminación recibida, iniciando graceful shutdown", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("error durante graceful shutdown del servidor", "error", err)
			_ = server.Close()
		}
		logger.Info("servidor detenido limpiamente")
	}
}
