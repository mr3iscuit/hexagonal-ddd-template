package application

import (
	"database/sql"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/other/domain"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/adapters/outbound/memory"
)

type WidgetRepository interface {
	Save(database *sql.DB) error
}

type WidgetUseCasesPort interface {
	Load() domain.Widget
}

type GhostRepositoryPort interface {
	Save() error
}

type CreateWidgetCommandDTO struct {
	Widget domain.Widget
}

type WidgetApplicationService struct {
	store memory.WidgetStore
}

func (service WidgetApplicationService) save() error { return nil }

func persistWidget() error { return nil }
