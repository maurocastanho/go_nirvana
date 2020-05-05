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

// ERR Default error message
var ERR []ResultsT

type ResultsT struct {
	val  string
	vars map[string]string
}

func NewResult(val string) ResultsT {
	var result ResultsT
	result.val = val
	result.vars = make(map[string]string)
	return result
}

func NewResultVars(val string, key string, value string) ResultsT {
	var result ResultsT
	result.val = val
	result.vars = make(map[string]string)
	result.vars[key] = value
	return result
}

// FunctionDict is the relation between the operation name and the function
var FunctionDict map[string]func(string, lineT, jsonT, optionsT) ([]ResultsT, error)

// InitFunctions maps the user functions
func InitFunctions() {
	FunctionDict = map[string]func(string, lineT, jsonT, optionsT) ([]ResultsT, error){
		"fixed":           Fixed,
		"field":           Field,
		"field_raw":       FieldRaw,
		"field_validated": FieldValidated,
		"field_noacc":     FieldNoAccents,
		"field_trim":      FieldTrim,
		"field_no_quotes": FieldNoQuotes,
		"field_money":     FieldMoney,
		"field_date":      FieldDate,
		"field_suffix":    Suffix,
		"assetid":         AssetID,
		"episode_id":      EpisodeID,
		"date":            Date,
		"convert_date":    ConvertDate,
		"timestamp":       Utc,
		"seconds":         Seconds,
		"surname_name":    SurnameName,
		"condition":       Condition,
		"filter":          FilterCondition,
		"option":          Option,
		"eval":            Eval,
		"split":           Split,
		"uuid":            UUID,
		"map":             MapField,
		"convert":         Convert,
		"janela_repasse":  JanelaRepasse,
		"attr_map":        AttrMap,
		"box_technology":  BoxTechnology,
		"empty":           EmptyFunc,
		"first_name":      FirstName,
		"middle_name":     MiddleName,
		"last_name":       LastName,
		"set_var":         SetVar,
	}
}

// Process process one element from json config
func Process(funcName string, lines []lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	ERR := []ResultsT{NewResult("#ERRO#")}
	// fmt.Printf("=> %s\n", funcName)
	if funcName == "" {
		return ERR, fmt.Errorf("'function' nao especificada")
	}
	function, ok := FunctionDict[funcName]
	if !ok {
		fmt.Printf("Warning: funcao [%s] nao existe!\n", funcName)
		result, _ := Undefined("", nil, json, options)
		return result, fmt.Errorf("funcao nao definida: [%s]", funcName)
	}
	result := make([]ResultsT, 0)
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
func Fixed(value string, _ lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	val, _ := json["Value"].(string)
	//if val != "" && val[0:1] == "$" {
	//	result := options[val]
	//	return NewResult([]string{result}), nil
	//}
	val, err := getValue("Value", json)
	return []ResultsT{NewResult(val)}, err
}

// FieldMoney returns field formatted as money
func FieldMoney(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	val, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := formatMoney(val)
	if err != nil {
		return ERR, err
	}
	return []ResultsT{NewResult(result)}, nil
}

// Field returns field from line after truncating max size
func Field(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	value, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	value, err = truncate(value, line, json, options)
	return []ResultsT{NewResult(value)}, err
}

// FieldRaw returns field without further processing
func FieldRaw(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	value, err := getField(value, "Name", line, json, options)
	return []ResultsT{NewResult(value)}, err
}

// FieldDate returns a date field after formatting
func FieldDate(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	value, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := parseDate(value)
	if err != nil {
		return ERR, err
	}
	value = formatDate(t)
	return []ResultsT{NewResult(value)}, err
}

func getField(value string, _ string, line lineT, json jsonT, _ optionsT) (string, error) {
	// fmt.Printf("field=%#v, json=%#v, line=%#v, options=%#v\n", field, json, line, options)
	if value != "" {
		return value, nil
	}
	fieldName, err := getValue("field", json)
	if err != nil {
		return ERR[0].val, err
	}
	value, ok := line[fieldName]
	if !ok {
		return ERR[0].val, fmt.Errorf("elemento '%s' inexistente na linha", fieldName)
	}
	// fmt.Printf("field(%v) = [%v]\n", field, value)
	return value, nil
}

// FieldValidated validates a field against a list and returns the value if valid
func FieldValidated(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}
	val, err := getValue("Options", json)
	if err != nil {
		return ERR, err
	}
	opts := strings.Split(val, ",")
	for _, opt := range opts {
		if strings.TrimSpace(opt) == field[0].val {
			return field, nil
		}
	}
	return ERR, fmt.Errorf("falha na validacao do elemento '%s': %s, valores possiveis: %v", json["Name"], field, opts)
}

