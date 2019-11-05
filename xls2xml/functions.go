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

const ERR = "#ERRO#"

// FunctionDict is the relation between the operation name and the function
var FunctionDict = map[string]func(map[string]string, map[string]interface{}, map[string]string) (string, error){
	"fixed":           Fixed,
	"field":           Field,
	"field_validated": FieldValidated,
	"field_noacc":     FieldNoAccents,
	"field_trim":      FieldTrim,
	"field_no_quotes": FieldNoQuotes,
	"field_money":     FieldMoney,
	"suffix":          Suffix,
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
}

func Process(funcName string, line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	// fmt.Printf("=> %s\n", funcName)
	if funcName == "" {
		return ERR, fmt.Errorf("'function' nao especificada")
	}
	function, ok := FunctionDict[funcName]
	if ok {
		return function(line, json, options)
	}
	fmt.Printf("Warning: funcao [%s] nao existe!\n", funcName)
	result, _ := Undefined(line, json, options)
	return result, fmt.Errorf("Funcao nao definida: [%s]", funcName)
}

func truncate(value string, suffix string, line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return value, nil
	}
	max, err := strconv.Atoi(val)
	if err != nil {
		return ERR, fmt.Errorf("Valor nao numerico em maxlenght: [%v]", val)
	}
	valLen := len(value)
	sufLen := len(suffix)
	if valLen+sufLen <= max {
		return value, nil
	}
	if sufLen+1 >= max {
		return ERR, fmt.Errorf("Sufixo [%s] nao pode ser aplicado porque estoura o tamanho maximo [%d] no elemento [%s]", suffix, max, value)
	}
	runes := []rune(value)
	safeSubstring := string(runes[0:max])
	return safeSubstring, nil
}

// Fixed returns the same value
func Fixed(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	val, err := getValue("Value", json)
	return val, err
}

func FieldMoney(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	val, err := getValue("Value", json)
	if err != nil {
		return ERR, err
	}
	return formatMoney(val)
}

func Field(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	value, err := getField("Name", line, json, options)
	return value, err
}

func getField(field string, line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	// fmt.Printf("field=%#v, json=%#v, line=%#v, options=%#v\n", field, json, line, options)
	val, err := getValue("field", json)
	if err != nil {
		return ERR, err
	}
	value, ok := line[val]
	if !ok {
		return ERR, fmt.Errorf("Element '%s' inexistente na linha [%v]", val, line)
	}
	// fmt.Printf("field(%v) = [%v]\n", field, value)
	return value, nil
}

func FieldValidated(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func FieldNoAccents(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	if err != nil {
		return ERR, err
	}
	result, err := RemoveAccents(field)
	return result, err
}

func FieldTrim(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	if err != nil {
		return ERR, err
	}
	return strings.TrimSpace(field), nil
}

func FieldNoQuotes(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	if err != nil {
		return ERR, err
	}
	return RemoveQuotes(field), nil
}

func Suffix(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	suffix, ok := json["suffix"].(string)
	if !ok || suffix == "" {
		return ERR, fmt.Errorf("Sufixo nao encontrado: [%v]", json)
	}
	field, err := Field(line, json, options)
	if err != nil {
		return ERR, err
	}
	// fmt.Printf("-->> %v\n", field)
	noacc, err := RemoveAccents(field)
	if err != nil {
		return ERR, err
	}
	val := ReplaceAllNonAlpha(noacc)
	val, err = truncate(val, suffix, line, json, options)
	if err != nil {
		return ERR, err
	}
	result := fmt.Sprintf("%s_%s", val, suffix)
	//	fmt.Printf("-->> %v\n", result)
	return result, nil
}

func AssetId(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func Date(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	return formatDate(time.Now()), nil
}

func ConvertDate(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := time.Parse("01/02/06", field)
	if err != nil {
		return ERR, err
	}
	return formatDate(t), nil
}

func Condition(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	condition, ok := json["condition"].(string)
	if !ok {
		return ERR, fmt.Errorf("Element '%s' inexistente na linha [%v]", "condition", line)
	}
	result, err := evalCondition(condition, line)
	if err != nil {
		return ERR, err
	}
	if result {
		value, ok := json["if_true"].(string)
		if !ok {
			return ERR, fmt.Errorf("Element '%s' inexistente na linha [%v]", "if_true", line)
		}
		return value, nil
	} else {
		value, ok := json["if_false"].(string)
		if !ok {
			return ERR, fmt.Errorf("Element '%s' inexistente na linha [%v]", "if_false", line)
		}
		return value, nil
	}
}

func FilterCondition(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	_, ok := json["condition"].(string)
	if !ok {
		return ERR, fmt.Errorf("Element '%s' inexistente na linha [%v]", "Value", line)
	}
	condition, ok := json["condition"].(string)
	if !ok {
		return ERR, fmt.Errorf("Element '%s' inexistente na linha [%v]", "condition", line)
	}
	_, err := evalCondition(condition, line)
	if err != nil {
		return ERR, err
	}
	return "", nil
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
		return false, fmt.Errorf("Expressao invalida (%v) na linha [%v]", expr, line)
	}
	params := make(map[string]interface{})
	for k, v := range line {
		params[k] = v
	}
	result, err := expression.Evaluate(params)
	return result.(bool), err
}

func IsHD(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func ScreenFormat(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func Exclude(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func Actors(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func ActorsNet(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func BitRate(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func Seconds(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	if err != nil {
		return ERR, err
	}
	t, err := time.Parse("03:04:05", field)
	if err != nil {
		return ERR, err
	}
	return formatHMS(t), nil
}

func SurnameName(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func EpisodeId(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func EpisodeName(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	field, err := Field(line, json, options)
	return field, err
}

func Option(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	optField, ok := json["field"].(string)
	if !ok {
		return ERR, fmt.Errorf("Elemento '%s' inexistente na linha [%v]", "field", line)
	}
	value, ok := options[optField]
	if !ok {
		return ERR, fmt.Errorf("Elemento '%s' inexistente nas options [%v]", optField, options)
	}
	return value, nil
}

func Undefined(line map[string]string, json map[string]interface{}, options map[string]string) (string, error) {
	return "##UNDEFINED##", fmt.Errorf("Funcao indefinida: [%s]")
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

func timestamp() string {
	// Mon Jan 2 15:04:05 MST 2006
	now := time.Now()
	// %y%m%d%H%M%S
	return formatTimestamp(now)
}

func formatTimestamp(t time.Time) string {
	return t.Format("20060102150405")
}

func formatHMS(t time.Time) string {
	return t.Format("15:04:05")
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}
