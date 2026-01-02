package structs

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"os"
)

func RespondXML(w http.ResponseWriter, code int, payload any) error {
	out, _ := xml.MarshalIndent(payload, " ", "  ")
	log.Print(">>> RespondXML >>>", string(out))

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(code)
	w.Write([]byte(out))

	return nil
}

func Respond(w http.ResponseWriter, code int, headers map[string]string, body []byte) error {
	if len(headers) > 0 {
		for k, v := range headers {
			w.Header().Set(k, v)
			log.Println("metadata >>", k, v)
		}
	}

	w.WriteHeader(code)
	if len(body) > 0 {
		w.Write(body)
	}

	return nil
}

func RespondFile(w http.ResponseWriter, code int, headers map[string]string, file *os.File) error {
	if len(headers) > 0 {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
	}

	w.WriteHeader(code)
	defer file.Close()
	io.Copy(w, file)

	return nil
}

type Error struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestId string   `xml:"RequestId"`
}

func RespondError(w http.ResponseWriter, httpcode int, awscode string, err error, resource string) error {
	e := Error{
		Code:     awscode,
		Message:  awscode,
		Resource: resource,
	}

	log.Printf(">>> RespondError >>> %s, %s", err, awscode)

	out, _ := xml.MarshalIndent(e, " ", "  ")
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(httpcode)
	w.Write([]byte(out))

	return nil
}