// FieldNoAccents returns the field after replacing accented characters for its non-accented correspondents
func FieldNoAccents(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := RemoveAccents(field)
	if err != nil {
		return ERR, err
	}
	result, err = truncate(result, line, json, options)
	return []ResultsT{NewResult(result)}, err
}

// FieldTrim returns the field after removing spaces from left and right
func FieldTrim(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result := strings.TrimSpace(field)
	result, err = truncate(result, line, json, options)
	if err != nil {
		return ERR, err
	}
	return []ResultsT{NewResult(result)}, nil
}

// FieldNoQuotes removes all quotation symbols from the field and returns it
func FieldNoQuotes(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	value = RemoveQuotes(field)
	result, err := truncate(value, line, json, options)
	return []ResultsT{NewResult(result)}, err
}

// Suffix removes the extension and appends a suffix to a field, returning the result
func Suffix(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}

	// fmt.Printf("-->> %v\n", field)
	noacc, err := RemoveAccents(field[0].val)
	if err != nil {
		return ERR, err
	}

	suffix, _ := json["suffix"].(string)
	if suffix == "" {
		suffix = path.Ext(noacc)
	}

	extIdx := strings.LastIndex(noacc, ".")
	if extIdx > 0 {
		noacc = noacc[0:extIdx]
	}
	prefix, _ := options["inpPrefix"]
	val := ReplaceAllNonAlpha(noacc)
	val, err = truncateSuffix(val, suffix, line, json, options)
	if err != nil {
		return ERR, err
	}
	result := fmt.Sprintf("%s%s%s", prefix, val, suffix)
	//	fmt.Printf("-->> %v\n", result)
	return []ResultsT{NewResult(result)}, nil
}

// AssetID returns the Asset ID
func AssetID(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	fProvider, ok := json["prefix"].(string)
	if !ok || fProvider == "" {
		return ERR, fmt.Errorf("field prefixo do Asset ID nao encontrado (prefix): [%v]", json)
	}
	prov, ok := line[fProvider]
	if !ok || prov == "" {
		return ERR, fmt.Errorf("provider assetid nao encontrado (provider): [%v]", line)
	}
	provider := strings.ToUpper(RemoveSpaces(prov))
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
		return ERR, fmt.Errorf("numero do Sufixo do assetid (suffix_number) nao encontrado: [%v]", json)
	}
	suffix := int(suffixF)
	timestamp := options["timestamp"]
	if !ok || timestamp == "" {
		return ERR, fmt.Errorf("timestamp nao encontrada (timestamp): [%v]", options)
	}
	fileNum, ok := line["file_number"]
	if !ok || fileNum == "" {
		return ERR, fmt.Errorf("numero do arquivo nao encontrado (file_number): [%v]", line)
	}
	result := fmt.Sprintf("%s%d%s%03s", provider, suffix, timestamp, fileNum)
	return []ResultsT{NewResult(result)}, nil
}

// EpisodeID returns the Episode ID, make of (season number | episode number)
func EpisodeID(value string, line lineT, _ jsonT, options optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	fSeason, ok := options["season_field"]
	if !ok || fSeason == "" {
		return ERR, fmt.Errorf("config para campo 'season_field' nao encontrado: [%v]", options)
	}
	season, ok := line[fSeason]
	if !ok || season == "" {
		return ERR, fmt.Errorf("temporada do episode_id nao encontrada (%v): [%v]", fSeason, line)
	}
	fEpisodeID := options["episode_field"]
	if !ok || fEpisodeID == "" {
		return ERR, fmt.Errorf("config para campo 'episode_field' nao encontrado: [%v]", options)
	}
	episode, ok := line[fEpisodeID]
	if !ok || season == "" {
		return ERR, fmt.Errorf("valor do episode_id nao encontrado (%v): [%v]", fEpisodeID, line)
	}
	result := fmt.Sprintf("%02s%03s", season, episode)
	return []ResultsT{NewResult(result)}, nil
}

