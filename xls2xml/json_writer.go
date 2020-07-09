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
	categLines []lineT
	serieLines []lineT
	testing    bool
}

// newJSONWriter creates a new struct
func newJSONWriter(filename string, categLines []lineT, serieLines []lineT, jType int) (*jsonWriter, error) {
	var err error
	w := jsonWriter{fileName: filename, categLines: categLines, serieLines: serieLines}
	switch jType {
	case assetsT:
		return &w, nil
	case categsT:
		_, err = w.initCateg()
	case seriesT:
		_, err = w.initSeries()
	}
	return &w, err
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
func (wr *jsonWriter) StartMap() error {
	wr.getElement("", mapT)
	return nil
}

// EndMap closes a map element
func (wr *jsonWriter) EndMap() error {
	err := wr.EndElem("", mapT)
	return err
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

// insertElement inserts a new element into the structure
func (wr *jsonWriter) insertElement(name string, elem interface{}, elType elemType) error {
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
			return fmt.Errorf("ERRO: InsertElement(interface{}): %#v\n", c)
		default:
			//fmt.Printf("**** %#v\n", c)
		}
	}
	return nil
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
					// fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = errorMessage[0].val
					break
				}
				c[name] = val
			case "timestamp":
				var val, err = toTimestamp(value)
				if err != nil {
					// fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = errorMessage[0].val
					break
				}
				c[name] = val
			case "boolean":
				if value != "true" && value != "false" {
					c[name] = errorMessage[0].val
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
	return wr.insertElement(name, el, elType)
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

// WriteConsolidated writes additional files
func (wr *jsonWriter) WriteConsolidated(mode int) (bufAssets []byte, bufCategs []byte, bufSeries []byte, err error) {
	bufAssets, err = js.MarshalIndent(consolidated, "", "  ")
	if err != nil {
		return
	}
	if !wr.testing {
		fileAssets := path.Join(wr.fileName, "assets.json")
		log("Writing " + fileAssets)
		err = ioutil.WriteFile(fileAssets, bufAssets, 0644)
		if err != nil {
			err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileAssets, err)
			return
		}
	}
	if mode == 1 {
		if wr.categLines == nil {
			return
		}
		bufCategs, err = js.MarshalIndent(wr.root, "", "  ")
		if err != nil {
			return
		}
		if !wr.testing {
			fileCateg := path.Join(wr.fileName, "categories.json")
			log("Writing " + fileCateg)
			err = ioutil.WriteFile(fileCateg, bufCategs, 0644)
			if err != nil {
				err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileCateg, err)
				return
			}
		}
	} else if mode == 2 {
		bufSeries, err = js.MarshalIndent(wr.root, "", "  ")
		if err != nil {
			return
		}
		if !wr.testing {
			fileSeries := path.Join(wr.fileName, "series.json")
			log("Writing " + fileSeries)
			err = ioutil.WriteFile(fileSeries, bufSeries, 0644)
			if err != nil {
				err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileSeries, err)
				return
			}
		}
	}
	return
}

func (wr *jsonWriter) initCateg() (map[string]interface{}, error) {
	root := make(map[string]interface{})
	cat := make([]map[string]interface{}, 0)
	//fmt.Printf("Categories: [%#v]\n", wr.categLines)
	for _, line := range wr.categLines {
		el := make(map[string]interface{})
		id, ok := line["id"]
		if !ok || id == "" {
			name, ok2 := line["name"]
			if ok2 {
				fmt.Printf("WARNING: categoria [%s] nao existente na aba 'categories'", name)
			}
			continue
		}
		el["id"] = line["id"]
		elName := make(map[string]interface{})
		strNames := strings.Split(line["name"], "|")
		for _, l := range strNames {
			vals := strings.Split(l, ":")
			if len(vals) < 2 {
				err := fmt.Errorf("erro ao ler categoria, valor invalido [%s]", l)
				return nil, err
			}
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
	return root, nil
}

func (wr *jsonWriter) initSeries() (map[string]interface{}, error) {
	root := make(map[string]interface{})
	cat := make([]map[string]interface{}, 0)
	fmt.Printf("Series: [%#v]\n", wr.serieLines)
	for _, line := range wr.serieLines {
		el := make(map[string]interface{})
		name, _ := line["title"]
		id, ok := line["id"]
		if !ok || id == "" {
			return nil, fmt.Errorf("serie [%s] nao existente na aba 'series'", name)
		}
		el["id"] = line["id"]
		elName := make(map[string]interface{})
		strNames := strings.Split(line["title"], "|")
		for _, l := range strNames {
			vals := strings.Split(l, ":")
			if len(vals) < 2 {
				err := fmt.Errorf("erro ao ler serie, valor invalido [%s]", name)
				return nil, err
			}
			elName[vals[0]] = vals[1]
		}
		el["external_ids"] = make([]interface{}, 0)
		el["title"] = elName
		el["synopsys"] = line["synopsys"]
		el["images"] = make([]interface{}, 0)
		el["seasons"] = make([]interface{}, 0)
		cat = append(cat, el)
	}
	root["series"] = cat
	wr.root = root
	return root, nil
}

// addToCategories adds an asset to the categories list
func (wr *jsonWriter) addToCategories(id string, categName string, rootEl string) error {
	if categName == "" {
		return nil
	}
	r := wr.root.(map[string]interface{})
	categs := r[rootEl].([]map[string]interface{})
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

// addToSeries adds an asset to the series list
func (wr *jsonWriter) addToSeries(id string, serieName string) error {
	if serieName == "" {
		return nil
	}
	r := wr.root.(map[string]interface{})
	series := r["series"].([]map[string]interface{})
	for _, serie := range series {
		name := serie["title"].(map[string]interface{})["por"]
		if name == serieName {
			assets := serie["assets"].([]interface{})
			assets = append(assets, id)
			serie["assets"] = assets
			return nil
		}
	}
	return nil
}

// IMPORTANT: JsonWriter must be created separately for the categories file - do not use this method for the assets file
func (wr *jsonWriter) processCategPack(row lineT, idField string, categField1 string, categField2 string) int {
	success := 0
	categ1 := row[categField1]
	if categ1 == "" {
		success = -2
		logError(fmt.Errorf("categoria 1 em branco na linha [%v]", row))
	}
	err := wr.addToCategories(row[idField], categ1, "categories")
	if err != nil {
		logError(err)
		success = -1
	}
	categ2 := row[categField2]
	if categ1 == "" {
		success = -2
		logError(fmt.Errorf("categoria 2 em branco na linha [%v]", row))
	}
	err = wr.addToCategories(row[idField], categ2, "categories")
	if err != nil {
		logError(err)
		success = -1
	}
	return success
}

// IMPORTANT: JsonWriter must be created separately for the series file - do not use this method for the assets file
func (wr *jsonWriter) processSeriesPack(row lineT, idField string, categField1 string) int {
	success := 0
	nEpis := row[categField1]
	err := wr.addToSeries(row[idField], nEpis)
	if err != nil {
		logError(err)
		success = -1
	}
	return success
}
