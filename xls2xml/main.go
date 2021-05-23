package main

import (
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"strconv"
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

type lineT struct {
	fields map[string]string
	idx    int
}

func newLineT(idx int) lineT {
	return lineT{fields: make(map[string]string), idx: idx}
}

type jsonT map[string]interface{}
type optionsT map[string]map[string]string

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
	Testing() bool
}

var options map[string]map[string]string
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
			log("")
			log("--------------------------------------")
			log(" Processamento terminado com sucesso. ")
			log("--------------------------------------")
		} else {
			log("*******************************************")
			log("*    ATENCAO: ERROS NO PROCESSAMENTO      *")
			log("*       VERIFIQUE MENSAGENS ACIMA         *")
			log("*******************************************")
		}
	}()

	log("-------------------------")
	log(" Iniciando processamento ")
	log("-------------------------")
	// defer func() { os.Exit(success) }()
	inputXls := ""
	confFile := ""
	outType := ""
	outDir := ""
	inputXlsCat := ""
	forceGenreCat := false
	flag.StringVar(&inputXls, "xls", "", "Arquivo XLS de entrada")
	flag.StringVar(&confFile, "config", "", "Arquivo JSON de configuracao")
	flag.StringVar(&outType, "outtype", "xml", "Tipo de output (xml ou json). Default: xml")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
	flag.StringVar(&inputXlsCat, "xlscat", "", "Arquivo Xls de categorias")
	flag.BoolVar(&forceGenreCat, "genrecat", false, "Forca a inserir Generos como categorias em caso de series")
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
	if outType == "json" {
		if inputXlsCat == "" {
			success = exitWithError("arquivo XLS de categorias deve ser especificado na linha de comando", 1)
			return
		}

	}
	log(fmt.Sprintf("Planilha de entrada: [%s]", inputXls))
	log(fmt.Sprintf("Arquivo config: [%s]", confFile))
	log(fmt.Sprintf("Diretorio de saida: [%s]", outDir))
	if outType == "json" {
		log(fmt.Sprintf("Planilha de categorias: [%s]", inputXlsCat))
	}
	log("-------------------------")

	// open input xls file and read main sheet
	var lines []lineT

	var spreadSheet *xlsx.Spreadsheet
	if spreadSheet, err = xlsx.Open(inputXls); err != nil {
		return
	}
	defer closeSheet(spreadSheet)
	lines, err = readSheetByName(spreadSheet, "dados")
	if err != nil {
		success = 1
		return
	}
	var linesCat []lineT
	if outType == "json" {
		var sheetCat *xlsx.Spreadsheet
		if sheetCat, err = xlsx.Open(inputXlsCat); err != nil {
			return
		}
		defer closeSheet(sheetCat)
		linesCat, err = readSheetByName(sheetCat, "categories")
		if err != nil {
			success = 1
			return
		}
	}
	// read config file
	var json map[string]interface{}
	if json, err = readConfig(confFile); err != nil {
		success = 1
		return
	}
	// init option vars
	initVars(json)
	var errs []error
	success, errs = processSpreadSheet(json, outType, spreadSheet, outDir, lines, linesCat, forceGenreCat)
	if len(errs) > 0 {
		for _, e := range errs {
			logError(e)
		}
		return
	}
	if success == 0 {
		log(fmt.Sprintf("Gravado arquivo de report: %s", xlsFilePath))
	}
}

func exitWithError(errMessage string, errCode int) int {
	logError(fmt.Errorf(errMessage))
	flag.Usage()
	return errCode
}

