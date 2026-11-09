package registry

import (
	"database/sql"
	"sync"

	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/handler"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/repository"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/service"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/validator"
)

// Registry contiene e inicializa las dependencias requeridas para el dominio Item.
type Registry struct {
	Handler    *handler.ItemHandler
	Service    domain.ItemService
	Repository domain.ItemRepository
	Clock      clock.Clock
}

var (
	instance *Registry
	once     sync.Once
)

// GetItemRegistry inicializa de manera thread-safe (singleton con sync.Once) y retorna
// la instancia única del Registry para el dominio Item con el reloj del sistema.
func GetItemRegistry(db *sql.DB, val *validator.CustomValidator) *Registry {
	once.Do(func() {
		clk := clock.NewRealClock()
		repo := repository.NewSQLiteItemRepository(db, clk)
		svc := service.NewItemService(repo, clk)
		h := handler.NewItemHandler(svc, val)

		instance = &Registry{
			Handler:    h,
			Service:    svc,
			Repository: repo,
			Clock:      clk,
		}
	})
	return instance
}
