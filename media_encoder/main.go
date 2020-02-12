package main

import (
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
	flag.StringVar(&inFile, "in", "", "Arquivo de entrada")
	flag.StringVar(&assetID, "assetid", "", "Asset ID")
	flag.StringVar(&confFile, "options", "", "Arquivo JSON de configuracao")
	flag.StringVar(&profFile, "profiles", "", "Arquivo JSON com os profiles")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
	flag.StringVar(&strChan, "audiochan", "", "Mapeamento dos canais de audio\n"+
		"formato: <mapeamento>|<mapeamento>|...\n"+
		"mapeamento: s<stream inicial>i<entrada>o<saida><linguagem (3 letras)>")
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
	audioChans := make([]audioChanT, 0)
	strChans := strings.Split(strChan, "|")
	count := 0
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

	fmt.Printf("AUDIO CHANNELS --> %#v\t", audioChans)

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

	defer os.RemoveAll(tempDir)

	encoding := profiles["encodings"].(map[string]interface{})
	videoJ := encoding["ffmpeg_video"].(map[string]interface{})
	presetsV := videoJ["presets"].([]interface{})
	// Process video
	for i := range presetsV {
		break
		encoding = profiles["encodings"].(map[string]interface{})
		videoJ = encoding["ffmpeg_video"].(map[string]interface{})
		videoTs := videoJ["templates"].(map[string]interface{})
		videoT := videoTs["default"].([]interface{})
		profiles = readJSON(profFile) // resets profile
		presetV1 := presetsV[i].(map[string]interface{})
		replacer := strings.NewReplacer("%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
			"%o", path.Join(outDir, path.Base(inFile)), "%s", presetV1["suffix"].(string))
		fragJ := encoding["mp4fragment"].(map[string]interface{})
		fragT := fragJ["template"].([]interface{})
		var presetfrag1 map[string]interface{}
		applyTemplate(presetV1, videoT, replacer)
		applyTemplate(presetfrag1, fragT, replacer)
		// Encode files
		flatV, err := buildCommand(options["ffmpeg_exe"].(string), videoT)
		err = execCommand(flatV)
		if err != nil {
			logError(err)
			return
		}
		// Fragment files
		mp4fragExe := path.Join(options["mp4box_dir"].(string), options["mp4fragment_exe"].(string))
		flatF, err := buildCommand(mp4fragExe, fragT)
		err = execCommand(flatF)
		if err != nil {
			logError(err)
			return
		}
	}
	// Process audio
	audioJ := encoding["ffmpeg_audio"].(map[string]interface{})
	presetsA := audioJ["presets"].([]interface{})
	for i := range presetsA {
		for _, ch := range audioChans {
			profiles = readJSON(profFile) // resets profile
			encoding = profiles["encodings"].(map[string]interface{})
			audioJ = encoding["ffmpeg_audio"].(map[string]interface{})
			audioTs := audioJ["templates"].(map[string]interface{})
			audioT := audioTs["default"].([]interface{})
			presetA1 := presetsA[i].(map[string]interface{})
			replacerWords := []string{"%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
				"%o", path.Join(outDir, path.Base(inFile)), "%s", presetA1["suffix"].(string)}
			for j := 0; j < 9; j++ {
				replacerWords = append(replacerWords, fmt.Sprintf("%%%d", j), fmt.Sprintf("%d", ch.start+j))
			}
			replacerWords = append(replacerWords, "%l", ch.lang)
			replacer := strings.NewReplacer(replacerWords...)
			fragJ := encoding["mp4fragment"].(map[string]interface{})
			fragT := fragJ["template"].([]interface{})
			applyTemplate(presetA1, audioT, replacer)
			channels := audioTs["channels"].(map[string]interface{})
			channel, ok1 := channels[ch.mappping].([]interface{})
			if !ok1 {
				logError(fmt.Errorf("mapeamento de canal de audio nao encontrado no arquivo de profiles: [%s]", ch.mappping))
				return
			}
			for _, chTemp := range channel {
				chTemplate := chTemp.(map[string]interface{})
				applyTemplate(chTemplate, audioT, replacer)
			}
			var presetfrag1 map[string]interface{}
			applyTemplate(presetfrag1, fragT, replacer)
			// Encode files
			flatA, err := buildCommand(options["ffmpeg_exe"].(string), audioT)
			err = execCommand(flatA)
			if err != nil {
				logError(err)
				return
			}
			// Fragment files
			mp4fragExe := path.Join(options["mp4box_dir"].(string), options["mp4fragment_exe"].(string))
			flatF, err := buildCommand(mp4fragExe, fragT)
			err = execCommand(flatF)
			if err != nil {
				logError(err)
				return
			}
		}
	}

	// Generate MPD
	var presetdash1 map[string]interface{}
	dashJ := encoding["mp4dash"].(map[string]interface{})
	dashT := dashJ["template"].([]interface{})
	replacer := strings.NewReplacer("%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
		"%o", path.Join(outDir, path.Base(inFile)))
	applyTemplate(presetdash1, dashT, replacer)
	mp4dashExe := path.Join(options["mp4box_dir"].(string), options["mp4dash_exe"].(string))
	flatB, err := buildCommand(mp4dashExe, dashT)
	err = execCommand(flatB)
	if err != nil {
		logError(err)
		return
	}

	// ~/bin/Bento4-SDK-1-5-1-629.x86_64-unknown-linux/bin/mp4dash --no-split --use-segment-list --no-media --mpd-name=teste.mpd -f -o . Robot-stream1.mp4 Robot-stream3.mp4
}

func execCommand(flat []string) error {
	var cmdS []string
	if runtime.GOOS == "Windows" {
		cmdS = []string{"cmd.exe", "/c"}
	} else {
		cmdS = []string{"sh", "-c"}
	}
	cmd := exec.Command(cmdS[0], cmdS[1], strings.Join(flat, " "))
	fmt.Printf("Exec: [%v]\n", strings.Join(cmd.Args[:], "|"))
	cmdOut, err := cmd.CombinedOutput()
	fmt.Printf("output = [\n%v]\n", string(cmdOut))
	return err
}

func buildCommand(cmdExe string, template []interface{}) ([]string, error) {
	flat := make([]string, 0, 100)
	flat = append(flat, cmdExe)
	err := flattenArray(template, &flat)
	fmt.Printf("%s ---> %#v\n", cmdExe, flat)
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

func applyTemplate(preset map[string]interface{}, templs []interface{}, replacer *strings.Replacer) {
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
						vv2 := replacer.Replace(vv)
						if vv2 != vv {
							t[k] = vv2
						}
					}
					continue
				}
				t[k] = replacer.Replace(val.(string)) // Max level of embedding: must be a string
			}
		case string:
			t2 := replacer.Replace(t)
			if t2 != t {
				templs[i] = t2
			}
		}
	}
}
