package handlers

import (
	"log"
	"net/http"
	"os"

	S "github.com/autovia/tri/structs"
)

func Get(w http.ResponseWriter, r *http.Request) error {
	log.Printf(">>> GET %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if r.URL.Path == "/" {
		return ListBuckets(w, r)
	}

	stat, err := os.Stat(s3.Path)
	if os.IsNotExist(err) {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}

	if r.URL.Query().Has("versioning") {
		return GetBucketVersioning(w, r)
	}

	if stat.IsDir() {
		return ListObjectsV2(w, r)
	}

	if r.URL.Query().Has("versions") {
		return ListObjectVersions(w, r)
	}

	return GetObject(w, r)
}

func Put(w http.ResponseWriter, r *http.Request) error {
	log.Printf(">>> PUT %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if len(s3.Key) > 0 {
		if len(r.Header.Get("X-Amz-Copy-Source")) > 0 {
			return CopyObject(w, r)
		}
		return PutObject(w, r)
	}

	return CreateBucket(w, r)
}

func Post(w http.ResponseWriter, r *http.Request) error {
	log.Printf(">>> POST %v\n", r)

	if r.URL.Query().Has("uploads") {
		return CreateMultipartUpload(w, r)
	}

	if r.URL.Query().Has("delete") {
		return DeleteObjects(w, r)
	}

	return S.RespondError(w, 500, "InternalError", nil, "")
}

func Delete(w http.ResponseWriter, r *http.Request) error {
	log.Printf(">>> DELETE %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if len(s3.Key) > 0 {
		return DeleteObject(w, r)
	}

	return DeleteBucket(w, r)
}

func Head(w http.ResponseWriter, r *http.Request) error {
	log.Printf(">>> HEAD %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if len(s3.Key) > 0 {
		return HeadObject(w, r)
	}
	return HeadBucket(w, r)
}