func processSpreadSheet(json map[string]interface{}, outType string, f *xlsx.Spreadsheet, outDir string, lines []lineT, linesCat []lineT, forceGenreCats bool) (success int, errs []error) {
	filenameField, okf := options["options"]["filename_field"]
	if !okf || filenameField == "" {
		return 2, []error{fmt.Errorf("ERRO ao procurar filename_field nas options [%#v]", options)}
	}
	nameField, okN := options["options"]["name_field"]
	if !okN || nameField == "" {
		return 2, []error{fmt.Errorf("ERRO ao procurar name_field nas options [%#v]", options)}
	}
	nameField = strings.ToLower(nameField)
	// fmt.Printf("**> [%v]: %#v\n", filenameField, options)
	// fmt.Printf("***> [%v]: %#v\n", filename, line)

	// fmt.Printf("--> %v\n", objmap)
	success = 0
	lName := ""
	filePath := ""
	var curr lineT
	name := ""
	idField, _ := options["options"]["id_field"]
	// TODO Check categ fields
	cField1 := strings.ToLower(options["options"]["categ_field1"])
	cField2 := strings.ToLower(options["options"]["categ_field2"])
	cField3 := strings.ToLower(options["options"]["categ_field3"])
	categFields := []string{cField1, cField2, cField3}
	var wrCategs *jsonWriter
	var wrSeries *jsonWriter
	var categLines []lineT
	var serieLines []lineT
	var err error
	if outType == "json" {
		// Read categories sheet for Box format
		categLines = linesCat
		if serieLines, err = readSheetByName(f, "series"); err != nil {
			return -1, []error{err}
		}
		if err = populateSerieIds(serieLines, options); err != nil {
			return -1, []error{err}
		}
		// TODO usar createwriter
		if wrCategs, err = newJSONWriter(outDir, categLines, serieLines, categsT); err != nil {
			return -1, []error{err}
		}
		if wrSeries, err = newJSONWriter(outDir, nil, serieLines, seriesT); err != nil {
			return -1, []error{err}
		}
	}
	nLines := len(lines)
	log("------------------------------")
	log("Iniciando geracao de arquivos:")
	log("------------------------------")
	var wr writer
	for i := 0; i < nLines; {
		log(fmt.Sprintf("Processando linha %d...", i+1))
		var pack []lineT
		// Groups lines with the same filename or empty filename
		j := i
		for ; j < nLines; j++ {
			curr = lines[j]
			name = curr.fields[nameField]
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
			return -1, []error{err}
		}
		if filePath = strings.TrimSuffix(filePath, path.Ext(filePath)); filePath == "" {
			logError(fmt.Errorf("ERRO ao procurar filename na linha [%#v], field [%v]", curr, filenameField))
			log("#ERRO FILENAME#")
			continue
		}
		filePath = path.Join(outDir, filePath)
		log(fmt.Sprintf("Arquivo: %s", filePath))
		if wr, err = createWriter(outType, filePath, "", 0, 0,
			categLines, serieLines, assetsT); err != nil {
			return -1, []error{err}
		}
		log("Escrevendo " + filePath)
		if errs = processAssets(json, pack, wr); len(errs) > 0 {
			// Do not stop: log error and continue to other files
			for _, e := range errs {
				logError(e)
			}
			success = -1
		}
		lName = name
		// publisher output
		if jsonXls, hasPubOutput := json["xls_output"]; hasPubOutput && success == 0 {
			JsonXlsMap := jsonXls.(map[string]interface{})
			var suc int
			if xlsFilePath, suc, errs = processPublisherXLS(JsonXlsMap, outDir, nLines, pack); len(errs) > 0 {
				return -1, errs
			} else if suc != 0 {
				success = suc
			}
			if rs != nil {
				if err = rs.WriteAndClose(""); err != nil {
					return -1, []error{err}
				}
			}
		}
		log("------------------------------------")
	}
	if wr != nil && wrCategs != nil {
		dirCategs := path.Dir(wrCategs.Filename())
		dirSeries := path.Dir(wrSeries.Filename())
		fileCateg := path.Join(dirCategs, "categories.json")
		fileSeries := path.Join(dirSeries, "series.json")
		if err = os.Remove(fileCateg); err != nil {
			switch err.(type) {
			case *os.PathError: // ok, file don't exist anyway
			default:
				return -1, []error{err}
			}
		}
		if err = os.Remove(fileSeries); err != nil {
			switch err.(type) {
			case *os.PathError: // ok, file don't exist anyway
			default:
				return -1, []error{err}
			}
		}
	}

	if success == 0 {
		// Main file successfully processed, process other outputs
		if rs != nil {
			// publisher report
			if _, _, _, err = rs.WriteConsolidated(assetsT); err != nil {
				return -1, []error{err}
			}
		}
		if wrCategs != nil || wrSeries != nil {
			// extra files
			suc, errors := processSeries(lines, wrSeries, "id")
			if len(errors) > 0 {
				return -1, errors
			} else if suc != 0 {
				success = suc
			}
			catSeason, ok := options["options"]["categ_season"]
			if !ok {
				return -1, []error{fmt.Errorf("categ_season nao encontrada em options no config")}
			}
			categSeason, errc := strconv.Atoi(catSeason)
			if errc != nil {
				return -1, []error{err}
			}
			suc, errors = processCategs(lines, wrCategs, wrSeries, idField, categFields, categSeason, forceGenreCats)
			if len(errors) > 0 {
				return -1, errors
			} else if suc != 0 {
				success = suc
			}
			// categories.json
			if _, _, _, err = wrCategs.WriteConsolidated(categsT); err != nil {
				return -1, []error{err}
			}
			// series.json
			if _, _, _, err = wrSeries.WriteConsolidated(seriesT); err != nil {
				return -1, []error{err}
			}
		}
	}
	return
}