// Date returns the present date, formatted
func Date(value string, _ lineT, _ jsonT, _ optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	return []ResultsT{NewResult(formatDate(time.Now()))}, nil
}

// EmptyFunc returns always a empty value
func EmptyFunc(_ string, _ lineT, _ jsonT, _ optionsT) ([]ResultsT, error) {
	return []ResultsT{NewResult("")}, nil
}

// SetVar sets a variable in the options
func SetVar(value string, _ lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	name, ok := json["var"].(string)
	if !ok || name == "" {
		return ERR, fmt.Errorf("campo 'var' nao encontrado: [%v]", json)
	}
	return []ResultsT{NewResultVars("", "$"+name, value)}, nil
}

// ConvertDate converts a date string from the mm/dd/yy format to the default format
func ConvertDate(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := time.Parse("01/02/06", field[0].val)
	if err != nil {
		return ERR, err
	}
	return []ResultsT{NewResult(formatDate(t))}, nil
}

// Condition returns one of two given values according to a boolean condition
func Condition(value string, line lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	var condition string
	var ok bool
	if value == "" {
		condition, ok = json["condition"].(string)
		if !ok {
			return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "condition")
		}
	} else {
		condition = value
	}
	result, err := EvalCondition(condition, line)
	if err != nil {
		return ERR, err
	}
	if result {
		value, ok = json["if_true"].(string)
		if !ok {
			return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "if_true")
		}
		return []ResultsT{NewResult(value)}, nil
	}
	value, ok = json["if_false"].(string)
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "if_false")
	}
	return []ResultsT{NewResult(value)}, nil
}

// Eval evaluates an expression
func Eval(value string, line lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	var expr string
	var ok bool
	if value == "" {
		expr, ok = json["expression"].(string)
		if !ok {
			return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "expression")
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
		return ERR, fmt.Errorf("expressao invalida (%v) na linha", expr)
	}
	params := make(map[string]interface{})
	for k, v := range line {
		params[RemoveSpaces(k)] = v
	}
	result, err := expression.Evaluate(params)
	if result == nil {
		result = ""
	}
	if err != nil {
		result = fmt.Sprintf("Erro na expressao [%s] com parametros [%#v]", expr, params)
		fmt.Print(result)
	}
	return []ResultsT{NewResult(result.(string))}, err
}

// FilterCondition returns an empty string if a condition is false, but continues the processing if it is true
func FilterCondition(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	var condition string
	var ok bool
	if value == "" {
		condition, ok = json["filter"].(string)
		if !ok {
			return ERR, fmt.Errorf("elemento '%s' inexistente na linha %v", "Value", json)
		}
	} else {
		condition = value
	}
	cond, err := EvalCondition(condition, line)
	if err != nil {
		return ERR, err
	}
	if cond {
		funcName, ok1 := json["function"].(string)
		if !ok1 {
			return ERR, fmt.Errorf("condicao sem elemento 'function' na linha %v", json)
		}
		function, ok1 := FunctionDict[funcName]
		if !ok1 {
			fmt.Printf("Warning: funcao [%s] nao existe!\n", funcName)
			result, _ := Undefined("", nil, json, options)
			return result, fmt.Errorf("funcao nao definida: [%s]", funcName)
		}
		return function("", line, json, options)
	}
	return []ResultsT{}, nil
}

// Split splits a list of arguments and calls a function for each one of those
func Split(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	funcName, ok := json["function2"].(string)
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente na linha %v", "function2", line)
	}
	func2, ok := FunctionDict[funcName]
	if !ok {
		return ERR, fmt.Errorf("funcao2 '%s' invalida na linha", json)
	}
	var field string
	var err error
	if funcName == "fixed" {
		field = json["Value"].(string)
		if !ok {
			return ERR, fmt.Errorf("funcao fixed precisa de elemento 'value' na linha %v", line)
		}
	} else {
		field, err = getField(value, "field", line, json, options)
		if err != nil {
			return ERR, err
		}
	}
	result := make([]ResultsT, 0)
	values := strings.Split(field, ",")
	for _, val := range values {
		val = strings.TrimSpace(val)
		res, err1 := func2(val, line, json, options)
		if err1 != nil {
			return ERR, err1
		}
		result = append(result, res...)
	}
	return result, nil
}

