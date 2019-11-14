package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	// "github.com/mgutz/ansi"
	"flag"

	xw "github.com/shabbyrobe/xmlwriter"
	"golang.org/x/text/encoding/charmap"
)

// type Ams struct {
// 	Provider     string `xml:"Provider,attr"`
// 	Product      string `xml:"Product,attr"`
// 	AssetName    string `xml:"Asset_Name,attr"`
// 	VersionMajor int    `xml:"Version_Major,attr"`
// 	VersionMinor int    `xml:"Version_Minor,attr"`
// 	Description  string `xml:"Description,attr"`
// 	CreationDate string `xml:"Creation_Date,attr"`
// 	ProviderID   string `xml:"Provider_ID,attr"`
// 	AssetID      string `xml:"Asset_ID,attr"`
// 	AssetClass   string `xml:"Asset_Class,attr"`
// }
// type AppData struct {
// 	App   string `xml:"App,attr"`
// 	Name  string `xml:"Name,attr"`
// 	Value string `xml:"Value,attr"`
// }
// type Metadata struct {
// 	Ams      Ams       `xml:"AMS,allowempty"`
// 	AppDatas []AppData `xml:"App_Data"`
// }
// type Content struct {
// 	Value string `xml:"value,attr"`
// }
// type Asset struct {
// 	Metadata Metadata `xml:"Metadata"`
// 	Content  `xml:"Content,omitempty"`
// }
// type Adi struct {
// 	Xmlns    string   `xml:"xmlns,attr"`
// 	Metadata Metadata `xml:"Metadata"`
// 	Assets   []Asset  `xml:"Asset"`
// }

