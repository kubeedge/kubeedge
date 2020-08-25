package swagger

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type SwaggerService struct {
	config            Config
	apiDeclarationMap *ApiDeclarationList
}

func newSwaggerService(config Config) *SwaggerService {
	sws := &SwaggerService{
		config:            config,
		apiDeclarationMap: new(ApiDeclarationList)}

	// Build all ApiDeclarations
	for _, each := range config.WebServices {
		rootPath := each.RootPath()
		// skip the api service itself
		if rootPath != config.ApiPath {
			sws.apiDeclarationMap.Put(each.RootPath(), sws.composeDeclaration(each, each.RootPath()))
		}
	}

	// if specified then call the PostBuilderHandler
	if config.PostBuildHandler != nil {
		config.PostBuildHandler(sws.apiDeclarationMap)
	}
	return sws
}

// LogInfo is the function that is called when this package needs to log. It defaults to log.Printf
var LogInfo = func(format string, v ...interface{}) {
	// use the restful package-wide logger
	log.Printf(format, v...)
}

// InstallSwaggerService add the WebService that provides the API documentation of all services
// conform the Swagger documentation specifcation. (https://github.com/wordnik/swagger-core/wiki).
func InstallSwaggerService(aSwaggerConfig Config) {
	RegisterSwaggerService(aSwaggerConfig, restful.DefaultContainer)
}
func (sws SwaggerService) WriteToFile() {
	for _, listing := range sws.apiDeclarationMap.List {
		if sws.config.FileStyle == "json" {
			list, err := json.Marshal(listing)
			if err == nil {
				err1 := ioutil.WriteFile(sws.config.SwaggerFilePath, list, 0644)
				if err1 != nil {
					log.Printf("fail to write to file", err1)
				}
			}
		} else {
			list, err := yaml.Marshal(listing)
			if err == nil {
				err1 := ioutil.WriteFile(sws.config.SwaggerFilePath, list, 0644)
				if err1 != nil {
					log.Printf("fail to write to file", err1)
				}
			}
		}

	}
}

// RegisterSwaggerService add the WebService that provides the API documentation of all services
// conform the Swagger documentation specifcation. (https://github.com/wordnik/swagger-core/wiki).
func RegisterSwaggerService(config Config, wsContainer *restful.Container) *SwaggerService {
	sws := newSwaggerService(config)
	sws.WriteToFile()
	if sws.config.OpenService == true {
		ws := new(restful.WebService)
		ws.Path(config.ApiPath)
		ws.Produces(restful.MIME_JSON)
		if config.DisableCORS {
			ws.Filter(enableCORS)
		}
		ws.Route(ws.GET("/").To(sws.getListing))
		ws.Route(ws.GET("/{a}").To(sws.getDeclarations))
		ws.Route(ws.GET("/{a}/{b}").To(sws.getDeclarations))
		ws.Route(ws.GET("/{a}/{b}/{c}").To(sws.getDeclarations))
		ws.Route(ws.GET("/{a}/{b}/{c}/{d}").To(sws.getDeclarations))
		ws.Route(ws.GET("/{a}/{b}/{c}/{d}/{e}").To(sws.getDeclarations))
		ws.Route(ws.GET("/{a}/{b}/{c}/{d}/{e}/{f}").To(sws.getDeclarations))
		ws.Route(ws.GET("/{a}/{b}/{c}/{d}/{e}/{f}/{g}").To(sws.getDeclarations))
		LogInfo("[restful/swagger] listing is available at %v%v", config.WebServicesUrl, config.ApiPath)
		wsContainer.Add(ws)

		// Check paths for UI serving
		if config.StaticHandler == nil && config.SwaggerFilePath != "" && config.SwaggerPath != "" {
			swaggerPathSlash := config.SwaggerPath
			// path must end with slash /
			if "/" != config.SwaggerPath[len(config.SwaggerPath)-1:] {
				LogInfo("[restful/swagger] use corrected SwaggerPath ; must end with slash (/)")
				swaggerPathSlash += "/"
			}

			LogInfo("[restful/swagger] %v%v is mapped to folder %v", config.WebServicesUrl, swaggerPathSlash, config.SwaggerFilePath)
			wsContainer.Handle(swaggerPathSlash, http.StripPrefix(swaggerPathSlash, http.FileServer(http.Dir(config.SwaggerFilePath))))

			//if we define a custom static handler use it
		} else if config.StaticHandler != nil && config.SwaggerPath != "" {
			swaggerPathSlash := config.SwaggerPath
			// path must end with slash /
			if "/" != config.SwaggerPath[len(config.SwaggerPath)-1:] {
				LogInfo("[restful/swagger] use corrected SwaggerFilePath ; must end with slash (/)")
				swaggerPathSlash += "/"

			}
			LogInfo("[restful/swagger] %v%v is mapped to custom Handler %T", config.WebServicesUrl, swaggerPathSlash, config.StaticHandler)
			wsContainer.Handle(swaggerPathSlash, config.StaticHandler)

		} else {
			LogInfo("[restful/swagger] Swagger(File)Path is empty ; no UI is served")
		}
	}
	return sws
}

