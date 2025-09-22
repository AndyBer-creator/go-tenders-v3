package api

import "net/http"

type ServiceHandler struct{}

func NewServiceHandler() *ServiceHandler {
	return &ServiceHandler{}
}
func (h *ServiceHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}

func (h *ServiceHandler) CheckServer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
