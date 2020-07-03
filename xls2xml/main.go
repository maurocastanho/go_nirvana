package main

import (
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/plandem/xlsx"

	// "github.com/mgutz/ansi"
	"flag"
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
	WriteExtras()
	StartMap()
	EndMap()
}

var options map[string]string

const errSuffix = "_ERRO"

func main() {
	options = make(map[string]string)
	success := 0
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
		st, err := os.Stat(outDir)
		if err != nil || !st.IsDir() {
			logError(fmt.Errorf("diretorio [%s] nao e' valido", outDir))
			success = 1
			return
		}
	}

	f, err := xlsx.Open(inputXls)
	if err != nil {
		logError(err)
		success = 2
		return
	}
	defer closeSheet(f)
	lines := readSheetIdx(f, "dados")
	json := readConfig(confFile)
	initVars(json)
	success = processSpreadSheet(json, outType, f, outDir, lines, err)
	if success == 0 {
		log("------------------------------------")
		log("Processamento terminado com sucesso.")
		log("------------------------------------")
	} else {
		log("*******************************************")
		log("*    ATENCAO: ERROS NO PROCESSAMENTO      *")
		log("*******************************************")
	}
}

func exitWithError(errMessage string, errCode int) int {
	logError(fmt.Errorf(errMessage))
	flag.Usage()
	return errCode
}

