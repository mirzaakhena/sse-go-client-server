package utility

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

type QueryParam struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type ExampleResponse struct {
	StatusCode int
	Content    interface{}
}

type APIData struct {
	Method string
	Url    string
	// Access             model.Access
	Body               any
	QueryParams        []QueryParam
	Summary            string
	Description        string
	Tag                string
	Examples           []ExampleResponse
	MultipartFormParam []MultipartFormParam
}

type MultipartFormParam struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

func (a APIData) GetMethodUrl() string {
	return a.Method + " " + a.Url
}

type ApiPrinter struct {
	urls []APIData
}

func (r *ApiPrinter) Add(apiData APIData) *ApiPrinter {
	r.urls = append(r.urls, apiData)
	return r
}

func (r ApiPrinter) Print() ApiPrinter {
	for _, v := range r.urls {
		// fmt.Printf("%s %s %s\n", v.Method, v.Url, v.Access)
		fmt.Printf("%s %s\n", v.Method, v.Url)
	}
	return r
}

func (r ApiPrinter) PrintAPIDataTable() ApiPrinter {
	// Define colors
	headerColor := color.New(color.FgHiCyan, color.Bold)
	// adminColor := color.New(color.FgRed)
	// anonymousColor := color.New(color.FgYellow)
	// userColor := color.New(color.FgGreen)
	defaultColor := color.New(color.FgWhite)

	// Define column widths
	tagWidth := 28
	accessWidth := 15
	summaryWidth := 35
	methodWidth := 8
	urlWidth := 40

	// Print table header
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%s\n", tagWidth, accessWidth, summaryWidth, methodWidth)
	headerColor.Printf(headerFormat, "Tag", "Access", "Summary", "Method", "URL")
	headerColor.Println(strings.Repeat("-", tagWidth+accessWidth+summaryWidth+methodWidth+urlWidth+4))

	// Print each row
	rowFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%s\n", tagWidth, accessWidth, summaryWidth, methodWidth)
	for _, item := range r.urls {
		// var rowColor *color.Color
		// switch item.Access {
		// case model.ADMIN_OPERATION:
		// 	rowColor = adminColor
		// case model.ANONYMOUS:
		// 	rowColor = anonymousColor
		// case model.DEFAULT_OPERATION:
		// 	rowColor = userColor
		// default:
		// rowColor = defaultColor
		// }

		tag := truncateOrPad(item.Tag, tagWidth)
		access := truncateOrPad(getDescriptionFromAccess(), accessWidth)
		summary := truncateOrPad(item.Summary, summaryWidth)
		method := truncateOrPad(item.Method, methodWidth)
		url := truncateOrPad(item.Url, urlWidth)

		// rowColor.Printf(rowFormat, tag, access, summary, method, url)
		defaultColor.Printf(rowFormat, tag, access, summary, method, url)
	}

	return r
}

func getDescriptionFromAccess() string {
	// if access == model.ANONYMOUS {
	// 	return "ANONYMOUS"
	// }

	// if access == model.ADMIN_OPERATION {
	// 	return "ADMIN"
	// }

	// if access == model.DEFAULT_OPERATION {
	// 	return "USER"
	// }

	return "OTHERS"
}

func truncateOrPad(s string, width int) string {
	if len(s) > width {
		return s[:width]
	}
	return fmt.Sprintf("%-*s", width, s)
}

