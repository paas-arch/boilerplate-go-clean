// Code generator: reads api/openapi.yaml, writes per-endpoint packages.
// Usage: go run ./internal/util/webserver/codegen -spec api/openapi.yaml -out internal/generated/api/schema -module app
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ---------- OpenAPI spec structs ----------

type Spec struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Components ComponentsSpec      `yaml:"components"`
}

type ComponentsSpec struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

type PathItem struct {
	Get    *Operation `yaml:"get"`
	Post   *Operation `yaml:"post"`
	Put    *Operation `yaml:"put"`
	Delete *Operation `yaml:"delete"`
	Patch  *Operation `yaml:"patch"`
}

type Operation struct {
	OperationID string                  `yaml:"operationId"`
	Parameters  []Parameter             `yaml:"parameters"`
	RequestBody *RequestBodySpec        `yaml:"requestBody"`
	Responses   map[string]ResponseSpec `yaml:"responses"`
}

type Parameter struct {
	Name     string  `yaml:"name"`
	In       string  `yaml:"in"`
	Required bool    `yaml:"required"`
	Schema   *Schema `yaml:"schema"`
}

type RequestBodySpec struct {
	Required bool                    `yaml:"required"`
	Content  map[string]MediaTypeObj `yaml:"content"`
}

type MediaTypeObj struct {
	Schema *Schema `yaml:"schema"`
}

type ResponseSpec struct {
	Description string                  `yaml:"description"`
	Content     map[string]MediaTypeObj `yaml:"content"`
}

type Schema struct {
	Type                 string             `yaml:"type"`
	Format               string             `yaml:"format"`
	Ref                  string             `yaml:"$ref"`
	Properties           map[string]*Schema `yaml:"properties"`
	Items                *Schema            `yaml:"items"`
	AdditionalProperties *Schema            `yaml:"additionalProperties"`
	Required             []string           `yaml:"required"`
	Enum                 []string           `yaml:"enum"`
	Default              interface{}        `yaml:"default"`
	Description          string             `yaml:"description"`
}

// ---------- naming helpers ----------

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnake(s string) string {
	snake := matchFirstCap.ReplaceAllString(s, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func toPascal(s string) string {
	parts := regexp.MustCompile(`[_\- ]+`).Split(s, -1)
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

func statusName(code string) string {
	m := map[string]string{
		"200": "OK",
		"201": "Created",
		"202": "Accepted",
		"204": "NoContent",
		"400": "BadRequest",
		"401": "Unauthorized",
		"403": "Forbidden",
		"404": "NotFound",
		"409": "Conflict",
		"422": "UnprocessableEntity",
		"500": "InternalServerError",
	}
	if n, ok := m[code]; ok {
		return n
	}
	return "Status" + code
}

func refName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// ---------- Go type resolution ----------

func goType(s *Schema, inComponents bool) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		name := refName(s.Ref)
		if inComponents {
			return name
		}
		return "components." + name
	}
	switch s.Type {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]" + goType(s.Items, inComponents)
	case "object":
		if s.AdditionalProperties != nil {
			return "map[string]" + goType(s.AdditionalProperties, inComponents)
		}
		return "map[string]any"
	}
	return "any"
}

// ---------- template data ----------

type HandleTmplData struct {
	Pkg     string
	Path    string
	Method  string
	HasBody bool
	Module  string
}

type ParamField struct {
	GoName  string
	JsonTag string
	FormTag string
	GoType  string
	Comment string
	Ptr     bool // pointer type (optional param)
}

type RequestTmplData struct {
	Pkg         string
	HasQuery    bool
	QueryFields []ParamField
	HasPath     bool
	PathFields  []ParamField
	HasBody     bool
	BodyType    string
	Module      string
}

type ResponseMethod struct {
	Name     string
	Code     int
	DataType string
}

type ResponseTmplData struct {
	Pkg     string
	Methods []ResponseMethod
	Module  string
}

