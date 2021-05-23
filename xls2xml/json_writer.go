package main

import (
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"github.com/golang-collections/collections/stack"
)

var consolidated interface{} // TODO refactor global var

// jsonWriter represents a writer to a JSON file
type jsonWriter struct {
	fileName   string
	root       interface{}
	st         stack.Stack
	categLines []lineT
	serieLines []lineT
	testing    bool
}

// newJSONWriter creates a new struct
func newJSONWriter(filename string, categLines []lineT, serieLines []lineT, jType int) (*jsonWriter, error) {
	var err error
	w := jsonWriter{fileName: filename, categLines: categLines, serieLines: serieLines}
	switch jType {
	case assetsT:
		return &w, nil
	case categsT:
		_, err = w.initCateg()
	case seriesT:
		_, err = w.initSeries()
	}
	return &w, err
}

// Suffix returns output file suffix
func (wr *jsonWriter) Suffix() string {
	return ".json"
}

// Filename returns output filename
func (wr *jsonWriter) Filename() string {
	return wr.fileName
}

// StartMap starts a map element
func (wr *jsonWriter) StartMap() error {
	wr.getElement("", mapT)
	return nil
}

// EndMap closes a map element
func (wr *jsonWriter) EndMap() error {
	err := wr.EndElem("", mapT)
	return err
}

// StartElem starts a JSON element
func (wr *jsonWriter) StartElem(name string, elType elemType) error {
	//fmt.Printf("%s -> %v\n", name, elType)
	el := wr.getElement(name, elType)
	wr.st.Push(el)
	// fmt.Printf("--> %#v - %T\n", wr.st, current)
	return nil
}

// getElement returns a new variable for a given element
func (wr *jsonWriter) getElement(name string, elType elemType) interface{} {
	var el interface{}
	switch elType {
	case mapT, mapNoArrT, mapArrayT:
		el = make(map[string]interface{})
	case arrayT:
		el = make([]interface{}, 0)
	case singleT:
		el = name
	case emptyT:
		el = nil

	default:
	}
	return el
}

// insertElement inserts a new element into the structure
func (wr *jsonWriter) insertElement(name string, elem interface{}, elType elemType) error {
	var el interface{}
	if elType == mapT || elem == nil {
		arr := make([]interface{}, 0)
		if elem != nil {
			arr = append(arr, elem)
		}
		el = arr
	} else {
		el = elem
	}
	current := wr.st.Peek()
	if current == nil {
		m := make(map[string]interface{})
		wr.root = m
		m[name] = el
	} else {
		switch c := current.(type) {
		case map[string]interface{}:
			c[name] = el
			// fmt.Printf("** %#v\n", c)
		case []interface{}:
			wr.st.Pop()
			c = append(c, el)
			wr.st.Push(c)
			//fmt.Printf("** %#v\n", c)
		case interface{}:
			return fmt.Errorf("ERRO: InsertElement(interface{}): %#v\n", c)
		default:
			//fmt.Printf("**** %#v\n", c)
		}
	}
	return nil
}

func (wr *jsonWriter) Write(_ string) error {
	return nil
}

// WriteAttr writes a subelement
func (wr *jsonWriter) WriteAttr(name string, value string, vtype string, _ string) error {
	current := wr.st.Peek()
	if current == nil {
		wr.root = value
	} else {
		switch c := current.(type) {
		case map[string]interface{}:
			switch vtype {
			case "", "string":
				c[name] = value
			case "int":
				val, err := strconv.Atoi(value)
				if err != nil {
					// fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = errorMessage[0].val
					break
				}
				c[name] = val
			case "float":
				if value == "" {
					log(fmt.Sprintf("ERRO field [%s]: [empty]", name))
					c[name] = errorMessage[0].val
					break
				}
				fl, err := strconv.ParseFloat(value, 64)
				if err != nil {
					c[name] = errorMessage[0].val
					log(fmt.Sprintf("ERRO field [%s]: [%v]", name, err))
					break
				}
				//val := strconv.FormatFloat(fl, 'f', 2, 32)
				c[name] = fl
			case "timestamp":
				var val, err = toTimestamp(value)
				if err != nil {
					// fmt.Printf("%s *--------> %#v\n", name, val)
					c[name] = errorMessage[0].val
					break
				}
				c[name] = val
			case "boolean":
				if value != "true" && value != "false" {
					c[name] = errorMessage[0].val
					break
				}
				c[name] = value == "true"
			}
			// fmt.Printf("** %#v\n", c)
		case []interface{}:
			c = append(c, value)
			wr.st.Pop()
			wr.st.Push(c)
			//fmt.Printf("** %#v\n", c)
		case interface{}:
			//fmt.Printf("**== %#v\n", c)
		default:
			//fmt.Printf("*********** %#v\n", c)
		}
	}
	//fmt.Printf("\"%s\": %s\n", name, value)
	return nil
}