func processSpreadSheet(json map[string]interface{}, outType string, f *xlsx.Spreadsheet, outDir string, lines []map[string]string, err error) int {
	filenameField, ok := options["filename_field"]
	if !ok || filenameField == "" {
		logError(fmt.Errorf("ERRO ao procurar filename_field nas options [%#v]", options))
		return 2
	}
	nameField, ok := options["name_field"]
	if !ok || nameField == "" {
		logError(fmt.Errorf("ERRO ao procurar name_field nas options [%#v]", options))
		return 2
	}
	idField, _ := options["id_field"]

	categField1, _ := options["categ_field1"]
	categField2, _ := options["categ_field2"]

	// fmt.Printf("**> [%v]: %#v\n", filenameField, options)
	// fmt.Printf("***> [%v]: %#v\n", filename, line)

	// fmt.Printf("--> %v\n", objmap)
	success := 0
	lName := ""
	filePath := ""
	var curr lineT
	name := ""

	jsonXls, okXls := json["xls_output"]
	JsonXlsMap := jsonXls.(map[string]interface{})
	var rs *reportSheet

	var wrCateg *jsonWriter
	if outType == "json" {
		categLines := readSheetIdx(f, "categories")
		wrCateg = newJSONWriter(outDir, categLines)
	}
	nLines := len(lines)
	log("------------------------------")
	log("Iniciando geracao de arquivos:")
	log("------------------------------")
	for i := 0; i < nLines; {
		log(fmt.Sprintf("Processando linha %d...", i+1))
		var pack []lineT
		j := i
		// Groups lines with the same filename or empty filename
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
		filePath = path.Base(lName)
		filePath = strings.TrimSuffix(filePath, path.Ext(filePath))
		filePath = replaceAllNonAlpha(filePath)
		if filePath == "" {
			logError(fmt.Errorf("ERRO ao procurar filename linha [%#v], field [%v]", curr, filenameField))
			log("#ERRO FILENAME#")
			continue
		}
		filePath = path.Join(outDir, filePath)
		log(fmt.Sprintf("Arquivo: %s", filePath))
		wr := createWriter(outType, filePath, "", 0, 0)
		err = processLines(json, pack, wr)
		if err != nil {
			logError(err)
			success = -1
		}
		lName = name

		// publisher output
		if okXls {
			jsonCols, okCols := JsonXlsMap["columns"].([]interface{})
			if !okCols {
				logError(fmt.Errorf("elemento 'columns' nao existe em xls_output no arquivo json"))
				success = -1
				break
			}
			xlsFile := JsonXlsMap["filename"].(string)
			if !ok {
				logError(fmt.Errorf("elemento 'filename' nao existe em xls_output no arquivo json"))
				success = -1
				break
			}
			sheetName := JsonXlsMap["sheet"].(string)
			if !ok {
				logError(fmt.Errorf("elemento 'sheet' nao existe em xls_output no arquivo json"))
				success = -1
				break
			}
			xlsFilepath := path.Join(outDir, xlsFile)
			if rs == nil {
				nCols := len(jsonCols)
				rs = newReportSheet(xlsFilepath, sheetName, nCols, nLines)
				err = rs.OpenOutput()
				if err != nil {
					logError(err)
					break
				}
			}
			errs := processAttrs("", jsonCols, pack, rs)
			if len(errs) > 0 {
				for _, errA := range errs {
					log(fmt.Sprintf("ERRO: [%s]", errA.Error()))
				}
			}
			rs.newLine()
			//_, err = fmt.Fprintf(os.Stderr, "%s\n", filename)
			//if err != nil {
			//	logError(err)
			//	success = -1
			//}
		}
		if rs != nil && okXls {
			log("Escrevendo " + filePath)
			err = rs.WriteAndClose("")
			if err != nil {
				logError(err)
				success = -1
			}
		}
		if wrCateg != nil {
			log("Processando categorias...")
			for k := range pack {
				categ1 := pack[k][categField1]
				if categ1 == "" {
					success = -2
					logError(fmt.Errorf("categoria 1 em branco na linha [%v]", pack[k]))
				}
				err = wrCateg.addAsset(pack[k][idField], categ1)
				if err != nil {
					logError(err)
					success = -1
				}
				categ2 := pack[k][categField2]
				if categ1 == "" {
					success = -2
					logError(fmt.Errorf("categoria 2 em branco na linha [%v]", pack[k]))
				}
				err = wrCateg.addAsset(pack[k][idField], categ2)
				if err != nil {
					logError(err)
					success = -1
				}
			}
		}
		log("------------------------------------")
	}
	if success == 0 {
		if rs != nil {
			rs.WriteExtras()
		}
		if wrCateg != nil {
			wrCateg.WriteExtras()
		}
	}
	return success
}

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
func readSheetIdx(f *xlsx.Spreadsheet, sName string) []map[string]string {
	header := make([]string, 0)
	lines := make([]map[string]string, 0)
	// Get all the rows in the Sheet1.
	idx := 1
	sheet := f.SheetByName(sName)
	//redBold := styles.New(
	//	styles.NumberFormatID(15),
	//)
	//
	//// Add formatting to xlsx
	//styleID := f.AddStyles(redBold)

	// Removes blank lines at the end of sheet
	lines = readSheet(sheet, header, lines, idx)
	lenLines := len(lines)
	lastLine := lenLines - 1
	for ; lastLine >= 0; lastLine-- {
		if !blankLine(lines[lastLine]) {
			fmt.Printf("input: %d lines\n", lastLine+1)
			break
		}
	}
	return lines[:lastLine+1]
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

// Reads the spreadsheet as an array of map[<line name>] = <value>
func readSheet(sheet xlsx.Sheet, header []string, lines []map[string]string, idx int) []map[string]string {
	ncols, nrows := sheet.Dimension()
	empty := 0
	for row := 0; row < nrows && empty < 10; row++ {
		if row == 0 {
			// row 0 == header
			var col int
			for col = 0; col < ncols; col++ {
				colCell := sheet.Cell(col, row)
				if colCell.String() == "" {
					break
				}
				header = append(header, strings.TrimSpace(colCell.String()))
			}
			ncols = col
		} else {
			line := make(map[string]string)
			lines = append(lines, line)
			for col := 0; col < ncols; col++ {
				colCell := sheet.Cell(col, row)
				cellF := ""
				x, err1 := colCell.Date()
				// TODO evitar o teste de prefixo
				if err1 != nil || !strings.HasPrefix(header[col], "Data") {
					cellF = strings.TrimSpace(colCell.String())
				} else {
					cellF = x.UTC().Format(dateformat)
				}
				//fmt.Printf("+++> %s\n", cellF)
				line[header[col]] = cellF
				if col == 0 {
					if cellF != "" {
						empty = 0
					} else {
						empty++
					}
				}
			}
			line["file_number"] = fmt.Sprintf("%d", idx)
			idx++
		}
	}
	return lines
}

// Reads config file
func readConfig(confFile string) map[string]interface{} {
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
	return json
}

// Factory for creating the writer
func createWriter(outType string, filename string, sheetname string, ncols int, nlines int) writer {
	var wr writer
	switch outType {
	case "xml":
		systemID := options["doctype_system"]
		wr = newXMLWriter(filename, systemID)
	case "json":
		wr = newJSONWriter(filename, nil)
	case "report":
		wr = newReportSheet(filename, sheetname, ncols, nlines)
	}
	return wr
}

// Process the config file against the lines of the sheet
func processLines(json jsonT, lines []lineT, wr writer) (err error) {
	err = wr.OpenOutput()
	if err != nil {
		return err
	}
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

// Process a JSON map element
func processMap(json jsonT, lines []lineT, wr writer) (err2 []error) {
	var name string
	name, hasName := json["Name"].(string)
	if !hasName {
		if name != "" {
			log(fmt.Sprintf("name not found: [%s]\n", name))
		}
	}
	commonAttrs, _ := json["common_attrs"].(map[string]interface{})
	filter, okFilter := json["filter"].(string)
	if okFilter {
		filterOk, err := evalCondition(filter, lines[0])
		if err != nil {
			err2 = append(err2, err)
			return
		}
		if !filterOk {
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
		el, okElArray = json["elements_array"]
		if okElArray {
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

	at, ok := json["attrs"]
	if ok {
		attrs := at.([]interface{})
		err2 = appendErrors(err2, processAttrs(name, attrs, lines, wr)...)
	}
	atgr, ok := json["group_attrs"]
	if ok {
		attrs := atgr.([]interface{})
		err2 = appendErrors(err2, processGroupAttrs(name, attrs, lines, wr)...)
	}
	if okSattr {
		sAttrs := sAux.([]interface{})
		err2 = appendErrors(err2, processSingleAttrs(name, sAttrs, lines, commonAttrs, wr)...)
	}
	if len(elements) > 0 {
		if okEl || okElArray {
			err2 = appendErrors(err2, processArray(name, elements, lines, wr)...)
		}
	}

	co, okComm := json["comments"]
	if okComm {
		err := wr.StartComment("DTH")
		if err != nil {
			err2 = appendErrors(err2, err)
			return
		}
		comm := co.([]interface{})
		err2 = appendErrors(err2, processSingleElements(name, comm, lines, wr)...)
		err = wr.EndComment("DTH")
		if err != nil {
			err2 = appendErrors(err2, err)
			return
		}
	}

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
	if isMap {
		err := wr.EndElem(name, elType)
		if err != nil {
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
			_, okEl := vv["elements"]
			if okEl {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			_, okElArr := vv["elements_array"]
			if okElArr {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
		}
	}
	return
}

// Process group of attrs in the JSON
func processGroupAttrs(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	err2 = appendErrors(err2, wr.StartElem("a", mapArrayT))
	if len(err2) > 0 {
		return
	}
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			_, okEl := vv["elements"]
			if okEl {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			_, okElArr := vv["elements_array"]
			if okElArr {
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
		return val
	}
	result, ok := vars[val]
	if ok {
		return result
	}
	return val
}

// Process option vars
func processVars(vars map[string]string, options map[string]string) {
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
	if !ok || function == "empty" {
		return
	}
	filter, okFilter := json["filter"].(string)
	if okFilter {
		filterOk, err3 := evalCondition(filter, lines[0])
		if err3 != nil {
			err = append(err, err3)
			return
		}
		if !filterOk {
			return
		}
	}

	vtype, _ := json["type"].(string)
	procVals, err2 := process(function, lines, json, options)
	err = appendErrors(err, err2)
	for _, procVal := range procVals {
		processVars(procVal.vars, options)
		isOtt := attrType == "ott"
		if isOtt {
			err = appendErrors(err, wr.StartElem(name, mapT))
		}
		at, okA := json["attrs"]
		if okA {
			attrs := at.([]interface{})
			err = appendErrors(err, processAttrs(name, attrs, lines, wr)...)
		}
		var err1 error
		f2, okf2 := json["function2"]
		if !okf2 || f2 != "set_var" {
			err1 = wr.WriteAttr(name, processVal(procVal.val, procVal.vars), vtype, attrType)
		}
		if err1 != nil {
			err = appendErrors(err, err1)
		}
		if isOtt {
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
	var name string
	name, ok := json["Name"].(string)
	_, okf := json["filter"].(string)
	var filterFunc string
	if okf {
		filterFunc = "filter"
	} else {
		filterFunc, ok = json["function"].(string)
	}
	isOtt := false
	elType, okt := json["at_type"].(string)
	if okt {
		isOtt = elType == "ott"
	}

	var procVals []resultsT
	if ok {
		vtype, _ := json["type"].(string)
		procVals, err = process(filterFunc, lines, json, options)
		if err != nil {
			return err
		}
		done := false
		var err2 error
		for _, procVal := range procVals {
			if isOtt {
				at, oka := json["attrs"]
				var attrs []interface{}
				if oka {
					attrs = at.([]interface{})
				}
				err2, done = writeElem(wr, attrs, lines, name, procVal.val)
			} else {
				err2, done = writeAttr(wr, nameElem, commonAttrs, name, procVal.val, vtype, elType)
			}
			if done {
				return err2
			}
		}
		attrs, oka := json["single_attrs"].([]interface{})
		if oka {
			processAttrs("", attrs, lines, wr)
			return nil
		}

		return
	}
	value, _ := json["Value"].(string)
	err = fmt.Errorf("erro no atributo %s, value [%s]", name, value)
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
			_, ok := vv["elements2"]
			if ok {
				err2 = appendErrors(err2, processMap(vv, lines, wr)...)
				continue
			}
			err2 = appendErrors(err2, processSingleElement(vv, lines, wr)...)
		}
	}
	return
}

func processSingleElement(json jsonT, lines []lineT, wr writer) (err []error) {
	var name string
	name, _ = json["Name"].(string)
	function, ok := json["function"].(string)
	if ok {
		procVals, err2 := process(function, lines, json, options)
		err = appendErrors(err, err2)
		err2 = wr.StartElem(name, singleT)
		if err2 != nil {
			err = appendErrors(err, err2)
			return
		}
		for _, procVal := range procVals {
			err1 := wr.Write(procVal.val)
			if err1 != nil {
				err = appendErrors(err, err1)
				_ = wr.EndElem(name, singleT)
				return
			}
		}
		err2 = wr.EndElem(name, singleT)
		if err2 != nil {
			err = appendErrors(err, err2)
			return
		}
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

func closeSheet(file *xlsx.Spreadsheet) {
	err := file.Close()
	if err != nil {
		logError(err)
	}
}
