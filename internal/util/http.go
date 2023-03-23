package util

import "net/http"

var (
	okResponseCodes = []int{http.StatusAccepted, http.StatusOK, http.StatusCreated}
)

// ResponseHasOKCode checks if the response is one of the accepted OK responses.
func ResponseHasOKCode(response *http.Response) bool {
	for _, code := range okResponseCodes {
		if code == response.StatusCode {
			return true
		}
	}
	return false
}

// CloseResponseBody closes the response body if the response is not nil.
// As per https://pkg.go.dev/net/http - The http Client and Transport guarantee that Body is always
// non-nil, even on responses without a body or responses with a zero-length body. It is the caller's responsibility to
// close Body. The default HTTP client's Transport may not reuse HTTP/1.x "keep-alive" TCP connections if the Body is
// not read to completion and closed.
func CloseResponseBody(response *http.Response) {
	if response != nil {
		_ = response.Body.Close()
	}
}
