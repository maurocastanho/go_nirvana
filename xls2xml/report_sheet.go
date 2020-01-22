package main

import (
	"fmt"

	ex "github.com/360EntSecGroup-Skylar/excelize"
	// "github.com/mgutz/ansi"
)

type ReportSheet struct {
	filepath string
}

func NewReportSheet(filename string) *ReportSheet {
	w := ReportSheet{filepath: filename}
	return &w
}

func (rs *ReportSheet) processColumns(_ string, json []interface{}, lines []lineT, wr Writer) (err2 []error) {
	for _, v := range json {
		switch vv := v.(type) {
		case map[string]interface{}:
			err2 = appendErrors(err2, rs.processColumn(vv, lines, wr)...)
		}
	}
	return
}

func (rs *ReportSheet) processColumn(json jsonT, lines []lineT, _ Writer) (err []error) {
	name := json["Name"].(string)
	function, ok := json["function"].(string)
	if ok {
		procVals, err2 := Process(function, lines, json, options)
		err = appendErrors(err, err2)
		for _, procVal := range procVals {
			fmt.Printf("%s: %#v\n", name, procVal)
		}
	}
	return
}

// OpenFile opens a xls file for writing
func (rs *ReportSheet) OpenFile(sheetName string) (*ex.File, error) {
	result, err := ex.OpenFile(rs.filepath)
	if err != nil {
		return nil, err
	}
	index := result.NewSheet(sheetName)
	result.SetActiveSheet(index)
	return result, nil
}

// WriteHeader writes the header line
func (rs *ReportSheet) WriteHeader(f *ex.File, columns []string) {

}

// WriteLine writes a line into a sheet
func (rs *ReportSheet) WriteLine(f *ex.File, columns []string) {

}

// Save saves file to disk
func (rs *ReportSheet) Save(f *ex.File, filepath string) error {
	return f.SaveAs(filepath)
}

func (rs *ReportSheet) Filename() string {
	return rs.filepath
}

func (rs *ReportSheet) Suffix() string {
	return ".xlsx"
}

func OpenOutput() {

}

func StartElem(_ string, _ ElemType) {

}

func EndElem(_ string) {

}

func WriteAttr(name string, value string) {

}

func WriteAndClose(filename string) error {
	return nil
}
