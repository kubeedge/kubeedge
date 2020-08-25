# go-restful-swagger20

## overview
[openapi](https://www.openapis.org) extension to the go-restful package, 
[OpenAPI-Specification version 2.0](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md)

## dependencies
- [go-restful](https://github.com/emicklei/go-restful)
- [gopkg.in/yaml.v2](https://github.com/go-yaml/yaml/tree/v2)

## how to use
use it to replace [go-restful-swagger12](https://github.com/emicklei/go-restful-swagger12)
```go
config := swagger.Config{
		WebServices:    restful.DefaultContainer.RegisteredWebServices(), 
		FileStyle:	"json", //optional, default is yaml
		OpenService:     true,  //should show it in rest API service
		WebServicesUrl: "http://localhost:8080",
        ApiPath:        "/apidocs.json", //swagger doc api path
		SwaggerPath:     "/apidocs/", //local file path
		SwaggerFilePath: os.Getenv("SWAGGERFILEPATH"),
} 
swagger.RegisterSwaggerService(config, restful.DefaultContainer)
```
### How to change or ignore the name of a field 

go struct 
```go
	type X struct {
		A int
		B int `json:"C"`  //Will generate C here
		D int `json:"-"`  //Will ignore it
	}
```
result
```json
	  "X": {
		"type": "object",
	   "properties": {
		"A": {
		 "type": "integer",
		 "format": "int32"
		},
		"C": {
		 "type": "integer",
		 "format": "int32"
		}
	   }
	  }
```


[Example](./examples)

