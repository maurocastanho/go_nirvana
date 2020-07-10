package main

import (
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"runtime/debug"
	"strings"

	"flag"

	"github.com/plandem/xlsx"
)

// Element types
const (
	singleT   = 0
	mapT      = iota
	arrayT    = iota
	emptyT    = iota
	mapNoArrT = iota
	mapArrayT = iota
)

// Json file types
const (
	assetsT = 0
	categsT = iota
	seriesT = iota
)

// Dateformat is the default date format
const dateformat = "01-02-06"

type lineT map[string]string

type jsonT map[string]interface{}
type optionsT map[string]string

// ElemType element types
type elemType int

// Writer controls output
type writer interface {
	Filename() string
	Suffix() string
	OpenOutput() error
	StartElem(string, elemType) error
	EndElem(string, elemType) error
	StartComment(string) error
	EndComment(string) error
	Write(string) error
	WriteAttr(string, string, string, string) error
	WriteAndClose(string) error
	WriteConsolidated(int) ([]byte, []byte, []byte, error)
	StartMap() error
	EndMap() error
}

var options map[string]string
var rs *reportSheet
var xlsFilePath string

const errSuffix = "_ERRO"

func main() {
	success := 0
	var err error
	defer func() {
		if msg := recover(); msg != nil {
			success = -4
			switch msg.(type) {
			case string:
				logError(fmt.Errorf("%v\n", msg))
			case error:
				logError(fmt.Errorf("%v:\n%v\n", msg, string(debug.Stack())))
			}
		}
		if err != nil {
			success = -3
			logError(err)
		}
		if success == 0 {
			log("--------------------------------------")
			log(" Processamento terminado com sucesso. ")
			log("--------------------------------------")
		} else {
			log("*******************************************")
			log("*    ATENCAO: ERROS NO PROCESSAMENTO      *")
			log("*******************************************")
		}

	}()

	log("-------------------------")
	log(" Iniciando processamento ")
	log("-------------------------")
	options = make(map[string]string)
	// defer func() { os.Exit(success) }()
	inputXls := ""
	confFile := ""
	outType := ""
	outDir := ""
	flag.StringVar(&inputXls, "xls", "", "Arquivo XLS de entrada")
	flag.StringVar(&confFile, "config", "", "Arquivo JSON de configuracao")
	flag.StringVar(&outType, "outtype", "xml", "Tipo de output (xml ou json). Default: xml")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
	flag.Parse()
	// test command line parameters
	if inputXls == "" {
		success = exitWithError("arquivo XLS deve ser especificado na linha de comando", 1)
		return
	}
	if confFile == "" {
		success = exitWithError("arquivo JSON de configuracao deve ser especificado na linha de comando", 1)
		return
	}
	if outType != "xml" && outType != "json" {
		success = exitWithError(fmt.Sprintf("tipo de arquivo de saida invalido: outType = [%s]", outType), 1)
		return
	}
	if outDir != "" {
		st, errS := os.Stat(outDir)
		if errS != nil || !st.IsDir() {
			logError(fmt.Errorf("diretorio [%s] nao e' valido", outDir))
			success = 1
			return
		}
	}
	// open input xls file and read main sheet
	var spreadSheet *xlsx.Spreadsheet
	if spreadSheet, err = xlsx.Open(inputXls); err != nil {
		return
	}
	defer closeSheet(spreadSheet)
	var lines []lineT
	lines, err = readSheetByName(spreadSheet, "dados")
	if err != nil {
		success = 1
		return
	}
	// read config file
	var json map[string]interface{}
	if json, err = readConfig(confFile); err != nil {
		success = 1
		return
	}
	// init option vars
	initVars(json)
	success, err = processSpreadSheet(json, outType, spreadSheet, outDir, lines)
	log(fmt.Sprintf("Gravado arquivo de report: %s", xlsFilePath))
}

func exitWithError(errMessage string, errCode int) int {
	logError(fmt.Errorf(errMessage))
	flag.Usage()
	return errCode
}

