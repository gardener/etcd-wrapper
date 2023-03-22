package util

import "net/http"

var (
	okResponseCodes = []int{http.StatusAccepted, http.StatusOK, http.StatusCreated}
)

// ResponseHasOKCode checks if the response is one the accepted OK responses
func ResponseHasOKCode(response *http.Response) bool {
	for _, code := range okResponseCodes {
		if code == response.StatusCode {
			return true
		}
	}
	return false
}
