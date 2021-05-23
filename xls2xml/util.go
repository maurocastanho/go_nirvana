package main

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"hash/crc32"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

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
func replaceAllNonAlpha(val string) (string, error) {
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		return "", err
	}
	val = reg.ReplaceAllString(val, "_")
	r := []rune(val)
	var b strings.Builder
	last := ' '
	l := len(val)
	blank := true
	for i, c := range r {
		const UNDERSCORE = 95                                         // 95 is unicode for '_'
		if !(c == UNDERSCORE && (last == UNDERSCORE || (i == l-1))) { // prevents duplicates for '_'
			if c != UNDERSCORE {
				// tests when a char different from '_' appears
				blank = false
			}
			if !blank {
				// first char cannot be '_'
				b.WriteRune(c)
			}
		}
		last = c
	}
	return b.String(), nil
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

// truncates element in max chars, preserving suffix
func truncateSuffix(value string, suffix string, max int) (string, error) {
	valLen := len(value)
	sufLen := len(suffix)
	if valLen+sufLen <= max {
		return value, nil
	}
	if sufLen+1 >= max {
		return errorMessage[0].val, fmt.Errorf("sufixo [%s] nao pode ser aplicado porque estoura "+
			"o tamanho maximo [%d] no elemento [%s]", suffix, max, value)
	}
	r := []rune(value)
	if l := len(r); max > l {
		max = l
	}
	for l := max - 1; l >= 0 && r[l] == '_'; l-- {
		max--
	}
	safeSubstring := string(r[0:max])
	return testInvalidChars(safeSubstring)
}

func truncate(value string, _ *lineT, json jsonT, _ optionsT) (string, error) {
	val, err := getValue("maxlength", json)
	if val == "" || err != nil {
		return testInvalidChars(value)
	}
	max, errA := strconv.Atoi(val)
	if errA != nil {
		return errorMessage[0].val, fmt.Errorf("valor nao numerico em maxlenght: [%v]", val)
	}
	r := []rune(value)
	if len(r) <= max {
		// size ok, return
		return testInvalidChars(value)
	}
	// truncate
	safeSubstring := string(r[0:max])
	return testInvalidChars(safeSubstring)
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
		// conversion from float failed: try to convert from string
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

func formatNumberString(s string) (string, error) {
	switch len(s) {
	case 0:
		return "00", nil
	case 1:
		return "0" + s, nil
	default:
		return s, nil
	}
}

func testInvalidChars(s string) (string, error) {
	_, err := charmap.ISO8859_1.NewEncoder().String(s)
	if err != nil {
		result := make([]rune, 0)
		//fmt.Printf("%s\n", err)
		ok := false
		rs1 := []rune(s)
		var r rune
		var b rune
		errors := make([]rune, 0)
		for _, r = range rs1 {
			b = r
			_, ok = charmap.ISO8859_1.EncodeRune(r)
			if ok != true {
				if r >= 8208 && r <= 8213 { // test if it is a kind of dash (en dash, em dash, etc)
					b = '-' // replace with hifen
				} else {
					b = '?' // error mark
					fmt.Printf("%d %s\n", r, string(r))
					errors = append(errors, r)
				}
			}
			result = append(result, b)
		}
		if len(errors) > 0 {
			return "#ERRO#", fmt.Errorf("%d caracter(es) invalido(s) [%v] na string [%s]", len(errors), string(errors), s)
		}
		return string(result), nil
	}
	return s, nil
}

func UUIDfromString(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	b := []byte(s)
	l := len(s)
	h := crc32.ChecksumIEEE(b) // take string checksum
	b2 := make([]byte, 16)     // prepare hash
	// insert checksum into hash
	for i := 0; i < 4; i++ {
		r := h % 8
		h /= 8
		b2[i] = byte(r)
	}
	// complete hash with characters of b
	for i := 4; i < 16 && i < l+4; i++ {
		b2[i] = b[i-4]
	}
	// uses hash to compute uuid
	uu, err := uuid.FromBytes(b2)
	if err != nil {
		return "", err
	}
	return uu.String(), nil
}