func processSpreadSheet(json map[string]interface{}, outType string, f *xlsx.Spreadsheet, outDir string, lines []lineT) (int, error) {
	filenameField, ok := options["filename_field"]
	if !ok || filenameField == "" {
		return 2, fmt.Errorf("ERRO ao procurar filename_field nas options [%#v]", options)
	}
	nameField, okN := options["name_field"]
	if !okN || nameField == "" {
		return 2, fmt.Errorf("ERRO ao procurar name_field nas options [%#v]", options)
	}

	// fmt.Printf("**> [%v]: %#v\n", filenameField, options)
	// fmt.Printf("***> [%v]: %#v\n", filename, line)

	// fmt.Printf("--> %v\n", objmap)
	success := 0
	lName := ""
	filePath := ""
	var curr lineT
	name := ""
	idField, _ := options["id_field"]
	categField1, _ := options["categ_field1"]
	categField2, _ := options["categ_field2"]
	var wrCategs *jsonWriter
	var wrSeries *jsonWriter
	var categLines []lineT
	var serieLines []lineT
	var err error
	if outType == "json" {
		// Read categories sheet for Box format
		if categLines, err = readSheetByName(f, "categories"); err != nil {
			return -1, err
		}
		if serieLines, err = readSheetByName(f, "series"); err != nil {
			return -1, err
		}
		// TODO usar createwriter
		if wrCategs, err = newJSONWriter(outDir, categLines, nil, categsT); err != nil {
			return -1, err
		}
		if wrSeries, err = newJSONWriter(outDir, nil, serieLines, seriesT); err != nil {
			return -1, err
		}
	}
	nLines := len(lines)
	log("------------------------------")
	log("Iniciando geracao de arquivos:")
	log("------------------------------")
	for i := 0; i < nLines; {
		log(fmt.Sprintf("Processando linha %d...", i+1))
		var pack []lineT
		// Groups lines with the same filename or empty filename
		j := i
		for ; j < nLines; j++ {
			curr = lines[j]
			name = curr[nameField]
			if lName == "" {
				lName = name
			}
			if name != "" && name != lName {
				break
			}
			pack = append(pack, curr)
		}
		i = j
		// fmt.Printf("== %v\n", pack)
		if filePath, err = replaceAllNonAlpha(path.Base(lName)); err != nil {
			return -1, err
		}
		if filePath = strings.TrimSuffix(filePath, path.Ext(filePath)); filePath == "" {
			logError(fmt.Errorf("ERRO ao procurar filename na linha [%#v], field [%v]", curr, filenameField))
			log("#ERRO FILENAME#")
			continue
		}
		filePath = path.Join(outDir, filePath)
		log(fmt.Sprintf("Arquivo: %s", filePath))
		var wr writer
		if wr, err = createWriter(outType, filePath, "", 0, 0,
			categLines, serieLines, assetsT); err != nil {
			return -1, err
		}
		log("Escrevendo " + filePath)
		if err = processLines(json, pack, wr); err != nil {
			// Do not stop: log error and continue to other files
			logError(err)
			success = -1
		}
		lName = name
		// publisher output
		if jsonXls, hasPubOutput := json["xls_output"]; hasPubOutput {
			JsonXlsMap := jsonXls.(map[string]interface{})
			var suc int
			xlsFilePath, suc, err = processPublisherXLS(JsonXlsMap, outDir, nLines, pack)
			if err != nil {
				return -1, err
			} else if suc != 0 {
				success = suc
			}
			if rs != nil {
				if err = rs.WriteAndClose(""); err != nil {
					return -1, err
				}
			}
		}
		log("------------------------------------")
	}
	if success == 0 {
		// Main file successfully processed, process other outputs
		if rs != nil {
			// publisher report
			if _, _, _, err = rs.WriteConsolidated(0); err != nil {
				return -1, err
			}
		}
		if wrCategs != nil || wrSeries != nil {
			// extra files
			suc, errors := processCategs(lines, categField1, wrCategs, idField, categField2)
			if len(errors) > 0 {
				return -1, errors[0] // TODO retornar todos os erros
			} else if suc != 0 {
				success = suc
			}
			if suc, errors = processSeries(lines, wrSeries, "id"); len(errors) > 0 {
				return -1, errors[0] // TODO retornar todos os erros
			} else if suc != 0 {
				success = suc
			}

			// categories.json
			if _, _, _, err = wrCategs.WriteConsolidated(1); err != nil {
				return -1, err
			}
			if _, _, _, err = wrSeries.WriteConsolidated(2); err != nil {
				return -1, err
			}
		}
	}
	return success, err
}