// EndElem closes a JSON element
func (wr *jsonWriter) EndElem(name string, elType elemType) error {
	//fmt.Printf("End: %s\n", name)
	el := wr.st.Pop()
	return wr.insertElement(name, el, elType)
}

// StartComment marks the start of a comment section
func (wr *jsonWriter) StartComment(_ string) error {
	return nil
}

// EndComment closes a comment section
func (wr *jsonWriter) EndComment(_ string) error {
	return nil
}

// OpenOutput opens a new output file
func (wr *jsonWriter) OpenOutput() error {
	return nil
}

// WriteAndClose writes the structure to an external file
func (wr *jsonWriter) WriteAndClose(_ string) error {
	if consolidated == nil {
		consolidated = wr.root
	} else {
		arrCons := consolidated.(map[string]interface{})["assets"].([]interface{})
		arrNew := wr.root.(map[string]interface{})["assets"].([]interface{})[0]
		consolidated.(map[string]interface{})["assets"] = append(arrCons, arrNew)
		wr.root = consolidated
	}
	//result, err := js.MarshalIndent(wr.root, "", "  ")
	//if err != nil {
	//	logError(err)
	//}
	//fmt.Printf("RESULT %v\n", string(result))
	return nil
}

// WriteConsolidated writes additional files
func (wr *jsonWriter) WriteConsolidated(mode int) (bufAssets []byte, bufCategs []byte, bufSeries []byte, err error) {
	bufAssets, err = js.MarshalIndent(consolidated, "", "  ")
	if err != nil {
		return
	}
	if !wr.testing {
		fileAssets := path.Join(wr.fileName, "assets.json")
		log("Salvando " + fileAssets)
		err = ioutil.WriteFile(fileAssets, bufAssets, 0644)
		if err != nil {
			err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileAssets, err)
			return
		}
	}
	switch mode {
	case categsT:
		if wr.categLines == nil {
			return
		}
		bufCategs, err = js.MarshalIndent(wr.root, "", "  ")
		if err != nil {
			return
		}
		if !wr.testing {
			fileCateg := path.Join(wr.fileName, "categories.json")
			log("Salvando " + fileCateg)
			err = ioutil.WriteFile(fileCateg, bufCategs, 0644)
			if err != nil {
				err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileCateg, err)
				return
			}
		}
	case seriesT:
		bufSeries, err = js.MarshalIndent(wr.root, "", "  ")
		if err != nil {
			return
		}
		if !wr.testing {
			fileSeries := path.Join(wr.fileName, "series.json")
			log("Salvando " + fileSeries)
			err = ioutil.WriteFile(fileSeries, bufSeries, 0644)
			if err != nil {
				err = fmt.Errorf("ERRO ao criar arquivo [%#v]: %v", fileSeries, err)
				return
			}
		}
	}
	return
}

func (wr *jsonWriter) initCateg() (map[string]interface{}, error) {
	root := make(map[string]interface{})
	cat := make([]map[string]interface{}, 0)
	//fmt.Printf("Categories: [%#v]\n", wr.categLines)
	for _, line := range wr.categLines {
		el := make(map[string]interface{})
		id, ok := line.fields["id"]
		if !ok || id == "" {
			name, ok2 := line.fields["name"]
			if ok2 {
				fmt.Printf("WARNING: categoria [%s] nao existente na aba 'categories'", name)
			}
			continue
		}
		m, err, done := newCateg(el, line.fields["name"], line.fields["id"], "", "null", "null")
		el["hidden"] = line.fields["hidden"] == "true"
		el["morality_level"] = line.fields["morality_level"]
		el["parental_control"] = line.fields["parental_control"] == "true"
		el["adult"] = line.fields["adult"] == "true"
		el["downloadable"] = line.fields["downloadable"] == "true"
		el["offline"] = line.fields["offline"] == "true"
		if done {
			return m, err
		}
		cat = append(cat, el)
	}
	root["categories"] = cat
	wr.root = root
	return root, nil
}