type DtoTmplData struct {
	Pkg                string
	HasQuery           bool
	QueryFields        []ParamField
	HasPath            bool
	PathFields         []ParamField
	HasInlineSchemas   bool
	InlineSchemas      []CompSchema
	DtoNeedsComponents bool
}

// ---------- component schema ----------

type CompField struct {
	GoName    string
	JsonTag   string
	FormTag   string
	GoType    string
	Comment   string
	OmitEmpty bool
}

type EnumValue struct {
	ConstName string
	Value     string
}

type EnumDef struct {
	TypeName    string
	ParentType  string // e.g. "Feed" — used for naming
	FieldName   string // e.g. "Status"
	Values      []EnumValue
	AllowedList string // "\"v1\", \"v2\""
}

type CompSchema struct {
	GoName      string
	Fields      []CompField
	Enums       []EnumDef
	RequiredSet map[string]bool
	RefFields   []string // field GoNames that reference other structs
	ArrayFields []string // field GoNames that are arrays
}

type ComponentsTmplData struct {
	Module  string
	Schemas []CompSchema
}

// ---------- templates ----------

const handleTmpl = `// Code generated by cupfeed-codegen. DO NOT EDIT.
package {{.Pkg}}

import (
	context "context"
	http "net/http"

	hctx "{{.Module}}/internal/util/webserver/hctx"
	request "{{.Module}}/internal/util/webserver/request"
	response "{{.Module}}/internal/util/webserver/response"
	router "{{.Module}}/internal/util/webserver/router"
)

type Handler interface {
	Handle(context.Context, {{if .HasBody}}JsonRequest{{else}}Parameters{{end}}, Response) error
}

const (
	HandlerPath   = "{{.Path}}"
	HandlerMethod = "{{.Method}}"
)

func Handle(m Handler) func(rt router.ApiHandlerRouter, internalErr router.InternalErrHandler, badReq router.BadRequestErrHandler) {
	return func(rt router.ApiHandlerRouter, internalErr router.InternalErrHandler, badReq router.BadRequestErrHandler) {
		rt.ApiHandler(HandlerMethod, HandlerPath, func(w http.ResponseWriter, r *http.Request) {
			_ = hctx.MakeRequestContext(r)
			resp := response.NewHandlerResponse()

			req, err := request.NewRequest(r)
			if err != nil {
				badReq(err, resp)
				_ = resp.Send(w)
				return
			}

			{{if .HasBody -}}
			params, err := NewJsonRequest(req, rt.URLParameters(r))
			{{- else -}}
			params, err := NewParameters(req, rt.URLParameters(r))
			{{- end}}
			if err != nil {
				badReq(err, resp)
				_ = resp.Send(w)
				return
			}

			if err := m.Handle(r.Context(), params, NewResponse(resp)); err != nil {
				internalErr(err, resp)
				_ = resp.Send(w)
				return
			}

			_ = resp.Send(w)
		})
	}
}
`

