package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"time"

	xw "github.com/shabbyrobe/xmlwriter"
	"golang.org/x/text/encoding/charmap"
)

const (
	normalS  = 0
	commentS = 1
)

// xmlWriter writes XML files
type xmlWriter struct {
	fileName string
	systemID string
	w        *xw.Writer
	b        *bytes.Buffer
	ec       *xw.ErrCollector
	status   int
	testing  bool
	time     time.Time
}

// StartMap starts a map element
func (wr *xmlWriter) StartMap() {

}

// EndMap closes a map element
func (wr *xmlWriter) EndMap() {

}

// NewXMLWriter creates a new struct
func newXMLWriter(filename string, systemID string) *xmlWriter {
	w := xmlWriter{fileName: filename, systemID: systemID, testing: false}
	return &w
}

// Suffix returns the output file extension
func (wr *xmlWriter) Suffix() string {
	return ".xml"
}

// Filename returns the output file name
func (wr *xmlWriter) Filename() string {
	return wr.fileName
}

// StartElem starts a XML element
func (wr *xmlWriter) StartElem(name string, _ elemType) error {
	wr.ec.Do(wr.w.StartElem(xw.Elem{Name: name}))
	if wr.ec.Err != nil {
		return fmt.Errorf(wr.ec.Error())
	}
	return nil
}

func (wr *xmlWriter) Write(value string) error {
	wr.ec.Do(wr.w.WriteText(value))
	if wr.ec.Err != nil {
		return fmt.Errorf(wr.ec.Error())
	}
	return nil
}

// WriteAttr adds an attribute to the current XML attribute
func (wr *xmlWriter) WriteAttr(name string, value string, vtype string, attrType string) error {
	ERRS := "#ERRO#"
	var val string
	switch vtype {
	case "", "string":
		val = value
	case "int":
		val = value
	case "time":
		sec, err := toTimeSeconds(value)
		if err != nil {
			//fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERRS
			break
		}
		hours := sec / 3600
		minutes := int64(math.Ceil((float64(sec) - float64(hours)*3600) / 60))
		val = fmt.Sprintf("%02d:%02d", hours, minutes)
	case "time_s":
		sec, err := toTimeSeconds(value)
		if err != nil {
			//fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERRS
			break
		}
		val = fmt.Sprintf("%d", sec)
	case "time_m":
		sec, err := toTimeSeconds(value)
		if err != nil {
			//fmt.Printf("%s *--------> %#v\n", name, val)
			val = ERRS
			break
		}
		min := int64(math.Ceil(float64(sec) / 60.0))
		val = fmt.Sprintf("%d", min)
	case "boolean":
		if value != "true" && value != "false" {
			val = ERRS
			break
		}
		val = value
	}

	if attrType == "ott" {
		if val != "" {
			wr.ec.Do(
				wr.Write(val),
			)
			if wr.ec.Err != nil {
				return fmt.Errorf(wr.ec.Error())
			}
		}
	} else {
		wr.ec.Do(wr.w.WriteAttr(xw.Attr{Name: name, Value: val}))
		if wr.ec.Err != nil {
			return fmt.Errorf(wr.ec.Error())
		}
	}
	return nil
}

// EndElem closes a XML element
func (wr *xmlWriter) EndElem(name string, _ elemType) error {
	wr.ec.Do(wr.w.EndElem(name))
	if wr.ec.Err != nil {
		return fmt.Errorf(wr.ec.Error())
	}
	return nil
}

// StartComment marks the start of a comment section
func (wr *xmlWriter) StartComment(name string) error {
	wr.status = commentS
	wr.ec.Do(wr.w.WriteRaw(fmt.Sprintf("\n<!-- %s\n", name)))
	if wr.ec.Err != nil {
		return fmt.Errorf(wr.ec.Error())
	}
	return nil

}

// EndComment closes a comment section
func (wr *xmlWriter) EndComment(_ string) error {
	wr.status = normalS
	wr.ec.Do(wr.w.WriteRaw(" -->\n"))
	if wr.ec.Err != nil {
		return fmt.Errorf(wr.ec.Error())
	}
	return nil

}

// OpenOutput prepares to write a XML file
func (wr *xmlWriter) OpenOutput() (err error) {
	encod := charmap.ISO8859_1.NewEncoder()
	wr.b = &bytes.Buffer{}
	wr.w = xw.OpenEncoding(wr.b, "ISO-8859-1", encod, xw.WithIndentString("\t"))
	wr.ec = &xw.ErrCollector{}
	doc := xw.Doc{}
	err = wr.w.StartDoc(doc)
	if err != nil {
		return
	}
	if wr.systemID != "" {
		wr.ec.Do(
			wr.w.StartDTD(xw.DTD{Name: "ADI", SystemID: wr.systemID}),
			wr.w.EndDTD(),
		)
	}
	if wr.ec.Err != nil {
		return fmt.Errorf(wr.ec.Error())
	}
	return nil
}

// WriteAndClose write all data to the extenal file
func (wr *xmlWriter) WriteAndClose(filename string) (err error) {
	err = wr.w.EndAllFlush()
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", filename, err)
		return
	}
	if wr.testing {
		return
	}
	err = ioutil.WriteFile(filename, wr.b.Bytes(), 0644)
	if err != nil {
		err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", filename, err)
		return
	}
	return
}

func (wr *xmlWriter) getBuffer() []byte {
	return wr.b.Bytes()
}

// WriteExtras writes additional files
func (wr *xmlWriter) WriteExtras() {
}
