package main

import (
	"bytes"
	"strings"
	"text/template"
)

var httpTemplate = `
{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}

// {{.ServiceComments}}
type {{.ServiceType}}XHTTPServer interface {
{{- range .MethodSets}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
{{- end}}
}

func Register{{.ServiceType}}XHTTPServer(s *xhttp.Server, srv {{.ServiceType}}XHTTPServer) {
	s.Route(func(r fiber.Router) {
		api := r.Group("{{.ServicePrefix}}")
	    {{- if .ServiceAnnotation}}
		{
		{{- if .ServiceAnnotation.Auth}}
		api.Use(middleware.Authenticator(),middleware.Authorizer())
		{{- end}}
		{{- if .ServiceAnnotation.Operations}}
		api.Use(middleware.Operations())
		{{- end}}
		{{- if .ServiceAnnotation.Operations}}
		api.Use(middleware.Validator())
		{{- end}}
		{{- range .ServiceAnnotation.Customs}}
		api.Use(middleware.CustomMiddleware({{.}}))
		{{- end}}
		}
		{{- end}}
	
		{{- range .Methods}}
		{{- if .Annotation}}
		api.{{.Method}}("{{.Path}}", _{{$svrType}}_{{.Name}}{{.Num}}_XHTTP_Handler(srv),
		{{- if .Annotation.Auth}}
			middleware.Authenticator(),middleware.Authorizer(),
		{{- end}}
		{{- if .Annotation.Operations}}
			middleware.Operations(),
		{{- end}}
		{{- if .Annotation.Validate}}
			middleware.Validator(),
		{{- end}}
		{{- if .Annotation.Customs}}
			{{- range .Annotation.Customs}}
				middleware.CustomMiddleware({{.}},
			{{- end}}
		{{- end}}
		)
		{{- else}}
		api.{{.Method}}("{{.Path}}", _{{$svrType}}_{{.Name}}{{.Num}}_XHTTP_Handler(srv))
		{{- end}}
		
		
		{{- end}}
	})
}

{{range .Methods}}
// {{.Comments}}
func _{{$svrType}}_{{.Name}}{{.Num}}_XHTTP_Handler(srv {{$svrType}}XHTTPServer) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var in {{.Request}}
		{{- if .HasBody}}
		if err := binding.BindBody(ctx, &in{{.Body}}); err != nil {
			return err
		}
		
		{{- if not (eq .Body "")}}
		if err := binding.BindQuery(ctx, &in); err != nil {
			return err
		}
		{{- end}}
		{{- else if not .HasParams}}
		if err := binding.BindQuery(ctx, &in); err != nil {
			return err
		}
		{{- end}}
		{{- if .HasParams}}
		if err := binding.BindParams(ctx, &in); err != nil {
			return err
		}
		{{- end}}
		
		reply, err := srv.{{.Name}}(ctx.Context(), &in)
		if err != nil {
			return err
		}
		return ApiState.Success().SendData(ctx, reply)
	}
}
{{end}}
`

type serviceDesc struct {
	ServiceType       string // Greeter
	ServiceName       string // helloworld.Greeter
	ServicePrefix     string // /api/helloworld/greeter
	ServiceComments   string // Greeter service
	ServiceAnnotation *annotation
	Metadata          string // api/helloworld/helloworld.proto
	Methods           []*methodDesc
	MethodSets        map[string]*methodDesc
}

type methodDesc struct {
	// method
	Name       string
	Num        int
	Request    string
	Reply      string
	Comments   string
	Annotation *annotation
	// http_rule
	Path         string
	Method       string
	HasBody      bool
	HasParams    bool
	Body         string
	ResponseBody string
}

func (s *serviceDesc) execute() string {
	s.MethodSets = make(map[string]*methodDesc)
	for _, m := range s.Methods {
		s.MethodSets[m.Name] = m
	}
	buf := new(bytes.Buffer)
	tmpl, err := template.New("http").Parse(strings.TrimSpace(httpTemplate))
	if err != nil {
		panic(err)
	}
	if err := tmpl.Execute(buf, s); err != nil {
		panic(err)
	}
	return strings.Trim(buf.String(), "\r\n")
}

type annotation struct {
	Comment    string
	Path       string
	Auth       bool
	Operations bool
	Validate   bool
	Customs    []string
}
