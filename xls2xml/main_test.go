package main

import (
	"fmt"
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

func TestSurnameName(t *testing.T) {
	t1, err := surnameName(" ", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = surnameName("first", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("first")}, t1)
	t1, err = surnameName("first second", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("second, first")}, t1)
	t1, err = surnameName("first second third", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("second third, first")}, t1)
	t1, err = surnameName("first second third fourth", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("second third fourth, first")}, t1)
}

func TestFirstname(t *testing.T) {
	t1, err := firstName(" ", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = firstName("first", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("first")}, t1)
	t1, err = firstName("first second", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("first")}, t1)
	t1, err = firstName("first second third", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("first")}, t1)
	t1, err = firstName("first second third fourth", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("first")}, t1)
}

func TestLastname(t *testing.T) {
	t1, err := lastName(" ", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = lastName("first", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = lastName("first second", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("second")}, t1)
	t1, err = lastName("first second third", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("third")}, t1)
	t1, err = lastName("first second third fourth", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("fourth")}, t1)
}

func TestMiddlename(t *testing.T) {
	t1, err := middleName(" ", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = middleName("first", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = middleName("first second", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("")}, t1)
	t1, err = middleName("first second third", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("second")}, t1)
	t1, err = middleName("first second third fourth", nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, []resultsT{newResult("second")}, t1)
}

func TestBuildAssetID(t *testing.T) {
	aID := buildAssetID("ABCD", 7, "200702102255", "1")
	assert.Equal(t, "ABCD7200702102255001", aID)
	aID = buildAssetID("A", 7, "200702102255", "1")
	assert.Equal(t, "AAAA7200702102255001", aID)
	aID = buildAssetID("ABCDEF", 7, "200702102255", "1")
	assert.Equal(t, "ABCD7200702102255001", aID)
}

