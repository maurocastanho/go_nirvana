package main

import (
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"github.com/golang-collections/collections/stack"
)

var consolidated interface{}

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
	wr.st.Push(el)
	// fmt.Printf("--> %#v - %T\n", wr.st, current)
	return nil
}

// getElement returns a new variable for a given element
func (wr *JSONWriter) getElement(name string, elType ElemType) interface{} {
	var el interface{}
	switch elType {
	case Map, MapNoarr:
		el = make(map[string]interface{})
	case Array:
		el = make([]interface{}, 0)
	case Single:
		el = name
	case Empty:
		el = nil

	default:
	}
	return el
}

// insertElement inserts a new elemnt into the structure
func (wr *JSONWriter) insertElement(name string, current interface{}, elem interface{}, elType ElemType) {
	var el interface{}
	if elType == Map || elem == nil {
		arr := make([]interface{}, 0)
		if elem != nil {
			arr = append(arr, elem)
		}
		el = arr
	} else {
		el = elem
	}
	if current == nil {
		m := make(map[string]interface{})
		wr.root = m
		m[name] = el
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
			wr.st.Pop()
			wr.st.Push(c)
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
func (wr *JSONWriter) EndElem(name string, elType ElemType) error {
	fmt.Printf("End: %s\n", name)
	el := wr.st.Pop()
	current := wr.st.Peek()
	wr.insertElement(name, current, el, elType)

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

	if consolidated == nil {
		consolidated = wr.root
	} else {

		arrCons := consolidated.(map[string]interface{})["assets"].([]interface{})
		arrNew := wr.root.(map[string]interface{})["assets"].([]interface{})[0]

		consolidated.(map[string]interface{})["assets"] = append(arrCons, arrNew)
		wr.root = consolidated
	}

	//result, err := js.MarshalIndent(wr.root, "", "  ")
	//if err != nil {
	//	logError(err)
	//}
	//fmt.Printf("RESULT %v\n", string(result))
	return nil
}

// WriteExtras writes additional files
func (wr *JSONWriter) WriteExtras() {
	if wr.categLines == nil {
		return
	}
	res, _ := js.MarshalIndent(consolidated, "", "  ")
	fileAssets := path.Join(wr.fileName, "assets.json")
	err := ioutil.WriteFile(fileAssets, res, 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileAssets, err)
		return
	}

	fmt.Printf("RESULT %v\n", string(res))

	result, err := js.MarshalIndent(wr.root, "", "  ")
	fileCateg := path.Join(wr.fileName, "categories.json")
	err = ioutil.WriteFile(fileCateg, result, 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileCateg, err)
		return
	}

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
