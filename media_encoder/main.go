package main

import (
	"bufio"
	js "encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
)

type audioChanT struct {
	start    int
	lenght   int
	mappping string
	lang     string
}

func main() {

	inFile := ""
	confFile := ""
	profFile := ""
	outDir := ""
	assetID := ""
	strChan := ""
	ignoreAudio := false
	ignoreVideo := false
	time := ""
	flag.StringVar(&inFile, "in", "", "Arquivo de entrada")
	flag.StringVar(&assetID, "assetid", "", "Asset ID")
	flag.StringVar(&confFile, "options", "", "Arquivo JSON de configuracao")
	flag.StringVar(&profFile, "profiles", "", "Arquivo JSON com os profiles")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
	flag.BoolVar(&ignoreAudio, "ignoreaudio", false, "Ignorar canais de audio")
	flag.BoolVar(&ignoreVideo, "ignorevideo", false, "Ignorar canais de audio")
	flag.StringVar(&strChan, "audiochan", "", "Mapeamento dos canais de audio\n"+
		"formato: <mapeamento>|<mapeamento>|...\n"+
		"mapeamento: s<stream inicial>i<entrada>o<saida><linguagem (3 letras)>")
	flag.StringVar(&time, "time", "", "tempo em segundos para restringir o encoding (para testes)")
	flag.Parse()

	if inFile == "" {
		logError(fmt.Errorf("arquivo de entrada deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if assetID == "" {
		logError(fmt.Errorf("asset ID deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if confFile == "" {
		logError(fmt.Errorf("arquivo JSON de configuracao deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if profFile == "" {
		logError(fmt.Errorf("arquivo JSON de profiles deve ser especificado na linha de comando"))
		flag.Usage()
		os.Exit(1)
	}
	if outDir != "" {
		st, err := os.Stat(outDir)
		if err != nil || !st.IsDir() {
			logError(fmt.Errorf("diretorio [%s] nao e' valido", outDir))
			os.Exit(1)
		}
	}
	if outDir == "" {
		outDir = "."
	}
	outDir += string(os.PathSeparator)

	audioChans := make([]audioChanT, 0)
	strChans := strings.Split(strChan, "|")
	count := 1
	for _, ch := range strChans {
		l := len(ch)
		if l == 0 {
			continue
		}
		if l < 4 {
			logError(fmt.Errorf("mapeamento de audio invalido: [%s]\t", strChan))
			os.Exit(1)
		}
		var err error
		st := count
		if strings.HasPrefix(ch, "s") {
			st, err = strconv.Atoi(string(ch[1]))
			if err != nil {
				logError(fmt.Errorf("mapeamento de audio invalido (stream inicial): [%s]\t", strChan))
				os.Exit(1)
			}
			ch = ch[2:]
			l -= 2
		}

		d1, err := strconv.Atoi(string(ch[1]))
		if err != nil {
			logError(fmt.Errorf("mapeamento de audio invalido (numero de canal): [%s]\t", strChan))
			os.Exit(1)
		}
		d2, err := strconv.Atoi(string(ch[2]))
		if err != nil {
			logError(fmt.Errorf("mapeamento de audio invalido (numero de canal): [%s]\t", strChan))
			os.Exit(1)
		}

		audioChans = append(audioChans, audioChanT{
			mappping: ch[:l-3],
			lang:     ch[l-3:],
			start:    st,
			lenght:   d1 + d2,
		})
		count += d1 + d2
	}

	// fmt.Printf("AUDIO CHANNELS --> %#v\n", audioChans)

	allOptions := readJSON(confFile)
	if allOptions == nil {
		logError(fmt.Errorf("arquivo [%s] nao e' um JSON no formato correto", confFile))
		os.Exit(1)
	}
	// fmt.Println(allOptions)

	opt, ok := allOptions[runtime.GOOS]
	if !ok {
		logError(fmt.Errorf("sistema operacional invalido: '%s'. Deve ser 'Windows' ou 'linux'", runtime.GOOS))
		fmt.Println(allOptions[runtime.GOOS])
		os.Exit(2)
	}
	options := opt.(map[string]interface{})

	profiles := readJSON(profFile)
	// fmt.Println(profiles)

	tempDir, err := ioutil.TempDir("", assetID)
	if err != nil {
		logError(fmt.Errorf("falha ao criar diretorio temporario: [%v]", err.Error()))
		os.Exit(1)
	}
	tempDir += string(os.PathSeparator)

	defer func() { _ = os.RemoveAll(tempDir) }()
	encoding := profiles["encodings"].(map[string]interface{})

	fmt.Println("=== Iniciando encoding ===")

	done := false
	// Process audio
	if !ignoreAudio {
		err, done = encodeAudio(audioChans, profFile, inFile, assetID, outDir, tempDir, time, err, options)
		if done {
			return
		}
	}
	if !ignoreVideo {
		err, done = encodeVideo(profFile, inFile, assetID, outDir, tempDir, time, err, options)
		if done {
			return
		}
	}

	// Generate MPD
	var presetdash1 map[string]interface{}
	encoding = profiles["encodings"].(map[string]interface{})
	dashJ := encoding["mp4dash"].(map[string]interface{})
	dashT := dashJ["template"].([]interface{})
	replacer := strings.NewReplacer("%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
		"%o", path.Join(outDir, path.Base(inFile)))
	err = applyTemplate(presetdash1, dashT, replacer)
	if err != nil {
		logError(err)
		return
	}
	mp4dashExe := path.Join(options["mp4box_dir"].(string), options["mp4dash_exe"].(string))
	flatB, err := buildCommand(mp4dashExe, dashT)
	err = execCommand(flatB)
	if err != nil {
		logError(err)
		return
	}
	log("\n=== Processo de encoding terminado ===")
}

func encodeAudio(audioChans []audioChanT, profFile string, inFile string, assetID string, outDir string, tempDir string, time string, err error, options map[string]interface{}) (error, bool) {
	profiles := readJSON(profFile) // resets profile
	encoding := profiles["encodings"].(map[string]interface{})
	audioJ := encoding["ffmpeg_audio"].(map[string]interface{})
	presetsA := audioJ["presets"].([]interface{})
	for i := range presetsA {
		for _, ch := range audioChans {
			profiles = readJSON(profFile) // resets profile
			encoding = profiles["encodings"].(map[string]interface{})
			audioJ = encoding["ffmpeg_audio"].(map[string]interface{})
			audioTs := audioJ["templates"].(map[string]interface{})
			audioT := audioTs["default"].([]interface{})
			preset := presetsA[i].(map[string]interface{})
			replacerWords := []string{"%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
				"%o", path.Join(outDir, path.Base(inFile)), "%s", preset["suffix"].(string)}
			for j := 0; j < 9; j++ {
				replacerWords = append(replacerWords, fmt.Sprintf("%%%d", j), fmt.Sprintf("%d", ch.start+j))
			}
			replacerWords = append(replacerWords, "%l", ch.lang)
			replacer := strings.NewReplacer(replacerWords...)
			//fragJ := encoding["mp4fragment"].(map[string]interface{})
			//fragT := fragJ["template"].([]interface{})
			if time != "" {
				preset["-t"] = time
			}
			err = applyTemplate(preset, audioT, replacer)
			if err != nil {
				logError(err)
				return err, true
			}
			channelout, okc := preset["channelout"]
			if !okc {
				logError(fmt.Errorf("preset %d nao contem channelout", i))
				return err, true

			}
			channels := audioTs["channels"].(map[string]interface{})
			chMapping := ch.mappping + channelout.(string)
			channel, ok1 := channels[chMapping].([]interface{})
			if !ok1 {
				err = fmt.Errorf("mapeamento de canal de audio nao encontrado no arquivo de profiles (channels): [%s]", chMapping)
				logError(err)
				return err, true
			}
			for _, chTemp := range channel {
				chTemplate := chTemp.(map[string]interface{})
				err = applyTemplate(chTemplate, audioT, replacer)
				if err != nil {
					logError(err)
					return err, true
				}
			}
			//var presetfrag1 map[string]interface{}
			//err = applyTemplate(presetfrag1, fragT, replacer)
			//if err != nil {
			//	logError(err)
			//	return err, true
			//}

			// Encode files
			flatA, err1 := buildCommand(options["ffmpeg_exe"].(string), audioT)
			err1 = execCommand(flatA)
			if err1 != nil {
				logError(err1)
				return err1, true
			}
			//// Fragment files
			//mp4fragExe := path.Join(options["mp4box_dir"].(string), options["mp4fragment_exe"].(string))
			//flatF, err1 := buildCommand(mp4fragExe, fragT)
			//err1 = execCommand(flatF)
			//if err1 != nil {
			//	logError(err1)
			//	return err1, true
			//}
		}
	}
	return err, false
}

func encodeVideo(profFile string, inFile string, assetID string, outDir string, tempDir string, time string, err error, options map[string]interface{}) (error, bool) {
	profiles := readJSON(profFile) // resets profile
	encoding := profiles["encodings"].(map[string]interface{})
	videoJ := encoding["ffmpeg_video"].(map[string]interface{})
	presetsV := videoJ["presets"].([]interface{})
	// Process video
	var wg sync.WaitGroup
	errorList := make([]error, 0)
	for i := range presetsV {
		encoding = profiles["encodings"].(map[string]interface{})
		videoJ = encoding["ffmpeg_video"].(map[string]interface{})
		videoTs := videoJ["templates"].(map[string]interface{})
		videoT := videoTs["default"].([]interface{})
		profiles = readJSON(profFile) // resets profile
		presetV1 := presetsV[i].(map[string]interface{})

		replacerWords := []string{"%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
			"%o", path.Join(outDir, path.Base(inFile)), "%s", presetV1["suffix"].(string)}

		replacer := strings.NewReplacer(replacerWords...)
		//fragJ := encoding["mp4fragment"].(map[string]interface{})
		//fragT := fragJ["template"].([]interface{})
		if time != "" {
			presetV1["-t"] = time
		}
		err = applyTemplate(presetV1, videoT, replacer)
		if err != nil {
			logError(err)
			return err, true
		}
		//var presetfrag1 map[string]interface{}
		//err = applyTemplate(presetfrag1, fragT, replacer)
		//if err != nil {
		//	logError(err)
		//	return err, true
		//}
		// Encode files
		flatV, err1 := buildCommand(options["ffmpeg_exe"].(string), videoT)
		wg.Add(1)
		go func(group *sync.WaitGroup) {
			defer group.Done()
			err1 = execCommand(flatV)
			if err1 != nil {
				logError(err1)
				errorList = append(errorList, err1)
				// return err1, true
			}
		}(&wg)
		//// Fragment files
		//mp4fragExe := path.Join(options["mp4box_dir"].(string), options["mp4fragment_exe"].(string))
		//flatF, err1 := buildCommand(mp4fragExe, fragT)
		//err1 = execCommand(flatF)
		//if err1 != nil {
		//	logError(err1)
		//	return err1, true
		//}
	}
	wg.Wait()
	if len(errorList) > 0 {
		return errorList[0], true
	}
	return nil, false
}

func execCommand(flat []string) error {
	var cmdS []string
	if runtime.GOOS == "Windows" {
		cmdS = []string{"cmd.exe", "/c"}
	} else {
		cmdS = []string{"sh", "-c"}
	}
	cmd := exec.Command(cmdS[0], cmdS[1], strings.Join(flat, " "))
	log(fmt.Sprintf("\nExec: [%v]\n", strings.Join(cmd.Args[:], "|")))
	// cmdOut, err := cmd.CombinedOutput()
	cmdReader, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmdReaderO, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	scannerO := bufio.NewScanner(cmdReaderO)
	doneO := make(chan bool)
	go func() {
		for scannerO.Scan() {
			color.Cyan("%v\n", scannerO.Text())
		}
		doneO <- true
	}()
	scanner := bufio.NewScanner(cmdReader)
	done := make(chan bool)
	go func() {
		for scanner.Scan() {
			color.Yellow("%v\n", scanner.Text())
		}
		done <- true
	}()
	err = cmd.Start()
	if err != nil {
		return err
	}
	<-done
	<-doneO
	err = cmd.Wait()
	return err
}

func buildCommand(cmdExe string, template []interface{}) ([]string, error) {
	flat := make([]string, 0, 100)
	flat = append(flat, cmdExe)
	err := flattenArray(template, &flat)
	// fmt.Printf("%s ---> %#v\n", cmdExe, flat)
	return flat, err
}

func readJSON(confFile string) map[string]interface{} {
	cFile, err := os.Open(confFile)
	if err != nil {
		logError(err)
		os.Exit(3)
	}

	buf, err := ioutil.ReadAll(cFile)
	if err != nil {
		logError(err)
		os.Exit(3)
	}

	newBuf := latinToUTF8(buf)
	var json map[string]interface{}
	err = js.Unmarshal([]byte(newBuf), &json)
	if err != nil {
		logError(err)
		os.Exit(4)
	}
	return json
}

func flattenMap(m map[string]interface{}, result *[]string) error {
	for k, el := range m {
		if !strings.HasPrefix(k, "-") {
			continue
		}
		switch t := el.(type) {
		case map[string]interface{}:
			*result = append(*result, k)
			err := flattenMap(t, result)
			if err != nil {
				return err
			}
		case []interface{}:
			*result = append(*result, k)
			err := flattenArray(t, result)
			if err != nil {
				return err
			}
		case string:
			*result = append(*result, k, t)
		default:
			if t != nil {
				return fmt.Errorf("elemento nao string no arquivo de profiles: [%v]", t)
			}
		}
	}
	return nil
}

func flattenArray(arr []interface{}, result *[]string) error {
	for _, el := range arr {
		switch t := el.(type) {
		case map[string]interface{}:
			err := flattenMap(t, result)
			if err != nil {
				return err
			}
		case []interface{}:
			err := flattenArray(t, result)
			if err != nil {
				return err
			}
		case string:
			*result = append(*result, t)
		default:
			return fmt.Errorf("elemento nao string no arquivo de profiles: [%v]", t)
		}
	}
	return nil
}

func applyTemplate(preset map[string]interface{}, templs []interface{}, replacer *strings.Replacer) error {
	if preset == nil {
		preset = make(map[string]interface{})
	}
	for i, templ := range templs {
		switch t := templ.(type) {
		case map[string]interface{}: // TODO []interface{}
			for k, v := range t {
				val, ok := preset[k]
				if !ok {
					switch vv := v.(type) {
					case string:
						vv2 := replaceParams(vv, replacer)
						if vv2 != vv {
							t[k] = vv2
						}
					}
					continue
				}
				switch v1 := val.(type) {
				case string:
					t[k] = replaceParams(v1, replacer) // Max level of embedding: must be a string
				default:
					return fmt.Errorf("parametro [%s] tem que ser string", k)
				}
			}
		case string:
			t2 := replaceParams(t, replacer)
			if t2 != t {
				templs[i] = t2
			}
		}
	}
	return nil
}

func replaceParams(str string, replacer *strings.Replacer) string {
	const LIMIT = 5 // limit of interactions
	result := str
	count := 0
	for {
		old := result
		result = replacer.Replace(result)
		if old == result {
			break
		}
		count++
		if count == LIMIT {
			log(fmt.Sprintf("WARNING: limite de substituicoes atingido na string [%s], original[%s]", result, str))
			break
		}
	}
	return result
}