func staticPathFromRoute(r restful.Route) string {
	static := r.Path
	bracket := strings.Index(static, "{")
	if bracket <= 1 {
		// result cannot be empty
		return static
	}
	if bracket != -1 {
		static = r.Path[:bracket]
	}
	if strings.HasSuffix(static, "/") {
		return static[:len(static)-1]
	} else {
		return static
	}
}

func enableCORS(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if origin := req.HeaderParameter(restful.HEADER_Origin); origin != "" {
		// prevent duplicate header
		if len(resp.Header().Get(restful.HEADER_AccessControlAllowOrigin)) == 0 {
			resp.AddHeader(restful.HEADER_AccessControlAllowOrigin, origin)
		}
	}
	chain.ProcessFilter(req, resp)
}

func (sws SwaggerService) getListing(req *restful.Request, resp *restful.Response) {
	for _, listing := range sws.apiDeclarationMap.List {
		resp.WriteAsJson(listing)
	}
}

func (sws SwaggerService) getDeclarations(req *restful.Request, resp *restful.Response) {
	decl, ok := sws.produceDeclarations(composeRootPath(req))
	if !ok {
		resp.WriteErrorString(http.StatusNotFound, "ApiDeclaration not found")
		return
	}
	// unless WebServicesUrl is given
	if len(sws.config.WebServicesUrl) == 0 {
		// update base path from the actual request
		// TODO how to detect https? assume http for now
		var host string
		// X-Forwarded-Host or Host or Request.Host
		hostvalues, ok := req.Request.Header["X-Forwarded-Host"] // apache specific?
		if !ok || len(hostvalues) == 0 {
			forwarded, ok := req.Request.Header["Host"] // without reverse-proxy
			if !ok || len(forwarded) == 0 {
				// fallback to Host field
				host = req.Request.Host
			} else {
				host = forwarded[0]
			}
		} else {
			host = hostvalues[0]
		}
		// inspect Referer for the scheme (http vs https)
		scheme := "http"
		if referer := req.Request.Header["Referer"]; len(referer) > 0 {
			if strings.HasPrefix(referer[0], "https") {
				scheme = "https"
			}
		}
		decl.BasePath = fmt.Sprintf("%s://%s", scheme, host)
	}
	resp.WriteAsJson(decl)
}

func (sws SwaggerService) produceDeclarations(route string) (*APIDefinition, bool) {
	decl, ok := sws.apiDeclarationMap.At(route)
	if !ok {
		return nil, false
	}
	decl.BasePath = sws.config.WebServicesUrl
	return &decl, true
}

