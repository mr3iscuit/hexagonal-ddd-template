package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/domain"
)

func main() {
	_ = chi.NewRouter()
	_ = domain.Widget{}
}
