package swagger

import (
	"reflect"
	"regexp"
	"strings"
)

func (b modelBuilder) addModel(st reflect.Type, nameOverride string) *Items {
	// Turn pointers into simpler types so further checks are
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	modelName := b.keyFrom(st)
	if nameOverride != "" {
		modelName = nameOverride
	}
	reg := regexp.MustCompile("[a-z]+\\.")
	name := reg.ReplaceAllString(modelName, "")
	// no models needed for primitive types
	if b.isPrimitiveType(modelName) {
		return nil
	}

	if (st.Kind() == reflect.Slice || st.Kind() == reflect.Array) &&
		st.Elem().Kind() == reflect.Uint8 {
		return nil
	}
	// see if we already have visited this model
	if _, ok := atMap(name, b.Definitions); ok {
		return nil
	}
	sm := Items{
		Type:       "object",
		Properties: map[string]*Items{}}

	(*b.Definitions)[name] = &sm
	// check for slice or array
	if st.Kind() == reflect.Slice || st.Kind() == reflect.Array {
		b.addModel(st.Elem(), "")
		return &sm
	}
	// check for structure or primitive type
	if st.Kind() != reflect.Struct {
		return &sm
	}

	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		//add tag,if =="" ignore
		if b.nameOfField(field) == "" {
			continue
		} else {
			field.Name = b.nameOfField(field)

		}
		sm.Properties[field.Name] = &Items{}
		ft := field.Type
		isCollection, ft := detectCollectionType(ft)
		fieldName := modelBuilder{}.keyFrom(ft)
		if !isCollection {
			if ft.Kind() == reflect.Struct {
				if fieldName == "time.Time" {
					sm.Properties[field.Name].Type = getOtherName(fieldName)
					sm.Properties[field.Name].Format = getFormat(fieldName)
				} else if len(ft.Name()) == 0 {
					anonType := name + "." + field.Name
					sm.Properties[field.Name].Ref = "#/definitions/" + anonType
					b.addModel(ft, anonType)
				} else {
					sm.Properties[field.Name].Ref = getModelName(fieldName)
					b.addModel(ft, "")
				}
			} else if ft.Kind() == reflect.Map {
				ft = ft.Elem()
				if ft.Kind() == reflect.Struct {
					sm.Properties[field.Name].Type = "object"
					sm.Properties[field.Name].AdditionalProperties = &Items{}
					modelName = modelBuilder{}.keyFrom(ft)
					sm.Properties[field.Name].AdditionalProperties.Ref = getModelName(modelName)
					b.addModel(ft, "")
				} else {
					sm.Properties[field.Name].Type = "object"
					sm.Properties[field.Name].AdditionalProperties = &Items{}
					modelName = modelBuilder{}.keyFrom(ft)
					sm.Properties[field.Name].AdditionalProperties.Type = getOtherName(modelName)
					if getOtherName(modelName) == "integer" || getOtherName(modelName) == "number" {
						sm.Properties[field.Name].AdditionalProperties.Format = getFormat(modelName)
					}
				}

			} else {
				sm.Properties[field.Name].Type = getOtherName(fieldName)
				if getOtherName(fieldName) == "integer" || getOtherName(fieldName) == "number" {
					sm.Properties[field.Name].Format = getFormat(fieldName)
				}
			}
		} else {
			if ft.Kind() == reflect.Struct {
				sm.Properties[field.Name].Type = "array"
				sm.Properties[field.Name].Items = &Items{}
				sm.Properties[field.Name].Items.Ref = getModelName(fieldName)
				b.addModel(ft, "")
			} else {
				sm.Properties[field.Name].Type = "array"
				sm.Properties[field.Name].Items = &Items{}
				sm.Properties[field.Name].Items.Type = getOtherName(fieldName)
				if getOtherName(fieldName) == "integer" || getOtherName(fieldName) == "number" {
					sm.Properties[field.Name].Items.Format = getFormat(fieldName)
				}
			}
		}
	}
	(*b.Definitions)[name] = &sm
	return &sm
}

type modelBuilder struct {
	Definitions *map[string]*Items
	Config      *Config
}

func atMap(name string, mapItem *map[string]*Items) (m *Items, ok bool) {
	for key, value := range *mapItem {
		if key == name {
			return value, true
		}
	}
	return m, false
}

func (b modelBuilder) keyFrom(st reflect.Type) string {
	key := st.String()
	if b.Config != nil && b.Config.ModelTypeNameHandler != nil {
		if name, ok := b.Config.ModelTypeNameHandler(st); ok {
			key = name
		}
	}
	if len(st.Name()) == 0 {
		key = strings.Replace(key, "[]", "", -1)
	}
	return key
}

////// see also https://golang.org/ref/spec#Numeric_types
func (b modelBuilder) isPrimitiveType(modelName string) bool {
	if len(modelName) == 0 {
		return false
	}
	return strings.Contains("uint uint8 uint16 uint32 uint64 int int8 int16 int32 int64 float32 float64 bool string byte rune time.Time", modelName)
}

// nameOfField returns the name of the field as it should appear in JSON format
// An empty string indicates that this field is not part of the JSON representation
func (b modelBuilder) nameOfField(field reflect.StructField) string {
	if tag := field.Tag.Get("json"); tag != "" {
		s := strings.Split(tag, ",")
		if s[0] == "-" {
			// empty name signals skip property
			return ""
		} else if s[0] != "" {
			return s[0]
		}
	}
	return field.Name
}