func processPublisherXLS(JsonXlsMap map[string]interface{}, outDir string, nLines int, pack []lineT) (string, int, []error) {
	var err error
	success := 0
	xlsFile, ok := JsonXlsMap["filename"].(string)
	if !ok {
		return "", -1, []error{fmt.Errorf("elemento 'filename' nao existe em xls_output no arquivo json")}
	}
	jsonCols, okCols := JsonXlsMap["columns"].([]interface{})
	if !okCols {
		return "", -1, []error{fmt.Errorf("elemento 'columns' nao existe em xls_output no arquivo json")}
	}
	sheetName, okS := JsonXlsMap["sheet"].(string)
	if !okS {
		return "", -1, []error{fmt.Errorf("elemento 'sheet' nao existe em xls_output no arquivo json")}
	}
	xlsFilepath := path.Join(outDir, xlsFile)
	if rs == nil {
		nCols := len(jsonCols)
		if rs, err = newReportSheet(xlsFilepath, sheetName, nCols, nLines); err != nil {
			return xlsFilepath, -1, []error{err}
		}
		if err = rs.OpenOutput(); err != nil {
			return xlsFilepath, -1, []error{err}
		}
	}
	if errs := processAttrs("", jsonCols, pack, rs); len(errs) > 0 {
		success = -3
		return xlsFilepath, -1, errs
	}
	rs.newLine()
	return xlsFilepath, success, nil
}

