package util

import "net/http"

var (
	okResponseCodes = []int{http.StatusAccepted, http.StatusOK, http.StatusCreated}
)

func ResponseHasOKCode(response *http.Response) bool {
	for _, code := range okResponseCodes {
		if code == response.StatusCode {
			return true
		}
	}
	return false
}
