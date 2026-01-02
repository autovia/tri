package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/autovia/tri/fs"
	S "github.com/autovia/tri/structs"
)

const ISO8601UTCFormat = "2006-01-02T15:04:05.000Z"
const RFC822Format = "Mon, 2 Jan 2006 15:04:05 GMT"

func ListObjectsV2(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#ListObjectsV2 %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}

	contents, err := os.ReadDir(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}

	objects := []S.Object{}
	prefixes := []S.CommonPrefix{}
	for _, file := range contents {
		fileInfo, _ := file.Info()
		if !file.IsDir() {
			t := fileInfo.ModTime()
			etag, err := fs.Getxattr(filepath.Join(s3.Path, fileInfo.Name()), "etag")
			if err != nil {
				return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
			}
			objects = append(objects, S.Object{
				Key:          fileInfo.Name(),
				LastModified: t.Format(ISO8601UTCFormat),
				Size:         fileInfo.Size(),
				ETag:         "\"" + etag + "\"",
				StorageClass: "STANDARD"})
		} else {
			prefixes = append(prefixes, S.CommonPrefix{Prefix: fileInfo.Name() + "/"})
		}
	}

	listBucketResult := S.ListBucketResult{
		Name:           s3.Bucket,
		KeyCount:       len(objects),
		MaxKeys:        1000,
		IsTruncated:    false,
		Contents:       objects,
		CommonPrefixes: prefixes,
		Prefix:         s3.Key,
	}

	return S.RespondXML(w, http.StatusOK, listBucketResult)
}

func CopyObject(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#CopyObject: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	source := r.Header.Get("X-Amz-Copy-Source")
	sourcePath, err := url.QueryUnescape(source)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Bucket)
	}
	etag, err := fs.Getxattr(filepath.Join(s3.Mount, sourcePath), "etag")
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	if s3.Path != filepath.Join(s3.Mount, sourcePath) {
		sourceFile, err := os.Open(filepath.Join(s3.Mount, sourcePath))
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
		defer sourceFile.Close()

		if _, err := os.Stat(filepath.Dir(s3.Path)); os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(s3.Path), os.ModePerm)
			if err != nil {
				return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
			}
		}

		targetFile, err := os.Create(s3.Path)
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
		defer targetFile.Close()
		_, err = io.Copy(targetFile, sourceFile)
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}

		err = targetFile.Sync()
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}

		err = fs.Setxattr(s3.Path, "etag", etag)
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
	}

	stats, err := os.Stat(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	t := stats.ModTime()
	return S.RespondXML(w, http.StatusOK, S.CopyObjectResult{
		LastModified: t.Format(ISO8601UTCFormat),
		ETag:         "\"" + etag + "\"",
	})
}

func CreateMultipartUpload(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#CreateMultipartUpload: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); !os.IsNotExist(err) {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	if strings.HasSuffix(s3.Path, "/") {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", errors.New("path is a directory"), s3.Key)
	}

	uploadID := generate(50)
	metapath := filepath.Join(s3.Mount, Metadata, uploadID)
	if err := os.MkdirAll(metapath, os.ModePerm); err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	return S.RespondXML(w, http.StatusOK, S.InitiateMultipartUploadResponse{
		Bucket:   s3.Bucket,
		Key:      s3.Key,
		UploadID: uploadID,
	})
}

func CompleteMultipartUpload(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#CompleteMultipartUpload: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	metapath := filepath.Join(s3.Mount, Metadata, r.URL.Query().Get("uploadId"))
	if _, err := os.Stat(metapath); os.IsNotExist(err) {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	body, _ := io.ReadAll(r.Body)
	var cmu S.CompleteMultipartUpload
	err := xml.Unmarshal(body, &cmu)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	outFile, err := os.Create(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	defer outFile.Close()

	var n int64
	h := md5.New()
	for _, part := range cmu.PartNumbers {
		fn := filepath.Join(metapath, fmt.Sprintf("%v", part))
		etag, err := fs.Getxattr(fn, "etag")
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
		h.Write([]byte(etag))
		n++
		partFile, err := os.Open(fn)
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
		_, err = io.Copy(outFile, partFile)
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
		partFile.Close()
		err = os.Remove(fn)
		if err != nil {
			return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
		}
	}
	err = os.Remove(metapath)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	dashTag := append(h.Sum(nil), '-')
	finalETag := strconv.AppendInt(dashTag, n, 10)

	return S.RespondXML(w, http.StatusOK, S.CompleteMultipartUpload{
		Bucket: s3.Bucket,
		Key:    s3.Key,
		ETag:   fmt.Sprintf("\"%s\"", finalETag),
	})
}

func PutObject(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#PutObject: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); !os.IsNotExist(err) {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}

	if strings.HasSuffix(s3.Path, "/") {
		err := os.MkdirAll(s3.Path, os.ModePerm)
		if err != nil {
			return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
		}
		return S.Respond(w, http.StatusOK, nil, nil)
	}

	if _, err := os.Stat(filepath.Dir(s3.Path)); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(s3.Path), os.ModePerm)
		if err != nil {
			return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
		}
	}

	targetFile, err := os.Create(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}
	defer targetFile.Close()

	defer r.Body.Close()

	fileBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}
	hash := md5.Sum(fileBytes)
	etag := hex.EncodeToString(hash[:])

	_, err = targetFile.Write(fileBytes)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}

	err = fs.Setxattr(s3.Path, "etag", etag)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}

	return S.Respond(w, http.StatusOK, nil, nil)
}

