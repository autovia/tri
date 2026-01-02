package handlers

import (
	"log"
	"net/http"
	"os"

	S "github.com/autovia/tri/structs"
)

func ListBuckets(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#ListBuckets %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	files, err := os.ReadDir(s3.Mount)
	if err != nil {
		return S.RespondError(w, 500, "InternalError", err, "")
	}

	buckets := []S.Bucket{}
	for _, file := range files {
		fileInfo, _ := file.Info()
		if file.IsDir() && fileInfo.Name() != Metadata {
			buckets = append(buckets, S.Bucket{Name: fileInfo.Name(), CreationDate: fileInfo.ModTime()})
		}
	}

	bucketList := S.ListAllMyBucketsResult{
		Owner:   &S.Owner{ID: "id", DisplayName: "name"},
		Buckets: buckets,
	}

	return S.RespondXML(w, http.StatusOK, bucketList)
}

func CreateBucket(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#CreateBucket: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); !os.IsNotExist(err) {
		return S.RespondError(w, 409, "BucketAlreadyOwnedByYou", err, s3.Bucket)
	}

	if err := os.Mkdir(s3.Path, os.ModePerm); err != nil {
		return S.RespondError(w, 500, "InternalError", err, s3.Bucket)
	}

	w.Header().Set("Location", s3.Key)
	w.Header().Set("Content-Length", "0")
	w.Header().Set("Server", "AmazonS3")
	return S.Respond(w, http.StatusOK, nil, nil)
}

func HeadBucket(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#HeadBucket: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, 400, "NoSuchBucket", err, s3.Bucket)
	}

	return S.Respond(w, http.StatusOK, nil, nil)
}

func GetBucketVersioning(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#GetBucketVersioning: %v\n", r)

	return S.RespondXML(w, http.StatusOK, S.VersioningConfiguration{Status: "Suspended"})
}

func DeleteBucket(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#DeleteBucket: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}

	contents, err := os.ReadDir(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}

	if len(contents) > 0 {
		return S.RespondError(w, http.StatusConflict, "BucketNotEmpty", err, s3.Bucket)
	}

	if err := os.Remove(s3.Path); err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}

	return S.RespondXML(w, http.StatusNoContent, nil)
}