func main() {

	FunctionDict = map[string]func(string, map[string]string, map[string]interface{}, map[string]string) ([]string, error){
		"fixed":           Fixed,
		"field":           Field,
		"field_validated": FieldValidated,
		"field_noacc":     FieldNoAccents,
		"field_trim":      FieldTrim,
		"field_no_quotes": FieldNoQuotes,
		"field_money":     FieldMoney,
		"field_suffix":    Suffix,
		"assetid":         AssetId,
		"date":            Date,
		"convert_date":    ConvertDate,
		"is_HD":           IsHD,
		"screen_format":   ScreenFormat,
		"exclude":         Exclude,
		"actors":          Actors,
		"bitrate":         BitRate,
		"seconds":         Seconds,
		"surname_name":    SurnameName,
		"episode_id":      EpisodeId,
		"episode_name":    EpisodeName,
		"condition":       Condition,
		"option":          Option,
		"eval":            Eval,
		"split":           Split,
	}

	inputXls := ""
	confFile := ""
	flag.StringVar(&inputXls, "xls", "", "Arquivo XLS de entrada")
	flag.StringVar(&confFile, "config", "", "Arquivo JSON de configuracao")
	flag.Parse()

	if inputXls == "" {
		logError(fmt.Errorf("Arquivo XLS deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if confFile == "" {
		logError(fmt.Errorf("Arquivo JSON de configuracao deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}

	f, err := excelize.OpenFile(inputXls)
	if err != nil {
		logError(err)
		os.Exit(2)
	}

	// var (
	// 	orientation excelize.PageLayoutOrientation
	// 	paperSize   excelize.PageLayoutPaperSize
	// )
	// if err := f.GetPageLayout("dados", &orientation); err != nil {
	// 	panic(err)
	// }
	// if err := f.GetPageLayout("dados", &paperSize); err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Defaults:")
	// fmt.Printf("- orientation: %q\n", orientation)
	// fmt.Printf("- paper size: %d\n", paperSize)

	header := make([]string, 0)
	lines := make([]map[string]string, 0)
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("dados")
	idx := 1
	for i, row := range rows {
		if i == 0 {
			for _, colCell := range row {
				header = append(header, colCell)
			}
		} else {
			line := make(map[string]string)
			lines = append(lines, line)
			for i, colCell := range row {
				line[header[i]] = colCell
			}
			line["file_number"] = fmt.Sprintf("%d", idx)
			idx++
		}
	}
	// fmt.Printf("--==>>> %#v\n", lines)

	file, err := os.Open(confFile)
	if err != nil {
		logError(err)
		os.Exit(3)
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		logError(err)
		os.Exit(3)
	}
	newBuf := latinToUTF8(buf)

	var objmap map[string]interface{}
	err = json.Unmarshal([]byte(newBuf), &objmap)
	if err != nil {
		logError(err)
		os.Exit(4)
	}
	// fmt.Printf("--> %v\n", objmap)
	success := 0
	for i, line := range lines {
		fmt.Fprintf(os.Stderr, "Processando linha %d...\n", i+1)
		err = process(objmap, line)
		if err != nil {
			logError(err)
			success = -1
		}
	}
	if success != 0 {
		log("*******************************************")
		log("*    ATENCAO: ERROS NO PROCESSAMENTO      *")
		log("*******************************************")
	}
	os.Exit(success)
}

var ec *xw.ErrCollector
var w *xw.Writer
var options map[string]string

const errSuffix = "_ERRO"

func process(json map[string]interface{}, line map[string]string) (err error) {
	options = make(map[string]string)
	opts := json["options"].([]interface{})
	for _, el := range opts {
		// fmt.Printf(">>>> %T\n", el)
		m := el.(map[string]interface{})
		name := m["name"].(string)
		value := m["value"].(string)
		options[name] = value
	}
	options["timestamp"] = Timestamp()
	systemID := options["doctype_system"]
	filename_field, ok := options["filename_field"]
	if !ok || filename_field == "" {
		err = fmt.Errorf("ERRO ao procurar filename_field nas options [%#v]\n", options)
		return
	}
	// fmt.Printf("**> [%v]: %#v\n", filename_field, options)
	filename := ReplaceAllNonAlpha(line[filename_field])
	// fmt.Printf("***> [%v]: %#v\n", filename, line)
	if filename == "" {
		err = fmt.Errorf("ERRO ao procurar filename linha [%#v], field [%v]\n", line, filename_field)
		return
	}
	b := &bytes.Buffer{}
	encod := charmap.ISO8859_1.NewEncoder()
	w = xw.OpenEncoding(b, "ISO-8859-1", encod, xw.WithIndentString("\t"))
	ec = &xw.ErrCollector{}
	defer ec.Panic()
	doc := xw.Doc{}
	w.StartDoc(doc)
	if true {
		ec.Do(
			w.StartDTD(xw.DTD{Name: "ADI", SystemID: systemID}),
			w.EndDTD(),
		)
		ec.Panic()
	}
	// fmt.Println("----------")
	var err3 error
	err2 := processMap(json, line)
	if err2 != nil {
		errs := ""
		for _, e := range err2 {
			errs = errs + fmt.Sprintf("[%s]\n", e)
		}
		err3 = fmt.Errorf("Erros ao processar linha [%#v]:\n\n%s----------", line, errs)
	}
	// ec.Do(
	// 	w.StartElem(xw.Elem{Name: "foo"}),
	// 	w.WriteAttr(xw.Attr{Name: "a1", Value: "val1"}),
	// 	w.WriteAttr(xw.Attr{Name: "a2", Value: "áá"}),
	// 	w.StartElem(xw.Elem{Name: "bar"}),
	// 	w.WriteAttr(xw.Attr{Name: "a1", Value: "val1"}),
	// 	w.WriteAttr(xw.Attr{Name: "a2", Value: "val2"}),
	// 	w.EndAllFlush(),
	// )
	w.EndAllFlush()
	// fmt.Println("----------")

	rightFile := filename + ".xml"
	wrongFile := filename + errSuffix + ".xml"
	fileOut := rightFile
	if err3 != nil {
		fileOut = wrongFile
	}
	os.Remove(rightFile)
	os.Remove(wrongFile)
	err = ioutil.WriteFile(fileOut, b.Bytes(), 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v\n", filename, err)
		return
	}
	if err3 != nil {
		err = err3
	}
	return
}

func processMap(json map[string]interface{}, line map[string]string) (err2 []error) {
	var name string
	name, hasName := json["name"].(string)
	if !hasName {
		if name != "" {
			fmt.Printf("name not found: [%s]\n", name)
		}
	}
	_, ok := json["single_attrs"]
	if hasName && !ok {
		w.StartElem(xw.Elem{Name: name})
	}
	commonAttrs, _ := json["common_attrs"].(map[string]interface{})

	for k, v := range json {
		switch vv := v.(type) {
		// case string:
		// 	fmt.Println(k, "is", vv)
		// case float64:
		// 	fmt.Println(k, "is", vv)
		case []map[string]interface{}:
			fmt.Println(k, ":")
			for _, u := range vv {
				err2 = append(err2, processMap(u, line)...)
			}
		case []interface{}:
			// fmt.Printf("%s:", k)
			// fmt.Printf("(%T)", v)
			// fmt.Println()
			// fmt.Printf("---> %s\n", name)
			switch k {
			case "attrs":
				err2 = appendErrors(err2, processAttrs(name, vv, line)...)
			case "single_attrs":
				err2 = appendErrors(err2, processSingleAttrs(name, vv, line, commonAttrs)...)
			case "elements":
				err2 = appendErrors(err2, processArray(name, vv, line)...)
			default:
				//fmt.Printf("[%s]\n", k)
			}
		case map[string]interface{}:
			switch k {
			case "options":
				processOptions(vv, line)
			}

		default:
			// fmt.Printf("\n%v is type %T\n", k, v)
		}
	}
	w.EndElem(name)
	return
}

func processOptions(json map[string]interface{}, line map[string]string) error {
	for k, v := range json {
		switch v.(type) {
		case string:
			options[k] = v.(string)
		default:
			return fmt.Errorf("Opcao tem que ser string, chave: [%s]", k)
		}
	}
	return nil
}

func processAttrs(nameElem string, json []interface{}, line map[string]string) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processAttr(vv, line)...)
		}
	}
	return
}

func processAttr(json map[string]interface{}, line map[string]string) (err []error) {
	var name string
	name, ok := json["Name"].(string)
	function, ok := json["function"].(string)
	if ok {
		procVals, err2 := Process(function, line, json, options)
		err = appendErrors(err, err2)
		for _, procVal := range procVals {
			w.WriteAttr(xw.Attr{Name: name, Value: procVal})
		}
	}
	return
}

func processSingleAttrs(name string, json []interface{}, line map[string]string, commonAttrs map[string]interface{}) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processSingleAttr(name, vv, line, commonAttrs))
		}
	}
	return
}

