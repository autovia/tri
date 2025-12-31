package structs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const emptyBody = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func (app *App) ValidSignatureV4(r *http.Request) (bool, *http.Request) {
	//log.Printf("---%s---", r.Header.Get("Authorization"))

	headers := authorizationHeader(r.Header, r.Host, r.Header.Get("Authorization"))
	if headers == nil {
		return false, nil
	}

	payload := []byte(r.Header.Get("X-Amz-Content-Sha256"))
	if r.Header.Get("X-Amz-Content-Sha256") == emptyBody {
		payload = []byte("")
	}

	query := r.URL.Query()
	canonicalRequest := canonicalRequest(r.Method, r.URL, query.Encode(), payload, headers)
	if canonicalRequest == "" {
		return false, nil
	}

	stringToSign := stringToSign(canonicalRequest, *app.AccessKey, headers)
	if stringToSign == "" {
		return false, nil
	}

	signature := signingKeySignature(*app.SecretKey, stringToSign, headers)
	if signature == "" {
		return false, nil
	}

	return signature == headers["Signature"], r
}

func authorizationHeader(header http.Header, host string, req string) map[string]string {
	authHeader, found := strings.CutPrefix(req, "AWS4-HMAC-SHA256")
	if !found {
		return nil
	}

	authHeaderSplit := strings.Split(strings.TrimSpace(authHeader), ",")
	if len(authHeaderSplit) != 3 {
		return nil
	}

	headers := make(map[string]string)
	for _, h := range authHeaderSplit {
		tuple := strings.Split(strings.TrimSpace(h), "=")
		if len(tuple) != 2 {
			return nil
		}
		headers[tuple[0]] = tuple[1]
	}

	headers["host"] = host
	for k, v := range header {
		headers[strings.ToLower(k)] = strings.Join(v, ",")
	}

	return headers
}

func getSignedHeaders(signedHeaders []string) string {
	var headers []string
	for _, k := range signedHeaders {
		headers = append(headers, strings.ToLower(k))
	}
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

func canonicalRequest(method string, requestURI *url.URL, rawQuery string, payload []byte, headers map[string]string) string {
	signedHeaders := strings.Split(headers["SignedHeaders"], ";")

	canonicalRequest := method + "\n"                                   // <HTTPMethod>
	canonicalRequest += requestURI.EscapedPath() + "\n"                 // <CanonicalURI>
	canonicalRequest += strings.ReplaceAll(rawQuery, "+", "%20") + "\n" // <CanonicalQueryString>

	for _, v := range signedHeaders {
		canonicalRequest += v + ":" + strings.Join(strings.Fields(headers[v]), " ") + "\n" // <CanonicalHeaders>
	}
	canonicalRequest += "\n"
	canonicalRequest += getSignedHeaders(signedHeaders) + "\n" // <SignedHeaders>
	if len(payload) == 0 {
		canonicalRequest += HexSHA256Hash(payload)
	} else {
		canonicalRequest += string(payload)
	}

	return canonicalRequest
}

func stringToSign(canonicalRequest string, accessKey string, headers map[string]string) string {
	scope, ok := strings.CutPrefix(headers["Credential"], accessKey+"/")
	if !ok {
		return ""
	}

	stringToSign := "AWS4-HMAC-SHA256\n"
	stringToSign += headers["x-amz-date"] + "\n"
	stringToSign += scope + "\n"
	stringToSign += HexSHA256Hash([]byte(canonicalRequest))

	return stringToSign
}

func signingKeySignature(secret string, stringToSign string, headers map[string]string) string {
	signPayload := strings.Split(headers["Credential"], "/")
	if len(signPayload) != 5 {
		return ""
	}

	dateKey := HmacSHA256([]byte("AWS4"+secret), []byte(signPayload[1]))
	dateRegionKey := HmacSHA256(dateKey, []byte(signPayload[2]))
	dateRegionServiceKey := HmacSHA256(dateRegionKey, []byte(signPayload[3]))
	signingKey := HmacSHA256(dateRegionServiceKey, []byte(signPayload[4]))

	return hex.EncodeToString(HmacSHA256(signingKey, []byte(stringToSign)))
}

func HexSHA256Hash(b []byte) string {
	bs := sha256.Sum256([]byte(b))
	return hex.EncodeToString(bs[:])
}

func HmacSHA256(key []byte, data []byte) []byte {
	hmac := hmac.New(sha256.New, key)
	hmac.Write(data)
	return hmac.Sum(nil)
}
