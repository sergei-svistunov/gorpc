package handler_foreign_return_values

import ()

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Caption() string {
	return "Test handler with foreign return values"
}

func (h *Handler) Description() string {
	return "Handler for tests: has return value structure described in other package"
}