func processPublisherXLS(JsonXlsMap map[string]interface{}, outDir string, nLines int, pack []lineT) (string, int, error) {
	var err error
	success := 0
	xlsFile, ok := JsonXlsMap["filename"].(string)
	if !ok {
		return "", -1, fmt.Errorf("elemento 'filename' nao existe em xls_output no arquivo json")
	}
	jsonCols, okCols := JsonXlsMap["columns"].([]interface{})
	if !okCols {
		return "", -1, fmt.Errorf("elemento 'columns' nao existe em xls_output no arquivo json")
	}
	sheetName, okS := JsonXlsMap["sheet"].(string)
	if !okS {
		return "", -1, fmt.Errorf("elemento 'sheet' nao existe em xls_output no arquivo json")
	}
	xlsFilepath := path.Join(outDir, xlsFile)
	if rs == nil {
		nCols := len(jsonCols)
		if rs, err = newReportSheet(xlsFilepath, sheetName, nCols, nLines); err != nil {
			return xlsFilepath, -1, err
		}
		if err = rs.OpenOutput(); err != nil {
			return xlsFilepath, -1, err
		}
	}
	if errs := processAttrs("", jsonCols, pack, rs); len(errs) > 0 {
		success = -3
		for _, errA := range errs {
			log(fmt.Sprintf("ERRO: [%s]", errA.Error()))
		}
	}
	rs.newLine()
	return xlsFilepath, success, nil
}

func processCategs(pack []lineT, categField1 string, wrCateg *jsonWriter, idField string, categField2 string) (int, []error) {
	log("Processando categorias...")
	success := 0
	errors := make([]error, 0, 0)
	for k := range pack {
		row := pack[k]
		succ, err := wrCateg.processCategPack(row, idField, categField1, categField2)
		if err != nil {
			return succ, appendErrors(errors, err)
		}
		if succ != 0 {
			success = succ
		}
	}
	return success, errors
}

func processSeries(pack []lineT, wrSeries *jsonWriter, idField string) (int, []error) {
	log("Processando series...")
	errors := make([]error, 0, 0)
	for k := range pack {
		row := pack[k]
		success, err := wrSeries.processSeriesPack(row, idField, "Número do Episódio") // TODO parametrizar
		if err != nil {
			errors = appendErrors(errors, err)
			return success, errors
		}
	}
	wrSeries.cleanSeries()
	return 0, errors
}

// init option vars
func initVars(json map[string]interface{}) {
	initFunctions()
	options = make(map[string]string)
	opts := json["options"].([]interface{})
	for _, el := range opts {
		// fmt.Printf(">>>> %T\n", el)
		m := el.(map[string]interface{})
		name := m["Name"].(string)
		value := m["Value"].(string)
		options[name] = value
	}
	options["timestamp"] = timestamp()
}

// Reads the spreadsheet as an array of map[<line name>] = <value>
func readSheetByName(f *xlsx.Spreadsheet, sName string) ([]lineT, error) {
	header := make([]string, 0)
	// Get all the rows in the Sheet1.
	idx := 1
	var sheet xlsx.Sheet
	// reads sheet in stream mode (much faster)
	if sheet = f.SheetByName(sName, xlsx.SheetModeStream); sheet == nil {
		return nil, fmt.Errorf("aba nao existente na planilha: [%s]", sName)
	}
	return readSheet(sheet, header, idx)
}

