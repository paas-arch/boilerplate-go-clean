package response

import (
	"encoding/json"
	"net/http"
)

type ResponseBuilder interface {
	SetCode(code int) ResponseBuilder
	SetHeader(key, value string) ResponseBuilder
	SetCookie(cookie *http.Cookie) ResponseBuilder
	SetBodyJsonData(data any) ResponseBuilder
	Send(w http.ResponseWriter) error
}

type builder struct {
	code    int
	headers [][2]string
	cookies []*http.Cookie
	body    any
}

func NewHandlerResponse() ResponseBuilder {
	return &builder{code: 200}
}

func (b *builder) SetCode(code int) ResponseBuilder {
	b.code = code
	return b
}

func (b *builder) SetHeader(key, value string) ResponseBuilder {
	b.headers = append(b.headers, [2]string{key, value})
	return b
}

func (b *builder) SetCookie(cookie *http.Cookie) ResponseBuilder {
	if cookie != nil {
		b.cookies = append(b.cookies, cookie)
	}
	return b
}

func (b *builder) SetBodyJsonData(data any) ResponseBuilder {
	b.body = data
	return b
}

func (b *builder) Send(w http.ResponseWriter) error {
	for _, h := range b.headers {
		w.Header().Set(h[0], h[1])
	}
	for _, c := range b.cookies {
		http.SetCookie(w, c)
	}
	w.WriteHeader(b.code)
	if b.body != nil {
		return json.NewEncoder(w).Encode(b.body)
	}
	return nil
}
