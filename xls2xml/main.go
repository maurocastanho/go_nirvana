package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	xw "github.com/shabbyrobe/xmlwriter"
	"golang.org/x/text/encoding/charmap"
)

type configType struct {
	Options []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"options"`

	Elements []struct { // ADI
		Name  string `json:"name"`
		Attrs []struct {
			Name     string `json:"Name"`
			Value    string `json:"Value"`
			Function string `json:"function"`
		} `json:"attrs"`
		Elements []struct { // ADI Metadata
			Name     string     `json:"name"`
			Elements []struct { // Asset
				Name    string `json:"name"`
				Options []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"options,omitifempty"`
				Elements []struct { // Metadata
					Name  string     `json:"name"`
					Attrs []struct { // AMS
						Name         string `json:"Name"`
						Value        string `json:"Value"`
						Field        string `json:"field"`
						Function     string `json:"function"`
						Maxlength    string `json:"maxlength,omitempty"`
						Suffix       string `json:"suffix,omitempty"`
						SuffixNumber int    `json:"suffix_number,omitempty"`
						Prefix       string `json:"prefix,omitempty"`
					} `json:"attrs"`
					SingleAttrs []struct { // AppData
						App      string `json:"App"`
						Name     string `json:"Name"`
						Value    string `json:"Value"`
						Field    string `json:"field"`
						Function string `json:"function"`
					} `json:"single_attrs"`
				} `json:"elements"`
			} `json:"elements"`
		} `json:"elements"`
	} `json:"elements"`
}

