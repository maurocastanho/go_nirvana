package main

import (
	js "encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type jsonT map[string]interface{}

func main() {

	inFile := ""
	confFile := ""
	profFile := ""
	outDir := ""
	flag.StringVar(&inFile, "in", "", "Arquivo de entrada")
	flag.StringVar(&confFile, "options", "", "Arquivo JSON de configuracao")
	flag.StringVar(&profFile, "profiles", "", "Arquivo JSON com os profiles")
	flag.StringVar(&outDir, "outdir", "", "Diretorio de saida")
	flag.Parse()

	if inFile == "" {
		logError(fmt.Errorf("arquivo de entrada deve ser especificado na linha de comando"))
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

	options := readJSON(confFile)
	if options == nil {
		logError(fmt.Errorf("arquivo [%s] nao e' um JSON no formato correto", confFile))
		os.Exit(1)
	}
	fmt.Println(options)

	profiles := readJSON(profFile)

	fmt.Println(profiles)

	encoding := profiles["encodings"].(map[string]interface{})
	videoJ := encoding["ffmpeg_video"].(map[string]interface{})
	videoT := videoJ["template"].([]interface{})
	presetsV := videoJ["presets"].([]interface{})
	presetV1 := presetsV[1].(map[string]interface{})

	applyTemplate(presetV1, videoT)

	flat := make([]string, 0, 100)
	err := flattenArray(videoT, &flat)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Printf("---> %#v", flat)

	/*
		staticArgs := []string{"run", // roda o docker
			"--rm",                                 // remove imagem depois da execucao
			"-v", fmt.Sprintf("%s:/pdf", tempIDir), // monta diretorio de entrada em /pdf
			"-v", fmt.Sprintf("%s:/pdftest", tempDir), // monta diretorio de saida em /pdftest
			"bwits/pdf2htmlex", "pdf2htmlEX", "--dest-dir=/pdftest", // monta comando
		}
		// Acrescenta argumentos da linha de comando do docker
		cmdArgs := append(staticArgs, tempFileName)
		cmdArgs = append(cmdArgs, args[1:]...)
		cmd := exec.Command("ffmpeg", cmdArgs...)
		fmt.Printf("%v\n", strings.Join(cmd.Args[:], " "))
		// Executa pdf2htmlex
		cmdOut, err := cmd.CombinedOutput()
		if err != nil {
			err = stacktrace.Propagate(err, "erro ao executar o comando [docker], saida do comando: [%s]", cmdOut)
			status = 4
			return
		}
	*/

	// ~/bin/Bento4-SDK-1-5-1-629.x86_64-unknown-linux/bin/mp4dash --no-split --use-segment-list --no-media --mpd-name=teste.mpd -f -o . Robot-stream1.mp4 Robot-stream3.mp4
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
			if t == nil {
				*result = append(*result, k, "NULL")
			} else {
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
		case string:
			*result = append(*result, t)
		default:
			return fmt.Errorf("elemento nao string no arquivo de profiles: [%v]", t)
		}
	}
	return nil
}

func applyTemplate(preset map[string]interface{}, templs []interface{}) {
	for _, templ := range templs {
		t := templ.(map[string]interface{})
		for k := range t {
			val, ok := preset[k]
			if !ok {
				continue
			}
			t[k] = val
		}
	}
}
