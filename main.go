package main

import (
	"flag"
	"html/template"
	"io"
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
const TEMPLATE_INDEX string = `<!DOCTYPE html><html><head><title>SideGate</title></head><body><form action="/upload" method="POST" enctype="multipart/form-data"><div><input type="file" name="file"></div><div><input type="submit" value="Upload"></div></form></body></html>`
const TEMPLATE_UPLOAD string = `<!DOCTYPE html><html><head><title>SideGate</title></head><body><p>Upload successful!</p></body></html>`

const DEFAULT_LISTEN_PORT int = 8000

func uploadHandler(w http.ResponseWriter, r *http.Request, destinationDir string) {
	r.ParseMultipartForm(math.MaxInt32)

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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.New("index-page").Parse(TEMPLATE_INDEX)
	t.Execute(w, nil)
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

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadHandler(w, r, *destinationDir)
	})

	log.Printf("Saving uploads to %s", *destinationDir)
	log.Printf("Listening on %s", listenAddress)

	log.Fatal(http.ListenAndServe(listenAddress, nil))
}
