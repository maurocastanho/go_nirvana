package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/plandem/xlsx/format/styles"
	colOptions "github.com/plandem/xlsx/types/options/column"

	"github.com/plandem/xlsx"
	// "github.com/mgutz/ansi"
)

// ReportSheet represents the sheet generated for the provider
type reportSheet struct {
	filepath    string
	sheetName   string
	xlsFile     *xlsx.Spreadsheet
	sheet       xlsx.Sheet
	currentRow  int
	currentCol  int
	numCols     int
	numLines    int
	headerStyle styles.DirectStyleID
	bodyStyle   styles.DirectStyleID
	timeStyle   styles.DirectStyleID
	moneyStyle  styles.DirectStyleID
}

// StartMap starts a map element
func (rs *reportSheet) StartMap() {

}

// EndMap closes a map element
func (rs *reportSheet) EndMap() {

}

// NewReportSheet creates a new struct
func newReportSheet(filename string, sheetName string, nCols int, nLines int) *reportSheet {
	rs := reportSheet{
		filepath:   filename,
		sheetName:  sheetName,
		numCols:    nCols,
		numLines:   nLines + 1,
		currentRow: 1,
	}
	return &rs
}

// OpenOutput opens the output file for writing
func (rs *reportSheet) OpenOutput() (err error) {
	st, err := os.Stat(rs.filepath)
	if err == nil {
		if st.IsDir() {
			err = fmt.Errorf("arquivo [%s] nao pode ser aberto pois e' um diretorio", rs.filepath)
			return
		}
		err2 := os.Remove(rs.filepath)
		if err2 != nil {
			err = fmt.Errorf("arquivo [%s] nao pode ser sobrescrito", rs.filepath)
			return
		}
	}
	err = nil
	// Open the XLSX file using file name
	rs.xlsFile = xlsx.New()
	rs.sheet = rs.xlsFile.AddSheet(rs.sheetName)
	rs.sheet.SetDimension(rs.numCols, rs.numLines)

	co := colOptions.New(colOptions.Width(30))
	for i := 0; i < rs.numCols; i++ {
		rs.sheet.Col(i).SetOptions(co)
	}

	// Add styles
	redBold := styles.New(
		styles.Fill.Color("#a0a0a0"),
		styles.Fill.Background("#a0a0a0"),
		styles.Border.Bottom.Color("#000000"),
		styles.Alignment.HAlign(styles.HAlignCenter),
		styles.Font.Bold,
	)
	rs.headerStyle = rs.xlsFile.AddStyles(redBold)

	bodyStyle := styles.New(
		styles.Fill.Color("#ffffff"),
		styles.Fill.Background("#ffffff"),
		styles.Fill.Type(styles.PatternTypeSolid),
	)
	rs.bodyStyle = rs.xlsFile.AddStyles(bodyStyle)

	timeStyle := styles.New(
		styles.Fill.Color("#ffffff"),
		styles.Fill.Background("#ffffff"),
		styles.Fill.Type(styles.PatternTypeSolid),
		styles.NumberFormat("HH:MM:SS"),
	)
	rs.timeStyle = rs.xlsFile.AddStyles(timeStyle)

	moneyStyle := styles.New(
		styles.Fill.Color("#ffffff"),
		styles.Fill.Background("#ffffff"),
		styles.Fill.Type(styles.PatternTypeSolid),
		styles.NumberFormat("\"R$\" #,##0.00"),
	)
	rs.moneyStyle = rs.xlsFile.AddStyles(moneyStyle)

	return
}

// StartElem is unused
func (rs *reportSheet) StartElem(_ string, _ elemType) error {
	return nil
}

// EndElem is unused
func (rs *reportSheet) EndElem(_ string, _ elemType) error {
	return nil
}

func (rs *reportSheet) writeCell(col int, row int, value interface{}, style styles.DirectStyleID) error {
	cell := rs.sheet.Cell(col, row)
	cell.SetStyles(style)
	if cell == nil {
		return fmt.Errorf("celula [%d, %d] na planilha [%s], aba [%s] nao existe", col, row, rs.filepath, rs.sheet.Name())
	}
	cell.SetValue(value)
	return nil
}

func (rs *reportSheet) Write(_ string) error {
	return nil
}

// WriteAttr writes a new attribute as a sheet cell
func (rs *reportSheet) WriteAttr(name string, value string, vtype string, _ string) error {
	//fmt.Printf("name:[%v], value:[%v], vtype:[%v]\n", name, value, vtype)
	style := rs.bodyStyle
	ERRS := "#ERRO#"
	var val interface{}
	switch vtype {
	case "", "string":
		val = value
	case "int":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			val = ERRS
			break
		}
		val = v
	case "money":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			val = ERRS
			break
		}
		val = v
		style = rs.moneyStyle
	case "time":
		time, err := toTimeSeconds(value)
		if err != nil {
			fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERRS
			break
		}
		hours := time / 3600
		minutes := int64(math.Ceil((float64(time) - float64(hours)*3600) / 60))
		val = fmt.Sprintf("%02d:%02d", hours, minutes)
	case "time_s":
		sec, err := toTimeSeconds(value)
		if err != nil {
			fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERRS
			break
		}
		val = float64(sec) / (24.0 * 3600.0)
		style = rs.timeStyle
	case "boolean":
		if value != "true" && value != "false" {
			val = ERRS
			break
		}
		val = value
	}

	if rs.currentRow == 1 {
		err := rs.writeCell(rs.currentCol, 0, name, rs.headerStyle)
		if err != nil {
			logError(err)
			return err
		}
	}
	err := rs.writeCell(rs.currentCol, rs.currentRow, val, style)
	if err != nil {
		logError(err)
		return err
	}
	rs.currentCol++
	return nil
}

// WriteAndClose writes the xls file and closes it
func (rs *reportSheet) WriteAndClose(_ string) (err error) {
	err = rs.xlsFile.SaveAs(rs.filepath)
	if err != nil {
		return
	}
	err = rs.xlsFile.Close()
	if err != nil {
		return
	}
	return
}

func (rs *reportSheet) processColumns(_ string, json []interface{}, lines []lineT, wr writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, rs.processColumn(vv, lines, wr)...)
		}
	}
	return
}

func (rs *reportSheet) processColumn(json jsonT, lines []lineT, _ writer) (err []error) {
	name := json["Name"].(string)
	function, ok := json["function"].(string)
	if ok {
		procVals, err2 := process(function, lines, json, options)
		err = appendErrors(err, err2)
		for _, procVal := range procVals {
			fmt.Printf("%s: %#v\n", name, procVal.val)
		}
	}
	return
}

// StartComment marks the start of a comment section
func (rs *reportSheet) StartComment(_ string) error {
	return nil
}

// EndComment closes a comment section
func (rs *reportSheet) EndComment(_ string) error {
	return nil
}

// Filename returns the output file name
func (rs *reportSheet) Filename() string {
	return rs.filepath
}

// Suffix returns the output file extension
func (rs *reportSheet) Suffix() string {
	return ".xlsx"
}

// NewLine sets the output to a new line in the current sheet
func (rs *reportSheet) newLine() {
	rs.currentRow++
	rs.currentCol = 0
}

// WriteExtras writes additional files
func (rs *reportSheet) WriteExtras() ([]byte, []byte, error) {
	return nil, nil, nil
}