func processCategs(lines []lineT, wrCateg *jsonWriter, wrSeries *jsonWriter, idField string, categFields []string, categSeason int, forceGenteCategs bool) (int, []error) {
	log("Processando categorias...")
	success := 0
	errors := make([]error, 0, 0)
	for k := range lines {
		succ, err := wrCateg.processCategPack(lines, k, idField, categFields, categSeason, wrSeries.serieLines, wrCateg.categLines, forceGenteCategs)
		if err != nil {
			return succ, appendErrors("", errors, err)
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
		success, err := wrSeries.processSeriesPack(&row, wrSeries.serieLines, idField, "Número do Episódio") // TODO parametrizar
		if err != nil {
			errors = appendErrors("", errors, err)
			return success, errors
		}
	}
	wrSeries.cleanSeries()
	return 0, errors
}

// init option vars
func initVars(json map[string]interface{}) {
	initFunctions()
	options = make(optionsT)
	options["options"] = make(map[string]string)
	opts := json["options"].([]interface{})
	for _, el := range opts {
		// fmt.Printf(">>>> %T\n", el)
		m := el.(map[string]interface{})
		name := m["Name"].(string)
		value := m["Value"].(string)
		options["options"][name] = value
	}
	options["options"]["timestamp"] = timestamp()
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
		colCell := sheet.Cell(col, 0)
		//fmt.Printf("%s, ", colCell.String())
		if colCell.String() != "" {
			break
		}
	}
	lastCol := col
	// Reading Header
	for col = 0; col < lastCol+1; col++ {
		colCell := sheet.Cell(col, 0)
		hName := strings.ToLower(strings.TrimSpace(colCell.String()))
		if contains(header, hName) {
			return nil, fmt.Errorf("header da planilha duplicado: [%s]", hName)
		}
		header = append(header, hName)
	}
	// Reading other lines
	lines := make([]lineT, 0)
	line := newLineT(1)
	for row := 1; row < nrows && empty < 5; row++ {
		for c := 0; c < lastCol+1; c++ {
			colCell := sheet.Cell(c, row)
			cellF := ""
			x, err1 := colCell.Date()
			// TODO evitar o teste de prefixo
			if err1 != nil || !strings.HasPrefix(header[c], "data") {
				f, err2 := colCell.Float()
				if err2 != nil || math.IsNaN(f) {
					st := colCell.String()
					re := regexp.MustCompile(`\r?\n`)
					st = re.ReplaceAllString(st, " ")
					cellF = strings.TrimSpace(st)
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
			line.fields[header[c]] = cellF
			if c == 0 {
				if cellF != "" {
					empty = 0
				} else {
					empty++
				}
			}
		}
		if !blankLine(line.fields) {
			line.fields["file_number"] = fmt.Sprintf("%d", idx)
			lines = append(lines, line)
			line = newLineT(row)
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
		systemID, ok := options["options"]["doctype_system"]
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
func processAssets(json jsonT, lines []lineT, wr writer) (errs []error) {
	if err := wr.OpenOutput(); err != nil {
		return appendErrors("", errs, err)
	}
	// fmt.Println("----------")
	errs = appendErrors("", errs, processMap(json, lines, wr)...)
	errors := len(errs) > 0
	rightFile := wr.Filename() + wr.Suffix()             // filename in case of success
	wrongFile := wr.Filename() + errSuffix + wr.Suffix() // filename in case of errors
	var fileOut string
	if !errors {
		fileOut = rightFile
	} else {
		fileOut = wrongFile
	}
	// Remove previous file(s)
	if !wr.Testing() {
		_ = os.Remove(rightFile)
		_ = os.Remove(wrongFile)
	}
	// Write new file
	if err1 := wr.WriteAndClose(fileOut); err1 != nil {
		return appendErrors("", errs, err1)
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
		filter2 := strings.ToLower(filter)
		filterOk, err := evalCondition(filter2, &lines[0])
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
		if err2 = appendErrors(name, err2, wr.StartElem(name, elType)); len(err2) > 0 {
			return
		}

		defer func() {
			err2 = appendErrors(name, err2, wr.EndElem(name, elType))
		}()
	}
	// Default Attributes
	if at, ok := json["attrs"]; ok {
		attrs := at.([]interface{})
		err2 = appendErrors(name, err2, processAttrs(name, attrs, lines, wr)...)
	}
	if atgr, ok := json["group_attrs"]; ok {
		attrs := atgr.([]interface{})
		err2 = appendErrors(name, err2, processGroupAttrs(name, attrs, lines, wr)...)
	}
	if okSattr {
		sAttrs := sAux.([]interface{})
		err2 = appendErrors(name, err2, processSingleAttrs(name, sAttrs, lines, commonAttrs, wr)...)
	}
	if len(elements) > 0 && (okEl || okElArray) {
		err2 = appendErrors(name, err2, processArray(name, elements, lines, wr)...)
	}
	// Comment section
	if co, ok := json["comments"]; ok {
		if err2 = appendErrors(name, err2, wr.StartComment("DTH")); len(err2) > 0 {
			return
		}
		comm := co.([]interface{})
		err2 = appendErrors(name, err2, processSingleElements(name, comm, lines, wr)...)
		if err2 = appendErrors(name, err2, wr.EndComment("DTH")); len(err2) > 0 {
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
				err2 = appendErrors(name, err2, processMap(u, lines, wr)...)
			}
		case []interface{}:
			// fmt.Printf("%s:", k)
			// fmt.Printf("(%T)", v)
			// fmt.Printf("---> %s\n", name)
		case map[string]interface{}:
			switch k {
			case "options":
				err2 = appendErrors(name, err2, processOptions(vv))
			}
		default:
			// fmt.Printf("\n%v is type %T\n", k, v)
		}
	}
	return
}

// Process option section in the JSON
func processOptions(json jsonT) error {
	for k, v := range json {
		switch vv := v.(type) {
		case string:
			options["options"][k] = vv
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
			err2 = appendErrors("", err2, processAttr(vv, lines, wr)...)
			if _, okEl := vv["elements"]; okEl {
				err2 = appendErrors("", err2, processMap(vv, lines, wr)...)
				continue
			}
			if _, okElArr := vv["elements_array"]; okElArr {
				err2 = appendErrors("", err2, processMap(vv, lines, wr)...)
				continue
			}
		}
	}
	return
}

// Process group of attrs in the JSON
func processGroupAttrs(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	if err := wr.StartElem("a", mapArrayT); err != nil {
		err2 = appendErrors("", err2, err)
		return
	}
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			if _, okEl := vv["elements"]; okEl {
				err2 = appendErrors("", err2, processMap(vv, lines, wr)...)
				continue
			}
			if _, okElArr := vv["elements_array"]; okElArr {
				err2 = appendErrors("", err2, processMap(vv, lines, wr)...)
				continue
			}
			err2 = appendErrors("", err2, processAttr(vv, lines, wr)...)
		}
	}
	err2 = appendErrors("", err2, wr.EndElem("a", mapArrayT))
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
func populateOptions(vars map[string]string, options optionsT, key string) {
	for name, val := range vars {
		options[key][name] = val
	}
}

// Process attr element
func processAttr(json jsonT, lines []lineT, wr writer) (errs []error) {
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
		filter = strings.ToLower(filter)
		// there is a filter expression: evaluate
		if filterAllow, err3 := evalCondition(filter, &lines[0]); err3 != nil {
			// error in condition
			errs = append(errs, err3)
			return
		} else if !filterAllow {
			// filter expression excludes element: do nothing
			return
		}
	}
	// process function
	procVals, err2 := process(function, lines, json, options)
	errs = appendErrors(name, errs, err2)
	for _, procVal := range procVals {
		populateOptions(procVal.vars, options, "options")
		isOtt := attrType == "ott"
		if isOtt {
			// Ott type open a new element, line <elem>x<elem>
			errs = appendErrors(name, errs, wr.StartElem(name, mapT))
		}
		if at, okA := json["attrs"]; okA {
			attrs := at.([]interface{})
			errs = appendErrors(name, errs, processAttrs(name, attrs, lines, wr)...)
		}
		if f2, okf2 := json["function2"]; !okf2 || f2 != "set_var" { // test set_var
			vtype, _ := json["type"].(string)
			if err1 := wr.WriteAttr(name, processVal(procVal.val, procVal.vars), vtype, attrType); err1 != nil {
				errs = appendErrors(name, errs, err1)
			}
		}
		if isOtt {
			// Closes ott element
			errs = appendErrors(name, errs, wr.EndElem(name, mapT))
		}
	}
	return
}

// Process singleT attrs = one attr per line
func processSingleAttrs(name string, json []interface{}, lines []lineT, commonAttrs map[string]interface{}, wr writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors("", err2, processSingleAttr(name, vv, lines, commonAttrs, wr)...)
		}
	}
	return
}

