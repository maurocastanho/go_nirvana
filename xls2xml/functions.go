package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Knetic/govaluate"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// ERR Default error message
var ERR []string

// FunctionDict is the relation between the operation name and the function
var FunctionDict map[string]func(string, map[string]string, map[string]interface{}, map[string]string) ([]string, error)

func Process(funcName string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	ERR = []string{"#ERRO#"}
	// fmt.Printf("=> %s\n", funcName)
	if funcName == "" {
		return ERR, fmt.Errorf("'function' nao especificada")
	}
	function, ok := FunctionDict[funcName]
	if ok {
		return function("", line, json, options)
	}
	fmt.Printf("Warning: funcao [%s] nao existe!\n", funcName)
	result, _ := Undefined("", line, json, options)
	return result, fmt.Errorf("Funcao nao definida: [%s]", funcName)
}

func findValue(value string, field string, json map[string]interface{}) (string, error) {
	if value == "" {
		return getValue(field, json)
	}
	return value, nil
}

// Fixed returns the same value
func Fixed(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	val, err := getValue("Value", json)
	return []string{val}, err
}

func FieldMoney(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	val, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := formatMoney(val)
	if err != nil {
		return ERR, err
	}
	return []string{result}, nil
}

func Field(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	value, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	value, err = truncate(value, line, json, options)
	return []string{value}, err
}

func getField(value string, field string, line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
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
		return ERR[0], fmt.Errorf("Elemento '%s' inexistente na linha", val)
	}
	// fmt.Printf("field(%v) = [%v]\n", field, value)
	return value, nil
}

func FieldValidated(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
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
		if opt == field[0] {
			return field, nil
		}
	}
	return ERR, fmt.Errorf("Falha na validacao do elemento '%s': %s, valores possiveis: %v", json["Name"], field, opts)
}

func FieldNoAccents(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := RemoveAccents(field)
	if err != nil {
		return ERR, err
	}
	result, err = truncate(value, line, json, options)
	return []string{result}, err
}

func FieldTrim(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	result := strings.TrimSpace(field)
	if err != nil {
		return ERR, err
	}
	result, err = truncate(value, line, json, options)
	return []string{result}, nil
}

func FieldNoQuotes(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := getField(value, "Name", line, json, options)
	if err != nil {
		return ERR, err
	}
	value = RemoveQuotes(field)
	if err != nil {
		return ERR, err
	}
	result, err := truncate(value, line, json, options)
	return []string{result}, nil
}

