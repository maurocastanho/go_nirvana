package main

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Knetic/govaluate"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

//const (
//	//ISO8601 is format for ISO8601 dates. Like RFC3339, but without timezone
//	ISO8601 = "2006-01-02T15:04:05"
//)

// errorMessage Default error message
type resultsT struct {
	val  string
	vars map[string]string
}

func newResult(val string) resultsT {
	var result resultsT
	result.val = val
	result.vars = make(map[string]string)
	return result
}

func newResultVars(val string, key string, value string) resultsT {
	var result resultsT
	result.val = val
	result.vars = make(map[string]string)
	result.vars[key] = value
	return result
}

var errorMessage = []resultsT{newResult("#ERRO#")}

// FunctionDict is the relation between the operation name and the function
var functionDict map[string]func(string, lineT, jsonT, optionsT) ([]resultsT, error)

// InitFunctions maps the user functions
func initFunctions() {
	functionDict = map[string]func(string, lineT, jsonT, optionsT) ([]resultsT, error){
		"assetid":         assetID,
		"assetid_ott":     assetIDOtt,
		"attr_map":        attrMap,
		"box_technology":  boxTechnology,
		"condition":       condition,
		"convert":         convert,
		"convert_date":    convertDate,
		"date":            date,
		"date_ott":        dateRFC3339,
		"empty":           emptyFunc,
		"episode_id":      episodeID,
		"eval":            eval,
		"field":           fieldTrunc,
		"field_date":      fieldDate,
		"field_money":     fieldMoney,
		"field_no_quotes": fieldNoQuotes,
		"field_noacc":     fieldNoAccents,
		"field_raw":       fieldRaw,
		"field_suffix":    suffix,
		"field_trim":      fieldTrim,
		"field_validated": fieldValidated,
		"filter":          filterCondition,
		"first_name":      firstName,
		"fixed":           fixed,
		"janela_repasse":  janelaRepasse,
		"last_name":       lastName,
		"map":             mapField,
		"middle_name":     middleName,
		"option":          option,
		"seconds":         seconds,
		"set_var":         setVar,
		"split":           split,
		"surname_name":    surnameName,
		"timestamp":       utc,
		"uuid":            genUUID,
	}
}

// Process processLines one element from json config
func process(funcName string, lines []lineT, json jsonT, options optionsT) ([]resultsT, error) {

	// fmt.Printf("=> %s\n", funcName)
	if funcName == "" {
		return errorMessage, fmt.Errorf("'function' nao especificada")
	}
	function, ok := functionDict[funcName]
	if !ok {
		fmt.Printf("Warning: funcao [%s] nao existe!\n", funcName)
		result, _ := undefined("", nil, json, options)
		return result, fmt.Errorf("funcao nao definida: [%s]", funcName)
	}
	result := make([]resultsT, 0)
	for _, line := range lines {
		res, err := function("", line, json, options)
		if err != nil {
			return res, err
		}
		result = append(result, res...)
	}
	return result, nil
}

//func findValue(value string, field string, json jsonT) (string, error) {
//	if value == "" {
//		return getValue(field, json)
//	}
//	return value, nil
//}

// Fixed returns the same value
func fixed(value string, _ lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	val, _ := json["Value"].(string)
	//if val != "" && val[0:1] == "$" {
	//	result := options[val]
	//	return NewResult([]string{result}), nil
	//}
	val, err := getValue("Value", json)
	return []resultsT{newResult(val)}, err
}

