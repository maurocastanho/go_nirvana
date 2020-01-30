package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"

	xw "github.com/shabbyrobe/xmlwriter"
	"golang.org/x/text/encoding/charmap"
)

const (
	normalS  = 0
	commentS = 1
)

// XMLWriter writes XML files
type XMLWriter struct {
	fileName string
	systemID string
	w        *xw.Writer
	b        *bytes.Buffer
	comm     *xw.Writer
	commB    *bytes.Buffer
	ec       *xw.ErrCollector
	status   int
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
func (wr *XMLWriter) StartElem(name string, _ ElemType) error {
	switch wr.status {
	case normalS:
		return wr.w.StartElem(xw.Elem{Name: name})
	case commentS:
		return wr.comm.StartElem(xw.Elem{Name: name})
	}
	return fmt.Errorf("invalid state: %d", wr.status)
}

// WriteAttr adds an attribute to the current XML attribute
func (wr *XMLWriter) WriteAttr(name string, value string, vtype string) error {
	ERRS := "#ERRO#"
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
			val = ERRS
			break
		}
		hours := time / 3600
		minutes := int64(math.Ceil((float64(time) - float64(hours)*3600) / 60))
		val = fmt.Sprintf("%02d:%02d", hours, minutes)
	case "time_s":
		sec, err := ToTimeSeconds(value)
		if err != nil {
			fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERRS
			break
		}
		val = fmt.Sprintf("%d", sec)
	case "boolean":
		if value != "true" && value != "false" {
			val = ERRS
			break
		}
		val = value
	}
	switch wr.status {
	case normalS:
		return wr.w.WriteAttr(xw.Attr{Name: name, Value: val})
	case commentS:
		return wr.comm.WriteAttr(xw.Attr{Name: name, Value: val})
	}
	return fmt.Errorf("invalid state: %d", wr.status)
}

// EndElem closes a XML element
func (wr *XMLWriter) EndElem(name string) error {
	switch wr.status {
	case normalS:
		return wr.w.EndElem(name)
	case commentS:
		return wr.comm.EndElem(name)
	}
	return fmt.Errorf("invalid state: %d", wr.status)
}

// StartComment marks the start of a comment section
func (wr *XMLWriter) StartComment(name string) error {
	// doc := xw.Doc{}
	wr.status = commentS
	wr.w.WriteRaw(fmt.Sprintf("\n<!-- %s\n", name))
	//return wr.comm.StartDoc(doc)
	return nil
}

// EndComment closes a comment section
func (wr *XMLWriter) EndComment(_ string) error {
	//err := wr.comm.Flush()
	//if err != nil {
	//	return err
	//}
	//comments := wr.commB.String()
	//err = wr.w.WriteRaw("\n")
	//if err != nil {
	//	return err
	//}
	//err = wr.w.WriteRaw(comments)
	//if err != nil {
	//	return err
	//}
	//wr.commB.Reset()
	wr.w.WriteRaw("-->\n")
	wr.status = normalS
	return nil
}

// OpenOutput prepares to write a XML file
func (wr *XMLWriter) OpenOutput() (err error) {
	encod := charmap.ISO8859_1.NewEncoder()
	wr.b = &bytes.Buffer{}
	wr.w = xw.OpenEncoding(wr.b, "ISO-8859-1", encod, xw.WithIndentString("\t"))
	wr.commB = &bytes.Buffer{}
	wr.comm = xw.OpenEncoding(wr.commB, "ISO-8859-1", encod, xw.WithIndentString("\t"))
	wr.ec = &xw.ErrCollector{}
	defer wr.ec.Panic()
	doc := xw.Doc{}
	err = wr.w.StartDoc(doc)
	if err != nil {
		return
	}
	doc2 := xw.Doc{}
	err = wr.comm.StartDoc(doc2)
	if err != nil {
		return
	}
	if true {
		wr.ec.Do(
			wr.w.StartDTD(xw.DTD{Name: "ADI", SystemID: wr.systemID}),
			wr.w.EndDTD(),
		)
	}
	return nil
}

// WriteAndClose write all data to the extenal file
func (wr *XMLWriter) WriteAndClose(filename string) (err error) {
	err = wr.w.EndAllFlush()
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", filename, err)
		return
	}
	err = ioutil.WriteFile(filename, wr.b.Bytes(), 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", filename, err)
		return
	}
	return
}