const requestTmpl = `// Code generated by cupfeed-codegen. DO NOT EDIT.
package {{.Pkg}}

import (
	unsafe "unsafe"

	request "{{.Module}}/internal/util/webserver/request"
	validation "{{.Module}}/internal/util/webserver/validation"
	{{if .NeedsComponents -}}
	components "{{.Module}}/internal/generated/api/schema/components"
	{{- end}}
)

type Parameters interface {
	{{if .HasQuery}}Query() (Query, error){{end}}
	{{if .HasPath}}Path() (Path, error){{end}}
}

func NewParameters(req request.Request, urlParams map[string]string) (Parameters, error) {
	return &params{
		req: req,
		urlParams: urlParams,
	}, nil
}

type params struct {
	req       request.Request
	urlParams map[string]string
}

{{if .HasQuery}}
func (p *params) Query() (Query, error) {
	var param Query
	if err := p.req.ParseQuery(&param); err != nil {
		return param, err
	}
	param.FillDefault_(map[unsafe.Pointer]bool{})
	{
		val := param
		if err := val.Validate_([]unsafe.Pointer{}); err != nil {
			return param, err
		}
	}
	return param, nil
}
{{end}}

{{if .HasPath}}
func (p *params) Path() (Path, error) {
	var param Path
	if err := request.ParseUrlParams(p.urlParams, &param); err != nil {
		return param, err
	}
	param.FillDefault_(map[unsafe.Pointer]bool{})
	{
		val := param
		if err := val.Validate_([]unsafe.Pointer{}); err != nil {
			return param, err
		}
	}
	return param, nil
}
{{end}}

{{if .HasBody}}
type JsonRequest interface {
	Parameters
	DTO() ({{.BodyType}}, error)
}

func NewJsonRequest(req request.Request, urlParams map[string]string) (JsonRequest, error) {
	params, err := NewParameters(req, urlParams)
	if err != nil {
		return nil, err
	}
	if !req.HasBody() {
		return nil, validation.NewError("request body is required")
	}
	return &jsonRequest{
		Parameters: params,
		req: req,
	}, nil
}

type jsonRequest struct {
	Parameters
	req request.Request
}

func (r *jsonRequest) DTO() ({{.BodyType}}, error) {
	var dto {{.BodyType}}
	if err := r.req.JSONBody(&dto); err != nil {
		return dto, err
	}
	dto.FillDefault_(map[unsafe.Pointer]bool{})
	{
		val := dto
		if err := val.Validate_([]unsafe.Pointer{}); err != nil {
			return dto, err
		}
	}
	return dto, nil
}
{{end}}

var _ = unsafe.Pointer(nil)
var _ = validation.NewError
`

const responseTmpl = `// Code generated by cupfeed-codegen. DO NOT EDIT.
package {{.Pkg}}

import (
	unsafe "unsafe"

	response "{{.Module}}/internal/util/webserver/response"
	{{if .NeedsComponents -}}
	components "{{.Module}}/internal/generated/api/schema/components"
	{{- end}}
)

type Response interface {
	{{range .Methods}}{{.Name}}(data {{.DataType}}) response.ResponseBuilder
	{{end}}
}

type handlerResponse struct {
	resp response.ResponseBuilder
}

func NewResponse(resp response.ResponseBuilder) Response {
	return &handlerResponse{resp: resp}
}

{{range .Methods}}
func (h *handlerResponse) {{.Name}}(data {{.DataType}}) response.ResponseBuilder {
	h.resp = h.resp.SetCode({{.Code}})
	data.FillDefault_(map[unsafe.Pointer]bool{})
	h.resp = h.resp.
		SetHeader("Content-Type", "application/json").
		SetBodyJsonData(data)
	return h.resp
}
{{end}}

var _ = unsafe.Pointer(nil)
{{if .NeedsComponents}}var _ = components.ComponentsPackage{{end}}
`