func composeResponses(route restful.Route, decl *APIDefinition, config *Config) (res map[string]Response) {
	response := make(map[string]Response)
	if route.ResponseErrors == nil {
		return res
	}
	// sort by code
	codes := sort.IntSlice{}
	for code := range route.ResponseErrors {
		codes = append(codes, code)
	}
	codes.Sort()
	for _, code := range codes {
		each := route.ResponseErrors[code]
		message := Response{
			Description: each.Message}
		if each.Model != nil {
			message.Schema = &Items{}
			st := reflect.TypeOf(each.Model)
			isCollection, st := detectCollectionType(st)
			modelName := modelBuilder{}.keyFrom(st)
			if !isCollection {
				if st.Kind() == reflect.Struct {
					message.Schema.Ref = getModelName(modelName)
					modelBuilder{Definitions: &decl.Definitions, Config: config}.addModel(st, "")
				} else if st.Kind() == reflect.Map {
					st = st.Elem()
					if st.Kind() == reflect.Struct {
						message.Schema.Type = "object"
						message.Schema.AdditionalProperties = &Items{}
						modelName = modelBuilder{}.keyFrom(st)
						message.Schema.AdditionalProperties.Ref = getModelName(modelName)
						modelBuilder{Definitions: &decl.Definitions, Config: config}.addModel(st, "")
					} else {
						message.Schema.Type = "object"
						message.Schema.AdditionalProperties = &Items{}
						modelName = modelBuilder{}.keyFrom(st)
						message.Schema.AdditionalProperties.Type = getOtherName(modelName)
						if getOtherName(modelName) == "integer" || getOtherName(modelName) == "number" {
							message.Schema.AdditionalProperties.Format = getFormat(modelName)
						}
					}
				} else {
					message.Schema.Type = getOtherName(modelName)
					if getOtherName(modelName) == "integer" || getOtherName(modelName) == "number" {
						message.Schema.Format = getFormat(modelName)
					}
				}
			} else {
				if st.Kind() == reflect.Struct {
					message.Schema.Type = "array"
					message.Schema.Items = &Items{}
					message.Schema.Items.Ref = getModelName(modelName)
					modelBuilder{Definitions: &decl.Definitions, Config: config}.addModel(st, "")
				} else {
					message.Schema.Type = "array"
					message.Schema.Items = &Items{}
					message.Schema.Items.Type = getOtherName(modelName)
					if getOtherName(modelName) == "integer" || getOtherName(modelName) == "number" {
						message.Schema.Items.Format = getFormat(modelName)
					}
				}
			}
		}
		response[strconv.Itoa(code)] = message
	}
	return response
}

func (sws SwaggerService) addModelsFromRouteTo(route restful.Route, decl *APIDefinition) {
	if route.ReadSample != nil {
		sws.addModelFromSampleTo(route.ReadSample, &decl.Definitions)
	}
	if route.WriteSample != nil {
		sws.addModelFromSampleTo(route.WriteSample, &decl.Definitions)
	}
}

// addModelFromSample creates and adds (or overwrites) a Model from a sample resource
func (sws SwaggerService) addModelFromSampleTo(sample interface{}, items *map[string]*Items) {
	mb := modelBuilder{Definitions: items, Config: &sws.config}
	mb.addModel(reflect.TypeOf(sample), "")
}

func getOperation(route restful.Route) *Endpoint {
	return &Endpoint{
		Summary:     route.Doc,
		Description: route.Notes,
		OperationId: route.Operation,
		Consumes:    route.Consumes,
		Produces:    route.Produces,
		Parameters:  Parameters{}}
}

