package request

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

// Предел тела запроса: без него сервер читал бы в память сколько пришлют. Нужно больше — подними здесь.
const maxBodyBytes = 1 << 20

type Request interface {
	Header(name string) *string
	Query(name string) *string
	HasBody() bool
	JSONBody(dst any) error
	ParseHeaders(dst any) error
	ParseQuery(dst any) error
}

func NewRequest(r *http.Request) (Request, error) {
	return &impl{r: r}, nil
}

func ParseUrlParams(params map[string]string, dst any) error {
	return setFieldsFromMap(dst, func(tag string) (string, bool) {
		v, ok := params[tag]
		return v, ok
	})
}

type impl struct {
	r *http.Request
}

func (req *impl) Header(name string) *string {
	v := req.r.Header.Get(name)
	if v == "" {
		return nil
	}
	return &v
}

func (req *impl) Query(name string) *string {
	v := req.r.URL.Query().Get(name)
	if v == "" {
		return nil
	}
	return &v
}

func (req *impl) HasBody() bool {
	return req.r.Body != nil && req.r.ContentLength != 0
}

func (req *impl) JSONBody(dst any) error {
	defer req.r.Body.Close()
	return json.NewDecoder(http.MaxBytesReader(nil, req.r.Body, maxBodyBytes)).Decode(dst)
}

func (req *impl) ParseHeaders(dst any) error {
	return setFieldsFromMap(dst, func(tag string) (string, bool) {
		v := req.r.Header.Get(tag)
		return v, v != ""
	})
}

func (req *impl) ParseQuery(dst any) error {
	q := req.r.URL.Query()
	return setFieldsFromMap(dst, func(tag string) (string, bool) {
		v := q.Get(tag)
		return v, v != ""
	})
}

func setFieldsFromMap(dst any, lookup func(tag string) (string, bool)) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to struct")
	}
	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("form")
		if tag == "" {
			tag = f.Name
		}
		raw, ok := lookup(tag)
		if !ok {
			continue
		}
		if err := setField(v.Field(i), raw); err != nil {
			return fmt.Errorf("field %s: %w", f.Name, err)
		}
	}
	return nil
}

func setField(v reflect.Value, s string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Pointer:
		ptr := reflect.New(v.Type().Elem())
		if err := setField(ptr.Elem(), s); err != nil {
			return err
		}
		v.Set(ptr)
	}
	return nil
}