// Returns true if a line has all fields blank
func blankLine(line map[string]string) bool {
	for k, v := range line {
		if k != "file_number" && v != "" {
			// fmt.Printf("===>>> %v - %v\n", k, v)
			return false
		}
	}
	return true
}

// contains tests if an array contains a given string
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// readSheet reads the spreadsheet as an array of map[<line name>] = <value>
func readSheet(sheet xlsx.Sheet, header []string, idx int) ([]lineT, error) {
	ncols, nrows := sheet.Dimension()
	empty := 0
	// row 0 == header
	var col int
	// Seeking last header column
	for col = ncols - 1; col >= 0; col-- {
		if colCell := sheet.Cell(col, 0); colCell.String() != "" {
			break
		}
	}
	lastCol := col
	// Reading Header
	for col = 0; col < lastCol+1; col++ {
		colCell := sheet.Cell(col, 0)
		hName := strings.TrimSpace(colCell.String())
		if contains(header, hName) {
			return nil, fmt.Errorf("header da planilha duplicado: [%s]", hName)
		}
		header = append(header, hName)
	}
	// Reading other lines
	lines := make([]lineT, 0)
	line := make(map[string]string)
	for row := 1; row < nrows && empty < 5; row++ {
		for c := 0; c < lastCol+1; c++ {
			colCell := sheet.Cell(c, row)
			cellF := ""
			x, err1 := colCell.Date()
			// TODO evitar o teste de prefixo
			if err1 != nil || !strings.HasPrefix(header[c], "Data") {
				f, err2 := colCell.Float()
				if err2 != nil || math.IsNaN(f) {
					cellF = strings.TrimSpace(colCell.String())
				} else {
					if math.Ceil(f) == math.Floor(f) {
						cellF = fmt.Sprintf("%d", int64(f))
					} else {
						cellF = fmt.Sprintf("%f", f)
					}
				}
			} else {
				cellF = x.UTC().Format(dateformat)
			}
			//fmt.Printf("+++> %s\n", cellF)
			line[header[c]] = cellF
			if c == 0 {
				if cellF != "" {
					empty = 0
				} else {
					empty++
				}
			}
		}
		if !blankLine(line) {
			line["file_number"] = fmt.Sprintf("%d", idx)
			lines = append(lines, line)
			line = make(map[string]string)
			idx++
		}
	}
	log(fmt.Sprintf("Aba [%s]: %d linhas, %d colunas. Lidas: %d linhas, %d colunas.",
		sheet.Name(), nrows, ncols, idx, lastCol+1))
	return lines, nil
}

// Reads config file
func readConfig(confFile string) (map[string]interface{}, error) {
	var err error
	var file *os.File
	var buf []byte
	if file, err = os.Open(confFile); err != nil {
		return nil, err
	}
	if buf, err = ioutil.ReadAll(file); err != nil {
		return nil, err
	}
	newBuf := latinToUTF8(buf)
	// Unmarshal JSON file to data structure
	var json map[string]interface{}
	if err = js.Unmarshal([]byte(newBuf), &json); err != nil {
		return nil, err
	}
	return json, nil
}

// Factory for creating the writer
func createWriter(outType string, filename string, sheetname string, ncols int, nlines int, linesCateg []lineT, linesSeries []lineT, jType int) (writer, error) {
	var err error
	var wr writer
	switch outType {
	case "xml":
		systemID, ok := options["doctype_system"]
		if !ok {
			return nil, fmt.Errorf("acrescente a opcao 'doctype_system' no arquivo de config")
		}
		wr, err = newXMLWriter(filename, systemID)
	case "json":
		wr, err = newJSONWriter(filename, linesCateg, linesSeries, jType)
	case "report":
		wr, err = newReportSheet(filename, sheetname, ncols, nlines)
	}
	return wr, err
}