func (r ApiPrinter) generateOpenAPISchema(baseURL string) OpenAPISchema {

	schema := OpenAPISchema{
		OpenAPI: "3.0.0",
		Info: map[string]interface{}{
			"title":   "IAM API",
			"version": "1.0.0",
		},
		Servers: []map[string]interface{}{
			{
				"url":         baseURL,
				"description": "API server",
			},
		},
		Paths:      make(map[string]interface{}),
		Components: make(map[string]interface{}),
		Tags:       []map[string]string{},
	}

	uniqueTags := make(map[string]bool)

	for _, endpoint := range r.urls {
		path := endpoint.Url
		method := strings.ToLower(endpoint.Method)

		pathParams := []map[string]interface{}{}
		parts := strings.Split(path, "/")
		for i, part := range parts {
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				paramName := strings.Trim(part, "{}")
				parts[i] = "{" + paramName + "}"
				pathParams = append(pathParams, map[string]interface{}{
					"name":     paramName,
					"in":       "path",
					"required": true,
					"schema":   map[string]string{"type": "string"},
				})
			}
		}

		pathItem, ok := schema.Paths[path].(map[string]interface{})
		if !ok {
			pathItem = make(map[string]interface{})
			schema.Paths[path] = pathItem
		}

		operation := map[string]interface{}{
			"responses": make(map[string]interface{}),
		}

		if endpoint.Summary != "" {
			operation["summary"] = endpoint.Summary
		} else {
			operation["summary"] = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
		}

		if endpoint.Description != "" {
			operation["description"] = endpoint.Description
		}

		if endpoint.Tag != "" {
			operation["tags"] = []string{endpoint.Tag}
			uniqueTags[endpoint.Tag] = true
		}

		parameters := append(pathParams, []map[string]interface{}{}...)
		for _, param := range endpoint.QueryParams {
			queryParam := map[string]interface{}{
				"name":        param.Name,
				"in":          "query",
				"description": param.Description,
				"required":    param.Required,
				"schema": map[string]string{
					"type": param.Type,
				},
			}
			parameters = append(parameters, queryParam)
		}
		if len(parameters) > 0 {
			operation["parameters"] = parameters
		}

		if len(endpoint.MultipartFormParam) > 0 {

			operation["requestBody"] = map[string]interface{}{
				"content": map[string]interface{}{
					"multipart/form-data": map[string]interface{}{
						"schema": map[string]interface{}{
							"type":       "object",
							"properties": generateMultipartFormSchema(endpoint.MultipartFormParam),
						},
					},
				},
			}
		} else {

			if endpoint.Body != nil && method != "get" {

				bodySchema := generateBodySchema(endpoint.Body)
				operation["requestBody"] = map[string]interface{}{
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": bodySchema,
						},
					},
				}

			}

		}

		// Add example responses
		for _, example := range endpoint.Examples {
			statusCode := fmt.Sprintf("%d", example.StatusCode)
			operation["responses"].(map[string]interface{})[statusCode] = map[string]interface{}{
				"description": fmt.Sprintf("Status %s response", statusCode),
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"example": example.Content,
					},
				},
			}
		}

		// Add default 200 response if no examples provided
		if len(endpoint.Examples) == 0 {
			operation["responses"].(map[string]interface{})["200"] = map[string]interface{}{
				"description": "Successful operation",
			}
		}

		// if endpoint.Access != "0" {
		// 	operation["security"] = []map[string][]string{
		// 		{"bearerAuth": {}},
		// 	}
		// }

		pathItem[method] = operation
	}

	for tag := range uniqueTags {
		schema.Tags = append(schema.Tags, map[string]string{"name": tag})
	}

	schema.Components["securitySchemes"] = map[string]interface{}{
		"bearerAuth": map[string]interface{}{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		},
	}

	return schema
}

func generateBodySchema(body interface{}) map[string]interface{} {
	return generateSchema(reflect.TypeOf(body))
}

func generateSchema(t reflect.Type) map[string]interface{} {
	schema := map[string]interface{}{}

	switch t.Kind() {
	case reflect.Struct:
		schema["type"] = "object"
		properties := make(map[string]interface{})
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = field.Name
			}
			jsonTag = strings.Split(jsonTag, ",")[0]

			fieldSchema := generateSchema(field.Type)
			properties[jsonTag] = fieldSchema
		}
		schema["properties"] = properties

	case reflect.Slice:
		schema["type"] = "array"
		schema["items"] = generateSchema(t.Elem())

	case reflect.Ptr:
		return generateSchema(t.Elem())

	case reflect.String:
		schema["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.Bool:
		schema["type"] = "boolean"
	default:
		schema["type"] = "object"
	}

	return schema
}

func generateMultipartFormSchema(params []MultipartFormParam) map[string]interface{} {
	properties := make(map[string]interface{})
	for _, param := range params {
		if param.Type == "file" {
			properties[param.Name] = map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":   "string",
					"format": "binary",
				},
				"description": param.Description,
			}
		} else {
			properties[param.Name] = map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
		}
	}
	return properties
}

// // Update the getItemSchema function to use the new generateSchema function
// func getItemSchema(t reflect.Type) map[string]interface{} {
// 	return generateSchema(t)
// }

type OpenAPISchema struct {
	OpenAPI    string                   `json:"openapi"`
	Info       map[string]interface{}   `json:"info"`
	Servers    []map[string]interface{} `json:"servers"`
	Paths      map[string]interface{}   `json:"paths"`
	Components map[string]interface{}   `json:"components"`
	Tags       []map[string]string      `json:"tags,omitempty"`
}

func (r ApiPrinter) PublishAPI(mux *http.ServeMux, baseURL, apiURL string) ApiPrinter {

	handler := func(w http.ResponseWriter, req *http.Request) {

		obj := r.generateOpenAPISchema(baseURL)

		yamlData, err := yaml.Marshal(&obj)
		if err != nil {
			http.Error(w, "Error creating YAML", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(yamlData)
	}

	mux.HandleFunc("GET "+apiURL, handler)

	fmt.Printf("\nSWAGGER https://editor.swagger.io/?url=%s%s\n", baseURL, apiURL)

	return r
}

func NewApiPrinter() *ApiPrinter {
	return &ApiPrinter{
		urls: []APIData{},
	}
}
