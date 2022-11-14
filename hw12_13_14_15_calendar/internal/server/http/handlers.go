package internalhttp

import "net/http"

type HelloHandler struct{}

func (h HelloHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	_, err := writer.Write([]byte("Hello, World!"))
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
}