func Suffix(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	suffix, ok := json["suffix"].(string)
	if !ok || suffix == "" {
		return ERR, fmt.Errorf("Sufixo nao encontrado: [%v]", json)
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

func AssetId(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	provider, ok := options["provider"]
	if !ok || provider == "" {
		return ERR, fmt.Errorf("Provider assetid nao encontrado (provider): [%v]", options)
	}
	suffixF, ok := json["suffix_number"].(float64)
	if !ok {
		return ERR, fmt.Errorf("Numero do Sufixo do assetid (suffix_number) nao encontrado: [%v]", json)
	}
	suffix := int(suffixF)
	timestamp := options["timestamp"]
	if !ok || timestamp == "" {
		return ERR, fmt.Errorf("Timestamp nao encontrada (timestamp): [%v]", options)
	}
	fileNum, ok := line["file_number"]
	if !ok || fileNum == "" {
		return ERR, fmt.Errorf("Numero do arquivo nao encontrado (file_number): [%v]", line)
	}
	result := fmt.Sprintf("%s%d%s%04s", provider, suffix, timestamp, fileNum)
	return []string{result}, nil
}

func Date(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	return []string{formatDate(time.Now())}, nil
}

func ConvertDate(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
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

func Condition(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	var condition string
	var ok bool
	if value == "" {
		condition, ok = json["condition"].(string)
		if !ok {
			return ERR, fmt.Errorf("Elemento '%s' inexistente na linha", "condition")
		}
	} else {
		condition = value
	}
	result, err := evalCondition(condition, line)
	if err != nil {
		return ERR, err
	}
	if result {
		value, ok := json["if_true"].(string)
		if !ok {
			return ERR, fmt.Errorf("Elemento '%s' inexistente na linha", "if_true")
		}
		return []string{value}, nil
	} else {
		value, ok := json["if_false"].(string)
		if !ok {
			return ERR, fmt.Errorf("Element '%s' inexistente na linha", "if_false")
		}
		return []string{value}, nil
	}
}

func Eval(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	var expr string
	var ok bool
	if value == "" {
		expr, ok = json["expression"].(string)
		if !ok {
			return ERR, fmt.Errorf("Elemento '%s' inexistente na linha", "expression")
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
		return ERR, fmt.Errorf("Expressao invalida (%v) na linha", expr)
	}
	params := make(map[string]interface{})
	for k, v := range line {
		params[k] = v
	}
	result, err := expression.Evaluate(params)
	return []string{result.(string)}, err
}

func FilterCondition(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	var condition string
	var ok bool
	if value == "" {
		condition, ok = json["condition"].(string)
		if !ok {
			return ERR, fmt.Errorf("Elemento '%s' inexistente na linha", "Value")
		}
	} else {
		condition = value
	}
	_, err := evalCondition(condition, line)
	if err != nil {
		return ERR, err
	}
	return []string{""}, nil
}

func Split(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := getField(value, "field", line, json, options)
	if err != nil {
		return ERR, err
	}
	funcName, ok := json["function2"].(string)
	if !ok {
		return ERR, fmt.Errorf("Elemento '%s' inexistente na linha %v", "function2", line)
	}
	func2, ok := FunctionDict[funcName]
	if !ok {
		return ERR, fmt.Errorf("Funcao2 '%s' invalida na linha", json)
	}
	result := make([]string, 0)
	values := strings.Split(field, ",")
	for _, value := range values {
		res, err := func2(value, line, json, options)
		if err != nil {
			return ERR, err
		}
		result = append(result, res...)
	}
	return result, nil
}

func evalCondition(expr string, line map[string]string) (bool, error) {
	functions := map[string]govaluate.ExpressionFunction{
		"strlen": func(args ...interface{}) (interface{}, error) {
			length := len(args[0].(string))
			return fmt.Sprintf("%d", length), nil
		},
	}
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expr, functions)
	if err != nil {
		return false, fmt.Errorf("Expressao invalida (%v) na linha", expr)
	}
	params := make(map[string]interface{})
	for k, v := range line {
		params[k] = v
	}
	result, err := expression.Evaluate(params)
	return result.(bool), err
}

func IsHD(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func ScreenFormat(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func Exclude(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func Actors(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func ActorsNet(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func BitRate(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func Seconds(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
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

func SurnameName(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
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

func EpisodeId(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

func EpisodeName(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	field, err := Field(value, line, json, options)
	return field, err
}

// Option seeks the result in the JSON file, section "options"
func Option(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	optField, ok := json["field"].(string)
	if !ok {
		return ERR, fmt.Errorf("Elemento '%s' inexistente na linha", "field")
	}
	val, ok := options[optField]
	if !ok {
		return ERR, fmt.Errorf("Elemento '%s' inexistente nas options [%v]", optField, options)
	}
	return []string{val}, nil
}

func Undefined(value string, line map[string]string, json map[string]interface{}, options map[string]string) ([]string, error) {
	if value != "" {
		return []string{value}, nil
	}
	return []string{"##UNDEFINED##"}, fmt.Errorf("Funcao indefinida: [%s]")
}

func getValue(key string, json map[string]interface{}) (string, error) {
	value, ok := json[key].(string)
	if !ok {
		return "###", fmt.Errorf("Chave [%v] nao encontrada no elemento json [%v]", key, json)
	}
	return value, nil
}

func stripchars(str, chr string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(chr, r) < 0 {
			return r
		}
		return -1
	}, str)
}

func RemoveSpaces(val string) string {
	return strings.Join(strings.Fields(val), "_")
}

func RemoveQuotes(val string) string {
	r := strings.NewReplacer("\"", "", "'", "")
	return r.Replace(val)
}

func RemoveAccents(val string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, err := transform.String(t, val)
	if err != nil {
		return "##ERRO##", err
	}
	return output, nil
}

func RemovePunctuation(val string) string {
	return val
}

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

func date() string {
	now := time.Now()
	return formatDate(now)
}

func Timestamp() string {
	now := time.Now()
	return formatTimestamp(now)
}

func formatTimestamp(t time.Time) string {
	// Mon Jan 2 15:04:05 MST 2006
	// %y%m%d%H%M%S
	return t.Format("20060102150405")
}

func formatHMS(t time.Time) string {
	return t.Format("15:04:05")
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func truncateSuffix(value string, suffix string, line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return value, nil
	}
	max, err := strconv.Atoi(val)
	if err != nil {
		return ERR[0], fmt.Errorf("Valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	sufLen := len(suffix)
	if valLen+sufLen <= max {
		return value, nil
	}
	if sufLen+1 >= max {
		return ERR[0], fmt.Errorf("Sufixo [%s] nao pode ser aplicado porque estoura o tamanho maximo [%d] no elemento [%s]", suffix, max, value)
	}
	runes := []rune(value)
	safeSubstring := string(runes[0:max])
	return safeSubstring, nil
}

func truncate(value string, line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return value, nil
	}
	max, err := strconv.Atoi(val)
	if err != nil {
		return ERR[0], fmt.Errorf("Valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	if valLen <= max {
		return value, nil
	}
	runes := []rune(value)
	safeSubstring := string(runes[0:max])
	return safeSubstring, nil
}
