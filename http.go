package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// The name of the package that contains the generated code.
	contextPackage       = protogen.GoImportPath("context")
	transportHTTPPackage = protogen.GoImportPath("github.com/go-kratos/kratos/v2/transport/xhttp")
	transportPackage     = protogen.GoImportPath("github.com/go-kratos/kratos/v2/transport")
	bindingPackage       = protogen.GoImportPath("github.com/go-kratos/kratos/v2/transport/xhttp/binding")
	middlewarePackage    = protogen.GoImportPath("github.com/go-kratos/kratos/v2/middleware")
	fiberPackage         = protogen.GoImportPath("github.com/gofiber/fiber/v2")
	apistatePackage      = protogen.GoImportPath("github.com/go-kratos/kratos/v2/transport/xhttp/apistate")
)

var methodSets = make(map[string]int)

// generateFile generates a _xhttp.pb.go file containing kratos errors definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File, omitempty bool) *protogen.GeneratedFile {
	if len(file.Services) == 0 || (omitempty && !hasHTTPRule(file.Services)) {
		return nil
	}
	filename := file.GeneratedFilenamePrefix + "_xhttp.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-go-xhttp. DO NOT EDIT.")
	g.P("// versions:")
	g.P(fmt.Sprintf("// protoc-gen-go-xhttp %s", release))
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()
	g.P("import fiber \"github.com/gofiber/fiber/v2\"")
	generateFileContent(gen, file, g, omitempty)
	return g
}

// generateFileContent generates the kratos errors definitions, excluding the package statement.
func generateFileContent(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, omitempty bool) {
	if len(file.Services) == 0 {
		return
	}
	g.P("// This is a compile-time assertion to ensure that this generated file")
	g.P("// is compatible with the kratos package it is being compiled against.")
	g.P("var _ = new(", contextPackage.Ident("Context"), ")")
	g.P("var _ = ", bindingPackage.Ident("BindBody"))
	g.P("const _ = ", transportHTTPPackage.Ident("SupportPackageIsVersion1"))
	g.P("const _ = ", middlewarePackage.Ident("SupportPackageIsVersion1"))
	g.P("const _ = ", transportPackage.Ident("KindXHTTP"))
	g.P("var _ = new(", apistatePackage.Ident("Resp[any]"), ")")
	g.P()

	for _, service := range file.Services {
		genService(gen, file, g, service, omitempty)
	}
}

func genService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service, omitempty bool) {
	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P("//")
		g.P(deprecationComment)
	}
	// HTTP Server.
	sd := &serviceDesc{
		ServiceType: service.GoName,
		ServiceName: string(service.Desc.FullName()),
		Metadata:    file.Desc.Path(),
	}
	anno, err := buildAnnotation(service.Comments.Leading.String())
	if err != nil {
		log.Panicf("buildAnnotation: %v", err)
		return
	}
	sd.ServiceAnnotation = anno
	if anno != nil && anno.Comment != "" {
		sd.ServiceComments = anno.Comment
	}
	if len(anno.Path) != 0 {
		sd.ServicePrefix = "api/" + anno.Path
	} else {
		sd.ServicePrefix = buildPrefix(string(service.Desc.FullName()))
	}
	if anno != nil && len(anno.Name) == 0 {
		anno.Name = sd.ServiceType + "-XHTTPServer"
	}
	for _, method := range service.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue
		}
		rule, ok := proto.GetExtension(method.Desc.Options(), annotations.E_Http).(*annotations.HttpRule)
		if rule != nil && ok {
			for _, bind := range rule.AdditionalBindings {
				sd.Methods = append(sd.Methods, buildHTTPRule(g, method, bind))
			}
			sd.Methods = append(sd.Methods, buildHTTPRule(g, method, rule))
		} else if !omitempty {
			path := fmt.Sprintf("/%s/%s", service.Desc.FullName(), method.Desc.Name())
			sd.Methods = append(sd.Methods, buildMethodDesc(g, method, "Post", path))
		}
	}
	if len(sd.Methods) != 0 {
		g.P(sd.execute())
	}
}

func hasHTTPRule(services []*protogen.Service) bool {
	for _, service := range services {
		for _, method := range service.Methods {
			if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
				continue
			}
			rule, ok := proto.GetExtension(method.Desc.Options(), annotations.E_Http).(*annotations.HttpRule)
			if rule != nil && ok {
				return true
			}
		}
	}
	return false
}