func newCateg(el map[string]interface{}, name string, id string, idParent string, seriesId string, seasonId string) (map[string]interface{}, error, bool) {
	elName := make(map[string]interface{})
	strNames := strings.Split(name, "|")
	for _, l := range strNames {
		vals := strings.Split(l, ":")
		if len(vals) < 2 {
			elName["por"] = strings.TrimSpace(l)
		} else {
			elName[strings.TrimSpace(vals[0])] = strings.TrimSpace(vals[1])
		}
	}
	el["id"] = id
	el["name"] = elName
	el["hidden"] = true
	el["morality_level"] = "0"
	el["parental_control"] = true
	el["adult"] = true
	el["downloadable"] = false
	el["offline"] = false
	el["metadata"] = make(map[string]interface{})
	el["images"] = make([]interface{}, 0)
	el["parent_id"] = idParent
	el["assets"] = make([]interface{}, 0)
	el["series_id"] = seriesId
	el["season_id"] = seasonId
	return el, nil, false
}

func (wr *jsonWriter) initSeries() (map[string]interface{}, error) {
	root := make(map[string]interface{})
	cat := make([]map[string]interface{}, 0)
	//fmt.Printf("Series: [%#v]\n", wr.serieLines)
	current := make(map[string]interface{})
	currentId := ""
	for _, line := range wr.serieLines {
		toAppend := true
		name, _ := line.fields["title"]
		//fmt.Printf("serie:[%s]\n", name)
		id, ok := line.fields["id"]
		if !ok || id == "" {
			return nil, fmt.Errorf("serie [%s] nao existente na aba 'series'", name)
		}
		var el map[string]interface{}
		if id != currentId {
			el = make(map[string]interface{})
			current = el
			toAppend = currentId != ""
			currentId = id
		}
		el = current
		el["id"] = id
		elName := make(map[string]interface{})
		strNames := strings.Split(line.fields["title"], "|")
		for _, l := range strNames {
			vals := strings.Split(l, ":")
			if len(vals) < 2 {
				err := fmt.Errorf("erro ao ler serie, valor invalido [%s]", name)
				return nil, err
			}
			elName[strings.TrimSpace(vals[0])] = strings.TrimSpace(vals[1])
		}
		elSeasM := make(map[string]interface{})
		idSeason := elSeasM["id"]
		idSeason = line.fields["id season"]
		elSeasM["external_ids"] = make(map[string]interface{})
		elSeasM["images"] = make([]interface{}, 0)
		numSeason, erri := strconv.Atoi(line.fields["season"])

		el["external_ids"] = make(map[string]interface{})
		el["title"] = elName
		var err error
		el["synopsis"], err = splitLangName(line.fields["synopsis"])
		if err != nil {
			return nil, err
		}
		el["images"] = make([]interface{}, 0)
		elSeas, okS := el["seasons"].([]interface{})
		if !okS {
			elSeas = make([]interface{}, 0)
		}
		if erri != nil {
			return nil, err
		}
		elSeasM["season_number"] = numSeason
		titleEls, errt := splitLangName(line.fields["title"])
		if errt != nil {
			return nil, err
		}
		elSeasM["title"] = titleEls
		elSeasM["synopsis"], err = splitLangName(line.fields["season synopsis"])

		poster, okp := line.fields["capa"]
		arrIm := make([]interface{}, 0)
		if okp && poster != "" {
			imgEl := make(map[string]interface{})
			imgEl["id"] = idSeason.(string) + "_po"
			imgEl["type"] = "poster"
			imgEl["location"] = poster
			arrIm = append(arrIm, imgEl)
		}
		backg, okb := line.fields["landscape"]
		if okb && backg != "" {
			imgEl := make(map[string]interface{})
			imgEl["id"] = idSeason.(string) + "_bk"
			imgEl["type"] = "background"
			imgEl["location"] = backg
			arrIm = append(arrIm, imgEl)
		}
		elSeasM["images"] = arrIm

		if err != nil {
			return nil, err
		}
		elSeas = append(elSeas, elSeasM)
		el["seasons"] = elSeas
		if toAppend {
			cat = append(cat, el)
		} else {
			cat = append(cat, el)
		}
	}
	root["series"] = cat
	wr.root = root
	return root, nil
}

func splitLangName(str string) (map[string]string, error) {
	listLang := strings.Split(str, "|")
	result := make(map[string]string)
	for _, langEl := range listLang {
		langSplit := strings.Split(langEl, ":")
		numEls := len(langSplit)
		if numEls != 2 {
			return nil, fmt.Errorf("campo de linguagem tem %d elementos, deveria ter 2: [%s]", numEls, langEl)
		}
		lang := strings.TrimSpace(langSplit[0])
		text := strings.TrimSpace(langSplit[1])
		result[lang] = text
	}
	return result, nil
}

