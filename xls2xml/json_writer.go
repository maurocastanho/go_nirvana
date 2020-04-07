package main

import (
	js "encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang-collections/collections/stack"
)

// JSONWriter represents a writer to a JSON file
type JSONWriter struct {
	fileName   string
	root       interface{}
	st         stack.Stack
	categLines []map[string]string
}

// NewJSONWriter creates a new struct
func NewJSONWriter(filename string, categLines []map[string]string) *JSONWriter {
	w := JSONWriter{fileName: filename, categLines: categLines}
	w.initCateg()
	return &w
}

// Suffix returns output file suffix
func (wr *JSONWriter) Suffix() string {
	return ".json"
}

// Filename returns output filename
func (wr *JSONWriter) Filename() string {
	return wr.fileName
}

// StartElem starts a JSON element
func (wr *JSONWriter) StartElem(name string, elType ElemType) error {
	fmt.Printf("%s -> %v\n", name, elType)
	el := wr.getElement(name, elType)
	current := wr.st.Peek()
	wr.insertElement(name, current, el)
	wr.st.Push(el)
	// fmt.Printf("--> %#v - %T\n", wr.st, current)
	return nil
}

// getElement returns a new variable for a given element
func (wr *JSONWriter) getElement(name string, elType ElemType) interface{} {
	var el interface{}
	switch elType {
	case MAP:
		el = make(map[string]interface{})
	case ARRAY:
		el = make([]interface{}, 0)
	case SINGLE:
		el = name
	case EMPTY:
		el = nil

	default:
	}
	return el
}

// insertElement inserts a new elemnt into the structure
func (wr *JSONWriter) insertElement(name string, current interface{}, elem interface{}) {
	var el interface{}
	if name != "" {
		m := make(map[string]interface{})
		arr := make([]interface{}, 0)
		if elem != nil {
			arr = append(arr, elem)
		}
		m[name] = arr
		el = arr
	}
	if current == nil {
		wr.root = el
	} else {
		switch c := current.(type) {
		case map[string]interface{}:
			c[name] = el
			fmt.Printf("** %#v\n", c)
		case []interface{}:
			c = append(c, el)
			fmt.Printf("** %#v\n", c)
		case interface{}:
			fmt.Printf("**== %#v\n", c)
		default:
			fmt.Printf("**** %#v\n", c)
		}
	}
}

func (wr *JSONWriter) Write(_ string) error {
	return nil
}

// WriteAttr writes a subelement
func (wr *JSONWriter) WriteAttr(name string, value string, vtype string) error {
	current := wr.st.Peek()
	if current == nil {
		wr.root = value
	} else {
		switch c := current.(type) {
		case map[string]interface{}:
			switch vtype {
			case "", "string":
				c[name] = value
			case "int":
				val, err := strconv.Atoi(value)
				if err != nil {
					fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = ERR
					break
				}
				c[name] = val
			case "timestamp":
				var val, err = ToTimestamp(value)
				if err != nil {
					fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = ERR
					break
				}
				c[name] = val
			case "boolean":
				if value != "true" && value != "false" {
					c[name] = ERR
					break
				}
				c[name] = value == "true"
			}
			// fmt.Printf("** %#v\n", c)
		case []interface{}:
			c = append(c, value)
			fmt.Printf("** %#v\n", c)
		case interface{}:
			fmt.Printf("**== %#v\n", c)
		default:
			fmt.Printf("*********** %#v\n", c)
		}
	}
	fmt.Printf("\"%s\": %s\n", name, value)
	return nil
}

// EndElem closes a JSON element
func (wr *JSONWriter) EndElem(name string) error {
	fmt.Printf("End: %s\n", name)
	wr.st.Pop()
	return nil
}

// StartComment marks the start of a comment section
func (wr *JSONWriter) StartComment(_ string) error {
	return nil
}

// EndComment closes a comment section
func (wr *JSONWriter) EndComment(_ string) error {
	return nil
}

// OpenOutput opens a new output file
func (wr *JSONWriter) OpenOutput() error {
	return nil
}

// WriteAndClose writes the structure in an external file
func (wr *JSONWriter) WriteAndClose(_ string) error {
	result, err := js.MarshalIndent(wr.root, "", "  ")
	fmt.Printf("RESULT %v, %v\n", string(result), err)
	return nil
}

// WriteExtras writes additional files
func (wr *JSONWriter) WriteExtras() {
	if wr.categLines == nil {
		return
	}
	result, err := js.MarshalIndent(wr.root, "", "  ")
	fmt.Printf("CATEGORIES %v, %v\n", string(result), err)
}

func (wr *JSONWriter) initCateg() map[string]interface{} {
	root := make(map[string]interface{})
	cat := make([]map[string]interface{}, 0)
	for _, line := range wr.categLines {
		el := make(map[string]interface{})
		el["id"] = line["id"]
		elName := make(map[string]interface{})
		strNames := strings.Split(line["name"], "|")
		for _, l := range strNames {
			vals := strings.Split(l, ":")
			elName[vals[0]] = vals[1]
		}
		el["name"] = elName
		el["hidden"] = line["hidden"] == "true"
		el["morality_level"] = line["morality_level"]
		el["parental_control"] = line["parental_control"] == "true"
		el["adult"] = line["adult"] == "true"
		el["downloadable"] = line["downloadable"] == "true"
		el["offline"] = line["offline"] == "true"
		el["metadata"] = make(map[string]interface{})
		el["images"] = make([]interface{}, 0)
		el["parent_id"] = ""
		el["assets"] = make([]interface{}, 0)

		cat = append(cat, el)
	}
	root["categories"] = cat
	wr.root = root
	return root
}

// AddAsset adds an asset to the categories list
func (wr *JSONWriter) AddAsset(id string, categName string) error {
	r := wr.root.(map[string]interface{})
	categs := r["categories"].([]map[string]interface{})
	for _, categ := range categs {
		name := categ["name"].(map[string]interface{})["por"]
		if name == categName {
			assets := categ["assets"].([]interface{})
			assets = append(assets, id)
			categ["assets"] = assets
			return nil
		}
	}
	uuid, err := UUID("", nil, nil, nil)
	if err != nil {
		return err
	}
	return fmt.Errorf("Categoria nao existente: inclua na aba 'categories'. Nome: [%s], sugestao de id: [%s] ", categName, uuid[0])
}
