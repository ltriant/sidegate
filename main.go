package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// These are embedded because otherwise the binary needs to find template HTML
// files at runtime, which I don't wanna mess around with for now. And other
// solutions require third-party dependencies, which I'm trying to avoid unless
// absolutely necessary.
const TEMPLATE_INDEX string = `<!DOCTYPE html>
<html>
	<head>
		<title>SideGate</title>
		<style>
			body {
				line-height: 1.5;
				font-family: "Helvetica", "Arial", sans-serif;
				font-weight: 400;
				font-size: 12pt;
				color: #202020;

				padding: 0.5em 1em;
			}
			h1 { font-size: 14pt; }
			th { font-weight: 700; text-align: left; }
			td { padding-left: 5px; padding-right: 5px; font-family: monospace; }
			a:link    { color: #f99157; }
			a:visited { color: #f99157; }
			a:hover   { color: #faa61a; text-decoration: none; }
		</style>
	</head>
	<body>
		<h1>Upload a File</h1>
		<form action="/upload" method="POST" enctype="multipart/form-data">
			<div><input type="file" name="file"></div>
			<div><input type="submit" value="Upload"></div>
		</form>

		<h1>{{.ServePath}}</h1>
		<table>
			{{range $item := .Items}}
			<tr>
				<td>{{$item.Size}}</td>

				<td>
				{{if $item.IsDir}}
				{{$item.Name}}/
				{{else}}
				{{$item.Name}}
				{{end}}
				</td>
			</tr>
			{{end}}
		</table>
	</body>
</html>`

const TEMPLATE_UPLOAD string = `<!DOCTYPE html>
<html>
	<head>
		<title>SideGate</title>
		<style>
			body {
				line-height: 1.5;
				font-family: "Helvetica", "Arial", sans-serif;
				font-weight: 400;
				font-size: 12pt;
				color: #202020;

				padding: 0.5em 1em;
			}
			h1 { font-size: 14pt; }
			th { font-weight: 700; text-align: left; }
			td { padding-left: 5px; padding-right: 5px; font-family: monospace; }
			a:link    { color: #f99157; }
			a:visited { color: #f99157; }
			a:hover   { color: #faa61a; text-decoration: none; }
		</style>
	</head>

	<body>
		<p>Upload successful!</p>
		<p><a href="/">Go back</a></p>
	</body>
</html>`

const DEFAULT_LISTEN_PORT int = 8000

func uploadHandler(w http.ResponseWriter, r *http.Request, destinationDir string) {
	r.ParseMultipartForm(1024 * 1024 * 10)

	fin, header, err := r.FormFile("file")
	if err != nil {
		log.Fatalf("Unable to get `file` parameter: %v", err)
	}
	defer fin.Close()

	name := header.Filename

	// Create upload destination
	var outputFile strings.Builder
	outputFile.WriteString(destinationDir)
	outputFile.WriteString("/")
	outputFile.WriteString(name)

	filePath := outputFile.String()

	fout, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Unable to create file %s: %v", filePath, err)
	}
	defer fout.Close()

	// Stream file to disk
	bytes, err := io.Copy(fout, fin)
	if err != nil {
		log.Fatalf("Failed to save file to path %s: %v", filePath, err)
	}

	log.Printf("File uploaded to: %s (%d bytes)", filePath, bytes)

	t, _ := template.New("upload-page").Parse(TEMPLATE_UPLOAD)
	t.Execute(w, nil)
}

type Node struct {
	Name  string
	Size  string
	IsDir bool
}

type HomeDir struct {
	ServePath string
	Items     []Node
}

var suffixes = []string{"bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// This was borrowed from https://stackoverflow.com/a/25613067
func humanizeFileSize(size int64) string {
	if size == 1 {
		return "1 byte"
	}

	order := uint(math.Log2(float64(size)) / 10.0)
	denom := 1 << (order * 10)
	realSize := float32(size) / float32(denom)
	return fmt.Sprintf("%0.1f %s", realSize, suffixes[order])
}

func indexHandler(w http.ResponseWriter, r *http.Request, destDir string) {
	t, _ := template.New("index-page").Parse(TEMPLATE_INDEX)

	dirObjects, err := ioutil.ReadDir(destDir)
	if err != nil {
		log.Printf("Unable to read contents of directory %s: %v", destDir, err)
		t.Execute(w, nil)
		return
	}

	numObjects := len(dirObjects)
	dirContents := make([]Node, numObjects)
	for _, obj := range dirObjects {
		var fileSize string
		if obj.IsDir() {
			fileSize = ""
		} else {
			fileSize = humanizeFileSize(obj.Size())
		}

		dirContents = append(dirContents, Node{
			Name:  obj.Name(),
			Size:  fileSize,
			IsDir: obj.IsDir(),
		})
	}

	t.Execute(w, HomeDir{
		ServePath: destDir,
		Items:     dirContents,
	})
}

func main() {
	cwd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Unable to get current working directory: %v", err)
	}

	destinationDir := flag.String("destDir", cwd, "destination folder")
	listenPort := flag.Int("port", DEFAULT_LISTEN_PORT, "port to serve HTTP endpoint")
	flag.Parse()

	var listenAddrStr strings.Builder
	listenAddrStr.WriteString(":")
	listenAddrStr.WriteString(strconv.Itoa(*listenPort))
	listenAddress := listenAddrStr.String()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexHandler(w, r, *destinationDir)
	})
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadHandler(w, r, *destinationDir)
	})

	log.Printf("Saving uploads to %s", *destinationDir)
	log.Printf("Listening on %s", listenAddress)

	log.Fatal(http.ListenAndServe(listenAddress, nil))
}
