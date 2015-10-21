package handler_foreign_arguments

import ()

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Caption() string {
	return "Test handler with foreign arguments"
}

func (h *Handler) Description() string {
	return "Handler for tests: has argument structure described in other package"
}