func TestXmlNet(t *testing.T) {
	json, errCf := readConfig("config_net.json")
	if errCf != nil {
		t.Error(errCf)
	}
	initVars(json)
	expected := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<!DOCTYPE ADI SYSTEM \"ADI.DTD\">\n" +
		"<ADI xmlns=\"http://www.eventis.nl/PRODIS/ADI\">\n" +
		"\t<Metadata>\n" +
		"\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Package\" " +
		"Version_Major=\"1\" Version_Minor=\"0\" Description=\"Friends\" Creation_Date=\"2020-06-19\" Provider_ID=\"warner.com\" " +
		"Asset_ID=\"WARN1200619015447001\" Asset_Class=\"package\"/>\n" +
		"\t\t<App_Data App=\"MOD\" Name=\"Metadata_Spec_Version\" Value=\"CableLabsVOD1.1\"/>" +
		"\n" +
		"\t</Metadata>\n" +
		"\t<Asset>\n" +
		"\t\t<Metadata>\n" +
		"\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Title\" Version_Major=\"1\" " +
		"Version_Minor=\"0\" Description=\"Friends\" Creation_Date=\"2020-06-19\" Provider_ID=\"warner.com\" Asset_ID=\"WARN2200619015447001\" " +
		"Asset_Class=\"title\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"title\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Title_Sort_Name\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Title_Brief\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Title\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Episode_Name\" Value=\"Aquele onde tudo começou\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Episode_ID\" Value=\"01001\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Long\" Value=\"Depois que Rachel abandona o noivo no altar, " +
		"ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Medium\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que" +
		" não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" " +
		"Name=\"Summary_Short\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente," +
		" principalmente quando não pode contar com o cartão de crédito do papai\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Rating\" Value=\"12\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Closed_Captioning\" Value=\"N\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Run_Time\" Value=\"1369\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Display_Run_Time\" Value=\"00:23\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Year\" Value=\"1994\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Country_of_Origin\" Value=\"USA\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Studio\" Value=\"Warner Home Video\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Studio_Name\" Value=\"Warner Home Video\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors_Display\" Value=\"Jennifer Aniston, Courteney Cox, Lisa Kudrow, Matt LeBlanc, Matthew Perry, David Schwimmer\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Aniston, Jennifer\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Cox, Courteney\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Kudrow, Lisa\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"LeBlanc, Matt\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Perry, Matthew\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Schwimmer, David\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Director\" Value=\"Burrows, James\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Director_Display\" Value=\"James Burrows\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Category\" Value=\"Série\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Genre\" Value=\"Comédia\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Box_Office\" Value=\"\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Billing_ID\" Value=\"WBH2S\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Licensing_Window_Start\" Value=\"2020-06-10\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Licensing_Window_End\" Value=\"2049-12-31\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Preview_Period\" Value=\"0\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Provider_QA_Contact\" Value=\"MediaCenter\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Contract_Name\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Suggested_Price\" Value=\"1.49\"/>\n" +
		"\t\t</Metadata>\n" +
		"\t\t<Asset>\n" +
		"\t\t\t<Metadata>\n" +
		"\t\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Movie\" Version_Major=\"1\" Version_Minor=\"0\" Creation_Date=\"2020-06-19\" " +
		"Description=\"Friends\" Provider_ID=\"warner.com\" Asset_ID=\"WARN3200619015447001\" Asset_Class=\"movie\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"movie\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Screen_Format\" Value=\"Widescreen\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"HDContent\" Value=\"Y\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Bit_Rate\" Value=\"8000\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Watermarking\" Value=\"N\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Audio_Type\" Value=\"Stereo\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Viewing_Can_Be_Resumed\" Value=\"Y\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"CGMS_A\" Value=\"3\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Languages\" Value=\"en\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Languages\" Value=\"pt\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Subtitle_Languages\" Value=\"pt\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_FileSize\" Value=\"1814458124\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_CheckSum\" Value=\"609A5FBB1D0301719462BB798886D43F\"/>\n" +
		"\t\t\t</Metadata>\n" +
		"\t\t\t<Content Value=\"friends_s01ep01_hd_da_20_dvb.ts\"/>\n" +
		"\t\t</Asset>\n" +
		"\t\t<Asset>\n" +
		"\t\t\t<Metadata>\n" +
		"\t\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Poster\" Version_Major=\"1\" Version_Minor=\"0\" Creation_Date=\"2020-06-19\" " +
		"Description=\"Friends\" Provider_ID=\"warner.com\" Asset_ID=\"WARN4200619015447001\" Asset_Class=\"poster\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"poster\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_FileSize\" Value=\"56725\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_CheckSum\" Value=\"EDCB1162F24A29692343BBC415ECB528\"/>\n" +
		"\t\t\t</Metadata>\n" +
		"\t\t\t<Content Value=\"friends_s01ep01_hd_da_20_dvb.jpg\"/>\n" +
		"\t\t</Asset>\n" +
		"\t</Asset>\n</ADI>\n"

	lines := [][]string{
		{"Provider", "Provider id", "Título Original", "Título em Português", "Título em Português do Episódio",
			"Temporada", "Número do Episódio", "Categoria", "Ano",
			"Bilheteria", "Ranking", "Língua Original", "Estúdio",
			"Classificação Etária", "Genero 1", "Genero 2", "Elenco",
			"Diretor", "País de Origem",
			"Sinopse EPG",
			"Sinopse Resumo",
			"Duração", "Data início no NOW", "Data Fim no NOW", "Formato", "Audio", "Legendado", "Dublado",
			"Billing ID", "Extradata 1", "Extradata 2", "Produto", "Janela Repasse",
			"Canal", "Box Office", "Versao",
			"Cobrança", "ID",
			"Movie Size", "Movie MD5",
			"Poster Size", "Poster MD5",
			"DOWNLOAD TO GO", "DIAS DE DOWNLOAD",
			"PASTA FTP", "Movie Audio Type",
			"Trailer ID", "Trailer Size", "Trailer MD5", "Duração Trailer", "Trailer Audio Type"},

		{"WARNER", "warner.com", "Friends", "Friends", "Aquele onde tudo começou",
			"1", "1", "Série", "1994",
			"1000000", "9", "en", "Warner Home Video",
			"12", "Comédia", "Programa",
			"Jennifer Aniston, Courteney Cox, Lisa Kudrow, Matt LeBlanc, Matthew Perry, David Schwimmer",
			"James Burrows", "USA",
			"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai",
			"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai",
			"0.01584490740740740741", "06-10-20", "12-31-49", "HD", "en,pt", "pt", "não",
			"WBH2S", "", "", "TVOD - Catálogo", "S",
			"Nirvana - Catálogo", "200000", "Multi-language",
			"1.49", "friends_s01ep01_hd_da_20_dvb.ts",
			"1814458124", "609A5FBB1D0301719462BB798886D43F",
			"56725", "EDCB1162F24A29692343BBC415ECB528",
			"não", "0",
			"\\rhome\\nirvana\\warner_series_20200608\\dvb\\friends_s01ep01_hd_da_20_dvb", "Stereo",
			"", "", "", "", ""},
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
	options["creationDate"] = "2020-06-19"
	//fmt.Printf("%#v\n", maplines)
	xmlWr, errW := newXMLWriter("unit_tests.json", "ADI.DTD")
	if errW != nil {
		t.Error(errW)
	}
	xmlWr.testing = true
	if err := processLines(json, []lineT{maplines}, xmlWr); err != nil {
		t.Error(err)
	}
	result := xmlWr.getBuffer()
	res2 := decodeISO88599ToUTF8(result) // converting from windows encoding to UTF-8
	assert.Equal(t, expected, res2)
}

func TestXmlOiOtt(t *testing.T) {
	json, errCf := readConfig("config_oi_ott.json")
	if errCf != nil {
		t.Error(errCf)
	}
	initVars(json)
	expected := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n" +
		"<assetPackages xmlns:date=\"http://exslt.org/dates-and-times\" " +
		"xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xsi:noNamespaceSchemaLocation=\"VODmetadata.xsd\" " +
		"xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" formatVersion=\"1.0\">\n" +
		"\t<assetPackage name=\"Nirvana\" verb=\"\" product=\"SVOD\" providerName=\"OnDemand\" providerId=\"ODG\" type=\"SVOD\" " +
		"asset=\"A Música da Minha Vida\">\n" +
		"\t\t<metadata>\n" +
		"\t\t\t<assetID>200702102255001</assetID>\n" +
		"\t\t\t<providerID>warner.com</providerID>\n" +
		"\t\t\t<showType>movie</showType>\n" +
		"\t\t\t<title language=\"pt\">A Música da Minha Vida</title>\n" +
		"\t\t\t<title language=\"en\">Blinded by the Light</title>\n" +
		"\t\t\t<shortTitle language=\"pt\">A Música da Minha Vida</shortTitle>\n" +
		"\t\t\t<reducedTitle language=\"pt\">A Música da Minha Vida</reducedTitle>\n" +
		"\t\t\t<summary language=\"pt\">Do escritor, diretor e produtor Gurinder Chadha, Driblando o Destino, temos o drama A " +
		"Música da Minha Vida, baseado nas músicas e letras inspiradoras das canções atemporais de Bruce Springsteen.</summary>\n" +
		"\t\t\t<shortSummary language=\"pt\">Do escritor, diretor e produtor Gurinder Chadha, Driblando o Destino, temos o drama A " +
		"Música da Minha Vida, baseado nas músicas e letras inspiradoras das canções atemporais de Bruce Springsteen.</shortSummary>\n" +
		"\t\t\t<episodeNumber/>\n" +
		"\t\t\t<cgmsaLevel>copynever</cgmsaLevel>\n" +
		"\t\t\t<rating>12</rating>\n" +
		"\t\t\t<runTimeMinutes>23</runTimeMinutes>\n" +
		"\t\t\t<release_year>2019</release_year>\n" +
		"\t\t\t<countryRegionCode>USA</countryRegionCode>\n" +
		"\t\t\t<person mname=\"\" fname=\"Viveik\" lname=\"Kalra\" role=\"actor\"/>\n" +
		"\t\t\t<person mname=\"\" fname=\"Hayley\" lname=\"Atwell\" role=\"actor\"/>\n" +
		"\t\t\t<person mname=\"\" fname=\"Rob\" lname=\"Brydon\" role=\"actor\"/>\n" +
		"\t\t\t<person mname=\"\" fname=\"Kulvinder\" lname=\"Ghir\" role=\"actor\"/>\n" +
		"\t\t\t<person mname=\"\" fname=\"Nell\" lname=\"Williams\" role=\"actor\"/>\n" +
		"\t\t\t<person mname=\"\" fname=\"Aaron\" lname=\"Phagura\" role=\"actor\"/>\n" +
		"\t\t\t<person mname=\"\" fname=\"Gurinder\" lname=\"Chadha\" role=\"director\"/>\n" +
		"\t\t\t<studio>Warner Bros</studio>\n" +
		"\t\t\t<studioDisplayName>Warner Bros</studioDisplayName>\n\t\t\t<category>MSEPGC_MV_Musical</category>\n" +
		"\t\t\t<autoDeploy>false</autoDeploy>\n" +
		"\t\t\t<autoImport>false</autoImport>\n" +
		"\t\t\t<categorization>\n" +
		"\t\t\t\t<category1 name=\"MSEPGC_MV_Musical\"/>\n" +
		"\t\t\t\t<category2 name=\"MSEPGC_MV_Musical\"/>\n" +
		"\t\t\t</categorization>\n" +
		"\t\t\t<genre>MSEPGC_MV_Musical</genre>\n" +
		"\t\t\t<additionalInfo name=\"A Música da Minha Vida\">\n" +
		"\t\t\t\t<value param=\"RentableOn\">OI_IPTV,OI_PC,OI_Mobile,OI_DTH</value>\n" +
		"\t\t\t\t<value param=\"DownloadableOn\">OI_IPTV,OI_PC,OI_Mobile,OI_DTH</value>\n" +
		"\t\t\t\t<value param=\"CountryGrantRestriction\">BR</value>\n" +
		"\t\t\t\t<value param=\"ISPGrantRestriction\"/>\n" +
		"\t\t\t\t<value param=\"ReasonCode\"/>\n" +
		"\t\t\t</additionalInfo>\n" +
		"\t\t</metadata>\n" +
		"\t\t<businessMetadata>\n" +
		"\t\t\t<suggestedPrice>0.0</suggestedPrice>\n" +
		"\t\t\t<currency_iso3166-2>BR</currency_iso3166-2>\n" +
		"\t\t\t<billingID>movie</billingID>\n" +
		"\t\t</businessMetadata>\n" +
		"\t\t<rightsMetadata>\n" +
		"\t\t\t<licensingWindowStart>2021-10-16T00:00:00Z</licensingWindowStart>\n" +
		"\t\t\t<licensingWindowEnd-2>2044-12-31T00:00:00Z</licensingWindowEnd-2>\n\t\t</rightsMetadata>\n" +
		"\t\t<asset type=\"feature\" asset_name=\"A Música da Minha Vida\">\n" +
		"\t\t\t<metadata>\n" +
		"\t\t\t\t<assetID>200702102255001</assetID>\n" +
		"\t\t\t\t<providerID>warner.com</providerID>\n" +
		"\t\t\t\t<audio>Dolby 5.1</audio>\n" +
		"\t\t\t\t<HD>true</HD>\n" +
		"\t\t\t\t<language_iso639>en</language_iso639>\n" +
		"\t\t\t\t<language_iso639>pt</language_iso639>\n" +
		"\t\t\t\t<rating value=\"12\" rating_system=\"DJCTQ\"/>\n" +
		"\t\t\t</metadata>\n" +
		"\t\t\t<content>blindedbythelight_dub_ptbr.ts</content>\n" +
		"\t\t</asset>\n" +
		"\t\t<asset type=\"trailer\" asset_name=\"A Música da Minha Vida\">\n" +
		"\t\t\t<metadata>\n" +
		"\t\t\t\t<assetID>200702102255001</assetID>\n" +
		"\t\t\t\t<providerID>warner.com</providerID>\n" +
		"\t\t\t\t<audio>Dolby 5.1</audio>\n" +
		"\t\t\t\t<rating value=\"12\" rating_system=\"DJCTQ\"/>\n" +
		"\t\t\t</metadata>\n" +
		"\t\t\t<content>blindedbythelight_dub_ptbr_trailer.ts</content>\n" +
		"\t\t</asset>\n<!-- DTH\n<businessRule>0</businessRule>\n" +
		"\t\t<UserNibble2>2</UserNibble2>\n" +
		"\t\t<audioLanguage>en</audioLanguage>\n" +
		"\t\t<soundType>Surround Sound</soundType> -->\n</assetPackage>\n</assetPackages>\n"

	lines := [][]string{
		{"Show Type", "Provider", "Codigo Categoria 1", "Codigo Categoria 2", "Business rule id", "UserNibble2",
			"Provider id", "Título Original", "Título em Português", "Título em Português do Episódio", "Temporada",
			"Número do Episódio", "Categoria", "Ano", "Bilheteria", "Ranking", "Língua Original", "Estúdio",
			"Classificação Etária", "Genero 1", "Genero 2",
			"Elenco",
			"Diretor", "País de Origem",
			"Sinopse EPG",
			"Sinopse Resumo",
			"Duração", "Data Início", "Data Fim", "Formato", "Audio", "Legendado", "Dublado",
			"Billing ID", "Extradata 1", "Extradata 2", "Produto", "Janela Repasse", "Canal", "Box Office", "Versao",
			"Cobrança", "ID", "Movie Size", "Movie MD5",
			"Poster Size", "Poster MD5", "DOWNLOAD TO GO",
			"DIAS DE DOWNLOAD", "PASTA FTP", "Movie Audio Type",
			"Trailer ID", "Trailer Size", "Trailer MD5",
			"Duração Trailer", "Trailer Audio Type"},

		{"Movie", "WARNER", "MSEPGC_MV_Musical", "MSEPGC_MV_Musical", "0", "2",
			"warner.com", "Blinded by the Light", "A Música da Minha Vida", "", "",
			"", "Filme", "2019", "5000000", "9", "en", "Warner Bros",
			"12", "Drama", "Comédia",
			"Viveik Kalra, Hayley Atwell, Rob Brydon, Kulvinder Ghir, Nell Williams, Aaron Phagura",
			"Gurinder Chadha", "USA",
			"Do escritor, diretor e produtor Gurinder Chadha, Driblando o Destino, temos o drama A Música da Minha Vida, " +
				"baseado nas músicas e letras inspiradoras das canções atemporais de Bruce Springsteen.",
			"Do escritor, diretor e produtor Gurinder Chadha, Driblando o Destino, temos o drama A Música da Minha Vida, " +
				"baseado nas músicas e letras das canções atemporais de Bruce Springsteen.",
			"0.01584490740740740741", "10-16-21", "12-31-44", "HD", "en", "não", "pt",
			"WBH2L", "", "", "TVOD - Catálogo", "L", "Nirvana - Catálogo", "500000", "Dublado",
			"6.9", "blindedbythelight_dub_ptbr.ts", "7063535812", "6C985805922D3E8A1994A5ED674B641B",
			"80586", "D8D855B9FEE9D206223856E64C8DED01", "não",
			"0", "\\rhome\\nirvana\\warner_20200114", "Dolby 5.1",
			"blindedbythelight_dub_ptbr_trailer.ts", "144207844", "4DEC3B668C6DD1A5994E053587E4658B",
			"12:02:24", "Dolby 5.1"},
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
	options["timestamp"] = "200702102255"
	options["creationDate"] = "2020-06-19"
	//fmt.Printf("%#v\n", maplines)
	xmlWr, errW := newXMLWriter("unit_tests.json", "")
	if errW != nil {
		t.Error(errW)
	}
	xmlWr.testing = true
	if err := processLines(json, []lineT{maplines}, xmlWr); err != nil {
		t.Error(err)
		return
	}
	result := xmlWr.getBuffer()
	res2 := decodeISO88599ToUTF8(result) // converting from windows encoding to UTF-8
	assert.Equal(t, expected, res2)
}

func TestXmlVivo(t *testing.T) {
	json, errC := readConfig("config_vivo.json")
	if errC != nil {
		t.Error(errC)
	}
	initVars(json)
	expected := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<!DOCTYPE ADI SYSTEM \"ADI.DTD\">\n" +
		"<ADI xmlns=\"http://www.eventis.nl/PRODIS/ADI\">\n" +
		"\t<Metadata>\n" +
		"\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Package\" " +
		"Version_Major=\"1\" Version_Minor=\"0\" Description=\"Friends\" Creation_Date=\"2020-06-19\" Provider_ID=\"warner.com\" " +
		"Asset_ID=\"WARN1200619015447001\" Asset_Class=\"package\"/>\n" +
		"\t\t<App_Data App=\"MOD\" Name=\"Metadata_Spec_Version\" Value=\"CableLabsVOD1.1\"/>" +
		"\n" +
		"\t</Metadata>\n" +
		"\t<Asset>\n" +
		"\t\t<Metadata>\n" +
		"\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Title\" Version_Major=\"1\" " +
		"Version_Minor=\"0\" Description=\"Friends\" Creation_Date=\"2020-06-19\" Provider_ID=\"warner.com\" Asset_ID=\"WARN2200619015447001\" " +
		"Asset_Class=\"title\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"title\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Title_Sort_Name\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Title_Brief\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Title\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Episode_Name\" Value=\"Aquele onde tudo começou\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Episode_ID\" Value=\"01001\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Long\" Value=\"Depois que Rachel abandona o noivo no altar, " +
		"ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Summary_Medium\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que" +
		" não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" " +
		"Name=\"Summary_Short\" Value=\"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente," +
		" principalmente quando não pode contar com o cartão de crédito do papai\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Rating\" Value=\"12\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Closed_Captioning\" Value=\"N\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Run_Time\" Value=\"1369\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Display_Run_Time\" Value=\"00:23\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Year\" Value=\"1994\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Country_of_Origin\" Value=\"USA\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Studio\" Value=\"Warner Home Video\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Studio_Name\" Value=\"Warner Home Video\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors_Display\" Value=\"Jennifer Aniston, Courteney Cox, Lisa Kudrow, Matt LeBlanc, Matthew Perry, David Schwimmer\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Aniston, Jennifer\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Cox, Courteney\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Kudrow, Lisa\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"LeBlanc, Matt\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Perry, Matthew\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Actors\" Value=\"Schwimmer, David\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Director\" Value=\"Burrows, James\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Director_Display\" Value=\"James Burrows\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Category\" Value=\"Série\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Genre\" Value=\"Comédia\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Box_Office\" Value=\"\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Billing_ID\" Value=\"WBH2S\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Licensing_Window_Start\" Value=\"2020-06-10\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Licensing_Window_End\" Value=\"2049-12-31\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Preview_Period\" Value=\"0\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Provider_QA_Contact\" Value=\"MediaCenter\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Contract_Name\" Value=\"Friends\"/>\n" +
		"\t\t\t<App_Data App=\"MOD\" Name=\"Suggested_Price\" Value=\"1.49\"/>\n" +
		"\t\t</Metadata>\n" +
		"\t\t<Asset>\n" +
		"\t\t\t<Metadata>\n" +
		"\t\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Movie\" Version_Major=\"1\" Version_Minor=\"0\" Creation_Date=\"2020-06-19\" " +
		"Description=\"Friends\" Provider_ID=\"warner.com\" Asset_ID=\"WARN3200619015447001\" Asset_Class=\"movie\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"movie\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Screen_Format\" Value=\"Widescreen\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"HDContent\" Value=\"Y\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Bit_Rate\" Value=\"8000\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Watermarking\" Value=\"N\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Audio_Type\" Value=\"Stereo\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Viewing_Can_Be_Resumed\" Value=\"Y\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"CGMS_A\" Value=\"3\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Languages\" Value=\"en\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Languages\" Value=\"pt\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Subtitle_Languages\" Value=\"pt\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_FileSize\" Value=\"1814458124\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_CheckSum\" Value=\"609A5FBB1D0301719462BB798886D43F\"/>\n" +
		"\t\t\t</Metadata>\n" +
		"\t\t\t<Content Value=\"friends_s01ep01_hd_da_20_dvb.ts\"/>\n" +
		"\t\t</Asset>\n" +
		"\t\t<Asset>\n" +
		"\t\t\t<Metadata>\n" +
		"\t\t\t\t<AMS Provider=\"WARNER\" Product=\"\" Asset_Name=\"Friends_Poster\" Version_Major=\"1\" Version_Minor=\"0\" Creation_Date=\"2020-06-19\" " +
		"Description=\"Friends\" Provider_ID=\"warner.com\" Asset_ID=\"WARN4200619015447001\" Asset_Class=\"poster\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Type\" Value=\"poster\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_FileSize\" Value=\"56725\"/>\n" +
		"\t\t\t\t<App_Data App=\"MOD\" Name=\"Content_CheckSum\" Value=\"EDCB1162F24A29692343BBC415ECB528\"/>\n" +
		"\t\t\t</Metadata>\n" +
		"\t\t\t<Content Value=\"friends_s01ep01_hd_da_20_dvb.jpg\"/>\n" +
		"\t\t</Asset>\n" +
		"\t</Asset>\n</ADI>\n"

	lines := [][]string{
		{"Provider", "Provider id", "Título Original", "Título em Português", "Título em Português do Episódio",
			"Temporada", "Número do Episódio", "Categoria", "Ano",
			"Bilheteria", "Ranking", "Língua Original", "Estúdio",
			"Classificação Etária", "Genero 1", "Genero 2", "Elenco",
			"Diretor", "País de Origem",
			"Sinopse EPG",
			"Sinopse Resumo",
			"Duração", "Data início no NOW", "Data Fim no NOW", "Formato", "Audio", "Legendado", "Dublado",
			"Billing ID", "Extradata 1", "Extradata 2", "Produto", "Janela Repasse",
			"Canal", "Box Office", "Versao",
			"Cobrança", "ID",
			"Movie Size", "Movie MD5",
			"Poster Size", "Poster MD5",
			"DOWNLOAD TO GO", "DIAS DE DOWNLOAD",
			"PASTA FTP", "Movie Audio Type",
			"Trailer ID", "Trailer Size", "Trailer MD5", "Duração Trailer", "Trailer Audio Type"},

		{"WARNER", "warner.com", "Friends", "Friends", "Aquele onde tudo começou",
			"1", "1", "Série", "1994",
			"1000000", "9", "en", "Warner Home Video",
			"12", "Comédia", "Programa",
			"Jennifer Aniston, Courteney Cox, Lisa Kudrow, Matt LeBlanc, Matthew Perry, David Schwimmer",
			"James Burrows", "USA",
			"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai",
			"Depois que Rachel abandona o noivo no altar, ela vai morar com Monica e descobre que não é fácil ser independente, principalmente quando não pode contar com o cartão de crédito do papai",
			"0.01584490740740740741", "06-10-20", "12-31-49", "HD", "en,pt", "pt", "não",
			"WBH2S", "", "", "TVOD - Catálogo", "S",
			"Nirvana - Catálogo", "200000", "Multi-language",
			"1.49", "friends_s01ep01_hd_da_20_dvb.ts",
			"1814458124", "609A5FBB1D0301719462BB798886D43F",
			"56725", "EDCB1162F24A29692343BBC415ECB528",
			"não", "0",
			"\\rhome\\nirvana\\warner_series_20200608\\dvb\\friends_s01ep01_hd_da_20_dvb", "Stereo",
			"", "", "", "", ""},
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
	options["creationDate"] = "2020-06-19"
	//fmt.Printf("%#v\n", maplines)
	xmlWr, errX := newXMLWriter("unit_tests.json", "ADI.DTD")
	if errX != nil {
		t.Error(errX)
	}
	xmlWr.testing = true
	if err := processLines(json, []lineT{maplines}, xmlWr); err != nil {
		t.Error(err)
	}
	result := xmlWr.getBuffer()
	res2 := decodeISO88599ToUTF8(result) // converting from windows encoding to UTF-8
	assert.Equal(t, expected, res2)
}

func TestXmlBox1(t *testing.T) {
	json, errCf := readConfig("config_box.json")
	if errCf != nil {
		t.Error(errCf)
	}
	initVars(json)
	expectedAssets := "{\n" +
		"  \"assets\": [\n" +
		"    {\n" +
		"      \"adult\": false,\n" +
		"      \"available_from\": 1593043200000,\n" +
		"      \"available_to\": 1798675200000,\n      \"duration\": 1421000,\n" +
		"      \"genres\": [\n" +
		"        \"Show\"\n" +
		"      ],\n" +
		"      \"id\": \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\",\n" +
		"      \"images\": [\n" +
		"        {\n" +
		"          \"id\": \"e2830250-d3bf-451a-ba3b-d4ec3ce1da19\",\n" +
		"          \"location\": \"shows/ariana_grande_poster.jpg\",\n" +
		"          \"type\": \"vod-poster\"\n" +
		"        },\n" +
		"        {\n" +
		"          \"id\": \"8b841c15-a02f-4a23-b4d2-d4eb409becbe\",\n" +
		"          \"location\": \"shows/ariana_grande_landscape.jpg\",\n" +
		"          \"type\": \"vod-background\"\n" +
		"        }\n" +
		"      ],\n" +
		"      \"medias\": [\n" +
		"        {\n" +
		"          \"audio_languages\": [\n" +
		"            \"eng\"\n" +
		"          ],\n" +
		"          \"id\": \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\",\n" +
		"          \"location\": \"shows/ariana_grande.mp4\",\n" +
		"          \"metadata\": {},\n" +
		"          \"subtitles_languages\": [\n" +
		"            \"por\"\n          ],\n" +
		"          \"technology\": \"MP4\",\n" +
		"          \"title\": \"Ariana Grande\",\n" +
		"          \"type\": \"MEDIA\"\n" +
		"        }\n" +
		"      ],\n" +
		"      \"metadata\": {\n" +
		"        \"actors\": [\n" +
		"          \"\"\n" +
		"        ],\n" +
		"        \"country\": \"USA\",\n" +
		"        \"directors\": [\n" +
		"          \"\"\n" +
		"        ],\n" +
		"        \"release_year\": \"2016\",\n" +
		"        \"rights\": [],\n" +
		"        \"summary\": [\n" +
		"          \"Show da cantora Ariana Grande\"\n" +
		"        ]\n" +
		"      },\n" +
		"      \"morality_level\": 0,\n" +
		"      \"synopsis\": {\n" +
		"        \"por\": \"Ariana Grande se apresenta em Las Vegas, a cantora canta todos os seus sucessos." +
		" O show conta com participação especial de Zedd.\"\n" +
		"      },\n" +
		"      \"title\": {\n" +
		"        \"por\": \"Ariana Grande\"\n" +
		"      }\n" +
		"    }\n" +
		"  ]\n" +
		"}"

	expectedCategs := "{\n" +
		"  \"categories\": [\n" +
		"    {\n" +
		"      \"adult\": false,\n" +
		"      \"assets\": [\n" +
		"        \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\"\n" +
		"      ],\n" +
		"      \"downloadable\": false,\n" +
		"      \"hidden\": false,\n" +
		"      \"id\": \"d7d4b94e-6055-4400-8325-c7f754830573\",\n" +
		"      \"images\": [],\n" +
		"      \"metadata\": {},\n" +
		"      \"morality_level\": \"0\",\n" +
		"      \"name\": {\n" +
		"        \"eng\": \"Show\",\n" +
		"        \"por\": \"Show\"\n" +
		"      },\n" +
		"      \"offline\": false,\n" +
		"      \"parent_id\": \"\",\n" +
		"      \"parental_control\": false\n" +
		"    },\n" +
		"    {\n" +
		"      \"adult\": false,\n" +
		"      \"assets\": [\n" +
		"        \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\"\n" +
		"      ],\n" +
		"      \"downloadable\": false,\n" +
		"      \"hidden\": false,\n" +
		"      \"id\": \"2f7c576a-7212-4af7-ac90-cbd6df1e5f94\",\n" +
		"      \"images\": [],\n" +
		"      \"metadata\": {},\n" +
		"      \"morality_level\": \"0\",\n" +
		"      \"name\": {\n" +
		"        \"eng\": \"Music\",\n" +
		"        \"por\": \"Música\"\n" +
		"      },\n" +
		"      \"offline\": false,\n" +
		"      \"parent_id\": \"\",\n" +
		"      \"parental_control\": false\n" +
		"    }\n" +
		"  ]\n" +
		"}"

	lines := [][]string{
		{"uuid_box", "uuid_trailer",
			"uuid_poster",
			"uuid_landscape", "uuid_thumb",
			"Título Original",
			"Título em Português",
			"Numero Temporada", "Número do Episódio", "Título em Português do Episódio", "Temporada",
			"ID", "Movie Size", "Movie MD5",
			"Poster Size", "Poster MD5",
			"Versao", "Língua Original ", "Linguagem Áudio", "Linguagem Legenda",
			"Categoria", "Ano", "Bilheteria", "Ranking",
			"Estúdio", "Classificação Etária", "Genero 1", "Genero 2",
			"Elenco", "Diretor",
			"País de Origem",
			"Sinopse EPG",
			"Sinopse Resumo",
			"Duração",
			"Data Início", "Data Fim",
			"Formato",
			"Provider ID", "Billing ID", "Cobrança",
			"Movie Audio Type",
			"Trailer ID", "Trailer Size", "Trailer MD5", "Duração Trailer", "Trailer Audio Type",
			"subpasta",
		},

		{"198413c7-3d35-4c6d-9714-f80e92e9b7d0", "5c8d732a-d702-4000-9fce-7bd882fcaaaf",
			"e2830250-d3bf-451a-ba3b-d4ec3ce1da19",
			"8b841c15-a02f-4a23-b4d2-d4eb409becbe", "cfea92ec-3ce3-463c-b8e7-1cdbee532964",
			"Ariana Grande - Live in New York",
			"Ariana Grande",
			"", "", "", "",
			"ariana_grande.mp4", "", "",
			"", "",
			"Legendado", "eng", "eng", "por",
			"Música", "2016", "1000000", "9",
			"Media Solutions", "0", "Show", "Música",
			"", "",
			"USA",
			"Ariana Grande se apresenta em Las Vegas, a cantora canta todos os seus sucessos. " +
				"O show conta com participação especial de Zedd.",
			"Show da cantora Ariana Grande",
			"0:23:41",
			"06-25-20", "12-31-26",
			"HD",
			"", "", "",
			"stereo",
			"", "", "", "", "",
			"shows"},
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

	categs := [][]string{
		{"id", "name", "hidden", "morality_level",
			"parental_control", "adult", "downloadable", "offline"},
		{"d7d4b94e-6055-4400-8325-c7f754830573", "por:Show|eng:Show",
			"false", "0", "false", "false", "false", "false"},
		{"2f7c576a-7212-4af7-ac90-cbd6df1e5f94", "por:Música|eng:Music",
			"false", "0", "false", "false", "false", "false"},
	}

	lenLineCat := len(categs[1])
	categLine := make([]lineT, 3, 3)
	for iLin := 1; iLin < 3; iLin++ {
		categLine[iLin-1] = make(map[string]string)
		for i := range categs[0] {
			val := ""
			if i < lenLineCat {
				val = categs[iLin][i]
			}
			categLine[iLin-1][categs[0][i]] = val
		}
	}
	maplines["file_number"] = "1"
	options["timestamp"] = "200702102255"
	options["creationDate"] = "2020-06-19"
	//fmt.Printf("%#v\n", maplines)
	jsonWr, errA := newJSONWriter("unit_tests_assets.json", categLine, nil, assetsT)
	if errA != nil {
		t.Error(errA)
	}
	jsonWr.testing = true
	if err := processLines(json, []lineT{maplines}, jsonWr); err != nil {
		t.Error(err)
	}
	categWr, errC := newJSONWriter("unit_tests_categs.json", categLine, nil, categsT)
	if errC != nil {
		t.Error(errC)
	}
	categWr.testing = true
	categWr.processCategPack(maplines, "uuid_box", "Genero 1", "Genero 2")
	bufAssets, bufCategs, errE := categWr.WriteExtras()
	if errE != nil {
		t.Error(errE)
	}
	assetRes := string(bufAssets) // converting from windows encoding to UTF-8
	categRes := string(bufCategs) // converting from windows encoding to UTF-8
	assert.JSONEq(t, expectedAssets, assetRes)
	assert.JSONEq(t, expectedCategs, categRes)
}

func TestXmlBoxSeries(t *testing.T) {
	json, errConf := readConfig("config_box.json")
	if errConf != nil {
		t.Error(errConf)
	}
	initVars(json)
	expectedAssets := "{\n" +
		"  \"assets\": [\n" +
		"    {\n" +
		"      \"adult\": false,\n" +
		"      \"available_from\": 1593043200000,\n" +
		"      \"available_to\": 1798675200000,\n      \"duration\": 1421000,\n" +
		"      \"genres\": [\n" +
		"        \"Show\"\n" +
		"      ],\n" +
		"      \"id\": \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\",\n" +
		"      \"images\": [\n" +
		"        {\n" +
		"          \"id\": \"e2830250-d3bf-451a-ba3b-d4ec3ce1da19\",\n" +
		"          \"location\": \"shows/ariana_grande_poster.jpg\",\n" +
		"          \"type\": \"vod-poster\"\n" +
		"        },\n" +
		"        {\n" +
		"          \"id\": \"8b841c15-a02f-4a23-b4d2-d4eb409becbe\",\n" +
		"          \"location\": \"shows/ariana_grande_landscape.jpg\",\n" +
		"          \"type\": \"vod-background\"\n" +
		"        }\n" +
		"      ],\n" +
		"      \"medias\": [\n" +
		"        {\n" +
		"          \"audio_languages\": [\n" +
		"            \"eng\"\n" +
		"          ],\n" +
		"          \"id\": \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\",\n" +
		"          \"location\": \"shows/ariana_grande.mp4\",\n" +
		"          \"metadata\": {},\n" +
		"          \"subtitles_languages\": [\n" +
		"            \"por\"\n          ],\n" +
		"          \"technology\": \"MP4\",\n" +
		"          \"title\": \"Ariana Grande\",\n" +
		"          \"type\": \"MEDIA\"\n" +
		"        }\n" +
		"      ],\n" +
		"      \"metadata\": {\n" +
		"        \"actors\": [\n" +
		"          \"\"\n" +
		"        ],\n" +
		"        \"country\": \"USA\",\n" +
		"        \"directors\": [\n" +
		"          \"\"\n" +
		"        ],\n" +
		"        \"release_year\": \"2016\",\n" +
		"        \"rights\": [],\n" +
		"        \"summary\": [\n" +
		"          \"Show da cantora Ariana Grande\"\n" +
		"        ]\n" +
		"      },\n" +
		"      \"morality_level\": 0,\n" +
		"      \"synopsis\": {\n" +
		"        \"por\": \"Ariana Grande se apresenta em Las Vegas, a cantora canta todos os seus sucessos." +
		" O show conta com participação especial de Zedd.\"\n" +
		"      },\n" +
		"      \"title\": {\n" +
		"        \"por\": \"Ariana Grande\"\n" +
		"      }\n" +
		"    }\n" +
		"  ]\n" +
		"}"

	expectedCategs := "{\n" +
		"  \"categories\": [\n" +
		"    {\n" +
		"      \"adult\": false,\n" +
		"      \"assets\": [\n" +
		"        \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\"\n" +
		"      ],\n" +
		"      \"downloadable\": false,\n" +
		"      \"hidden\": false,\n" +
		"      \"id\": \"d7d4b94e-6055-4400-8325-c7f754830573\",\n" +
		"      \"images\": [],\n" +
		"      \"metadata\": {},\n" +
		"      \"morality_level\": \"0\",\n" +
		"      \"name\": {\n" +
		"        \"eng\": \"Show\",\n" +
		"        \"por\": \"Show\"\n" +
		"      },\n" +
		"      \"offline\": false,\n" +
		"      \"parent_id\": \"\",\n" +
		"      \"parental_control\": false\n" +
		"    },\n" +
		"    {\n" +
		"      \"adult\": false,\n" +
		"      \"assets\": [\n" +
		"        \"198413c7-3d35-4c6d-9714-f80e92e9b7d0\"\n" +
		"      ],\n" +
		"      \"downloadable\": false,\n" +
		"      \"hidden\": false,\n" +
		"      \"id\": \"2f7c576a-7212-4af7-ac90-cbd6df1e5f94\",\n" +
		"      \"images\": [],\n" +
		"      \"metadata\": {},\n" +
		"      \"morality_level\": \"0\",\n" +
		"      \"name\": {\n" +
		"        \"eng\": \"Music\",\n" +
		"        \"por\": \"Música\"\n" +
		"      },\n" +
		"      \"offline\": false,\n" +
		"      \"parent_id\": \"\",\n" +
		"      \"parental_control\": false\n" +
		"    }\n" +
		"  ]\n" +
		"}"

	lines := [][]string{
		{"uuid_box", "uuid_trailer",
			"uuid_poster",
			"uuid_landscape", "uuid_thumb",
			"Título Original",
			"Título em Português",
			"Numero Temporada", "Número do Episódio", "Título em Português do Episódio", "Temporada",
			"ID", "Movie Size", "Movie MD5",
			"Poster Size", "Poster MD5",
			"Versao", "Língua Original ", "Linguagem Áudio", "Linguagem Legenda",
			"Categoria", "Ano", "Bilheteria", "Ranking",
			"Estúdio", "Classificação Etária", "Genero 1", "Genero 2",
			"Elenco", "Diretor",
			"País de Origem",
			"Sinopse EPG",
			"Sinopse Resumo",
			"Duração",
			"Data Início", "Data Fim",
			"Formato",
			"Provider ID", "Billing ID", "Cobrança",
			"Movie Audio Type",
			"Trailer ID", "Trailer Size", "Trailer MD5", "Duração Trailer", "Trailer Audio Type",
			"subpasta",
		},
		{
			"198413c7-3d35-4c6d-9714-f80e92e9b7d0", "5c8d732a-d702-4000-9fce-7bd882fcaaaf",
			"e2830250-d3bf-451a-ba3b-d4ec3ce1da19",
			"8b841c15-a02f-4a23-b4d2-d4eb409becbe", "cfea92ec-3ce3-463c-b8e7-1cdbee532964",
			"Ariana Grande - Live in New York",
			"Ariana Grande",
			"1", "1", "Ep 1", "Estreia",
			"ariana_grande_s1e1.mp4", "", "",
			"", "",
			"Legendado", "eng", "eng", "por",
			"Música", "2016", "1000000", "9",
			"Media Solutions", "0", "Show", "Música",
			"", "",
			"USA",
			"Ariana Grande se apresenta em Las Vegas, a cantora canta todos os seus sucessos. " +
				"O show conta com participação especial de Zedd.",
			"Show da cantora Ariana Grande",
			"0:23:41",
			"06-25-20", "12-31-26",
			"HD",
			"", "", "",
			"stereo",
			"", "", "", "", "",
			"shows",
		},
		{
			"198413c7-3d35-4c6d-9714-f80e92e9b7d1", "5c8d732a-d702-4000-9fce-7bd882fcaaa0",
			"e2830250-d3bf-451a-ba3b-d4ec3ce1da1a",
			"8b841c15-a02f-4a23-b4d2-d4eb409becbe", "cfea92ec-3ce3-463c-b8e7-1cdbee532965",
			"Ariana Grande - Live in New York",
			"Ariana Grande",
			"1", "2", "Ep 1", "Estreia",
			"ariana_grande_s1e2.mp4", "", "",
			"", "",
			"Legendado", "eng", "eng", "por",
			"Música", "2016", "1000000", "9",
			"Media Solutions", "0", "Show", "Música",
			"", "",
			"USA",
			"Ariana Grande se apresenta em Las Vegas, a cantora canta todos os seus sucessos. " +
				"O show conta com participação especial de Zedd.",
			"Show da cantora Ariana Grande",
			"0:23:41",
			"06-25-20", "12-31-26",
			"HD",
			"", "", "",
			"stereo",
			"", "", "", "", "",
			"shows",
		},
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

	categs := [][]string{
		{"id", "name", "hidden", "morality_level",
			"parental_control", "adult", "downloadable", "offline"},
		{"d7d4b94e-6055-4400-8325-c7f754830573", "por:Show|eng:Show",
			"false", "0", "false", "false", "false", "false"},
		{"2f7c576a-7212-4af7-ac90-cbd6df1e5f94", "por:Música|eng:Music",
			"false", "0", "false", "false", "false", "false"},
	}

	lenLineCat := len(categs[1])
	categLine := make([]lineT, 3, 3)
	for iLin := 1; iLin < 3; iLin++ {
		categLine[iLin-1] = make(map[string]string)
		for i := range categs[0] {
			val := ""
			if i < lenLineCat {
				val = categs[iLin][i]
			}
			categLine[iLin-1][categs[0][i]] = val
		}
	}
	maplines["file_number"] = "1"
	options["timestamp"] = "200702102255"
	options["creationDate"] = "2020-06-19"
	//fmt.Printf("%#v\n", maplines)
	jsonWr, errA := newJSONWriter("unit_tests_assets.json", categLine, nil, assetsT)
	if errA != nil {
		t.Error(errA)
	}
	jsonWr.testing = true
	if err := processLines(json, []lineT{maplines}, jsonWr); err != nil {
		t.Error(err)
	}
	categWr, errC := newJSONWriter("unit_tests_categs.json", categLine, nil, categsT)
	if errC != nil {
		t.Error(errC)
	}
	categWr.testing = true
	categWr.processCategPack(maplines, "uuid_box", "Genero 1", "Genero 2")
	bufAssets, bufCategs, errE := categWr.WriteExtras()
	if errE != nil {
		t.Error(errE)
	}
	assetRes := string(bufAssets) // converting from windows encoding to UTF-8
	categRes := string(bufCategs) // converting from windows encoding to UTF-8
	fmt.Printf("%s\n", assetRes)
	assert.JSONEq(t, expectedAssets, assetRes)
	assert.JSONEq(t, expectedCategs, categRes)
}

func decodeISO88599ToUTF8(bytes []byte) string {
	encoded, _ := charmap.ISO8859_9.NewDecoder().Bytes(bytes)
	return string(encoded[:])
}

//func encodeUTF8ToISO88599(bytes []byte) string {
//	encoded, _ := charmap.ISO8859_9.NewEncoder().Bytes(bytes)
//	return string(encoded[:])
//}