const dtoTmpl = `// Code generated by cupfeed-codegen. DO NOT EDIT.
package {{.Pkg}}

import (
	"errors"
	fmt "fmt"
	unsafe "unsafe"

	tools "{{.Module}}/internal/util/webserver/tools"
	validation "{{.Module}}/internal/util/webserver/validation"
	{{if .DtoNeedsComponents -}}
	components "{{.Module}}/internal/generated/api/schema/components"
	{{- end}}
)

{{if .HasQuery}}
type Query struct {
{{range .QueryFields}}	{{.GoName}} {{if .Ptr}}*{{end}}{{.GoType}} ` + "`" + `json:"{{.JsonTag}}" form:"{{.FormTag}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}}

{{range .QueryFields}}
func (d *Query) Get{{.GoName}}() {{if .Ptr}}*{{end}}{{.GoType}} { return d.{{.GoName}} }
func (d *Query) Set{{.GoName}}(v {{if .Ptr}}*{{end}}{{.GoType}}) { d.{{.GoName}} = v }
{{end}}

func (d *Query) Validate_(visited []unsafe.Pointer) error {
	if d == nil { return nil }
	for _, p := range visited {
		if p == unsafe.Pointer(d) { return validation.NewError("struct is recursive") }
	}
	return nil
}

func (d *Query) FillDefault_(visited map[unsafe.Pointer]bool) {
	if d == nil || visited[unsafe.Pointer(d)] { return }
	visited[unsafe.Pointer(d)] = true
}
{{end}}

{{if .HasPath}}
type Path struct {
{{range .PathFields}}	{{.GoName}} {{if .Ptr}}*{{end}}{{.GoType}} ` + "`" + `json:"{{.JsonTag}}" form:"{{.FormTag}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}}

{{range .PathFields}}
func (d *Path) Get{{.GoName}}() {{if .Ptr}}*{{end}}{{.GoType}} { return d.{{.GoName}} }
func (d *Path) Set{{.GoName}}(v {{if .Ptr}}*{{end}}{{.GoType}}) { d.{{.GoName}} = v }
{{end}}

func (d *Path) Validate_(visited []unsafe.Pointer) error {
	if d == nil { return nil }
	for _, p := range visited {
		if p == unsafe.Pointer(d) { return validation.NewError("struct is recursive") }
	}
	return nil
}

func (d *Path) FillDefault_(visited map[unsafe.Pointer]bool) {
	if d == nil || visited[unsafe.Pointer(d)] { return }
	visited[unsafe.Pointer(d)] = true
}
{{end}}

{{range .InlineSchemas}}{{$schema := .}}
{{range .Enums}}{{$enum := .}}
type {{$enum.TypeName}} struct {
	initialized bool
	value       string
}

func (e *{{$enum.TypeName}}) Value() string { return e.value }

var allowedValues{{$enum.TypeName}} = map[string]struct{}{
{{range $enum.Values}}	"{{.Value}}": {},
{{end}}}

{{range $enum.Values}}
func {{$enum.TypeName}}_{{.ConstName}}() *{{$enum.TypeName}} {
	return &{{$enum.TypeName}}{
		initialized: true,
		value: "{{.Value}}",
	}
}
{{end}}

func (e *{{$enum.TypeName}}) UnmarshalJSON(data []byte) error {
	var wrapper *string
	if err := tools.JSON.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	if wrapper == nil {
		return nil
	}
	if _, ok := allowedValues{{$enum.TypeName}}[*wrapper]; !ok {
		return fmt.Errorf("unknown enum value %v", *wrapper)
	}
	e.initialized = true
	e.value = *wrapper
	return nil
}

func (e {{$enum.TypeName}}) MarshalJSON() ([]byte, error) {
	if !e.initialized {
		return nil, errors.New("uninitialized enum")
	}
	if _, ok := allowedValues{{$enum.TypeName}}[e.value]; !ok {
		return nil, fmt.Errorf("unknown enum value %v", e.value)
	}
	return tools.JSON.Marshal(e.value)
}

func (e *{{$enum.TypeName}}) DecodeForm(data []byte) error {
	value := string(data)
	if _, ok := allowedValues{{$enum.TypeName}}[value]; !ok {
		return fmt.Errorf("unknown enum value %v", value)
	}
	e.initialized = true
	e.value = value
	return nil
}

func Parse{{$enum.TypeName}}(value string) (*{{$enum.TypeName}}, error) {
	if _, ok := allowedValues{{$enum.TypeName}}[value]; !ok {
		return nil, fmt.Errorf("unknown enum value %v", value)
	}
	return &{{$enum.TypeName}}{
		initialized: true,
		value: value,
	}, nil
}
{{end}}

type {{$schema.GoName}} struct {
{{range $schema.Fields}}	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JsonTag}}{{if .OmitEmpty}},omitempty{{end}}" form:"{{.FormTag}}{{if .OmitEmpty}},omitempty{{end}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}}

{{range $schema.Fields}}
func (d *{{$schema.GoName}}) Get{{.GoName}}() {{.GoType}} { return d.{{.GoName}} }
func (d *{{$schema.GoName}}) Set{{.GoName}}(v {{.GoType}}) { d.{{.GoName}} = v }
{{end}}

func (d *{{$schema.GoName}}) Validate_(visited []unsafe.Pointer) error {
	if d == nil {
		return nil
	}
	for _, p := range visited {
		if p == unsafe.Pointer(d) {
			return validation.NewError("struct is recursive")
		}
	}
	visited = append(visited, unsafe.Pointer(d))
	return nil
}

func (d *{{$schema.GoName}}) FillDefault_(visited map[unsafe.Pointer]bool) {
	if d == nil {
		return
	}
	if visited[unsafe.Pointer(d)] {
		return
	}
	visited[unsafe.Pointer(d)] = true
}
{{end}}

var _ = errors.New
var _ = fmt.Sprintf
var _ = unsafe.Pointer(nil)
var _ = tools.JSON
var _ = validation.NewError
`

