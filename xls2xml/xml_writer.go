package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"

	xw "github.com/shabbyrobe/xmlwriter"
	"golang.org/x/text/encoding/charmap"
)

// XMLWriter writes XML files
type XMLWriter struct {
	fileName string
	systemID string
	b        *bytes.Buffer
	ec       *xw.ErrCollector
	w        *xw.Writer
}

// NewXMLWriter creates a new struct
func NewXMLWriter(filename string, systemID string) *XMLWriter {
	w := XMLWriter{fileName: filename, systemID: systemID}
	return &w
}

// Suffix returns the output file extension
func (wr *XMLWriter) Suffix() string {
	return ".xml"
}

// Filename returns the output file name
func (wr *XMLWriter) Filename() string {
	return wr.fileName
}

// StartElem starts a XML element
func (wr *XMLWriter) StartElem(name string, _ ElemType) {
	wr.w.StartElem(xw.Elem{Name: name})
}

// WriteAttr adds an attribute to the current XML attribute
func (wr *XMLWriter) WriteAttr(name string, value string, vtype string) {
	ERR := "#ERRO#"
	var val string
	switch vtype {
	case "", "string":
		val = value
	case "int":
		val = value
	case "time":
		time, err := ToTimeSeconds(value)
		if err != nil {
			fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERR
			break
		}
		hours := time / 3600
		minutes := int64(math.Ceil((float64(time) - float64(hours)*3600) / 60))
		val = fmt.Sprintf("%02d:%02d", hours, minutes)
	case "time_s":
		sec, err := ToTimeSeconds(value)
		if err != nil {
			fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERR
			break
		}
		val = fmt.Sprintf("%d", sec)
	case "boolean":
		if value != "true" && value != "false" {
			val = ERR
			break
		}
		val = value
	}

	wr.w.WriteAttr(xw.Attr{Name: name, Value: val})
}

// EndElem closes a XML element
func (wr *XMLWriter) EndElem(name string) {
	wr.w.EndElem(name)
}

// OpenOutput prepares to write a XML file
func (wr *XMLWriter) OpenOutput() {
	wr.b = &bytes.Buffer{}
	encod := charmap.ISO8859_1.NewEncoder()
	wr.w = xw.OpenEncoding(wr.b, "ISO-8859-1", encod, xw.WithIndentString("\t"))
	wr.ec = &xw.ErrCollector{}
	defer wr.ec.Panic()
	doc := xw.Doc{}
	wr.w.StartDoc(doc)
	if true {
		wr.ec.Do(
			wr.w.StartDTD(xw.DTD{Name: "ADI", SystemID: wr.systemID}),
			wr.w.EndDTD(),
		)
	}
}

// WriteAndClose write all data to the extenal file
func (wr *XMLWriter) WriteAndClose(filename string) error {
	wr.w.EndAllFlush()
	err := ioutil.WriteFile(filename, wr.b.Bytes(), 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", filename, err)
	}
	return err
}
