package main

import (
	"fmt"
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

const (
	//ISO8601 is format for ISO8601 dates. Like RFC3339, but without timezone
	ISO8601 = "2006-01-02T15:04:05"
)

// ERR Default error message
var ERR []string

// FunctionDict is the relation between the operation name and the function
var FunctionDict map[string]func(string, lineT, jsonT, optionsT) ([]string, error)

// InitFunctions maps the user functions
func InitFunctions() {
	FunctionDict = map[string]func(string, lineT, jsonT, optionsT) ([]string, error){
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
		"map":             Map,
	}
}

// Process process one element from json config
func Process(funcName string, lines []lineT, json jsonT, options optionsT) ([]string, error) {
	ERR = []string{"#ERRO#"}
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
	result := make([]string, 0)
	for _, line := range lines {
		res, err := function("", line, json, options)
		if err != nil {
			return res, err
		}
		result = append(result, res...)
	}
	return result, nil
}

func findValue(value string, field string, json jsonT) (string, error) {
	if value == "" {
		return getValue(field, json)
	}
	return value, nil
}

// Fixed returns the same value
func Fixed(value string, _ lineT, json jsonT, _ optionsT) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	val, err := getValue("value", json)
	return []string{val}, err
}

// FieldMoney returns field formatted as money
func FieldMoney(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	val, err := getField(value, "name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := formatMoney(val)
	if err != nil {
		return ERR, err
	}
	return []string{result}, nil
}

// Field returns field from line after truncating max size
func Field(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	value, err := getField(value, "name", line, json, options)
	if err != nil {
		return ERR, err
	}
	value, err = truncate(value, line, json, options)
	return []string{value}, err
}

// FieldRaw returns field without further processing
func FieldRaw(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	value, err := getField(value, "name", line, json, options)
	return []string{value}, err
}

// FieldDate returns a date field after formatting
func FieldDate(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	value, err := getField(value, "name", line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := parseDate(value)
	if err != nil {
		return ERR, err
	}
	value = formatDate(t)
	return []string{value}, err
}

func getField(value string, _ string, line lineT, json jsonT, _ optionsT) (string, error) {
	// fmt.Printf("field=%#v, json=%#v, line=%#v, options=%#v\n", field, json, line, options)
	if value != "" {
		return value, nil
	}
	val, err := getValue("field", json)
	if err != nil {
		return ERR[0], err
	}
	value, ok := line[val]
	if !ok {
		return ERR[0], fmt.Errorf("elemento '%s' inexistente na linha", val)
	}
	// fmt.Printf("field(%v) = [%v]\n", field, value)
	return value, nil
}

// FieldValidated validates a field against a list and returns the value if valid
func FieldValidated(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
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
		if strings.TrimSpace(opt) == field[0] {
			return field, nil
		}
	}
	return ERR, fmt.Errorf("falha na validacao do elemento '%s': %s, valores possiveis: %v", json["name"], field, opts)
}

// FieldNoAccents returns the field after replacing accented characters for its non-accented correspondents
func FieldNoAccents(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	field, err := getField(value, "name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := RemoveAccents(field)
	if err != nil {
		return ERR, err
	}
	result, err = truncate(result, line, json, options)
	return []string{result}, err
}

// FieldTrim returns the field after removing spaces from left and right
func FieldTrim(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	field, err := getField(value, "name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result := strings.TrimSpace(field)
	result, err = truncate(result, line, json, options)
	if err != nil {
		return ERR, err
	}
	return []string{result}, nil
}

// FieldNoQuotes removes all quotation symbols from the field and returns it
func FieldNoQuotes(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	field, err := getField(value, "name", line, json, options)
	if err != nil {
		return ERR, err
	}
	value = RemoveQuotes(field)
	result, err := truncate(value, line, json, options)
	return []string{result}, err
}

// Suffix removes the extension and appends a suffix to a field, returning the result
func Suffix(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	suffix, ok := json["suffix"].(string)
	if !ok || suffix == "" {
		return ERR, fmt.Errorf("sufixo nao encontrado: [%v]", json)
	}
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}
	// fmt.Printf("-->> %v\n", field)
	noacc, err := RemoveAccents(field[0])
	if err != nil {
		return ERR, err
	}

	extIdx := strings.LastIndex(noacc, ".")
	if extIdx > 0 {
		noacc = noacc[0:extIdx]
	}

	val := ReplaceAllNonAlpha(noacc)
	val, err = truncateSuffix(val, suffix, line, json, options)
	if err != nil {
		return ERR, err
	}
	result := fmt.Sprintf("%s%s", val, suffix)
	//	fmt.Printf("-->> %v\n", result)
	return []string{result}, nil
}

// AssetID returns the Asset ID
func AssetID(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	if value != "" {
		return []string{value}, nil
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
	return []string{result}, nil
}

// EpisodeID returns the Episode ID, make of (season number | episode number)
func EpisodeID(value string, line lineT, _ jsonT, options optionsT) ([]string, error) {
	if value != "" {
		return []string{value}, nil
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
	return []string{result}, nil
}

// Date returns the present date, formatted
func Date(value string, _ lineT, _ jsonT, _ optionsT) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	return []string{formatDate(time.Now())}, nil
}

// ConvertDate converts a date string from the mm/dd/yy format to the default format
func ConvertDate(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := time.Parse("01/02/06", field[0])
	if err != nil {
		return ERR, err
	}
	return []string{formatDate(t)}, nil
}

// Condition returns one of two given values according to a boolean condition
func Condition(value string, line lineT, json jsonT, _ optionsT) ([]string, error) {
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
		return []string{value}, nil
	}
	value, ok = json["if_false"].(string)
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "if_false")
	}
	return []string{value}, nil
}