const componentsTmpl = `// Code generated by cupfeed-codegen. DO NOT EDIT.
package components

import (
	"errors"
	fmt "fmt"
	unsafe "unsafe"

	tools "{{.Module}}/internal/util/webserver/tools"
	validation "{{.Module}}/internal/util/webserver/validation"
)

// ComponentsPackage is used by generated operation packages to import this package.
const ComponentsPackage = ""

{{range .Schemas}}{{$schema := .}}
{{range .Enums}}{{$enum := .}}
type {{$enum.TypeName}} struct {
	initialized bool
	value       string
}

func (e *{{$enum.TypeName}}) Value() string { return e.value }

var allowedValues{{$enum.TypeName}} = map[string]struct{}{
{{range $enum.Values}}	"{{.Value}}": {},
{{end}}}

{{range $enum.Values}}
func {{$enum.TypeName}}_{{.ConstName}}() *{{$enum.TypeName}} {
	return &{{$enum.TypeName}}{
		initialized: true,
		value: "{{.Value}}",
	}
}
{{end}}

func (e *{{$enum.TypeName}}) UnmarshalJSON(data []byte) error {
	var wrapper *string
	if err := tools.JSON.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	if wrapper == nil {
		return nil
	}
	if _, ok := allowedValues{{$enum.TypeName}}[*wrapper]; !ok {
		return fmt.Errorf("unknown enum value %v", *wrapper)
	}
	e.initialized = true
	e.value = *wrapper
	return nil
}

func (e {{$enum.TypeName}}) MarshalJSON() ([]byte, error) {
	if !e.initialized {
		return nil, errors.New("uninitialized enum")
	}
	if _, ok := allowedValues{{$enum.TypeName}}[e.value]; !ok {
		return nil, fmt.Errorf("unknown enum value %v", e.value)
	}
	return tools.JSON.Marshal(e.value)
}

func (e *{{$enum.TypeName}}) DecodeForm(data []byte) error {
	value := string(data)
	if _, ok := allowedValues{{$enum.TypeName}}[value]; !ok {
		return fmt.Errorf("unknown enum value %v", value)
	}
	e.initialized = true
	e.value = value
	return nil
}

func Parse{{$enum.TypeName}}(value string) (*{{$enum.TypeName}}, error) {
	if _, ok := allowedValues{{$enum.TypeName}}[value]; !ok {
		return nil, fmt.Errorf("unknown enum value %v", value)
	}
	return &{{$enum.TypeName}}{
		initialized: true,
		value: value,
	}, nil
}
{{end}}

type {{$schema.GoName}} struct {
{{range $schema.Fields}}	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JsonTag}}{{if .OmitEmpty}},omitempty{{end}}" form:"{{.FormTag}}{{if .OmitEmpty}},omitempty{{end}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{end}}}

{{range $schema.Fields}}
func (d *{{$schema.GoName}}) Get{{.GoName}}() {{.GoType}} { return d.{{.GoName}} }
func (d *{{$schema.GoName}}) Set{{.GoName}}(v {{.GoType}}) { d.{{.GoName}} = v }
{{end}}

func (d *{{$schema.GoName}}) Validate_(visited []unsafe.Pointer) error {
	if d == nil {
		return nil
	}
	for _, p := range visited {
		if p == unsafe.Pointer(d) {
			return validation.NewError("struct is recursive")
		}
	}
	visited = append(visited, unsafe.Pointer(d))
	return nil
}

func (d *{{$schema.GoName}}) FillDefault_(visited map[unsafe.Pointer]bool) {
	if d == nil {
		return
	}
	if visited[unsafe.Pointer(d)] {
		return
	}
	visited[unsafe.Pointer(d)] = true
}
{{end}}

var _ = errors.New
var _ = fmt.Sprintf
var _ = unsafe.Pointer(nil)
var _ = tools.JSON
var _ = validation.NewError
`

