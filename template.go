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
		// Register all service annotation
		{
		api.Name("{{.ServiceAnnotation.Name}}")
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
		api.{{.Method}}("{{.Path}}", 
		{{- if .Annotation.Auth}}
			middleware.Authenticator(),middleware.Authorizer(),
		{{- end}}
		{{- if .Annotation.Operations}}
			middleware.Operations(),
		{{- end}}
		{{- if .Annotation.Customs}}
			{{- range .Annotation.Customs}}
				middleware.CustomMiddleware({{.}},
			{{- end}}
		{{- end}}
		_{{$svrType}}_{{.Name}}{{.Num}}_XHTTP_Handler(srv)).Name("{{$svrType}}-{{.Annotation.Name}}")
		{{- else}}
		api.{{.Method}}("{{.Path}}", _{{$svrType}}_{{.Name}}{{.Num}}_XHTTP_Handler(srv)).Name("{{$svrType}}-{{.Name}}.{{.Num}}-XHTTP_Handler")
		{{- end}}
		
		
		{{- end}}
	})
}

{{range .Methods}}
{{- if not (eq .Comments "")}}
// {{.Comments}}
{{- end}}
func _{{$svrType}}_{{.Name}}{{.Num}}_XHTTP_Handler(srv {{$svrType}}XHTTPServer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var in {{.Request}}
		{{- if .HasBody}}
		if err := binding.BindBody(c, &in{{.Body}}); err != nil {
			return apistate.Error[any]().WithError(err).Send(c)
		}
		
		{{- if not (eq .Body "")}}
		if err := binding.BindQuery(c, &in); err != nil {
			return apistate.Error[any]().WithError(err).Send(c)
		}
		{{- end}}
		{{- else if not .HasParams}}
		if err := binding.BindQuery(c, &in); err != nil {
			return apistate.Error[any]().WithError(err).Send(c)
		}
		{{- end}}
		{{- if .HasParams}}
		if err := binding.BindParams(c, &in); err != nil {
			return apistate.Error[any]().WithError(err).Send(c)
		}
		{{- end}}
		{{- if .Annotation}}
		{{- if .Annotation.Validate}}
		if err := in.Validate(); err != nil {
			return apistate.InvalidError[any]().WithError(err).Send(c)
		}
		{{- end}}
		{{- end}}
		ctx := transport.NewFiberContext(context.Background(),c)
		reply, err := srv.{{.Name}}(ctx, &in)
		if err != nil {
			return apistate.Error[any]().WithError(err).Send(c)
		}
		return apistate.Success[*{{.Reply}}]().WithData(reply).Send(c)
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
	Name       string
	Auth       bool
	Operations bool
	Validate   bool
	Customs    []string
}
