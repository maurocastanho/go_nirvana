package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"golang.org/x/text/encoding/charmap"
)

//var line = map[string]string{
//	"id":   "1",
//	"name": "2",
//}

func TestFunctionDict(t *testing.T) {
	// x := FunctionDict["fixed"]
	// arg := "aaa"
	// exp := "aaa"
	// res, err := x(arg, line)
	// if err != nil {
	// 	t.Errorf("%v", err)
	// }
	// if exp != res {
	// 	t.Errorf("Fixed function wrong: fixed(\"%s\") = %s", arg, exp)
	// }
}

func TestRemoveSpaces(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"  ", ""},
		{" aa aa  aa   aa ", "aa_aa_aa_aa"},
	}

	for _, table := range tables {
		res := removeSpaces(table.arg)
		if res != table.exp {
			t.Errorf("RemoveSpaces(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestRemoveQuotes(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"'", ""},
		{"\"", ""},
		{"  ", "  "},
		{"'aa'aa'", "aaaa"},
		{"'aa\"'\"a\"a'", "aaaa"},
	}

	for _, table := range tables {
		res := removeQuotes(table.arg)
		if res != table.exp {
			t.Errorf("RemoveQuotes(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestRemoveAccents(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"ć", "c"},
		{"cçc", "ccc"},
		{"!@#$%¨&*()", "!@#$%¨&*()"},
		{"abcdefghijklmnopqrstuvwxyzáéíóúçãõâêîôûàèìòù", "abcdefghijklmnopqrstuvwxyzaeioucaoaeiouaeiou"},
		{"Titulo Original", "Titulo Original"},
	}

	for _, table := range tables {
		res, err := removeAccents(table.arg)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		if res != table.exp {
			t.Errorf("RemoveAccents(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"0", "0.00"},
		{"444", "444.00"},
		{"123.4567", "123.46"},
		{"-123.4567", "-123.46"},
	}

	for _, table := range tables {
		res, err := formatMoney(table.arg)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		if res != table.exp {
			t.Errorf("formatMoney(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2005-11-29T22:08:41+00:00")
	tables := []struct {
		arg time.Time
		exp string
	}{
		{t1, "051129220841"},
	}

	for _, table := range tables {
		res := formatTimestamp(table.arg)
		if res != table.exp {
			t.Errorf("formatTimeStamp(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestFormatDate(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2005-11-29T22:08:41+00:00")
	tables := []struct {
		arg time.Time
		exp string
	}{
		{t1, "2005-11-29"},
	}

	for _, table := range tables {
		res := formatDate(table.arg)
		if res != table.exp {
			t.Errorf("formatDate(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestCondition(t *testing.T) {
	line := map[string]string{"a": "1", "b": "22", "c": "1"}
	tables := []struct {
		arg string
		exp bool
	}{
		{"a=='1'", true},
		{"b=='22'", true},
		{"a==b", false},
		{"a==c", true},
		{"a==c && a!=b", true},
		{"a==c || a==b", true},
		{"a != c || (a==c && a!=b)", true},
		{"a == c && (a!=c || a==b)", false},
	}

	for _, table := range tables {
		res, err := evalCondition(table.arg, line)
		if err != nil {
			t.Error(err)
		}
		if res != table.exp {
			t.Errorf("evalCondition(\"%s\") = [%v], expected [%v]", table.arg, res, table.exp)
		}
	}
}

func TestXmlNet(t *testing.T) {
	json := readConfig("config_net.json")
	initVars(json)
	const expected string = "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<!DOCTYPE ADI SYSTEM \"ADI.DTD\">\n<ADI xmlns=\"http://www.eventis.nl/PRODIS/ADI\">\n\t<Metadata>\n\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Package\" Version_Major=\"1\" Version_Minor=\"0\" Description=\"Friends\" Creation_Date=\"2020-06-19\" Provider_ID=\"warner.com\" Asset_ID=\"WARN1200619015447001\" Asset_Class=\"package\"/>\n\t\t<App_Data App=\"MOD\" Name=\"Metadata_Spec_Version\" Value=\"CableLabsVOD1.1\"/>\n\t</Metadata>\n\t<Asset>\n\t\t<Metadata>\n\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Title\" Version_Major=\"1\" Version_Minor=\"0\" Description=\"Friends\" Creation_Date=\"2020-06-19\" Provider_ID=\"warner.com\" Asset_ID=\"WARN2200619015447001\" Asset_Class=\"title\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"title\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Title_Sort_Name\" Value=\"Friends\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Title_Brief\" Value=\"Friends\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Title\" Value=\"Friends\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Episode_Name\" Value=\"Aquele onde tudo começou\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Episode_ID\" Value=\"01001\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Long\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Medium\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Short\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Rating\" Value=\"12\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Closed_Captioning\" Value=\"N\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Run_Time\" Value=\"1369\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Display_Run_Time\" Value=\"00:23\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Year\" Value=\"1994\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Country_of_Origin\" Value=\"USA\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Studio\" Value=\"Warner Home Video\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Studio_Name\" Value=\"Warner Home Video\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors_Display\" Value=\"Jennifer Aniston, Courteney Cox, Lisa Kudrow, Matt LeBlanc, Matthew Perry, David Schwimmer\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Aniston, Jennifer\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Cox, Courteney\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Kudrow, Lisa\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"LeBlanc, Matt\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Perry, Matthew\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Schwimmer, David\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Director\" Value=\"Burrows, James\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Director_Display\" Value=\"James Burrows\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Category\" Value=\"Série\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Genre\" Value=\"Comédia\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Box_Office\" Value=\"\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Billing_ID\" Value=\"WBH2S\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Licensing_Window_Start\" Value=\"2020-06-10\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Licensing_Window_End\" Value=\"2049-12-31\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Preview_Period\" Value=\"0\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Provider_QA_Contact\" Value=\"MediaCenter\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Contract_Name\" Value=\"Friends\"/>\n\t\t\t<App_Data App=\"MOD\" Name=\"Suggested_Price\" Value=\"1.49\"/>\n\t\t</Metadata>\n\t\t<Asset>\n\t\t\t<Metadata>\n\t\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Movie\" Version_Major=\"1\" Version_Minor=\"0\" Creation_Date=\"2020-06-19\" Description=\"Friends\" Provider_ID=\"warner.com\" Asset_ID=\"WARN3200619010721001\" Asset_Class=\"movie\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"movie\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Screen_Format\" Value=\"Widescreen\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"HDContent\" Value=\"Y\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Bit_Rate\" Value=\"8000\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Watermarking\" Value=\"N\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Audio_Type\" Value=\"Stereo\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Viewing_Can_Be_Resumed\" Value=\"Y\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"CGMS_A\" Value=\"3\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Languages\" Value=\"en\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Languages\" Value=\"pt\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Subtitle_Languages\" Value=\"pt\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_FileSize\" Value=\"1814458124\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_CheckSum\" Value=\"609A5FBB1D0301719462BB798886D43F\"/>\n\t\t\t</Metadata>\n\t\t\t<Content Value=\"friends_s01ep01_hd_da_20_dvb.ts\"/>\n\t\t</Asset>\n\t\t<Asset>\n\t\t\t<Metadata>\n\t\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Poster\" Version_Major=\"1\" Version_Minor=\"0\" Creation_Date=\"2020-06-19\" Description=\"Friends\" Provider_ID=\"warner.com\" Asset_ID=\"WARN4200619010721001\" Asset_Class=\"poster\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"poster\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_FileSize\" Value=\"56725\"/>\n\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_CheckSum\" Value=\"EDCB1162F24A29692343BBC415ECB528\"/>\n\t\t\t</Metadata>\n\t\t\t<Content Value=\"friends_s01ep01_hd_da_20_dvb.jpg\"/>\n\t\t</Asset>\n\t</Asset>\n</ADI>"

	lines := [][]string{
		{"Provider", "Provider id", "Título Original", "Título em Português", "Título em Português do Episódio",
			"Temporada", "Número do Episódio", "Categoria", "Ano", "Bilheteria", "Ranking", "Língua Original", "Estúdio",
			"Classificação Etária", "Genero 1", "Genero 2", "Elenco", "Diretor", "País de Origem", "Sinopse EPG", "Sinopse Resumo",
			"Duração", "Data início no NOW", "Data Fim no NOW", "Formato", "Audio", "Legendado", "Dublado",
			"Billing ID", "Extradata 1", "Extradata 2", "Produto", "Janela Repasse", "Canal", "Box Office", "Versao",
			"Cobrança", "ID", "Movie Size", "Movie MD5", "Poster Size", "Poster MD5", "DOWNLOAD TO GO", "DIAS DE DOWNLOAD",
			"PASTA FTP", "Movie Audio Type", "Trailer ID", "Trailer Size", "Trailer MD5", "Duração Trailer", "Trailer Audio Type"},
		{"WARNER", "warner.com", "Friends", "Friends", "Aquele onde tudo começou", "1", "1", "Série", "1994",
			"1000000", "9", "en", "Warner Home Video", "12", "Comédia", "Programa",
			"Jennifer Aniston, Courteney Cox, Lisa Kudrow, Matt LeBlanc, Matthew Perry, David Schwimmer", "James Burrows", "USA",
			"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai",
			"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai",
			"00:22:49", "06-10-20", "12-31-49", "HD", "en,pt", "pt", "não", "WBH2S", "", "", "TVOD - Catálogo", "S",
			"Nirvana - Catálogo", "200000", "Multi-language", "1.49", "friends_s01ep01_hd_da_20_dvb.ts",
			"1814458124", "609A5FBB1D0301719462BB798886D43F", "56725", "EDCB1162F24A29692343BBC415ECB528", "não", "0",
			"\\rhome\\nirvana\\warner_series_20200608\\dvb\\friends_s01ep01_hd_da_20_dvb", "Stereo"},
	}
	lenLine1 := len(lines[1])
	//fmt.Printf("%#v\n%#v\n%d %d", lines[0], lines[1], len(lines[0]), lenLine1)

	var maplines lineT = make(map[string]string)
	for i := range lines[0] {
		val := ""
		if i < lenLine1 {
			val = lines[1][i]
		}
		maplines[lines[0][i]] = val
	}
	maplines["file_number"] = "1"
	options["timestamp"] = "200619015447"
	//fmt.Printf("%#v\n", maplines)
	xmlWr := newXMLWriter("unit_tests.json", "ADI.DTD")
	xmlWr.testing = true
	err := processLines(json, []lineT{maplines}, xmlWr)
	if err != nil {
		t.Error(err)
	}
	result := xmlWr.getBuffer()

	res2 := decodeISO88599ToUTF8(result)

	assert.Equal(t, expected, res2)
}

func decodeISO88599ToUTF8(bytes []byte) string {
	encoded, _ := charmap.ISO8859_9.NewDecoder().Bytes(bytes)
	return string(encoded[:])
}

//func encodeUTF8ToISO88599(bytes []byte) string {
//	encoded, _ := charmap.ISO8859_9.NewEncoder().Bytes(bytes)
//	return string(encoded[:])
//}