func buildHTTPRule(g *protogen.GeneratedFile, m *protogen.Method, rule *annotations.HttpRule) *methodDesc {
	var (
		path         string
		method       string
		body         string
		responseBody string
	)
	switch pattern := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		path = pattern.Get
		method = "Get"
	case *annotations.HttpRule_Put:
		path = pattern.Put
		method = "Put"
	case *annotations.HttpRule_Post:
		path = pattern.Post
		method = "Post"
	case *annotations.HttpRule_Delete:
		path = pattern.Delete
		method = "Delete"
	case *annotations.HttpRule_Patch:
		path = pattern.Patch
		method = "Patch"
	case *annotations.HttpRule_Custom:
		path = pattern.Custom.Path
		method = pattern.Custom.Kind
	}
	body = rule.Body
	responseBody = rule.ResponseBody
	md := buildMethodDesc(g, m, method, path)
	if hasPathParams(path) {
		md.HasParams = true
		md.Path = buildPath(path)
	}
	anno, err := buildAnnotation(m.Comments.Leading.String())
	if err != nil {
		log.Panicf("buildAnnotation error: %v", err)
	}
	md.Annotation = anno
	if anno != nil && anno.Comment != "" {
		md.Comments = anno.Comment
	}
	if anno != nil && len(anno.Name) == 0 {
		anno.Name = fmt.Sprintf("%s.%d-XHTTP_Handler", m.GoName, md.Num)
	}
	if method == "Get" || method == "Delete" {
		if body != "" {
			_, _ = fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: %s %s body should not be declared.\n", method, path)
		}
		md.HasBody = false
	} else if body == "*" {
		md.HasBody = true
		md.Body = ""
	} else if body != "" {
		md.HasBody = true
		md.Body = "." + camelCaseVars(body)
	} else {
		md.HasBody = false
		_, _ = fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: %s %s does not declare a body.\n", method, path)
	}
	if responseBody == "*" {
		md.ResponseBody = ""
	} else if responseBody != "" {
		md.ResponseBody = "." + camelCaseVars(responseBody)
	}
	return md
}

func buildMethodDesc(g *protogen.GeneratedFile, m *protogen.Method, method, path string) *methodDesc {
	defer func() { methodSets[m.GoName]++ }()

	md := &methodDesc{
		Name:    m.GoName,
		Num:     methodSets[m.GoName],
		Request: g.QualifiedGoIdent(m.Input.GoIdent),
		Reply:   g.QualifiedGoIdent(m.Output.GoIdent),
		Path:    path,
		Method:  method,
	}
	return md
}

func hasPathParams(path string) bool {
	return regexp.MustCompile(`(?i){([a-z\.0-9_\s]*)}`).MatchString(path)
}

func buildPath(path string) string {
	return regexp.MustCompile(`(?i){([a-z\.0-9_\s]*)}`).ReplaceAllStringFunc(path, func(s string) string {
		return ":" + strings.TrimSpace(s[1:len(s)-1])
	})
}

func buildAnnotation(comment string) (anno *annotation, err error) {
	if len(comment) == 0 {
		return
	}
	anno = &annotation{}
	comments := strings.Split(comment, "\n")
	for _, s := range comments {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}
		s = strings.TrimSpace(s[2:])
		if !strings.HasPrefix(s, "@") {
			anno.Comment = s
			continue
		}
		sub := regexp.MustCompile(`(?i)@([a-z_]*)\s*(.*)`).FindStringSubmatch(strings.ReplaceAll(s, " ", ""))
		switch sub[1] {
		case "Path":
			if len(sub[2]) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "\u001B[31mWARN\u001B[m: the path annotation is empty,use the default value.")
				continue
			}
			anno.Path = sub[2][2 : len(sub[2])-2]
		case "Name":
			if len(sub[2]) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "\u001B[31mWARN\u001B[m: the name annotation is empty,use the default value.")
				continue
			}
			anno.Name = sub[2][2 : len(sub[2])-2]
		case "Auth":
			anno.Auth = true
		case "Operations":
			anno.Operations = true
		case "Validate":
			anno.Validate = true
		case "Customs":
			if len(sub[2]) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "\u001B[31mWARN\u001B[m: the custom annotation is empty.")
				continue
			}
			anno.Customs = strings.Split(sub[2][2:len(sub[2])-2], ",")
		default:
			_, _ = fmt.Fprintf(os.Stderr, "\u001B[31mWARN\u001B[m: %s is not a valid annotation.\n", sub[1])
			err = fmt.Errorf("%s is not a valid annotation", sub[1])
			return nil, err
		}
	}
	return
}

func buildPrefix(serviceName string) string {
	prefix := strings.ReplaceAll(serviceName, ".", "/")
	return strings.ToLower(prefix)
}

func camelCaseVars(s string) string {
	vars := make([]string, 0)
	subs := strings.Split(s, ".")
	for _, sub := range subs {
		vars = append(vars, camelCase(sub))
	}
	return strings.Join(vars, ".")
}

// camelCase returns the CamelCased name.
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
// In short, _my_field_name_2 becomes XMyFieldName_2.
func camelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

const deprecationComment = "// Deprecated: Do not use."
