package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

const (
	defaultTemplate = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="content-type" content="text/html; charset=utf-8">
<title>{{ .Title }}</title>
</head>
<body>
<h2>You are previewing {{ .FileName }}</h2>
{{ .Body }}
</body>
</html>`
)

type content struct {
	Title    string
	FileName string
	Body     template.HTML
}

func main() {
	filename := flag.String("file", "", "Markdown file to preview")
	skipPreview := flag.Bool("s", false, "Skip auto-preview")
	tFname := flag.String("t", "", "Alternate template name")
	flag.Parse()

	if *tFname == "" && os.Getenv("TEMPLATE_FILENAME") != "" {
		*tFname = os.Getenv("TEMPLATE_FILENAME")
	}

	if err := run(*filename, *tFname, os.Stdin, os.Stdout, *skipPreview); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(filename string, tFname string, in io.Reader, out io.Writer, skipPreview bool) error {

	input, err := getInput(in, filename, flag.Args()...)
	if err != nil {
		return err
	}

	htmlData, err := parseContent(input, tFname, filename)
	if err != nil {
		return err
	}

	temp, err := os.CreateTemp("", "mdp*.html")
	if err != nil {
		return err
	}

	outName := temp.Name()
	fmt.Fprintln(out, outName)

	if err := saveHTML(outName, htmlData); err != nil {
		return err
	}

	if skipPreview {
		return nil
	}

	defer os.Remove(outName)

	return preview(outName)
}

func parseContent(input []byte, tFname string, fileName string) ([]byte, error) {
	output := blackfriday.Run(input)
	body := bluemonday.UGCPolicy().SanitizeBytes(output)

	t, err := template.New("mdp").Parse(defaultTemplate)
	if err != nil {
		return nil, err
	}

	if tFname != "" {
		t, err = template.ParseFiles(tFname)
		if err != nil {
			return nil, err
		}
	}

	c := content{
		Title:    "Markdown Preview Tool",
		FileName: fileName,
		Body:     template.HTML(body),
	}

	var buffer bytes.Buffer

	if err := t.Execute(&buffer, c); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func saveHTML(outFname string, data []byte) error {
	return os.WriteFile(outFname, data, 0644)
}

func preview(fname string) error {
	cName := "xdg-open"
	cParams := []string{fname}

	cPath, err := exec.LookPath(cName)
	if err != nil {
		return err
	}

	err = exec.Command(cPath, cParams...).Run()

	time.Sleep(2 * time.Second)

	return err
}

func getInput(in io.Reader, filename string, args ...string) ([]byte, error) {
	if len(args) > 0 {
		return []byte(strings.Join(args, " ")), nil
	}

	if filename != "" {
		input, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		return input, nil
	}

	s := bufio.NewScanner(in)
	s.Scan()
	if err := s.Err(); err != nil {
		return nil, err
	}

	if len(s.Text()) == 0 {
		return nil, errors.New("Input cannot be blank")
	}

	return []byte(s.Text()), nil
}