func processSingleAttr(nameElem string, json jsonT, lines []lineT, commonAttrs map[string]interface{}, wr writer) (errs []error) {
	_, okFil := json["filter"].(string)
	var function string
	var okFun bool
	if okFil {
		function, okFun = "filter", true
	} else {
		function, okFun = json["function"].(string)
	}
	function = strings.ToLower(function)
	var procVals []resultsT
	name, _ := json["Name"].(string)
	if !okFun {
		value, _ := json["Value"].(string)
		errs = []error{fmt.Errorf("erro no atributo %s, value [%s]", name, value)}
	}
	var err3 error
	if procVals, err3 = process(function, lines, json, options); err3 != nil {
		return appendErrors("", errs, err3)
	}
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
			errs, done = writeElem(wr, attrs, lines, name, procVal.val)
		} else {
			vtype, _ := json["type"].(string)
			errs, done = writeAttr(wr, nameElem, commonAttrs, name, procVal.val, vtype, elType)
		}
		if done {
			return errs
		}
	}
	if attrs, ok := json["single_attrs"].([]interface{}); ok {
		processAttrs("", attrs, lines, wr)
		return nil
	}
	return
}

// Write element to output
func writeElem(wr writer, attrs []interface{}, lines []lineT, name string, procVal string) (errs []error, done bool) {
	done = true
	var processed []error
	if processed = processAttrs(name, attrs, lines, wr); len(processed) > 0 {
		return processed, false
	}
	if errs = appendErrors(name, errs, wr.StartElem(name, singleT)); len(errs) > 0 {
		return
	}

	defer func() {
		if errs = appendErrors(name, errs, wr.EndElem(name, singleT)); len(errs) > 0 {
			done = true
		}
	}()

	if procVal != "" {
		if errs = appendErrors(name, errs, wr.Write(procVal)); len(errs) > 0 {
			return
		}
	}
	return nil, false
}