// Process the config file against the lines of the sheet
func processLines(json jsonT, lines []lineT, wr writer) (err error) {
	if err = wr.OpenOutput(); err != nil {
		return err
	}
	// fmt.Println("----------")
	var err3 error
	if err2 := processMap(json, lines, wr); err2 != nil {
		errs := ""
		for _, e := range err2 {
			errs = errs + fmt.Sprintf("[%s]\n", e)
		}
		err3 = fmt.Errorf("Erros ao processar linha [%#v]:\n\n%s----------", lines, errs)
	}
	rightFile := wr.Filename() + wr.Suffix()             // filename in case of success
	wrongFile := wr.Filename() + errSuffix + wr.Suffix() // filename in case of errors
	fileOut := rightFile
	if err3 != nil {
		fileOut = wrongFile
	}
	// Remove previous file(s)
	_ = os.Remove(rightFile)
	_ = os.Remove(wrongFile)
	// Write new file
	if err1 := wr.WriteAndClose(fileOut); err1 != nil {
		err = err1
		return
	}
	if err3 != nil {
		err = err3
	}
	return
}

// Process a JSON map element
func processMap(json jsonT, lines []lineT, wr writer) (err2 []error) {
	var name string
	var hasName bool
	if name, hasName = json["Name"].(string); !hasName {
		if name != "" {
			log(fmt.Sprintf("name not found: [%s]\n", name))
		}
	}
	commonAttrs, _ := json["common_attrs"].(map[string]interface{})
	// Test if there is a filter expression
	if filter, ok := json["filter"].(string); ok {
		filterOk, err := evalCondition(filter, lines[0])
		if err != nil {
			err2 = append(err2, err)
			return
		}
		if !filterOk {
			// filter element
			return
		}
	}
	var elType elemType = mapT
	_, onlyValues := json["only_values"]
	_, noArr := json["no_array"]
	_, okValattr := json["elem_val"]
	sAux, okSattr := json["single_attrs"]
	el, okEl := json["elements"]
	okElArray := false
	var elements []interface{}
	if okEl {
		elements = el.([]interface{})
		if len(elements) == 0 {
			elType = emptyT
		} else {
			if okValattr {
				elType = arrayT
			} else if noArr {
				elType = mapNoArrT
			} else {
				elType = mapT
			}
		}
		//log(fmt.Sprintf("ELEMENTS %s [%v] %d", name, elements, len(elements)))
	} else {
		if el, okElArray = json["elements_array"]; okElArray {
			elements = el.([]interface{})
			elType = arrayT
		}
	}

	isMap := hasName && !okSattr && !onlyValues
	if isMap {
		err := wr.StartElem(name, elType)
		err2 = appendErrors(err2, err)
		if err != nil {
			return
		}
	}
	// Default Attributes
	if at, ok := json["attrs"]; ok {
		attrs := at.([]interface{})
		err2 = appendErrors(err2, processAttrs(name, attrs, lines, wr)...)
	}
	if atgr, ok := json["group_attrs"]; ok {
		attrs := atgr.([]interface{})
		err2 = appendErrors(err2, processGroupAttrs(name, attrs, lines, wr)...)
	}
	if okSattr {
		sAttrs := sAux.([]interface{})
		err2 = appendErrors(err2, processSingleAttrs(name, sAttrs, lines, commonAttrs, wr)...)
	}
	if len(elements) > 0 && (okEl || okElArray) {
		err2 = appendErrors(err2, processArray(name, elements, lines, wr)...)
	}
	// Comment section
	if co, ok := json["comments"]; ok {
		if err := wr.StartComment("DTH"); err != nil {
			err2 = appendErrors(err2, err)
			return
		}
		comm := co.([]interface{})
		err2 = appendErrors(err2, processSingleElements(name, comm, lines, wr)...)
		if err := wr.EndComment("DTH"); err != nil {
			err2 = appendErrors(err2, err)
			return
		}
	}
	// Process other elements
	for k, v := range json {
		// Ignore already processed elements
		switch k {
		case "attrs",
			"single_attrs",
			"elements",
			"elements_array",
			"comments":
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
	if isMap {
		if err := wr.EndElem(name, elType); err != nil {
			err2 = appendErrors(err2, err)
			return
		}
	}
	return
}

// Process option section in the JSON
func processOptions(json jsonT) error {
	for k, v := range json {
		switch vv := v.(type) {
		case string:
			options[k] = vv
		default:
			return fmt.Errorf("opcao tem que ser string, chave: [%s]", k)
		}
	}
	return nil
}

// Process attr element in the JSON
func processAttrs(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processAttr(vv, lines, wr)...)
			if _, okEl := vv["elements"]; okEl {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			if _, okElArr := vv["elements_array"]; okElArr {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
		}
	}
	return
}

// Process group of attrs in the JSON
func processGroupAttrs(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	if err := wr.StartElem("a", mapArrayT); err != nil {
		err2 = appendErrors(err2, err)
		return
	}
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			if _, okEl := vv["elements"]; okEl {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			if _, okElArr := vv["elements_array"]; okElArr {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			err2 = appendErrors(err2, processAttr(vv, lines, wr)...)
		}
	}
	err2 = appendErrors(err2, wr.EndElem("a", mapArrayT))
	return
}

// Process a simple value
func processVal(val string, vars map[string]string) string {
	if val == "" || val[0:1] != "$" {
		// value does not begin with "$": do nothing
		return val
	}
	result, ok := vars[val]
	if ok {
		// found a variable named val, return its value
		return result
	}
	// variable not found, returns original value
	return val
}

// populate option vars
func populateOptions(vars map[string]string, options map[string]string) {
	for key, val := range vars {
		options[key] = val
	}
}

// Process attr element
func processAttr(json jsonT, lines []lineT, wr writer) (err []error) {
	var name string
	name, _ = json["Name"].(string)
	attrType, _ := json["at_type"].(string)
	function, ok := json["function"].(string)
	if !ok {
		// element does not have "function" attribute
		return []error{fmt.Errorf("elemento [%s] nao tem 'function'", name)}
	}
	if function == "empty" {
		// function empty does nothing
		return
	}
	// process filter
	if filter, okFilter := json["filter"].(string); okFilter {
		// there is a filter expression: evaluate
		if filterAllow, err3 := evalCondition(filter, lines[0]); err3 != nil {
			// error in condition
			err = append(err, err3)
			return
		} else if !filterAllow {
			// filter expression excludes element: do nothing
			return
		}
	}
	// process function
	procVals, err2 := process(function, lines, json, options)
	err = appendErrors(err, err2)
	for _, procVal := range procVals {
		populateOptions(procVal.vars, options)
		isOtt := attrType == "ott"
		if isOtt {
			// Ott type open a new element, line <elem>x<elem>
			err = appendErrors(err, wr.StartElem(name, mapT))
		}
		if at, okA := json["attrs"]; okA {
			attrs := at.([]interface{})
			err = appendErrors(err, processAttrs(name, attrs, lines, wr)...)
		}
		if f2, okf2 := json["function2"]; !okf2 || f2 != "set_var" {
			vtype, _ := json["type"].(string)
			if err1 := wr.WriteAttr(name, processVal(procVal.val, procVal.vars), vtype, attrType); err1 != nil {
				err = appendErrors(err, err1)
			}
		}
		if isOtt {
			// Closes ott element
			err = appendErrors(err, wr.EndElem(name, mapT))
		}
	}
	return
}

// Process singleT attrs = one attr per line
func processSingleAttrs(name string, json []interface{}, lines []lineT, commonAttrs map[string]interface{}, wr writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, processSingleAttr(name, vv, lines, commonAttrs, wr))
		}
	}
	return
}

