// Package swagger implements the structures of the Swagger
// https://github.com/wordnik/swagger-spec/blob/master/versions/1.2.md
package swagger

const swaggerVersion = "2.0"
//Swagger Object
type APIDefinition struct {
	Swagger     string                	`yaml:"swagger" json:"swagger"`
	Info        Info                  	`yaml:"info" json:"info"`
	Host        string                	`yaml:"host,omitempty" json:"host,omitempty"`
	Schemes     []string                	`yaml:"schemes,omitempty" json:"schemes,omitempty"`
	BasePath    string                	`yaml:"basePath" json:"basePath"`
	Produces    []string                	`yaml:"produces,omitempty" json:"produces,omitempty"`
	Paths       map[string]*Path        	`yaml:"paths" json:"paths"`
	Definitions map[string]*Items    	`yaml:"definitions,omitempty" json:"definitions,omitempty"`
	Parameters  map[string]*Items  		`yaml:"parameters,omitempty" json:"parameters,omitempty"`
}
type Info    struct {
	Title       string        `yaml:"title" json:"title"`
	Description string        `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string        `yaml:"version" json:"version"`
}

// Path represents all of the endpoints and parameters available for a single
type Path struct {
	Get        *Endpoint        `yaml:"get,omitempty" json:"get,omitempty"`
	Put        *Endpoint        `yaml:"put,omitempty" json:"put,omitempty"`
	Post       *Endpoint        `yaml:"post,omitempty" json:"post,omitempty"`
	Delete     *Endpoint        `yaml:"delete,omitempty" json:"delete,omitempty"`
	Patch	   *Endpoint	    `yaml:"patch,omitempty" json:"patch,omitempty"`
	Options    *Endpoint	    `yaml:"options,omitempty" json:"options,omitempty"`
	Head       *Endpoint	    `yaml:"head,omitempty" json:"head,omitempty"`
	Parameters Parameters       `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// Parameters is a slice of request parameters for a single endpoint.
type Parameters []*Items

// Response represents the response object in an OpenAPI spec.
type Response struct {
	Description string        `yaml:"description" json:"description"`
	Schema      *Items        `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// Endpoint represents an endpoint for a path in an OpenAPI spec.
type Endpoint struct {
	Summary     string                `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string                `yaml:"description,omitempty" json:"description,omitempty"`
	OperationId string                `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Parameters  Parameters            `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Consumes    []string              `yaml:"consumes,omitempty" json:"consumes,omitempty"`
	Produces    []string              `yaml:"produces,omitempty" json:"produces,omitempty"`
	Tags        []string              `yaml:"tags,omitempty" json:"tags,omitempty"`
	Responses   map[string]Response   `yaml:"responses,omitempty" json:"responses,omitempty"`
}


// Items represent Model properties in an OpenAPI spec.
type Items struct {
	Name                 string 		`yaml:"name,omitempty" json:"name,omitempty"`
	In                   string     	`yaml:"in,omitempty" json:"in,omitempty"`
	Description          string        	`yaml:"description,omitempty" json:"description,omitempty"`
	Required             bool            	`yaml:"required,omitempty" json:"required,omitempty"`
	Type                 interface{}      	`yaml:"type,omitempty" json:"type,omitempty"`
	Format               interface{}        `yaml:"format,omitempty" json:"format,omitempty"`
	Enum                 []string    	`yaml:"enum,omitempty" json:"enum,omitempty"`

	ProtoTag             int      		`yaml:"x-proto-tag,omitempty" json:"x-proto-tag,omitempty"`
	// Map type
	AdditionalProperties *Items     	`yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
	// ref another Model
	Ref                  string          	`yaml:"$ref,omitempty" json:"$ref,omitempty"`
	// is an array
	Items                *Items         	`yaml:"items,omitempty" json:"items,omitempty"`
	Schema               *Items             `yaml:"schema,omitempty" json:"schema,omitempty"`
	Properties           map[string]*Items  `yaml:"properties,omitempty" json:"properties,omitempty"`
}