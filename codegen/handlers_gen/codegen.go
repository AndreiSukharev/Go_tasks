package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

var (
	tmpServerHTTP = template.Must(template.New("ServeHTTP").Parse(`
func (srv *{{.StructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{range .Apis}}
	case {{.Path}}:
		srv.wrapper{{.WrapperName}}(w, r)
	{{end}}
	default:
		sendError(w, "unknown method", http.StatusNotFound)
	}
}
`))
	tmpWrapper = template.Must(template.New("Wrapper").Parse(`
func (srv *{{.StructName}}) wrapper{{.WrapperName}}(w http.ResponseWriter, r *http.Request) {
	if !checkMethod({{.Api.AllowedMethod}}, r.Method, w) {
		return
	}
	if !checkAuth({{.Api.Auth}}, w, r.Header.Get("X-Auth")) {
		return
	}
	paramsMap := getParams(r)
	param := {{.Api.ParamName}}{}
	if !param.unpack{{.WrapperName}}(paramsMap, w) {
		return
	}
	resApi, err := srv.{{.WrapperName}}(r.Context(), param)
	if isErrorApi(err, w) {
		return
	}
	responseStruct := ResponseStruct{
		"error":    "",
		"response": resApi,
	}
	response, _ := json.Marshal(responseStruct)
	w.Write(response)
}
`))

	tmpExtFunc = `
func checkEnum(status string, enum []string) string {
	for _, item := range enum {
		if item == status {
			return ""
		}
	}
	err := "["
	size := len(enum)
	for i := range enum {
		err += enum[i]
		if i + 1 < size {
			err += ", "
		}
	}
	err += "]"
	return err
}

func sendError(w http.ResponseWriter, err string, status int) {
	response := fmt.Sprintf(` + "`" + `{"error": "%s"}` + "`" + `, err)
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func isErrorApi(err error, w http.ResponseWriter) bool {
	switch err.(type) {
	case ApiError:
		apiError := err.(ApiError)
		sendError(w, err.Error(), apiError.HTTPStatus)
		return true
	case error:
		sendError(w, err.Error(), http.StatusInternalServerError)
		return true
	}
	return false
}

func checkAuth(auth bool, w http.ResponseWriter, xAuth string) bool {
	if !auth {
		return true
	}
	if xAuth == "100500" {
		return true
	}
	sendError(w, "unauthorized", http.StatusForbidden)
	return false
}

func checkMethod(allowedMethod string, method string, w http.ResponseWriter) bool {
	if allowedMethod == "" {
		return true
	}
	if allowedMethod == method {
		return true
	}
	sendError(w, "bad method", http.StatusNotAcceptable)
	return false
}
func getParams(r *http.Request) url.Values {
	var params url.Values

	switch r.Method {
	case http.MethodGet:
		params = r.URL.Query()
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		params = r.Form
	}
	return params
}
`
)

type ApiStruct struct {
	WrapperName   string
	Path          string
	ParamName     string
	AllowedMethod string
	Auth          string
}

type validateStruct struct {
	FieldName           string
	FieldType           string
	FieldNameCapitalize string
	Required            bool
	Paramname           string
	Enum                []string
	DefaultName         string
	Validate            string
	Min                 [2]int
	Max                 [2]int
}

type CommonStruct struct {
	Api         ApiStruct
	Validation  []validateStruct
	WrapperName string
	StructName  string
}

type generalStruct map[string]map[string]*CommonStruct

var codeGen = generalStruct{}

//go build handlers_gen/* && ./codegen Api.go api_handlers.go
func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	//ast.Print(fset, node)
	out, _ := os.Create(os.Args[2])
	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, `import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)`)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, "type ResponseStruct map[string]interface{}")
	fmt.Fprintln(out, tmpExtFunc)
	// create apigen
	for _, f := range node.Decls {
		manageFuncDecl(f)
	}
	for _, f := range node.Decls {
		manageGenDecl(f)
	}
	generateCode(out)
	//fmt.Println(codeGen)

}

func generateCode(out *os.File) {
	generateServer(out)
	for structName, structData := range codeGen {
		for wrapperName, common := range structData {
			common.StructName = structName
			common.WrapperName = wrapperName
			tmpWrapper.Execute(out, common)
			generateUnpack(out, common)
			//tmpUnpack.Execute(out, common)
		}
	}
}