func processSingleAttr(nameElem string, json jsonT, lines []lineT, commonAttrs map[string]interface{}, wr writer) (err error) {
	_, okFil := json["filter"].(string)
	var function string
	var okFun bool
	if okFil {
		function, okFun = "filter", true
	} else {
		function, okFun = json["function"].(string)
	}
	var procVals []resultsT
	name, _ := json["Name"].(string)
	if !okFun {
		value, _ := json["Value"].(string)
		err = fmt.Errorf("erro no atributo %s, value [%s]", name, value)
	}
	procVals, err = process(function, lines, json, options)
	if err != nil {
		return err
	}
	var err2 error
	isOtt := false
	elType, okt := json["at_type"].(string)
	if okt {
		// tests if attribute is of ott type
		isOtt = elType == "ott"
	}
	done := false
	for _, procVal := range procVals {
		if isOtt {
			at, oka := json["attrs"]
			var attrs []interface{}
			if oka {
				attrs = at.([]interface{})
			}
			err2, done = writeElem(wr, attrs, lines, name, procVal.val)
		} else {
			vtype, _ := json["type"].(string)
			err2, done = writeAttr(wr, nameElem, commonAttrs, name, procVal.val, vtype, elType)
		}
		if done {
			return err2
		}
	}
	if attrs, ok := json["single_attrs"].([]interface{}); ok {
		processAttrs("", attrs, lines, wr)
		return nil
	}
	return
}

