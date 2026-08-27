package http

import "net/http"

type ResponseWriter struct {
	Status        int
	BytesCount    int
	wrapped       http.ResponseWriter
	headerWritten bool
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		Status:  http.StatusOK,
		wrapped: w,
	}
}

func (rw *ResponseWriter) Header() http.Header {
	return rw.wrapped.Header()
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.wrapped.WriteHeader(status)

	if !rw.headerWritten {
		rw.Status = status
		rw.headerWritten = true
	}
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	rw.headerWritten = true

	n, err := rw.wrapped.Write(b)
	rw.BytesCount += n

	return n, err
}

func (rw *ResponseWriter) Unwrap() http.ResponseWriter {
	return rw.wrapped
}