var (
	tmpFunc = template.Must(template.New("tmpFunc").Parse(`
func (in *{{.Api.ParamName}}) unpack{{.WrapperName}}(params url.Values, w http.ResponseWriter) bool {
`))
	tmpStart = template.Must(template.New("tmpStart").Parse(`
	{{.FieldName}}Temp, ok := params["{{.FieldName}}"]
`))
	tmpRequired = template.Must(template.New("tmpRequired").Parse(`
	if !ok {
		sendError(w, "{{.FieldName}} must me not empty", http.StatusBadRequest)
		return false
	}
`))
	tmpDefault = template.Must(template.New("tmpRequired").Parse(`
	var {{.FieldName}} {{.FieldType}}
	if !ok {
		{{.FieldName}} = "{{.DefaultName}}"
	} else {
		{{.FieldName}} = {{.FieldName}}Temp[0]
	}	
`))
	tmpEnum = template.Must(template.New("tmpEnum").Parse(`
	if ok {
		err := checkEnum({{.FieldName}}, []string{ {{range .Enum}} "{{.}}",{{ end }}})
		if err != "" {
				sendError(w, "{{.FieldName}} must be one of " + err, http.StatusBadRequest)
				return false
		}
	}
`))
	tmpFieldName = template.Must(template.New("tmpFieldName").Parse(`
	{{ if (eq .FieldType "string") }}
	{{.FieldName}} := {{.FieldName}}Temp[0]
	{{ else }}
	{{.FieldName}}, err := strconv.Atoi({{.FieldName}}Temp[0])
	if err != nil {
		sendError(w, "{{.FieldName}} must be int", http.StatusBadRequest)
		return false
	}
{{ end }}`))

	tmpMinStr = template.Must(template.New("tmpMinStr").Parse(`
	if len({{.FieldName}}) < {{index .Min 1}} {
		sendError(w, "{{.FieldName}} len must be >= {{index .Min 1}}", http.StatusBadRequest)
		return false
	}
`))
	tmpMaxStr = template.Must(template.New("tmpMaxStr").Parse(`
	if len({{.FieldName}}) > {{index .Max 1}} {
		sendError(w, "{{.FieldName}} len must be <= {{index .Max 1}}", http.StatusBadRequest)
		return false
	}
`))
	tmpMinInt = template.Must(template.New("tmpMinInt").Parse(`
	if {{.FieldName}} < {{index .Min 1}} {
		sendError(w, "{{.FieldName}} must be >= {{index .Min 1}}", http.StatusBadRequest)
		return false
	}
`))
	tmpMaxInt = template.Must(template.New("tmpMaxInt").Parse(`
	if {{.FieldName}} > {{index .Max 1}} {
		sendError(w, "{{.FieldName}} must be <= {{index .Max 1}}", http.StatusBadRequest)
		return false
	}
`))
	tmpIn = template.Must(template.New("tmpIn").Parse(`
	in.{{.FieldNameCapitalize}} = {{.FieldName}}
`))
)

func generateUnpack(out *os.File, common *CommonStruct) {
	tmpFunc.Execute(out, common)

	for _, val := range common.Validation {

		tmpStart.Execute(out, val)

		if val.Required {
			tmpRequired.Execute(out, val)
		}

		//tmpFieldName.Execute(out, val)

		if val.FieldType == "string" {
			manageStr(out, val)
		} else {
			manageInt(out, val)
		}

		tmpIn.Execute(out, val)

	}
	fmt.Fprintf(out, `
	return true
}`)

}

func manageStr(out *os.File, val validateStruct) {
	if val.DefaultName != "" {
		tmpDefault.Execute(out, val)
	} else {
		tmpFieldName.Execute(out, val)
	}
	if len(val.Enum) != 0 {
		tmpEnum.Execute(out, val)
	}
	if val.Min[0] == 1 {
			tmpMinStr.Execute(out, val)
		}
	if val.Max[0] == 1 {
			tmpMaxStr.Execute(out, val)
	}
}

func manageInt(out *os.File, val validateStruct) {
	tmpFieldName.Execute(out, val)

	if val.Min[0] == 1 {
		tmpMinInt.Execute(out, val)
	}
	if val.Max[0] == 1 {
		tmpMaxInt.Execute(out, val)
	}
}