func main() {
	/*
			f := excelize.NewFile()
			var (
				orientation excelize.PageLayoutOrientation
				paperSize   excelize.PageLayoutPaperSize
				fitToHeight excelize.FitToHeight
				fitToWidth  excelize.FitToWidth
			)
			if err := f.GetPageLayout("Sheet1", &orientation); err != nil {
				panic(err)
			}
			if err := f.GetPageLayout("Sheet1", &paperSize); err != nil {
				panic(err)
			}
			if err := f.GetPageLayout("Sheet1", &fitToHeight); err != nil {
				panic(err)
			}

			if err := f.GetPageLayout("Sheet1", &fitToWidth); err != nil {
				panic(err)
			}
			fmt.Println("Defaults:")
			fmt.Printf("- orientation: %q\n", orientation)
			fmt.Printf("- paper size: %d\n", paperSize)
			fmt.Printf("- fit to height: %d\n", fitToHeight)
			fmt.Printf("- fit to width: %d\n", fitToWidth)

		type Ams struct {
			Provider     string `xml:"Provider,attr"`
			Product      string `xml:"Product,attr"`
			AssetName    string `xml:"Asset_Name,attr"`
			VersionMajor int    `xml:"Version_Major,attr"`
			VersionMinor int    `xml:"Version_Minor,attr"`
			Description  string `xml:"Description,attr"`
			CreationDate string `xml:"Creation_Date,attr"`
			ProviderID   string `xml:"Provider_ID,attr"`
			AssetID      string `xml:"Asset_ID,attr"`
			AssetClass   string `xml:"Asset_Class,attr"`
		}
		type AppData struct {
			App   string `xml:"App,attr"`
			Name  string `xml:"Name,attr"`
			Value string `xml:"Value,attr"`
		}
		type Metadata struct {
			Ams      Ams       `xml:"AMS,allowempty"`
			AppDatas []AppData `xml:"App_Data"`
		}
		type Content struct {
			Value string `xml:"value,attr"`
		}
		type Asset struct {
			Metadata Metadata `xml:"Metadata"`
			Content  `xml:"Content,omitempty"`
		}
		type Adi struct {
			Xmlns    string   `xml:"xmlns,attr"`
			Metadata Metadata `xml:"Metadata"`
			Assets   []Asset  `xml:"Asset"`
		}

		a := Adi{
			Xmlns: "http://www.eventis.nl/PRODIS/ADI",
			Metadata: Metadata{
				Ams: Ams{Provider: "NIRVANA", Product: "", AssetName: "Loris_na_TV_Serie_Emagracao_Ao_Ar_Livre_Package",
					VersionMajor: 1, VersionMinor: 0, Description: "Loris na TV - Serie Emagração - Ao Ar Livre",
					CreationDate: "2015-08-17", ProviderID: "nirvana.com", AssetID: "NIRV1150817160126003", AssetClass: "package"},
				AppDatas: []AppData{
					{App: "MOD", Name: "Metadata_Spec_Version", Value: "Metadata_Spec_Version"},
				},
			},
			Assets: []Asset{
				Asset{
					Metadata: Metadata{
						Ams{Provider: "NIRVANA", Product: "", AssetName: "Loris_na_TV_Serie_Emagracao_Ao_Ar_Livre_Title",
							VersionMajor: 1, VersionMinor: 0, Description: "Loris na TV - Serie Emagração - Ao Ar Livre",
							CreationDate: "2015-08-17", ProviderID: "nirvana.com", AssetID: "NIRV1150817160126003",
							AssetClass: "title"},
						[]AppData{
							{App: "MOD", Name: "Type", Value: "title"},
							{App: "MOD", Name: "Title_Sort_Name", Value: "Loris na TV - Serie Em"},
						},
					},
				},
				Asset{
					Metadata: Metadata{
						Ams{Provider: "NIRVANA", Product: "", AssetName: "Loris_na_TV_Serie_Emagracao_Ao_Ar_Livre_Movie",
							VersionMajor: 1, VersionMinor: 0, Description: "Loris na TV - Serie Emagração - Ao Ar Livre",
							CreationDate: "2015-08-17", ProviderID: "nirvana.com", AssetID: "NIRV1150817160126003",
							AssetClass: "movie"},
						[]AppData{
							{App: "MOD", Name: "Type", Value: "movie"},
							{App: "MOD", Name: "Screen_Format", Value: "Widescreen"},
						},
					},
					Content: Content{
						Value: "loris_na_tv_serie_emagracao_t1_e3.ts",
					},
				},
			},
		}

		enc := xml.NewEncoder(os.Stdout)
		enc.Indent("  ", "    ")
		if err := enc.Encode(a); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	*/

	file, err := os.Open("config.json")
	if err != nil {
		log(err)
		return
	}
	var config configType

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		log(err)
		return
	}
	newBuf := latinToUTF8(buf)

	err = json.Unmarshal(newBuf, &config)
	if err != nil {
		log(err)
		return
	}
	fmt.Printf("%v\n", config)

	var objmap interface{}
	err = json.Unmarshal(newBuf, &objmap)
	if err != nil {
		log(err)
		return
	}
	fmt.Printf("--> %v\n", objmap)

	b := &bytes.Buffer{}
	enc := charmap.ISO8859_1.NewEncoder()
	w := xw.OpenEncoding(b, "ISO-8859-1", enc, xw.WithIndentString("\t"))
	ec := &xw.ErrCollector{}
	defer ec.Panic()
	doc := xw.Doc{}
	ec.Do(
		w.StartDoc(doc),
		w.StartDTD(xw.DTD{Name: "ADI", SystemID: "ADI.DTD"}),
		w.EndDTD(),
		w.StartElem(xw.Elem{Name: "foo"}),
		w.WriteAttr(xw.Attr{Name: "a1", Value: "val1"}),
		w.WriteAttr(xw.Attr{Name: "a2", Value: "áá"}),
		w.StartElem(xw.Elem{Name: "bar"}),
		w.WriteAttr(xw.Attr{Name: "a1", Value: "val1"}),
		w.WriteAttr(xw.Attr{Name: "a2", Value: "val2"}),
		w.EndAllFlush(),
	)
	fmt.Println(b.String())
}

func latinToUTF8(buffer []byte) []byte {
	buf := make([]rune, len(buffer))
	for i, b := range buffer {
		buf[i] = rune(b)
	}
	return []byte(string(buf))
}

func log(err error) {
	fmt.Fprintf(os.Stderr, "ERRO: %v\n", err)
}

func fixed(val string) string {
	return val
}

//func field(idx string)
