package structs

import (
	"errors"
	"log"
	"net/http"
)

type Auth struct {
	*App
	R map[string]any
}

func (a Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("AuthHandler %v\n%v", r, a.R)

	if _, ok := a.R[r.Method]; !ok {
		log.Print("http method not allowed")
		RespondError(w, 405, "MethodNotAllowed", errors.New("MethodNotAllowed"), "")
		return
	}

	valid, r := a.ValidSignatureV4(r)
	if !valid {
		log.Print("signature not valid")
		RespondError(w, 401, "UnauthorizedAccess", errors.New("UnauthorizedAccess"), "")
		return
	}

	r, err := a.ParseRequest(r)
	if err != nil {
		RespondError(w, 500, "InternalError", err, "")
		return
	}

	err = a.R[r.Method].(func(w http.ResponseWriter, r *http.Request) error)(w, r)
	if err != nil {
		log.Print(err)
		return
	}
}
