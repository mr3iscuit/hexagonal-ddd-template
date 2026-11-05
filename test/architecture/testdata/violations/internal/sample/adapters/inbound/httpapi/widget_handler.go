package httpapi

import (
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/adapters/outbound/memory"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/application"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/domain"
)

type WidgetHandler struct {
	service application.WidgetApplicationService
}

type widgetBody struct{}

type WidgetResponseDTO struct{}

func toWidgetBody(widget domain.Widget) widgetBody {
	return WidgetResponseDTO(widget)
}

func saveErr(cfg memory.WidgetStore) {}

var _ = middleware.RequestID

var _ = middleware.Recoverer