// FieldMoney returns field formatted as money
func fieldMoney(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	val, err := getField(value, "Name", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	result, err := formatMoney(val)
	if err != nil {
		return errorMessage, err
	}
	return []resultsT{newResult(result)}, nil
}

// Field returns field from line after truncating max size
func fieldTrunc(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	value, err := getField(value, "Name", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	value, err = truncate(value, line, json, options)
	return []resultsT{newResult(value)}, err
}

// FieldRaw returns field without further processing
func fieldRaw(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	value, err := getField(value, "Name", line, json, options)
	return []resultsT{newResult(value)}, err
}

// FieldDate returns a date field after formatting
func fieldDate(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	value, err := getField(value, "Name", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	t, err2 := parseDate(value)
	if err2 != nil {
		fieldName, _ := getValue("field", json)
		err3 := fmt.Errorf("erro no campo '%s': [%s]", fieldName, err2.Error())
		return errorMessage, err3
	}
	value = formatDate(t)
	return []resultsT{newResult(value)}, err
}

func getField(value string, _ string, line lineT, json jsonT, _ optionsT) (string, error) {
	// fmt.Printf("field=%#v, json=%#v, line=%#v, options=%#v\n", field, json, line, options)
	if value != "" {
		return value, nil
	}
	fieldName, err := getValue("field", json)
	if err != nil {
		return errorMessage[0].val, err
	}
	value, ok := line[fieldName]
	if !ok {
		return fieldName, fmt.Errorf("elemento '%s' inexistente na linha", fieldName)
	}
	// fmt.Printf("field(%v) = [%v]\n", field, value)
	return value, nil
}

// FieldValidated validates a field against a list and returns the value if valid
func fieldValidated(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	f, err := fieldTrunc(value, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	val, err := getValue("Options", json)
	if err != nil {
		return errorMessage, err
	}
	opts := strings.Split(val, ",")
	for _, opt := range opts {
		if strings.TrimSpace(opt) == f[0].val {
			return f, nil
		}
	}
	return errorMessage, fmt.Errorf("falha na validacao do elemento '%s': %s, valores possiveis: %v", json["Name"], f, opts)
}

// FieldNoAccents returns the field after replacing accented characters for its non-accented correspondents
func fieldNoAccents(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	f, err := getField(value, "Name", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	result, err := removeAccents(f)
	if err != nil {
		return errorMessage, err
	}
	result, err = truncate(result, line, json, options)
	return []resultsT{newResult(result)}, err
}

// FieldTrim returns the field after removing spaces from left and right
func fieldTrim(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	result := strings.TrimSpace(field)
	result, err = truncate(result, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	return []resultsT{newResult(result)}, nil
}

// FieldNoQuotes removes all quotation symbols from the field and returns it
func fieldNoQuotes(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	value = removeQuotes(field)
	result, err := truncate(value, line, json, options)
	return []resultsT{newResult(result)}, err
}

// Suffix removes the extension and appends a suffix to a field, returning the result
func suffix(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	field, err := fieldTrunc(value, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	// fmt.Printf("-->> %v\n", field)
	noacc, err := removeAccents(field[0].val)
	if err != nil {
		return errorMessage, err
	}
	suf, _ := json["suffix"].(string)
	if suf == "" {
		suf = path.Ext(noacc)
	}
	extIdx := strings.LastIndex(noacc, ".")
	if extIdx > 0 {
		noacc = noacc[0:extIdx]
	}
	prefix, _ := line["subpasta"]
	if prefix != "" {
		prefix += "/"
	}
	val := replaceAllNonAlpha(noacc)
	val, err = truncateSuffix(val, suf, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	result := fmt.Sprintf("%s%s%s", prefix, val, suf)
	//	fmt.Printf("-->> %v\n", result)
	return []resultsT{newResult(result)}, nil
}

// AssetID returns the Asset ID
func assetID(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	fProvider, ok := json["prefix"].(string)
	if !ok || fProvider == "" {
		return errorMessage, fmt.Errorf("field prefixo do Asset ID nao encontrado (prefix): [%v]", json)
	}
	prov, ok := line[fProvider]
	if !ok || prov == "" {
		return errorMessage, fmt.Errorf("provider assetid nao encontrado (provider): [%v]", line)
	}
	provider := strings.ToUpper(removeSpaces(prov))
	leng := len(provider)
	if leng > 4 {
		provider = provider[0:4]
	} else {
		// repeats last character until length = 4
		last := provider[leng-1]
		for i := leng; i < 4; i++ {
			provider += string([]byte{last})
		}
	}
	suffixF, ok := json["suffix_number"].(float64)
	if !ok {
		return errorMessage, fmt.Errorf("numero do sufixo do assetid (suffix_number) nao encontrado: [%v]", json)
	}
	suf := int(suffixF)
	timest := options["timestamp"]
	if !ok || timest == "" {
		return errorMessage, fmt.Errorf("timestamp nao encontrada (timestamp): [%v]", options)
	}
	fileNum, ok := line["file_number"]
	if !ok || fileNum == "" {
		return errorMessage, fmt.Errorf("numero do arquivo nao encontrado (file_number): [%v]", line)
	}
	result := fmt.Sprintf("%s%d%s%03s", provider, suf, timest, fileNum)
	return []resultsT{newResult(result)}, nil
}

// AssetIDOtt returns the Asset ID for the ott format
func assetIDOtt(value string, line lineT, _ jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	timest, ok := options["timestamp"]
	if !ok || timest == "" {
		return errorMessage, fmt.Errorf("timestamp nao encontrada (timestamp): [%v]", options)
	}
	fileNum, ok := line["file_number"]
	if !ok || fileNum == "" {
		return errorMessage, fmt.Errorf("numero do arquivo nao encontrado (file_number): [%v]", line)
	}
	result := fmt.Sprintf("%s%03s", timest, fileNum)
	return []resultsT{newResult(result)}, nil
}

// EpisodeID returns the Episode ID, make of (season number | episode number)
func episodeID(value string, line lineT, _ jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	fSeason, ok := options["season_field"]
	if !ok || fSeason == "" {
		return errorMessage, fmt.Errorf("config para campo 'season_field' nao encontrado: [%v]", options)
	}
	season, ok := line[fSeason]
	if !ok || season == "" {
		return errorMessage, fmt.Errorf("temporada do episode_id nao encontrada (%v): [%v]", fSeason, line)
	}
	fEpisodeID := options["episode_field"]
	if !ok || fEpisodeID == "" {
		return errorMessage, fmt.Errorf("config para campo 'episode_field' nao encontrado: [%v]", options)
	}
	episode, ok := line[fEpisodeID]
	if !ok || season == "" {
		return errorMessage, fmt.Errorf("valor do episode_id nao encontrado (%v): [%v]", fEpisodeID, line)
	}
	result := fmt.Sprintf("%02s%03s", season, episode)
	return []resultsT{newResult(result)}, nil
}

// Date returns the present date, formatted
func date(value string, _ lineT, _ jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	crDate := options["creationDate"]
	if crDate != "" {
		return []resultsT{newResult(crDate)}, nil
	}
	return []resultsT{newResult(formatDate(time.Now()))}, nil
}

// EmptyFunc returns always a empty value
func emptyFunc(_ string, _ lineT, _ jsonT, _ optionsT) ([]resultsT, error) {
	return []resultsT{newResult("")}, nil
}

// SetVar sets a variable in the options
func setVar(value string, _ lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	name, ok := json["var"].(string)
	if !ok || name == "" {
		return errorMessage, fmt.Errorf("campo 'var' nao encontrado: [%v]", json)
	}
	return []resultsT{newResultVars("", "$"+name, value)}, nil
}

// ConvertDate converts a date string from the mm/dd/yy format to the default format
func convertDate(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	field, err := fieldTrunc(value, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	t, err := time.Parse("01/02/06", field[0].val)
	if err != nil {
		return errorMessage, err
	}
	return []resultsT{newResult(formatDate(t))}, nil
}

// DateRFC3339 converts a date string from the mm/dd/yy format to the RFC3339 format
func dateRFC3339(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	field, err := fieldTrunc(value, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	val := field[0].val
	t, err2 := time.Parse("01-02-06", val)
	if err2 != nil {
		return errorMessage, err2
	}
	return []resultsT{newResult(t.Format(time.RFC3339))}, nil
}

// Condition returns one of two given values according to a boolean condition
func condition(value string, line lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	var cond string
	var ok bool
	if value == "" {
		cond, ok = json["condition"].(string)
		if !ok {
			return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha", "condition")
		}
	} else {
		cond = value
	}
	result, err := evalCondition(cond, line)
	if err != nil {
		return errorMessage, err
	}
	if result {
		value, ok = json["if_true"].(string)
		if !ok {
			return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha", "if_true")
		}
		return []resultsT{newResult(value)}, nil
	}
	value, ok = json["if_false"].(string)
	if !ok {
		return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha", "if_false")
	}
	return []resultsT{newResult(value)}, nil
}

// Eval evaluates an expression
func eval(value string, line lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	var expr string
	var ok bool
	if value == "" {
		expr, ok = json["expression"].(string)
		if !ok {
			return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha", "expression")
		}
	} else {
		expr = value
	}
	functions := map[string]govaluate.ExpressionFunction{
		"strlen": func(args ...interface{}) (interface{}, error) {
			length := len(args[0].(string))
			return fmt.Sprintf("%d", length), nil
		},
		"replace": func(args ...interface{}) (interface{}, error) {
			orig := args[0].(string)
			from := args[1].(string)
			to := args[2].(string)
			return strings.ReplaceAll(orig, from, to), nil
		},
	}
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expr, functions)
	if err != nil {
		return errorMessage, fmt.Errorf("expressao invalida (%v) na linha", expr)
	}
	params := make(map[string]interface{})
	for k, v := range line {
		params[removeSpaces(k)] = v
	}
	result, err := expression.Evaluate(params)
	if err != nil {
		return errorMessage, fmt.Errorf("erro na expressao [%s] com parametros [%#v]", expr, params)
	}
	if result == nil {
		result = ""
	}
	return []resultsT{newResult(fmt.Sprintf("%v", result))}, nil
}

// FilterCondition returns an empty string if a condition is false, but continues the processing if it is true
func filterCondition(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	var cond string
	var ok bool
	if value == "" {
		cond, ok = json["filter"].(string)
		if !ok {
			return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha %v", "Value", json)
		}
	} else {
		cond = value
	}
	condResult, err := evalCondition(cond, line)
	if err != nil {
		return errorMessage, err
	}
	if condResult {
		funcName, ok1 := json["function"].(string)
		if !ok1 {
			return errorMessage, fmt.Errorf("condicao sem elemento 'function' na linha %v", json)
		}
		if funcName == "filter" {
			return errorMessage, fmt.Errorf("condicao recursiva (elemento 'filter' + 'function = filter') na linha %v", json)
		}
		function, ok2 := functionDict[funcName]
		if !ok2 {
			fmt.Printf("Warning: funcao [%s] nao existe!\n", funcName)
			result, _ := undefined("", nil, json, options)
			return result, fmt.Errorf("funcao nao definida: [%s]", funcName)
		}
		return function("", line, json, options)
	}
	return []resultsT{}, nil
}

// Split splits a list of arguments and calls a function for each one of those
func split(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	funcName, ok := json["function2"].(string)
	if !ok {
		return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha %v", "function2", line)
	}
	func2, ok := functionDict[funcName]
	if !ok {
		return errorMessage, fmt.Errorf("funcao2 '%s' invalida na linha", json)
	}
	var field string
	var err error
	if funcName == "fixed" {
		field = json["Value"].(string)
		if !ok {
			return errorMessage, fmt.Errorf("funcao fixed precisa de elemento 'value' na linha %v", line)
		}
	} else {
		field, err = getField(value, "field", line, json, options)
		if err != nil {
			return errorMessage, err
		}
	}
	result := make([]resultsT, 0)
	values := strings.Split(field, ",")
	for _, val := range values {
		val = strings.TrimSpace(val)
		res, err1 := func2(val, line, json, options)
		if err1 != nil {
			return errorMessage, err1
		}
		result = append(result, res...)
	}
	return result, nil
}

// MapField returns a map with a field for key and other for value
func mapField(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	key, err := getField(value, "field1", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	val, err := getField(value, "field2", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	return []resultsT{newResult(key), newResult(val)}, nil
}

// AttrMap returns a map with a field for attribute name and other for value
func attrMap(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	key, err := getField(value, "attr_list", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	//values := strings.Split(key, ",")
	//attrs, ok := json["attrs"].([]interface{})
	//if !ok {
	//	return errorMessage, fmt.Errorf("atributo 'attrs' nao encontrado em funcao attrmap")
	//}
	//var attrMap map[string]string
	//for _, s := range values {
	//	for _, at := range attrs {
	//		attr := at.(map[string]interface{})
	//		name, ok := attr["Name"].(string)
	//		if !ok {
	//			return errorMessage, fmt.Errorf("atributo 'Name' nao encontrado em funcao attrmap")
	//		}
	//		fun, err2 := getField(value, "attr_list", line, json, options)
	//		if err2 != nil {
	//			return errorMessage, err2
	//		}
	//
	//		res, err := function("", line, json, options)
	//		if err != nil {
	//			return res, err
	//		}
	//		result = append(result, res...)
	//		attrMap[name], err = Process(fun, line, json, options)
	//	}
	//}
	val, err := getField(value, "field2", line, json, options)
	if err != nil {
		return errorMessage, err
	}

	return []resultsT{newResult(key), newResult(val)}, nil
}

// Convert maps an element of a string array unto another
func convert(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	key, err := getField(value, "field", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	from, ok := json["from"].(string)
	if !ok {
		return errorMessage, err
	}
	to, ok := json["to"].(string)
	if !ok {
		return errorMessage, err
	}
	fArr := strings.Split(from, ",")
	tArr := strings.Split(to, ",")
	if len(fArr) == 0 || len(fArr) != len(tArr) {
		return errorMessage, fmt.Errorf("funcao 'convert' tem que ter parametros 'from' e 'to' com mesmo numero de elementos")
	}
	cMap := make(map[string]string)
	for i, fr := range fArr {
		cMap[fr] = tArr[i]
	}

	val, ok := cMap[key]
	if !ok {
		return errorMessage, fmt.Errorf("valor [%s] nao consta da string 'from' no elemento 'convert'", key)
	}
	return []resultsT{newResult(val)}, nil
}

// Utc returns a date in UTC format
func utc(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	val, err := getField(value, "field", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	dat, err := parseDate(val)
	if err != nil {
		return errorMessage, err
	}
	utcTime := timeToUTCTimestamp(dat)
	result := fmt.Sprintf("%d", utcTime)
	return []resultsT{newResult(result)}, nil
}

// EvalCondition evaluates a boolean expression
func evalCondition(expr string, line lineT) (bool, error) {
	functions := map[string]govaluate.ExpressionFunction{
		"strlen": func(args ...interface{}) (interface{}, error) {
			length := len(args[0].(string))
			return fmt.Sprintf("%d", length), nil
		},
	}
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expr, functions)
	if err != nil {
		return false, fmt.Errorf("expressao invalida (%v)", expr)
	}
	params := make(map[string]interface{})
	for k, v := range line {
		params[removeSpaces(k)] = v
	}
	result, err := expression.Evaluate(params)
	if result == nil {
		return false, fmt.Errorf("expressao invalida (%v), parametros (%v)", expr, params)
	}
	return result.(bool), err
}

// Seconds returns the total seconds from a time
func seconds(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	field, err := fieldTrunc(value, line, json, options)
	if err != nil {
		return errorMessage, err
	}
	t, err := time.Parse("03:04:05", field[0].val)
	if err != nil {
		return errorMessage, err
	}
	return []resultsT{newResult(formatHMS(t))}, nil
}

// SurnameName inverts name and surname
func surnameName(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	var field string
	if value == "" {
		f, err := fieldTrunc(value, line, json, options)
		if err != nil {
			return errorMessage, err
		}
		field = f[0].val
	} else {
		field = value
	}
	result := removeExtraSpaces(field)
	if result == "" {
		return []resultsT{newResult("")}, nil
	}
	names := strings.Split(result, " ")
	length := len(names)
	var newName strings.Builder
	for i := 1; i < length; i++ {
		if i > 1 {
			newName.WriteString(" ")
		}
		newName.WriteString(names[i])
	}
	newName.WriteString(", ")
	newName.WriteString(names[0])
	result = newName.String()
	return []resultsT{newResult(result)}, nil
}

// genUUID returns a random uuid number
func genUUID(_ string, _ lineT, _ jsonT, _ optionsT) ([]resultsT, error) {
	result := uuids()
	return []resultsT{newResult(result)}, nil
}

// Option returns the option as defined in the JSON file, section "options"
func option(value string, _ lineT, json jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	optField, ok := json["field"].(string)
	if !ok {
		return errorMessage, fmt.Errorf("elemento '%s' inexistente na linha", "field")
	}
	val, ok := options[optField]
	if !ok {
		return errorMessage, fmt.Errorf("elemento '%s' inexistente nas options [%v]", optField, options)
	}
	return []resultsT{newResult(val)}, nil
}

// JanelaRepasse returns the last character of the billing id
func janelaRepasse(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	billId, err := getField(value, "field", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	val := ""
	if billId != "" {
		idx := len(billId) - 1
		val = billId[idx : idx+1]
	}
	return []resultsT{newResult(val)}, nil
}

// BoxTechnology returns the technology of the encoding based on the extension
func boxTechnology(value string, line lineT, json jsonT, options optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	filename, err := getField(value, "field", line, json, options)
	if err != nil {
		return errorMessage, err
	}
	extension := strings.ToLower(path.Ext(filename))
	result := ""
	switch extension {
	case ".mp4":
		result = "MP4"
	case ".ts", ".mov":
		result = "TS"
	case ".m3u8":
		result = "HLS"
	case ".mpd":
		result = "DASH"
	case ".ism":
		result = "HSS"
	default:
		return errorMessage, fmt.Errorf("tecnologia indeterminada para a extensao: [%s]", extension)
	}
	return []resultsT{newResult(result)}, nil
}

// Undefined returns a value to indicate an undefined function
func undefined(value string, _ lineT, _ jsonT, _ optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	return []resultsT{newResult("##UNDEFINED##")}, fmt.Errorf("funcao indefinida: [%s]", value)
}

// FirstName returns the first name of a composite name
func firstName(value string, _ lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	val, _ := json["Value"].(string)
	if val != "" && val[0:1] == "$" {
		value = options[val]
	}
	names := strings.Split(value, " ")
	result := ""
	if len(names) >= 1 {
		result = names[0]
	}
	return []resultsT{newResult(result)}, nil
}

// LastName returns the first name of a composite name
func lastName(value string, _ lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	val, _ := json["Value"].(string)
	if val != "" && val[0:1] == "$" {
		value = options[val]
	}
	names := strings.Split(value, " ")
	result := ""
	length := len(names)
	if length > 1 {
		result = names[length-1]
	}
	return []resultsT{newResult(result)}, nil
}

// MiddleName returns the first name of a composite name
func middleName(value string, _ lineT, json jsonT, _ optionsT) ([]resultsT, error) {
	if value != "" {
		return []resultsT{newResult(value)}, nil
	}
	val, _ := json["Value"].(string)
	if val != "" && val[0:1] == "$" {
		value = options[val]
	}
	names := strings.Split(value, " ")
	result := ""
	length := len(names)
	if length > 2 {
		result = names[1]
	}
	return []resultsT{newResult(result)}, nil
}

///////////////////////////////
///////////////////////////////

func getValue(key string, json jsonT) (string, error) {
	value, ok := json[key].(string)
	if !ok {
		return "###", fmt.Errorf("chave [%v] nao encontrada no elemento json [%v]", key, json)
	}
	return value, nil
}

//func stripchars(str, chr string) string {
//	return strings.mapT(func(r rune) rune {
//		if !strings.ContainsRune(chr, r) {
//			return r
//		}
//		return -1
//	}, str)
//}

// RemoveSpaces replaces all whitespace with "_"
func removeSpaces(val string) string {
	return strings.Join(strings.Fields(val), "_")
}

// RemoveExtraSpaces removes any redundant spaces and trim spaces at left and right
func removeExtraSpaces(val string) string {
	return strings.Join(strings.Fields(val), " ")
}

// RemoveQuotes replaces all quotes with "_"
func removeQuotes(val string) string {
	r := strings.NewReplacer("\"", "", "'", "")
	return r.Replace(val)
}

// RemoveAccents Removes all accented characters from a string
func removeAccents(val string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, err := transform.String(t, val)
	if err != nil {
		return "##ERRO##", err
	}
	return output, nil
}

// ReplaceAllNonAlpha Replaces all non-alphanumeric characters with a "_"
func replaceAllNonAlpha(val string) string {
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		panic(err.Error())
	}
	return reg.ReplaceAllString(val, "_")
}

func formatMoney(val string) (string, error) {
	if val == "" {
		return "", nil
	}
	res, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return "##ERRO##", err
	}
	return fmt.Sprintf("%.2f", res), nil
}

//func date() string {
//	now := time.Now()
//	return formatDate(now)
//}

// Timestamp returns a timestamp from present time
func timestamp() string {
	now := time.Now()
	return formatTimestamp(now)
}

func formatTimestamp(t time.Time) string {
	// Mon Jan 2 15:04:05 MST 2006
	// %y%m%d%H%M%S
	return t.Format("060102150405")
}

func formatHMS(t time.Time) string {
	return t.Format("15:04:05")
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func parseDate(dateStr string) (time.Time, error) {
	// Mon Jan 2 15:04:05 MST 2006
	result, err := time.Parse("01-02-06", dateStr)
	return result, err
}

func truncateSuffix(value string, suffix string, _ lineT, json jsonT, _ optionsT) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return value, nil
	}
	max, err := strconv.Atoi(val)
	if err != nil {
		return errorMessage[0].val, fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	sufLen := len(suffix)
	if valLen+sufLen <= max {
		return value, nil
	}
	if sufLen+1 >= max {
		return errorMessage[0].val, fmt.Errorf("sufixo [%s] nao pode ser aplicado porque estoura o tamanho maximo [%d] no elemento [%s]", suffix, max, value)
	}
	r := []rune(value)
	l := len(r)
	if max > l {
		max = l
	}
	safeSubstring := string(r[0:max])
	return safeSubstring, nil
}

func truncate(value string, _ lineT, json jsonT, _ optionsT) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return value, nil
	}
	max, err := strconv.Atoi(val)
	if err != nil {
		return errorMessage[0].val, fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	if valLen <= max {
		return value, nil
	}
	r := []rune(value)
	safeSubstring := string(r[0:max])
	return safeSubstring, nil
}

//func appendIfNotNil(orig []string, values ...string) []string {
//	for _, val := range values {
//		if val != "" {
//			orig = append(orig, val)
//		}
//	}
//	return orig
//}

func uuids() string {
	u1 := uuid.NewV4()
	return u1.String()
}

func timeToUTCTimestamp(t time.Time) int64 {
	//fmt.Printf("::: %#v", t.Format("15:04:05"))
	return t.UnixNano() / int64(time.Millisecond)
}

// ToDate converts time to a standard string format
//func ToDate(value string) (time.Time, error) {
//	//is serial format?
//	serial, err := strconv.ParseFloat(value, 64)
//	if err == nil {
//		return time.Unix(int64((serial-25569)*86400), 0), nil
//	}
//	return time.Parse(ISO8601, value)
//}

// ToTimestamp converts a numeric string to a timestamp in milliseconds
func toTimestamp(value string) (int64, error) {
	//is serial format?
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t, err2 := time.Parse("15:04:05", value)
		if err2 != nil {
			return -1, err
		}
		return int64(uint64(t.Hour())*3600+uint64(t.Minute())*60+uint64(t.Second())) * 1000, nil
	}
	return int64(serial * 86400 * 1000), nil
}

// ToTimeSeconds converts a numeric string to a timestamp in seconds
func toTimeSeconds(value string) (int64, error) {
	//is serial format?
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return -1, err
	}
	return int64(serial * 86400), nil
}