// MapField returns a map with a field for key and other for value
func MapField(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	key, err := getField(value, "field1", line, json, options)
	if err != nil {
		return ERR, err
	}
	val, err := getField(value, "field2", line, json, options)
	if err != nil {
		return ERR, err
	}
	return []ResultsT{NewResult(key), NewResult(val)}, nil
}

// AttrMap returns a map with a field for attribute name and other for value
func AttrMap(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	key, err := getField(value, "attr_list", line, json, options)
	if err != nil {
		return ERR, err
	}
	//values := strings.Split(key, ",")
	//attrs, ok := json["attrs"].([]interface{})
	//if !ok {
	//	return ERR, fmt.Errorf("atributo 'attrs' nao encontrado em funcao attrmap")
	//}
	//var attrMap map[string]string
	//for _, s := range values {
	//	for _, at := range attrs {
	//		attr := at.(map[string]interface{})
	//		name, ok := attr["Name"].(string)
	//		if !ok {
	//			return ERR, fmt.Errorf("atributo 'Name' nao encontrado em funcao attrmap")
	//		}
	//		fun, err2 := getField(value, "attr_list", line, json, options)
	//		if err2 != nil {
	//			return ERR, err2
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
		return ERR, err
	}

	return []ResultsT{NewResult(key), NewResult(val)}, nil
}

// Convert maps an element of a string array unto another
func Convert(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	key, err := getField(value, "field", line, json, options)
	if err != nil {
		return ERR, err
	}
	from, ok := json["from"].(string)
	if !ok {
		return ERR, err
	}
	to, ok := json["to"].(string)
	if !ok {
		return ERR, err
	}
	fArr := strings.Split(from, ",")
	tArr := strings.Split(to, ",")
	if len(fArr) == 0 || len(fArr) != len(tArr) {
		return ERR, fmt.Errorf("funcao 'convert' tem que ter parametros 'from' e 'to' com mesmo numero de elementos")
	}
	cMap := make(map[string]string)
	for i, fr := range fArr {
		cMap[fr] = tArr[i]
	}

	val, ok := cMap[key]
	if !ok {
		return ERR, fmt.Errorf("valor [%s] nao consta da string 'from' no elemento 'convert'", key)
	}
	return []ResultsT{NewResult(val)}, nil
}

// Utc returns a date in UTC format
func Utc(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	val, err := getField(value, "field", line, json, options)
	if err != nil {
		return ERR, err
	}
	dat, err := parseDate(val)
	if err != nil {
		return ERR, err
	}
	utc := timeToUTCTimestamp(dat)
	result := fmt.Sprintf("%d", utc)

	return []ResultsT{NewResult(result)}, nil
}

// EvalCondition evaluates a boolean expression
func EvalCondition(expr string, line lineT) (bool, error) {
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
		params[RemoveSpaces(k)] = v
	}
	result, err := expression.Evaluate(params)
	if result == nil {
		return false, fmt.Errorf("expressao invalida (%v), parametros (%v)", expr, params)
	}
	return result.(bool), err
}

// Seconds returns the total seconds from a time
func Seconds(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := time.Parse("03:04:05", field[0].val)
	if err != nil {
		return ERR, err
	}
	return []ResultsT{NewResult(formatHMS(t))}, nil
}

// SurnameName inverts name and surname
func SurnameName(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	var field string
	if value == "" {
		f, err := Field(value, line, json, options)
		if err != nil {
			return ERR, err
		}
		field = f[0].val
	} else {
		field = value
	}
	result := RemoveExtraSpaces(field)
	if result == "" {
		return []ResultsT{NewResult("")}, nil
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
	return []ResultsT{NewResult(result)}, nil
}

// UUID returns a random uuid number
func UUID(_ string, _ lineT, _ jsonT, _ optionsT) ([]ResultsT, error) {
	result := uuids()
	return []ResultsT{NewResult(result)}, nil
}

// Option returns the option as defined in the JSON file, section "options"
func Option(value string, _ lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	optField, ok := json["field"].(string)
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "field")
	}
	val, ok := options[optField]
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente nas options [%v]", optField, options)
	}
	return []ResultsT{NewResult(val)}, nil
}

