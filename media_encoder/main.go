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
	"strings"
)

func main() {

	inFile := ""
	confFile := ""
	profFile := ""
	outDir := ""
	assetID := ""
	flag.StringVar(&inFile, "in", "", "Arquivo de entrada")
	flag.StringVar(&assetID, "assetid", "", "Asset ID")
	flag.StringVar(&confFile, "options", "", "Arquivo JSON de configuracao")
	flag.StringVar(&profFile, "profiles", "", "Arquivo JSON com os profiles")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
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

	allOptions := readJSON(confFile)
	if allOptions == nil {
		logError(fmt.Errorf("arquivo [%s] nao e' um JSON no formato correto", confFile))
		os.Exit(1)
	}
	fmt.Println(allOptions)

	opt, ok := allOptions[runtime.GOOS]
	if !ok {
		logError(fmt.Errorf("sistema operacional invalido: '%s'. Deve ser 'Windows' ou 'linux'", runtime.GOOS))
		fmt.Println(allOptions[runtime.GOOS])
		os.Exit(2)
	}
	options := opt.(map[string]interface{})

	profiles := readJSON(profFile)
	fmt.Println(profiles)

	tempDir, err := ioutil.TempDir("", assetID)
	if err != nil {
		logError(fmt.Errorf("falha ao criar diretorio temporario: [%v]", err.Error()))
		os.Exit(1)
	}
	tempDir += string(os.PathSeparator)

	encoding := profiles["encodings"].(map[string]interface{})
	videoJ := encoding["ffmpeg_video"].(map[string]interface{})
	presetsV := videoJ["presets"].([]interface{})
	// Process video
	for i := range presetsV {
		encoding = profiles["encodings"].(map[string]interface{})
		videoJ = encoding["ffmpeg_video"].(map[string]interface{})
		videoT := videoJ["template"].([]interface{})
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
		profiles = readJSON(profFile) // resets profile
		encoding = profiles["encodings"].(map[string]interface{})
		audioJ = encoding["ffmpeg_audio"].(map[string]interface{})
		audioT := audioJ["template"].([]interface{})
		presetA1 := presetsA[i].(map[string]interface{})
		replacer := strings.NewReplacer("%i", inFile, "%a", assetID, "%d", outDir, "%t", tempDir,
			"%o", path.Join(outDir, path.Base(inFile)), "%s", presetA1["suffix"].(string))
		fragJ := encoding["mp4fragment"].(map[string]interface{})
		fragT := fragJ["template"].([]interface{})
		applyTemplate(presetA1, audioT, replacer)
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
		case map[string]interface{}:
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
				t[k] = val
			}
		case string:
			t2 := replacer.Replace(t)
			if t2 != t {
				templs[i] = t2
			}
		}
	}
}