// addToCategories adds an asset to the categories list
func (wr *jsonWriter) addToCategories(id string, idCateg string, categName string, idParent string, rootEl string, seriesId string, seasonId string) error {
	if categName == "" {
		return nil
	}
	r := wr.root.(map[string]interface{})
	categs := r[rootEl].([]map[string]interface{})
	for _, categ := range categs {
		name := categ["name"].(map[string]interface{})["por"]
		if name == categName {
			for _, asset := range categ["assets"].([]interface{}) {
				if id == asset.(string) {
					return nil
				}
			}
			log(fmt.Sprintf("adicionando: [%s], id: [%s]", categName, id))
			categ["assets"] = append(categ["assets"].([]interface{}), id)
			return nil
		}
	}
	el := make(map[string]interface{})
	log(fmt.Sprintf("criando nova categoria: [%s], id: [%s], parent:[%s]", categName, id, idParent))
	categ, err, _ := newCateg(el, categName, idCateg, idParent, seriesId, seasonId)
	if err != nil {
		return err
	}
	r[rootEl] = append(categs, categ)
	////categs["assets"]= categ["assets"].([]interface{}), id)
	// return fmt.Errorf("Categoria nao existente: inclua na aba 'categories'. Nome: [%s], sugestao de id: [%s] ", categName, uuid[0].val)
	return nil
}

// addToSeries adds an asset to the series list in assets.json
func (wr *jsonWriter) addToSeries(id string, serieName string) error {
	if serieName == "" {
		return nil
	}
	r := wr.root.(map[string]interface{})
	series := r["series"].([]map[string]interface{})
	for _, serie := range series {
		name := serie["title"].(map[string]interface{})["por"]
		if name == serieName {
			serie["assets"] = append(serie["assets"].([]interface{}), id)
			return nil
		}
	}
	return nil
}

// IMPORTANT: JsonWriter must be created separately for the categories file - do not use this method for the assets file
func (wr *jsonWriter) processCategPack(lines []lineT, k int, idField string, categFields []string, categSeason int, series []lineT, categs []lineT, forceGenreCategs bool) (int, error) {
	var err error
	line := lines[k]

	studio := line.fields["estúdio"]
	idStudio, erru := UUIDfromString(studio)
	if erru != nil {
		return -1, err
	}

	// search serie
	studioField := categFields[1]
	seasonField := categFields[2]
	serie, _ := findSerie(series, studio, line.fields[studioField], line.fields[seasonField])
	if serie == nil || forceGenreCategs {
		// add genre 1 to categ
		genre1Field := "genero 1"
		_, errc1 := findCateg(categs, line.fields[genre1Field])
		if errc1 == nil {
			if errc := wr.addToCategories(line.fields[idField], "", line.fields[genre1Field], "", "categories", "null", "null"); errc != nil {
				return -1, errc
			}
		}

		// add genre 2 to categ
		genre2Field := "genero 2"
		_, errc2 := findCateg(categs, line.fields[genre2Field])
		if errc2 == nil {
			if errc := wr.addToCategories(line.fields[idField], "", line.fields[genre2Field], "", "categories", "null", "null"); errc != nil {
				return -1, errc
			}
		}
	}
	var idChild = ""
	if line.fields[categFields[categSeason]] != "" {
		idChild, err = wr.processSerie(&line, idField, categFields, series, idStudio)
	}
	if idChild == "" {
		idChild = idStudio
	}
	if errc := wr.addToCategories(idChild, idStudio, studio, "", "categories", "null", "null"); errc != nil {
		return -1, errc
	}

	return 0, err
}

func (wr *jsonWriter) processSerie(line *lineT, idField string, categFields []string, series []lineT, idStudio string) (string, error) {
	serie, err := findSerie(series, line.fields[categFields[0]], line.fields[categFields[1]], line.fields[categFields[2]])
	if serie == nil {
		return "", err
	}
	name := line.fields[categFields[1]]
	serId := serie.fields["id"]
	serieId1 := fmt.Sprintf("%s_s", serId)
	idSeason, errS := wr.processSeason(line, idField, categFields, series, idStudio)
	if errS != nil {
		return idSeason, errS
	}
	if errc := wr.addToCategories(idSeason, serieId1, name, idStudio, "categories", serId, "null"); errc != nil {
		return serieId1, errc
	}

	return serieId1, nil
}

