package common

import (
	"net/http"
)

func FormRequest(url string, headers map[string]string) (req *http.Request, err error) {
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	defaultHeaders := map[string]string{
		"Accept":     Accept,
		"user_agent": UserAgent,
	}
	for k, v := range defaultHeaders {
		req.Header.Set(k, v)
	}
	if headers != nil && len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	return
}