func generateServer(out *os.File) {
	type tmp struct {
		StructName string
		Apis       []*ApiStruct
	}
	for structName, structData := range codeGen {
		apis := make([]*ApiStruct, 0)
		for _, common := range structData {
			apis = append(apis, &common.Api)
		}
		templateStruct := tmp{
			StructName: structName,
			Apis:       apis,
		}
		err := tmpServerHTTP.Execute(out, templateStruct)
		if err != nil {
			fmt.Println("error", err)
		}
	}

}
func manageGenDecl(f ast.Decl) {
	genDecl, ok := f.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range genDecl.Specs {
		currType, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		ParamName := currType.Name.Name
		mainKey, wrapperKey := checkStructNameAndWrapperName(ParamName)
		if mainKey == "" {
			continue
		}
		currStruct, ok := currType.Type.(*ast.StructType)
		if !ok {
			continue
		}
		valStruct := getValidateStruct(currStruct)
		codeGen[mainKey][wrapperKey].Validation = valStruct
	}
}

func getValidateStruct(currStruct *ast.StructType) []validateStruct {
	valListStruct := make([]validateStruct, 0)

	for _, field := range currStruct.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
		validatorString := tag.Get("apivalidator")
		if validatorString == "" {
			continue
		}
		validatorSlice := strings.Split(validatorString, ",")
		fieldType, _ := field.Type.(*ast.Ident)
		valStruct := validateStruct{
			FieldName:           strings.ToLower(field.Names[0].Name),
			FieldType:           fieldType.Name,
			FieldNameCapitalize: strings.Title(field.Names[0].Name),
		}
		for _, item := range validatorSlice {
			if item == "required" {
				valStruct.Required = true
				continue
			}
			splitedItem := strings.Split(item, "=")
			key := splitedItem[0]
			val := splitedItem[1]
			switch key {
			case "paramname":
				valStruct.Paramname = val
			case "min":
				minVal, _ := strconv.Atoi(val)
				valStruct.Min[0] = 1
				valStruct.Min[1] = minVal
			case "max":
				maxVal, _ := strconv.Atoi(val)
				valStruct.Max[0] = 1
				valStruct.Max[1] = maxVal
			case "default":
				valStruct.DefaultName = val
			case "enum":
				valStruct.Enum = strings.Split(val, "|")
			}
		}
		if valStruct.Paramname != "" {
			valStruct.FieldName = valStruct.Paramname
		}
		valListStruct = append(valListStruct, valStruct)
	}
	return valListStruct
}

func checkStructNameAndWrapperName(ParamName string) (string, string) {
	for mainKey, mainVal := range codeGen {
		for wrapperKey, wrapperVal := range mainVal {
			if wrapperVal.Api.ParamName == ParamName {
				return mainKey, wrapperKey
			}
		}
	}
	return "", ""
}

func manageFuncDecl(f ast.Decl) {
	funcDecl, funcOk := f.(*ast.FuncDecl)
	if !funcOk {
		return
	}
	commentGroup := funcDecl.Doc
	if commentGroup == nil {
		return
	}
	comment := commentGroup.List[0].Text
	hasApigen := strings.HasPrefix(comment, "// apigen:api")
	if !hasApigen {
		return
	}
	structName := getStructName(funcDecl)
	if structName == "" {
		return
	}
	ParamName := funcDecl.Type.Params.List[1].Type.(*ast.Ident).Name
	WrapperName := funcDecl.Name.Name
	Api := fillApiStruct(structName, comment, ParamName)
	makeCodeGen(WrapperName, structName, Api)
}

func makeCodeGen(WrapperName, structName string, Api *ApiStruct) {
	Api.WrapperName = WrapperName
	comStruct := CommonStruct{Api: *Api,}
	if _, ok := codeGen[structName]; ok {
		codeGen[structName][WrapperName] = &comStruct
	} else {
		m := map[string]*CommonStruct{}
		m[WrapperName] = &comStruct
		codeGen[structName] = m
	}
}

func fillApiStruct(structName, comment, ParamName string) *ApiStruct {
	apigenCutComment := strings.Replace(comment, "// apigen:api {", "", 1)
	endCutComment := strings.Replace(apigenCutComment, "}", "", 1)
	splitedComment := strings.Split(endCutComment, ",")
	Api := ApiStruct{}
	for _, item := range splitedComment {
		splitedItem := strings.Split(item, ": ")
		val := splitedItem[1]
		key := strings.Replace(splitedItem[0], " ", "", 1)
		switch key {
		case `"url"`:
			Api.Path = val
		case `"auth"`:
			Api.Auth = val
		case `"method"`:
			Api.AllowedMethod = val
		}
	}
	if Api.AllowedMethod == "" {
		Api.AllowedMethod = `""`
	}
	Api.ParamName = ParamName
	return &Api
}

func getStructName(funcDecl *ast.FuncDecl) string {
	StarExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	structName := StarExpr.X.(*ast.Ident).Name
	return structName
}