func (wr *jsonWriter) processSeason(line *lineT, idField string, categFields []string, series []lineT, idStudio string) (string, error) {
	serie, err := findSerie(series, line.fields[categFields[0]], line.fields[categFields[1]], line.fields[categFields[2]])
	if serie == nil {
		return "", err
	}
	name := line.fields[categFields[2]]
	serId := serie.fields["id"]
	seasId := serie.fields["id season"]
	seasonId1 := fmt.Sprintf("%s_t", seasId)
	serieId1 := fmt.Sprintf("%s_s", serId)
	name = fmt.Sprintf("%s T%s", line.fields[categFields[1]], strings.TrimSpace(line.fields[categFields[2]]))
	id := line.fields[idField]
	if errc := wr.addToCategories(id, seasonId1, name, serieId1, "categories", serId, seasId); errc != nil {
		return id, errc
	}
	return seasonId1, nil
}

func findSerie(series []lineT, studio string, name string, season string) (*lineT, error) {
	for _, line := range series {
		title, err := splitLangName(line.fields["title"])
		if err != nil {
			return nil, err
		}
		if title["por"] != name {
			continue
		}
		s := line.fields["season"]
		if s != season {
			continue
		}
		return &line, nil
	}
	return nil, fmt.Errorf("serie com nome [%s] e temporada [%s] nao encontrada. Adicionar na aba 'series'", name, season)
}

func findCateg(categs []lineT, name string) (*lineT, error) {
	for _, line := range categs {
		title, err := splitLangName(line.fields["name"])
		if err != nil {
			return nil, err
		}
		if title["por"] != name {
			continue
		}
		return &line, nil
	}
	return nil, fmt.Errorf("categoria com nome [%s] nao encontrada. Adicionar na aba 'categs'", name)
}

// IMPORTANT: JsonWriter must be created separately for the series file - do not use this method for the assets file
func (wr *jsonWriter) processSeriesPack(line *lineT, series []lineT, idField string, idEpisodeField string) (int, error) {
	nEpis := line.fields[idEpisodeField]
	err := wr.addToSeries(line.fields["uuid_box"], nEpis)
	if err != nil {
		return -1, err
	}
	return 0, err
}

func (wr *jsonWriter) cleanSeries() {
}

// Testing returns true if is running in a testing environment
func (wr *jsonWriter) Testing() bool {
	return wr.testing
}

func populateSerieIds(lines []lineT, options optionsT) error {
	options["series"] = make(map[string]string)
	idF, okO := options["options"]["series_id_field"]
	if !okO {
		return fmt.Errorf("opcao 'season_id_field' nao encontrada")
	}
	titleF, okT := options["options"]["series_title_field"]
	if !okT {
		return fmt.Errorf("opcao 'series_title_field' nao encontrada")
	}
	nSeasonF, okN := options["options"]["season_num_field"]
	if !okN {
		return fmt.Errorf("opcao 'season_num_field' nao encontrada")
	}
	idSeasonF, okI := options["options"]["season_id_field"]
	if !okI {
		return fmt.Errorf("opcao 'season_id_field' nao encontrada")
	}
	options["series"] = make(map[string]string)
	for i, line := range lines {
		id, ok1 := line.fields[idF]
		if !ok1 {
			return fmt.Errorf("campo '%s' nao encontrado na planilha de series", idF)
		}
		title, ok2 := line.fields[titleF]
		if !ok2 {
			return fmt.Errorf("campo '%s' nao encontrado na planilha de series", titleF)
		}
		sNames, err := splitLangName(title)
		if err != nil {
			// Sums 2 to line because array begins with 0 and there is a header column
			return fmt.Errorf("erro na linha %d, coluna [%s]: %s", i+2, titleF, err.Error())
		}
		titlePor, ok := sNames["por"]
		if !ok {
			return fmt.Errorf("serie '%s' nao tem nome em portugues", titleF)
		}
		nSeason, ok3 := line.fields[nSeasonF]
		if !ok3 {
			return fmt.Errorf("campo '%s' nao encontrado na planilha de series", nSeason)
		}
		idSeason, ok4 := line.fields[idSeasonF]
		if !ok4 {
			return fmt.Errorf("campo '%s' nao encontrado na planilha de series", idSeasonF)
		}
		idSerie := fmt.Sprintf("%s|%s", titlePor, nSeason)
		valSerie := fmt.Sprintf("%s|%s", id, idSeason)
		options["series"][idSerie] = valSerie
	}
	return nil
}