func UploadPart(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#UploadPart: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	uploadID := filepath.Join(s3.Mount, Metadata, r.URL.Query().Get("uploadId"))
	partNumber := filepath.Join(uploadID, r.URL.Query().Get("partNumber"))
	if _, err := os.Stat(uploadID); os.IsNotExist(err) {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}

	targetFile, err := os.Create(partNumber)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}
	defer targetFile.Close()
	defer r.Body.Close()

	fileBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}
	hash := md5.Sum(fileBytes)
	etag := hex.EncodeToString(hash[:])

	_, err = targetFile.Write(fileBytes)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}

	err = fs.Setxattr(partNumber, "etag", etag)
	if err != nil {
		return S.RespondError(w, http.StatusBadRequest, "InternalError", err, s3.Key)
	}

	w.Header().Set("ETag", "\""+etag+"\"")
	return S.Respond(w, http.StatusOK, nil, nil)
}

func HeadObject(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#HeadObject: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, http.StatusNotFound, "NoSuchKey", err, s3.Key)
	}

	file, err := os.Stat(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	headers := make(map[string]string)
	t := file.ModTime()
	headers["Content-Length"] = fmt.Sprintf("%v", file.Size())
	headers["Last-Modified"] = t.Format(RFC822Format)
	etag, err := fs.Getxattr(s3.Path, "etag")
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	headers["ETag"] = "\"" + etag + "\""

	return S.Respond(w, http.StatusOK, headers, nil)
}

func GetObject(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#GetObject: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, 400, "NoSuchKey", err, s3.Key)
	}

	file, err := os.Open(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	stats, err := file.Stat()
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	if stats.IsDir() {
		return S.RespondError(w, 400, "NoSuchKey", err, s3.Key)
	}

	headers := make(map[string]string)
	t := stats.ModTime()
	headers["Content-Length"] = fmt.Sprintf("%v", stats.Size())
	headers["Last-Modified"] = t.Format(RFC822Format)

	return S.RespondFile(w, http.StatusOK, headers, file)
}

func ListObjectVersions(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#ListObjectVersions: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, 400, "NoSuchKey", err, s3.Key)
	}

	file, err := os.Open(s3.Path)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	stats, err := file.Stat()
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}
	t := stats.ModTime()
	return S.RespondXML(w, http.StatusOK, S.ListVersionsResult{
		Name:        s3.Bucket,
		Prefix:      s3.Key,
		MaxKeys:     1,
		IsTruncated: false,
		Version: []S.ObjectVersion{
			{
				Object: S.Object{
					Key:          s3.Key,
					LastModified: t.Format(ISO8601UTCFormat),
					ETag:         "xxx",
					Size:         stats.Size(),
					StorageClass: "STANDARD",
					Owner:        &S.Owner{ID: "id", DisplayName: "name"},
				},
				IsLatest:  true,
				VersionID: "xxx",
			},
		},
	})
}

func DeleteObject(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#DeleteObject: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	if _, err := os.Stat(s3.Path); os.IsNotExist(err) {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	if err := os.RemoveAll(s3.Path); err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	fs.CleanupEmptyDirs(s3.Path, filepath.Join(s3.Mount, s3.Bucket))

	headers := make(map[string]string)
	headers["Content-Length"] = "0"

	return S.Respond(w, http.StatusOK, headers, nil)
}

func DeleteObjects(w http.ResponseWriter, r *http.Request) error {
	log.Printf("#DeleteObjects: %v\n", r)
	s3 := r.Context().Value(S.Request{}).(S.Request)

	body, _ := io.ReadAll(r.Body)
	var delete S.Delete
	err := xml.Unmarshal(body, &delete)
	if err != nil {
		return S.RespondError(w, http.StatusInternalServerError, "InternalError", err, s3.Key)
	}

	objects := []S.DeletedObject{}
	errors := []S.DeleteError{}
	for _, file := range delete.Objects {
		path := filepath.Join(s3.Mount, s3.Bucket, file.Key)
		delErr := S.DeleteError{}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delErr = S.DeleteError{
				Code:    "NoSuchKey",
				Message: "NoSuchKey",
				Key:     file.Key,
			}
		}

		if err := os.RemoveAll(path); err != nil {
			delErr = S.DeleteError{
				Code:    "NoSuchKey",
				Message: "NoSuchKey",
				Key:     file.Key,
			}
		}

		fs.CleanupEmptyDirs(path, filepath.Join(s3.Mount, s3.Bucket))

		if delErr != (S.DeleteError{}) {
			errors = append(errors, delErr)
		} else {
			obj := S.DeletedObject{
				Key: file.Key,
			}
			objects = append(objects, obj)
		}
	}

	return S.RespondXML(w, http.StatusOK, S.DeleteObjectsResponse{
		DeletedObjects: objects,
		Errors:         errors,
	})
}