func (sws SwaggerService) composeDeclaration(ws *restful.WebService, pathPrefix string) APIDefinition {
	sws.config.Info.Version = ws.Version()
	decl := APIDefinition{
		Swagger:     swaggerVersion,
		BasePath:    ws.RootPath(),
		Paths:       map[string]*Path{},
		Info:        sws.config.Info,
		Definitions: map[string]*Items{}}

	pathToRoutes := newOrderedRouteMap()
	for _, other := range ws.Routes() {
		if strings.HasPrefix(other.Path, ws.RootPath()) {
			if len(ws.RootPath()) > 1 && len(other.Path) > len(ws.RootPath()) && other.Path[len(ws.RootPath())] != '/' {
				continue
			}
			path := strings.TrimPrefix(other.Path, pathPrefix)
			if ws.RootPath() == "" || ws.RootPath() == "/" {
				path = other.Path
			}
			pathToRoutes.Add(path, other)
		}
	}
	pathToRoutes.Do(func(path string, routes []restful.Route) {
		for _, route := range routes {
			op := buildEndpoint(route, decl, sws)
			switch route.Method {
			case "GET":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Get: &Endpoint{}}
					decl.Paths[path].Get = op
				} else {
					decl.Paths[path].Get = op
				}
			case "POST":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Post: &Endpoint{}}
					decl.Paths[path].Post = op
				} else {
					decl.Paths[path].Post = op
				}
			case "PUT":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Put: &Endpoint{}}
					decl.Paths[path].Put = op
				} else {
					decl.Paths[path].Put = op
				}
			case "DELETE":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Delete: &Endpoint{}}
					decl.Paths[path].Delete = op
				} else {
					decl.Paths[path].Delete = op
				}
			case "PATCH":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Patch: &Endpoint{}}
					decl.Paths[path].Patch = op
				} else {
					decl.Paths[path].Patch = op
				}
			case "OPTIONS":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Options: &Endpoint{}}
					decl.Paths[path].Options = op
				} else {
					decl.Paths[path].Options = op
				}
			case "HEAD":
				if decl.Paths[path] == nil {
					decl.Paths[path] = &Path{Head: &Endpoint{}}
					decl.Paths[path].Head = op
				} else {
					decl.Paths[path].Head = op
				}
			}
		}

	})
	return decl
}
func buildEndpoint(route restful.Route, decl APIDefinition, sws SwaggerService) *Endpoint {
	endpoint := getOperation(route)
	for _, param := range route.ParameterDocs {
		item := asSwaggerParameter(param.Data())
		optimizeParameter(item, param.Data(), route.ReadSample)
		endpoint.Parameters = append(endpoint.Parameters, item)
	}
	endpoint.Responses = composeResponses(route, &decl, &sws.config)
	sws.addModelsFromRouteTo(route, &decl)
	return endpoint
}
func optimizeParameter(item *Items, param restful.ParameterData, readSample interface{}) {
	if param.Kind == 2 {
		if reflect.TypeOf(readSample).Kind() == reflect.Slice || reflect.TypeOf(readSample).Kind() == reflect.Array {
			item.Type = "array"
			dataType := modelBuilder{}.keyFrom(reflect.TypeOf(readSample).Elem())
			item.Items = &Items{Ref: getModelName(dataType)}
		} else {
			item.Type = nil
			item.Schema = &Items{Ref: getModelName(param.DataType)}
		}
	}
	if param.DataFormat == "" {
		item.Format = nil
	}
}

func detectCollectionType(st reflect.Type) (bool, reflect.Type) {
	isCollection := false
	if st.Kind() == reflect.Slice || st.Kind() == reflect.Array {
		st = st.Elem()
		if st.Kind() == reflect.Ptr {
			st = st.Elem()
		}
		isCollection = true
	} else {
		if st.Kind() == reflect.Ptr {
			st = st.Elem()
			if st.Kind() == reflect.Slice || st.Kind() == reflect.Array {
				st = st.Elem()
				isCollection = true
			}
		}
	}
	return isCollection, st
}

func asSwaggerParameter(param restful.ParameterData) *Items {
	return &Items{
		Name:        param.Name,
		In:          asParamType(param.Kind),
		Description: param.Description,
		Required:    param.Required,
		Type:        getOtherName(param.DataType),
		Format:      asFormat(param.DataType, param.DataFormat)}
}

// Between 1..7 path parameters is supported
func composeRootPath(req *restful.Request) string {
	path := "/" + req.PathParameter("a")
	b := req.PathParameter("b")
	if b == "" {
		return path
	}
	path = path + "/" + b
	c := req.PathParameter("c")
	if c == "" {
		return path
	}
	path = path + "/" + c
	d := req.PathParameter("d")
	if d == "" {
		return path
	}
	path = path + "/" + d
	e := req.PathParameter("e")
	if e == "" {
		return path
	}
	path = path + "/" + e
	f := req.PathParameter("f")
	if f == "" {
		return path
	}
	path = path + "/" + f
	g := req.PathParameter("g")
	if g == "" {
		return path
	}
	return path + "/" + g
}

func asFormat(dataType string, dataFormat string) string {
	if dataFormat != "" {
		return getFormat(dataFormat)
	}
	return "" // TODO
}

func asParamType(kind int) string {
	switch {
	case kind == restful.PathParameterKind:
		return "path"
	case kind == restful.QueryParameterKind:
		return "query"
	case kind == restful.BodyParameterKind:
		return "body"
	case kind == restful.HeaderParameterKind:
		return "header"
	case kind == restful.FormParameterKind:
		return "form"
	}
	return ""
}

//Get Schema information
func (sws *SwaggerService) GetSchemaInfoList() ([]string, error) {
	var result []string
	for _, listing := range sws.apiDeclarationMap.List {
		list, err := yaml.Marshal(listing)
		if err != nil {
			return nil, err
		}
		result = append(result, string(list))
	}
	return result, nil
}
