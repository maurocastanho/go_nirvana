package main

import (
	js "encoding/json"
	"fmt"
	"strconv"

	"github.com/golang-collections/collections/stack"
)

// JSONWriter represents a writer to a JSON file
type JSONWriter struct {
	fileName string
	root     interface{}
	st       stack.Stack
}

// NewJSONWriter creates a new struct
func NewJSONWriter(filename string) *JSONWriter {
	w := JSONWriter{fileName: filename}
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
		arr = append(arr, elem)
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