// ---------- generation logic ----------

type Generator struct {
	spec   *Spec
	outDir string
	module string
}

func (g *Generator) run() {
	g.generateComponents()

	// Sort paths for deterministic output
	paths := make([]string, 0, len(g.spec.Paths))
	for p := range g.spec.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := g.spec.Paths[path]
		ops := map[string]*Operation{
			"GET":    item.Get,
			"POST":   item.Post,
			"PUT":    item.Put,
			"DELETE": item.Delete,
			"PATCH":  item.Patch,
		}
		for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
			op := ops[method]
			if op == nil {
				continue
			}
			g.generateOperation(op, path, method)
		}
	}
}

func (g *Generator) generateOperation(op *Operation, path, method string) {
	pkg := toSnake(op.OperationID)
	dir := filepath.Join(g.outDir, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", dir, err)
	}

	// Collect parameters
	var queryFields, pathFields []ParamField
	for _, p := range op.Parameters {
		f := paramField(p)
		switch p.In {
		case "query":
			queryFields = append(queryFields, f)
		case "path":
			pathFields = append(pathFields, f)
		}
	}

	// Body type
	bodyType := ""
	var inlineSchemas []CompSchema
	if op.RequestBody != nil {
		if mt, ok := op.RequestBody.Content["application/json"]; ok && mt.Schema != nil {
			if isInlineObject(mt.Schema) {
				bodyType = "Request"
				inlineSchemas = append(inlineSchemas, buildInlineSchema("Request", mt.Schema))
			} else {
				bodyType = goType(mt.Schema, false)
			}
		}
	}
	hasBody := bodyType != ""

	// Response methods
	var respMethods []ResponseMethod
	codes := sortedKeys(op.Responses)
	for _, code := range codes {
		resp := op.Responses[code]
		dt := ""
		if mt, ok := resp.Content["application/json"]; ok && mt.Schema != nil {
			if isInlineObject(mt.Schema) {
				localName := statusName(code) + "Response"
				dt = localName
				inlineSchemas = append(inlineSchemas, buildInlineSchema(localName, mt.Schema))
			} else {
				dt = goType(mt.Schema, false)
			}
		}
		if dt == "" {
			continue
		}
		codeInt, err := strconv.Atoi(code)
		if err != nil {
			log.Fatalf("operation %s has invalid response status %q: %v", op.OperationID, code, err)
		}
		respMethods = append(respMethods, ResponseMethod{
			Name:     statusName(code),
			Code:     codeInt,
			DataType: dt,
		})
	}

	needsComponents := false
	for _, m := range respMethods {
		if strings.HasPrefix(m.DataType, "components.") {
			needsComponents = true
		}
	}
	if strings.HasPrefix(bodyType, "components.") {
		needsComponents = true
	}

	dtoNeedsComponents := false
	for _, cs := range inlineSchemas {
		for _, f := range cs.Fields {
			if strings.HasPrefix(f.GoType, "components.") {
				dtoNeedsComponents = true
			}
		}
	}

	g.writeFile(filepath.Join(dir, "handle.go"), handleTmpl, HandleTmplData{
		Pkg:     pkg,
		Path:    path,
		Method:  method,
		HasBody: hasBody,
		Module:  g.module,
	})

	g.writeFile(filepath.Join(dir, "request.go"), requestTmpl, map[string]any{
		"Pkg":             pkg,
		"HasQuery":        len(queryFields) > 0,
		"QueryFields":     queryFields,
		"HasPath":         len(pathFields) > 0,
		"PathFields":      pathFields,
		"HasBody":         hasBody,
		"BodyType":        bodyType,
		"Module":          g.module,
		"NeedsComponents": strings.HasPrefix(bodyType, "components."),
	})

	g.writeFile(filepath.Join(dir, "response.go"), responseTmpl, map[string]any{
		"Pkg":             pkg,
		"Methods":         respMethods,
		"Module":          g.module,
		"NeedsComponents": needsComponents,
	})

	g.writeFile(filepath.Join(dir, "dto.go"), dtoTmpl, map[string]any{
		"Pkg":                pkg,
		"HasQuery":           len(queryFields) > 0,
		"QueryFields":        queryFields,
		"HasPath":            len(pathFields) > 0,
		"PathFields":         pathFields,
		"HasInlineSchemas":   len(inlineSchemas) > 0,
		"InlineSchemas":      inlineSchemas,
		"DtoNeedsComponents": dtoNeedsComponents,
		"Module":             g.module,
	})
}

