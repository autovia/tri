package structs

import (
	"net/http"
)

type App struct {
	Addr      *string
	Router    *http.ServeMux
	AccessKey *string
	SecretKey *string
	Mount     *string
}