// JanelaRepasse returns the last character of the billing id
func JanelaRepasse(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	billId, err := getField(value, "field", line, json, options)
	if err != nil {
		return ERR, err
	}
	val := ""
	if billId != "" {
		idx := len(billId) - 1
		val = billId[idx : idx+1]
	}
	return []ResultsT{NewResult(val)}, nil
}

// BoxTechnology returns the technology of the encoding based on the extension
func BoxTechnology(value string, line lineT, json jsonT, options optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	filename, err := getField(value, "field", line, json, options)
	if err != nil {
		return ERR, err
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
		return ERR, fmt.Errorf("tecnologia indeterminada para a extensao: [%s]", extension)
	}
	return []ResultsT{NewResult(result)}, nil
}

// Undefined returns a value to indicate an undefined function
func Undefined(value string, _ lineT, _ jsonT, _ optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
	}
	return []ResultsT{NewResult("##UNDEFINED##")}, fmt.Errorf("funcao indefinida: [%s]", value)
}

// FirstName returns the first name of a composite name
func FirstName(value string, _ lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
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
	return []ResultsT{NewResult(result)}, nil
}

// LastName returns the first name of a composite name
func LastName(value string, _ lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
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
	return []ResultsT{NewResult(result)}, nil
}

// MiddleName returns the first name of a composite name
func MiddleName(value string, _ lineT, json jsonT, _ optionsT) ([]ResultsT, error) {
	if value != "" {
		return []ResultsT{NewResult(value)}, nil
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
	return []ResultsT{NewResult(result)}, nil
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
//	return strings.Map(func(r rune) rune {
//		if !strings.ContainsRune(chr, r) {
//			return r
//		}
//		return -1
//	}, str)
//}

// RemoveSpaces replaces all whitespace with "_"
func RemoveSpaces(val string) string {
	return strings.Join(strings.Fields(val), "_")
}

// RemoveExtraSpaces removes any redundant spaces and trim spaces at left and right
func RemoveExtraSpaces(val string) string {
	return strings.Join(strings.Fields(val), " ")
}

// RemoveQuotes replaces all quotes with "_"
func RemoveQuotes(val string) string {
	r := strings.NewReplacer("\"", "", "'", "")
	return r.Replace(val)
}

// RemoveAccents Removes all accented characters from a string
func RemoveAccents(val string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, err := transform.String(t, val)
	if err != nil {
		return "##ERRO##", err
	}
	return output, nil
}

// ReplaceAllNonAlpha Replaces all non-alphanumeric characters with a "_"
func ReplaceAllNonAlpha(val string) string {
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		panic(err)
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
func Timestamp() string {
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
	if err != nil {
		fmt.Printf("ERRO: %v\n", dateStr)
	}
	return result, err
}

func truncateSuffix(value string, suffix string, _ lineT, json jsonT, _ optionsT) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return value, nil
	}
	max, err := strconv.Atoi(val)
	if err != nil {
		return ERR[0].val, fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	sufLen := len(suffix)
	if valLen+sufLen <= max {
		return value, nil
	}
	if sufLen+1 >= max {
		return ERR[0].val, fmt.Errorf("sufixo [%s] nao pode ser aplicado porque estoura o tamanho maximo [%d] no elemento [%s]", suffix, max, value)
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
		return ERR[0].val, fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
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
	fmt.Printf("::: %#v", t.Format("15:04:05"))
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
func ToTimestamp(value string) (int64, error) {
	//is serial format?
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return -1, err
	}
	return int64(serial * 86400 * 1000), nil
}

// ToTimeSeconds converts a numeric string to a timestamp in seconds
func ToTimeSeconds(value string) (int64, error) {
	//is serial format?
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return -1, err
	}
	return int64(serial * 86400), nil
}