func (g *Generator) generateComponents() {
	dir := filepath.Join(g.outDir, "components")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", dir, err)
	}

	names := sortedKeys(g.spec.Components.Schemas)
	schemas := make([]CompSchema, 0, len(names))
	for _, name := range names {
		s := g.spec.Components.Schemas[name]
		schemas = append(schemas, buildCompSchema(name, &s))
	}

	g.writeFile(filepath.Join(dir, "dto.go"), componentsTmpl, ComponentsTmplData{
		Module:  g.module,
		Schemas: schemas,
	})
}

func buildCompSchema(name string, s *Schema) CompSchema {
	cs := CompSchema{
		GoName:      name,
		RequiredSet: make(map[string]bool),
	}
	for _, r := range s.Required {
		cs.RequiredSet[r] = true
	}

	propNames := sortedKeys(s.Properties)
	for _, propName := range propNames {
		prop := s.Properties[propName]
		goName := toPascal(propName)
		gt := goTypeComponents(prop)
		omit := !cs.RequiredSet[propName]
		// Указатель отличает отсутствующее поле от пустого объекта или явного false.
		if omit && (prop.Ref != "" || prop.Type == "boolean") {
			gt = "*" + gt
		}

		cs.Fields = append(cs.Fields, CompField{
			GoName:    goName,
			JsonTag:   propName,
			FormTag:   propName,
			GoType:    gt,
			Comment:   prop.Description,
			OmitEmpty: omit,
		})

		// Check if we need to generate enum
		if len(prop.Enum) > 0 {
			typeName := "Enum" + name + goName
			var vals []EnumValue
			for _, v := range prop.Enum {
				vals = append(vals, EnumValue{
					ConstName: toPascal(strings.ReplaceAll(v, "-", "_")),
					Value:     v,
				})
			}
			allowed := make([]string, len(prop.Enum))
			for i, v := range prop.Enum {
				allowed[i] = fmt.Sprintf("%q", v)
			}
			cs.Enums = append(cs.Enums, EnumDef{
				TypeName:    typeName,
				ParentType:  name,
				FieldName:   goName,
				Values:      vals,
				AllowedList: strings.Join(allowed, ", "),
			})
		}
	}

	return cs
}

func goTypeComponents(s *Schema) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		return refName(s.Ref)
	}
	if len(s.Enum) > 0 {
		return "string" // placeholder; actual type is enum, handled separately
	}
	switch s.Type {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]" + goTypeComponents(s.Items)
	case "object":
		if s.AdditionalProperties != nil {
			return "map[string]" + goTypeComponents(s.AdditionalProperties)
		}
		return "map[string]any"
	}
	return "any"
}

