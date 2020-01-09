package main

import (
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	// "github.com/mgutz/ansi"
	"flag"
)

// Element types
const (
	SINGLE = 0
	MAP    = iota
	ARRAY  = iota
)

type lineT map[string]string

type jsonT map[string]interface{}
type optionsT map[string]string

// ElemType element types
type ElemType int

// Writer controls output
type Writer interface {
	Filename() string
	Suffix() string
	OpenOutput()
	StartElem(string, ElemType)
	EndElem(string)
	WriteAttr(string, string, string)
	WriteAndClose(string) error
}

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

var options map[string]string

const errSuffix = "_ERRO"

func main() {

	InitFunctions()

	inputXls := ""
	confFile := ""
	outType := ""
	outDir := ""
	flag.StringVar(&inputXls, "xls", "", "Arquivo XLS de entrada")
	flag.StringVar(&confFile, "config", "", "Arquivo JSON de configuracao")
	flag.StringVar(&outType, "outtype", "xml", "Tipo de output (xml ou json). Default: xml")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
	flag.Parse()

	if inputXls == "" {
		logError(fmt.Errorf("arquivo XLS deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if confFile == "" {
		logError(fmt.Errorf("arquivo JSON de configuracao deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if outType != "xml" && outType != "json" {
		logError(fmt.Errorf("tipo de arquivo de saida invalido: outType = [%s]", outType))
		flag.Usage()
		os.Exit(1)
	}

	if outDir != "" {
		st, err := os.Stat(outDir)
		if err != nil || !st.IsDir() {
			logError(fmt.Errorf("diretorio [%s] nao e' valido", outDir))
			os.Exit(1)
		}
	}

	f, err := excelize.OpenFile(inputXls)
	if err != nil {
		logError(err)
		os.Exit(2)
	}

	header := make([]string, 0)
	lines := make([]map[string]string, 0)
	// Get all the rows in the Sheet1.
	rows := f.GetRows("dados")
	idx := 1
	for i, row := range rows {
		if i == 0 {
			for _, colCell := range row {
				header = append(header, strings.TrimSpace(colCell))
			}
		} else {
			line := make(map[string]string)
			lines = append(lines, line)
			for i, colCell := range row {
				line[header[i]] = strings.TrimSpace(colCell)
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

	var json map[string]interface{}
	err = js.Unmarshal([]byte(newBuf), &json)
	if err != nil {
		logError(err)
		os.Exit(4)
	}

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
	filenameField, ok := options["filename_field"]
	if !ok || filenameField == "" {
		logError(fmt.Errorf("ERRO ao procurar filename_field nas options [%#v]", options))
		return
	}
	nameField, ok := options["name_field"]
	if !ok || nameField == "" {
		logError(fmt.Errorf("ERRO ao procurar name_field nas options [%#v]", options))
		return
	}
	// fmt.Printf("**> [%v]: %#v\n", filenameField, options)
	// fmt.Printf("***> [%v]: %#v\n", filename, line)

	// fmt.Printf("--> %v\n", objmap)
	success := 0
	lastName := ""
	filename := ""
	var curr lineT
	name := ""
	for i := 0; i < len(lines); {
		_, _ = fmt.Fprintf(os.Stderr, "Processando linha %d... ", i+1)
		var pack []lineT
		j := i
		// Groups lines with the same filename or empty filename
		for ; j < len(lines); j++ {
			curr = lines[j]
			name = curr[nameField]
			if lastName == "" {
				lastName = name
			}
			if name != "" && name != lastName {
				break
			}
			pack = append(pack, curr)
		}
		i = j
		// fmt.Printf("== %v\n", pack)
		filename = ReplaceAllNonAlpha(lastName)
		if filename == "" {
			logError(fmt.Errorf("ERRO ao procurar filename linha [%#v], field [%v]", curr, filenameField))
			log("#ERRO FILENAME#")
			continue
		}
		filename = path.Join(outDir, lastName)
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", filename)
		err = process(json, pack, createWriter(outType, filename))
		if err != nil {
			logError(err)
			success = -1
		}
		lastName = name
	}
	filename = path.Join(outDir, "planilha.xlsx")
	rs := NewReportSheet(filename)
	rs.OpenFile("teste")
	// err = process(json, pack, createWriter(outType, filename))
	// fmt.Fprintf(os.Stderr, "%s\n", filename)
	// if err != nil {
	// 	logError(err)
	// 	success = -1
	// }
	if success != 0 {
		log("*******************************************")
		log("*    ATENCAO: ERROS NO PROCESSAMENTO      *")
		log("*******************************************")
	}
	log("Fim.")
	os.Exit(success)
}

func createWriter(outType string, filename string) Writer {
	var writer Writer
	switch outType {
	case "xml":
		systemID := options["doctype_system"]
		writer = NewXMLWriter(filename, systemID)
	case "json":
		writer = NewJSONWriter(filename)
	}
	return writer
}

func process(json jsonT, lines []lineT, wr Writer) (err error) {
	wr.OpenOutput()
	// fmt.Println("----------")
	var err3 error
	err2 := processMap(json, lines, wr)
	if err2 != nil {
		errs := ""
		for _, e := range err2 {
			errs = errs + fmt.Sprintf("[%s]\n", e)
		}
		err3 = fmt.Errorf("Erros ao processar linha [%#v]:\n\n%s----------", lines, errs)
	}

	rightFile := wr.Filename() + wr.Suffix()
	wrongFile := wr.Filename() + errSuffix + wr.Suffix()
	fileOut := rightFile
	if err3 != nil {
		fileOut = wrongFile
	}
	_ = os.Remove(rightFile)
	_ = os.Remove(wrongFile)
	err1 := wr.WriteAndClose(fileOut)
	if err1 != nil {
		err = err1
		return
	}
	if err3 != nil {
		err = err3
	}
	return
}

func processMap(json jsonT, lines []lineT, wr Writer) (err2 []error) {
	var name string
	name, hasName := json["name"].(string)
	if !hasName {
		if name != "" {
			fmt.Printf("name not found: [%s]\n", name)
		}
	}
	commonAttrs, _ := json["common_attrs"].(map[string]interface{})
	filter, ok := json["filter"].(string)
	if ok {
		filterOk, err := EvalCondition(filter, lines[0])
		if err != nil {
			err2 = append(err2, err)
			return
		}
		if !filterOk {
			return
		}
	}
	if hasName && !ok {
		wr.StartElem(name, MAP)
	}

	at, ok := json["attrs"]
	if ok {
		attrs := at.([]interface{})
		err2 = appendErrors(err2, processAttrs(name, attrs, lines, wr)...)
	}
	sAux, ok := json["single_attrs"]
	if ok {
		sAttrs := sAux.([]interface{})
		err2 = appendErrors(err2, processSingleAttrs(name, sAttrs, lines, commonAttrs, wr)...)
	}
	el, ok := json["elements"]
	if ok {
		elements := el.([]interface{})
		err2 = appendErrors(err2, processArray(name, elements, lines, wr)...)
	}

	for k, v := range json {
		switch k {
		case "attrs":
		case "single_attrs":
		case "elements":
			continue
		}
		switch vv := v.(type) {
		// case string:
		// 	fmt.Println(k, "is", vv)
		// case float64:
		// 	fmt.Println(k, "is", vv)
		case []map[string]interface{}:
			fmt.Println(k, ":")
			for _, u := range vv {
				err2 = appendErrors(err2, processMap(u, lines, wr)...)
			}
		case []interface{}:
			// fmt.Printf("%s:", k)
			// fmt.Printf("(%T)", v)
			// fmt.Println()
			// fmt.Printf("---> %s\n", name)
		case map[string]interface{}:
			switch k {
			case "options":
				err2 = appendErrors(err2, processOptions(vv))
			}

		default:
			// fmt.Printf("\n%v is type %T\n", k, v)
		}
	}
	wr.EndElem(name)
	return
}

func processOptions(json jsonT) error {
	for k, v := range json {
		switch v := v.(type) {
		case string:
			options[k] = v
		default:
			return fmt.Errorf("opcao tem que ser string, chave: [%s]", k)
		}
	}
	return nil
}

func processAttrs(_ string, json []interface{}, lines []lineT, wr Writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processAttr(vv, lines, wr)...)
		}
	}
	return
}

func processAttr(json jsonT, lines []lineT, wr Writer) (err []error) {
	var name string
	name, _ = json["Name"].(string)
	function, ok := json["function"].(string)
	if ok {
		vtype, _ := json["type"].(string)
		procVals, err2 := Process(function, lines, json, options)
		err = appendErrors(err, err2)
		for _, procVal := range procVals {
			wr.WriteAttr(name, procVal, vtype)
		}
	}
	return
}

func processSingleAttrs(name string, json []interface{}, lines []lineT, commonAttrs map[string]interface{}, wr Writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processSingleAttr(name, vv, lines, commonAttrs, wr))
		}
	}
	return
}

func processSingleAttr(nameElem string, json jsonT, lines []lineT, commonAttrs map[string]interface{}, wr Writer) (err error) {
	var name string
	name, ok := json["Name"].(string)
	value, _ := json["Value"].(string)
	_, okf := json["filter"].(string)
	var function string
	if okf {
		function = "filter"
	} else {
		function, ok = json["function"].(string)
	}
	var procVals []string
	if ok {
		vtype, _ := json["type"].(string)
		procVals, err = Process(function, lines, json, options)
		for _, procVal := range procVals {
			wr.StartElem(nameElem, SINGLE)
			for k, v := range commonAttrs {
				wr.WriteAttr(k, v.(string), "string")
			}
			wr.WriteAttr("Name", name, "string")
			wr.WriteAttr("Value", procVal, vtype)
			wr.EndElem(nameElem)
		}
		return
	}
	err = fmt.Errorf("erro no atributo %s: %s", name, value)
	return
}

func processArray(_ string, json []interface{}, lines []lineT, wr Writer) (err2 []error) {
	//fmt.Printf(">>>%s<<<\n", nameElem)
	// w.StartElem(xw.Elem{Name: nameElem})
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processMap(vv, lines, wr)...)
		case []interface{}:
			err2 = appendErrors(err2, processArray("", vv, lines, wr)...)
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
	_, _ = fmt.Fprintln(os.Stderr, msg)
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