func writeAttr(wr writer, nameElem string, commonAttrs map[string]interface{}, name string, procVal string, vtype string, attrType string) (errs []error, done bool) {
	done = true
	if errs = appendErrors(name, errs, wr.StartElem(nameElem, singleT)); len(errs) > 0 {
		return
	}

	defer func() {
		errs = appendErrors(name, errs, wr.EndElem(nameElem, singleT))
	}()

	for k, v := range commonAttrs {
		if errs = appendErrors(name, errs, wr.WriteAttr(k, v.(string), "string", "")); len(errs) > 0 {
			return
		}
	}
	if errs = appendErrors(name, errs, wr.WriteAttr("Name", name, "string", "")); len(errs) > 0 {
		return
	}
	if errs = appendErrors(name, errs, wr.WriteAttr("Value", procVal, vtype, attrType)); len(errs) > 0 {
		return
	}
	return nil, false
}

func processArray(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	//fmt.Printf(">>>%s<<<\n", nameElem)
	// w.StartElem(xw.Elem{Name: nameElem})
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors("", err2, processMap(vv, lines, wr)...)
		case []interface{}:
			err2 = appendErrors("", err2, processArray("", vv, lines, wr)...)
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
				err2 = appendErrors("", err2, processMap(vv, lines, wr)...)
				continue
			}
			err2 = appendErrors("", err2, processSingleElement(vv, lines, wr)...)
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
	var procVals []resultsT
	var errsp error
	if procVals, errsp = process(function, lines, json, options); errsp != nil {
		return appendErrors("", errs, errsp)
	}
	if errs = appendErrors("", errs, wr.StartElem(name, singleT)); len(errs) > 0 {
		return
	}

	defer func() {
		errs = appendErrors("", errs, wr.EndElem(name, singleT))
	}()

	for _, procVal := range procVals {
		if errs = appendErrors("", errs, wr.Write(procVal.val)); len(errs) > 0 {
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
	msg := fmt.Sprintf("ERRO: %v", err.Error())
	log(msg)
}

func log(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
}

func appendErrors(name string, result []error, errors ...error) []error {
	for _, e := range errors {
		if e == nil {
			continue
		}
		if name == "" {
			result = append(result, e)
		} else {
			result = append(result, fmt.Errorf("[%s] %v", name, e.Error()))
		}
	}
	return result
}

func closeSheet(file *xlsx.Spreadsheet) {
	if err := file.Close(); err != nil {
		logError(err)
	}
}
