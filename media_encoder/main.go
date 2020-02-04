package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	inFile := ""
	confFile := ""
	outDir := ""
	flag.StringVar(&inFile, "in", "", "Arquivo de entrada")
	flag.StringVar(&confFile, "config", "", "Arquivo JSON de configuracao")
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

	if outDir != "" {
		st, err := os.Stat(outDir)
		if err != nil || !st.IsDir() {
			logError(fmt.Errorf("diretorio [%s] nao e' valido", outDir))
			os.Exit(1)
		}
	}

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

func logError(err error) {
	// phosphorize := ansi.ColorFunc("red")
	msg := fmt.Sprintf("ERRO: %v\n", err.Error())
	log(msg)
}

func log(msg string) {
	// phosphorize := ansi.ColorFunc("red")
	_, _ = fmt.Fprintln(os.Stderr, msg)
}
