package apiv1

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/config"
)

const Baseurl = "/api/v1"

func Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/test", GetTest)
	return router
}

/*
GetTest list of all possible icon names
*/
func GetTest(response http.ResponseWriter, request *http.Request) {
	icons := config.Get()
	render.JSON(response, request, icons)
}
