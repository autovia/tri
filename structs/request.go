package structs

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

type Request struct {
	Bucket string
	Key    string
	Path   string
	Mount  string
}

func (app *App) ParseRequest(r *http.Request) (*http.Request, error) {
	var bucket, key, path string

	if r.URL.Path != "/" {
		urlPath, found := strings.CutPrefix(r.URL.Path, "/")
		if !found {
			return nil, fmt.Errorf("prefix not found")
		}

		if len(urlPath) == 0 {
			return nil, fmt.Errorf("bucket missing")
		}

		uPath, err := url.QueryUnescape(urlPath)
		if err != nil {
			return nil, fmt.Errorf("can not unescape url")
		}

		split := strings.Split(uPath, "/")
		switch len(split) {
		case 0:
			return nil, fmt.Errorf("bucket missing")
		case 1:
			bucket = split[0]
			key = ""
			path = filepath.Clean(filepath.Join(*app.Mount, bucket))
		default:
			bucket = split[0]
			key, _ = strings.CutPrefix(uPath, bucket+"/")
			path = filepath.Clean(filepath.Join(*app.Mount, bucket, key))
			log.Printf(">>> bucket: %s, key: %s, path: %s, split: %v\n", bucket, key, path, split)
		}

		// check prefix
		prefix := r.URL.Query().Get("prefix")
		if len(prefix) > 0 {
			key = prefix
			path = filepath.Clean(filepath.Join(*app.Mount, bucket, key))
		}
	}

	req := Request{
		Bucket: bucket,
		Key:    key,
		Path:   path,
		Mount:  *app.Mount,
	}

	ctx := context.WithValue(r.Context(), Request{}, req)
	//log.Printf(">>> bucket: %s, key: %s, prefix: %v, path: %s, split: %v\n", req.Bucket, req.Key, len(prefix) > 0, req.Path, len(split))
	return r.WithContext(ctx), nil
}