func isInlineObject(s *Schema) bool {
	return s != nil && s.Ref == "" && (s.Type == "object" || len(s.Properties) > 0)
}

// goTypeInline resolves Go types for fields inside inline operation schemas.
// Unlike goTypeComponents, it prefixes $ref names with "components.".
func goTypeInline(s *Schema) string {
	if s == nil {
		return "any"
	}
	if s.Ref != "" {
		return "components." + refName(s.Ref)
	}
	if len(s.Enum) > 0 {
		return "string"
	}
	switch s.Type {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]" + goTypeInline(s.Items)
	case "object":
		if s.AdditionalProperties != nil {
			return "map[string]" + goTypeInline(s.AdditionalProperties)
		}
		return "map[string]any"
	}
	return "any"
}

func buildInlineSchema(name string, s *Schema) CompSchema {
	cs := CompSchema{
		GoName:      name,
		RequiredSet: make(map[string]bool),
	}
	for _, r := range s.Required {
		cs.RequiredSet[r] = true
	}
	propNames := sortedKeys(s.Properties)
	for _, propName := range propNames {
		prop := s.Properties[propName]
		goName := toPascal(propName)
		gt := goTypeInline(prop)
		omit := !cs.RequiredSet[propName]
		// Указатель отличает отсутствующее поле от пустого объекта или явного false.
		if omit && (prop.Ref != "" || prop.Type == "boolean") {
			gt = "*" + gt
		}
		cs.Fields = append(cs.Fields, CompField{
			GoName:    goName,
			JsonTag:   propName,
			FormTag:   propName,
			GoType:    gt,
			Comment:   prop.Description,
			OmitEmpty: omit,
		})
		if len(prop.Enum) > 0 {
			typeName := "Enum" + name + goName
			var vals []EnumValue
			for _, v := range prop.Enum {
				vals = append(vals, EnumValue{
					ConstName: toPascal(strings.ReplaceAll(v, "-", "_")),
					Value:     v,
				})
			}
			allowed := make([]string, len(prop.Enum))
			for i, v := range prop.Enum {
				allowed[i] = fmt.Sprintf("%q", v)
			}
			cs.Enums = append(cs.Enums, EnumDef{
				TypeName:    typeName,
				ParentType:  name,
				FieldName:   goName,
				Values:      vals,
				AllowedList: strings.Join(allowed, ", "),
			})
		}
	}
	return cs
}

func paramField(p Parameter) ParamField {
	gt := "string"
	ptr := !p.Required
	if p.Schema != nil {
		switch p.Schema.Type {
		case "integer":
			gt = "int64"
		case "number":
			gt = "float64"
		case "boolean":
			gt = "bool"
		}
	}
	return ParamField{
		GoName:  toPascal(p.Name),
		JsonTag: p.Name,
		FormTag: p.Name,
		GoType:  gt,
		Ptr:     ptr,
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------- file writer ----------

func (g *Generator) writeFile(path, tmplStr string, data any) {
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}).Parse(tmplStr))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("template %s: %v", path, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		_ = os.WriteFile(path, buf.Bytes(), 0o644)
		log.Printf("format error in %s: %v\n--- source ---\n%s", path, err, buf.String())
		return
	}

	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
	fmt.Printf("wrote %s\n", path)
}

// ---------- main ----------

func main() {
	specPath := flag.String("spec", "api/openapi.yaml", "path to openapi.yaml")
	outDir := flag.String("out", "internal/generated/api/schema", "output directory")
	module := flag.String("module", "app.internal", "Go module path")
	flag.Parse()

	data, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatalf("read spec: %v", err)
	}

	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		log.Fatalf("parse spec: %v", err)
	}

	g := &Generator{
		spec:   &spec,
		outDir: *outDir,
		module: *module,
	}
	g.run()
}