// Eval evaluates an expression
func Eval(value string, line lineT, json jsonT, _ optionsT) ([]string, error) {
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
	return []string{result.(string)}, err
}

// FilterCondition returns an empty string if a condition is false, but continues the processing if it is true
func FilterCondition(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
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
	return []string{}, nil
}

// Split splits a list of arguments and calls a function for each one of those
func Split(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
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
		field = json["value"].(string)
		if !ok {
			return ERR, fmt.Errorf("funcao fixed precisa de elemento 'value' na linha %v", line)
		}
	} else {
		field, err = getField(value, "field", line, json, options)
		if err != nil {
			return ERR, err
		}
	}
	result := make([]string, 0)
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

// Map returns a map with a field for key and other for value
func Map(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	key, err := getField(value, "field1", line, json, options)
	if err != nil {
		return ERR, err
	}
	val, err := getField(value, "field2", line, json, options)
	if err != nil {
		return ERR, err
	}

	return []string{key, val}, nil
}

// Utc returns a date in UTC format
func Utc(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
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

	return []string{result}, nil
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
func Seconds(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	field, err := Field(value, line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := time.Parse("03:04:05", field[0])
	if err != nil {
		return ERR, err
	}
	return []string{formatHMS(t)}, nil
}

// SurnameName inverts name and surname
func SurnameName(value string, line lineT, json jsonT, options optionsT) ([]string, error) {
	var field string
	if value == "" {
		f, err := Field(value, line, json, options)
		if err != nil {
			return ERR, err
		}
		field = f[0]
	} else {
		field = value
	}
	result := RemoveSpaces(field)
	return []string{result}, nil
}

// UUID returns a random uuid number
func UUID(_ string, _ lineT, _ jsonT, _ optionsT) ([]string, error) {
	result := uuids()
	return []string{result}, nil
}

// Option returns the option as defined in the JSON file, section "options"
func Option(value string, _ lineT, json jsonT, options optionsT) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	optField, ok := json["field"].(string)
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente na linha", "field")
	}
	val, ok := options[optField]
	if !ok {
		return ERR, fmt.Errorf("elemento '%s' inexistente nas options [%v]", optField, options)
	}
	return []string{val}, nil
}

// Undefined returns a value to indicate an undefined function
func Undefined(value string, _ lineT, _ jsonT, _ optionsT) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	return []string{"##UNDEFINED##"}, fmt.Errorf("funcao indefinida: [%s]", value)
}

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
		return ERR[0], fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	sufLen := len(suffix)
	if valLen+sufLen <= max {
		return value, nil
	}
	if sufLen+1 >= max {
		return ERR[0], fmt.Errorf("sufixo [%s] nao pode ser aplicado porque estoura o tamanho maximo [%d] no elemento [%s]", suffix, max, value)
	}
	r := []rune(value)
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
		return ERR[0], fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	if valLen <= max {
		return value, nil
	}
	r := []rune(value)
	safeSubstring := string(r[0:max])
	return safeSubstring, nil
}

func appendIfNotNil(orig []string, values ...string) []string {
	for _, val := range values {
		if val != "" {
			orig = append(orig, val)
		}
	}
	return orig
}

func uuids() string {
	u1 := uuid.NewV4()
	return u1.String()
}

func timeToUTCTimestamp(t time.Time) int64 {
	fmt.Printf("::: %#v", t.Format("15:04:05"))
	return t.UnixNano() / int64(time.Millisecond)
}

func ToDate(value string) (time.Time, error) {
	//is serial format?
	serial, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return time.Unix(int64((serial-25569)*86400), 0), nil
	}
	return time.Parse(ISO8601, value)
}

func ToTimestamp(value string) (int64, error) {
	//is serial format?
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return -1, err
	}
	return int64(serial * 86400 * 1000), nil
}
