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

// jsonWriter represents a writer to a JSON file
type jsonWriter struct {
	fileName   string
	root       interface{}
	st         stack.Stack
	categLines []map[string]string
}

// newJSONWriter creates a new struct
func newJSONWriter(filename string, categLines []map[string]string) *jsonWriter {
	w := jsonWriter{fileName: filename, categLines: categLines}
	w.initCateg()
	return &w
}

// Suffix returns output file suffix
func (wr *jsonWriter) Suffix() string {
	return ".json"
}

// Filename returns output filename
func (wr *jsonWriter) Filename() string {
	return wr.fileName
}

// StartMap starts a map element
func (wr *jsonWriter) StartMap() {
	wr.getElement("", mapT)
}

// EndMap closes a map element
func (wr *jsonWriter) EndMap() {
	err := wr.EndElem("", mapT)
	if err != nil {
		logError(err)
	}
}

// StartElem starts a JSON element
func (wr *jsonWriter) StartElem(name string, elType elemType) error {
	//fmt.Printf("%s -> %v\n", name, elType)
	el := wr.getElement(name, elType)
	wr.st.Push(el)
	// fmt.Printf("--> %#v - %T\n", wr.st, current)
	return nil
}

// getElement returns a new variable for a given element
func (wr *jsonWriter) getElement(name string, elType elemType) interface{} {
	var el interface{}
	switch elType {
	case mapT, mapNoArrT, mapArrayT:
		el = make(map[string]interface{})
	case arrayT:
		el = make([]interface{}, 0)
	case singleT:
		el = name
	case emptyT:
		el = nil

	default:
	}
	return el
}

// insertElement inserts a new elemnt into the structure
func (wr *jsonWriter) insertElement(name string, elem interface{}, elType elemType) {
	var el interface{}
	if elType == mapT || elem == nil {
		arr := make([]interface{}, 0)
		if elem != nil {
			arr = append(arr, elem)
		}
		el = arr
	} else {
		el = elem
	}
	current := wr.st.Peek()
	if current == nil {
		m := make(map[string]interface{})
		wr.root = m
		m[name] = el
	} else {
		switch c := current.(type) {
		case map[string]interface{}:
			c[name] = el
			// fmt.Printf("** %#v\n", c)
		case []interface{}:
			wr.st.Pop()
			c = append(c, el)
			wr.st.Push(c)
			//fmt.Printf("** %#v\n", c)
		case interface{}:
			fmt.Printf("**== %#v\n", c)
		default:
			//fmt.Printf("**** %#v\n", c)
		}
	}
}

func (wr *jsonWriter) Write(_ string) error {
	return nil
}

// WriteAttr writes a subelement
func (wr *jsonWriter) WriteAttr(name string, value string, vtype string, _ string) error {
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
					c[name] = errMsg
					break
				}
				c[name] = val
			case "timestamp":
				var val, err = toTimestamp(value)
				if err != nil {
					fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = errMsg
					break
				}
				c[name] = val
			case "boolean":
				if value != "true" && value != "false" {
					c[name] = errMsg
					break
				}
				c[name] = value == "true"
			}
			// fmt.Printf("** %#v\n", c)
		case []interface{}:
			c = append(c, value)
			wr.st.Pop()
			wr.st.Push(c)
			//fmt.Printf("** %#v\n", c)
		case interface{}:
			//fmt.Printf("**== %#v\n", c)
		default:
			//fmt.Printf("*********** %#v\n", c)
		}
	}
	//fmt.Printf("\"%s\": %s\n", name, value)
	return nil
}

// EndElem closes a JSON element
func (wr *jsonWriter) EndElem(name string, elType elemType) error {
	//fmt.Printf("End: %s\n", name)
	el := wr.st.Pop()
	wr.insertElement(name, el, elType)

	return nil
}

// StartComment marks the start of a comment section
func (wr *jsonWriter) StartComment(_ string) error {
	return nil
}

// EndComment closes a comment section
func (wr *jsonWriter) EndComment(_ string) error {
	return nil
}

// OpenOutput opens a new output file
func (wr *jsonWriter) OpenOutput() error {
	return nil
}

// WriteAndClose writes the structure in an external file
func (wr *jsonWriter) WriteAndClose(_ string) error {

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
func (wr *jsonWriter) WriteExtras() {
	if wr.categLines == nil {
		return
	}
	res, _ := js.MarshalIndent(consolidated, "", "  ")
	fileAssets := path.Join(wr.fileName, "assets.json")
	log("Writing " + fileAssets)
	err := ioutil.WriteFile(fileAssets, res, 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileAssets, err)
		return
	}
	result, err := js.MarshalIndent(wr.root, "", "  ")
	fileCateg := path.Join(wr.fileName, "categories.json")
	log("Writing " + fileCateg)
	err = ioutil.WriteFile(fileCateg, result, 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileCateg, err)
		return
	}
}

func (wr *jsonWriter) initCateg() map[string]interface{} {
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
func (wr *jsonWriter) addAsset(id string, categName string) error {
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
	uuid, err := genUUID("", nil, nil, nil)
	if err != nil {
		return err
	}
	return fmt.Errorf("Categoria nao existente: inclua na aba 'categories'. Nome: [%s], sugestao de id: [%s] ", categName, uuid[0].val)
}
