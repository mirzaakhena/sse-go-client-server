package controller

import (
	"net/http"
	"shared/utility"
)

type Controller struct {
	Mux *http.ServeMux
	JWT utility.JWTTokenizer
}