func processSingleAttr(nameElem string, json map[string]interface{}, line map[string]string, commonAttrs map[string]interface{}) (err error) {
	var name string
	name, ok := json["Name"].(string)
	value, ok := json["Value"].(string)
	function, ok := json["function"].(string)
	var procVals []string
	if ok {
		procVals, err = Process(function, line, json, options)
		for _, procVal := range procVals {
			w.StartElem(xw.Elem{Name: nameElem})
			for k, v := range commonAttrs {
				w.WriteAttr(xw.Attr{Name: k, Value: v.(string)})
			}
			w.WriteAttr(xw.Attr{Name: "Name", Value: name})
			w.WriteAttr(xw.Attr{Name: "Value", Value: procVal})
			w.EndElem(nameElem)
		}
		return
	}
	err = fmt.Errorf("Erro no atributo %s: %s", name, value)
	return
}

func processArray(nameElem string, json []interface{}, line map[string]string) (err2 []error) {
	//fmt.Printf(">>>%s<<<\n", nameElem)
	// w.StartElem(xw.Elem{Name: nameElem})
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processMap(vv, line)...)
		case []interface{}:
			err2 = appendErrors(err2, processArray("", vv, line)...)
		default:
			// fmt.Printf("\n%v is type %T\n", k, v)
		}
	}
	//w.EndElem(nameElem)
	return
}

func latinToUTF8(buffer []byte) string {
	buf := make([]rune, len(buffer))
	for i, b := range buffer {
		buf[i] = rune(b)
	}
	//fmt.Println(string(buf))
	return string(buf)
}

func logError(err error) {
	// phosphorize := ansi.ColorFunc("red")
	msg := fmt.Sprintf("ERRO: %v\n", err.Error())
	log(msg)
}

func log(msg string) {
	// phosphorize := ansi.ColorFunc("red")
	fmt.Fprintln(os.Stderr, msg)
}

func appendErrors(result []error, errors ...error) []error {
	for _, e := range errors {
		if e == nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

//func field(idx string)