// Write element to output
func writeElem(wr writer, attrs []interface{}, lines []lineT, name string, procVal string) (error, bool) {
	processed := processAttrs(name, attrs, lines, wr)
	if len(processed) == 0 {
		return nil, false
	}
	err := wr.StartElem(name, singleT)
	if err != nil {
		return nil, true
	}
	var err3 []error
	err3 = appendErrors(err3, processed...)
	if len(err3) > 0 {
		return err3[0], true
	}
	if procVal != "" {
		err1 := wr.Write(procVal)
		if err1 != nil {
			_ = wr.EndElem(name, singleT)
			return err1, true
		}
	}
	err2 := wr.EndElem(name, singleT)
	return err2, false
}

func writeAttr(wr writer, nameElem string, commonAttrs map[string]interface{}, name string, procVal string, vtype string, attrType string) (error, bool) {
	err := wr.StartElem(nameElem, singleT)
	if err != nil {
		return nil, true
	}
	for k, v := range commonAttrs {
		err = wr.WriteAttr(k, v.(string), "string", "")
		if err != nil {
			return nil, true
		}
	}
	err = wr.WriteAttr("Name", name, "string", "")
	if err != nil {
		return nil, true
	}
	err = wr.WriteAttr("Value", procVal, vtype, attrType)
	if err != nil {
		return nil, true
	}
	err = wr.EndElem(nameElem, singleT)
	if err != nil {
		return nil, true
	}
	return nil, false
}

func processArray(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
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

func processSingleElements(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			if _, ok := vv["elements2"]; ok {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			err2 = appendErrors(err2, processSingleElement(vv, lines, wr)...)
		}
	}
	return
}

func processSingleElement(json jsonT, lines []lineT, wr writer) (errs []error) {
	var name string
	name, _ = json["Name"].(string)
	function, ok := json["function"].(string)
	if !ok {
		return []error{fmt.Errorf("elemento sem atributo 'function': [%s]", name)}
	}
	procVals, err := process(function, lines, json, options)
	if err != nil {
		return appendErrors(errs, err)
	}
	if err = wr.StartElem(name, singleT); err != nil {
		return appendErrors(errs, err)
	}
	for _, procVal := range procVals {
		if err1 := wr.Write(procVal.val); err1 != nil {
			_ = wr.EndElem(name, singleT)
			return appendErrors(errs, err1)
		}
	}
	if err = wr.EndElem(name, singleT); err != nil {
		return appendErrors(errs, err)
	}
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
	msg := fmt.Sprintf("ERRO: %v\n", err.Error())
	log(msg)
}

func log(msg string) {
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

func closeSheet(file *xlsx.Spreadsheet) {
	if err := file.Close(); err != nil {
		logError(err)
	}
}
