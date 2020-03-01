package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const DEFAULT_LISTEN_PORT int = 8000

// This is embedded because otherwise the binary needs to find template HTML
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
			h1 { font-size: 14pt; font-weight: 700; }
			h2 { font-size: 12pt; font-weight: 700; font-family: monospace; }
			th { font-weight: 700; text-align: left; }
			td { padding-left: 5px; padding-right: 5px; font-family: monospace; }
			a:link    { color: #67ce2c; }
			a:visited { color: #67ce2c; }
			a:hover   { color: #97ee4c; text-decoration: none; }
		</style>
	</head>
	<body>
		<h1>Upload a File</h1>
		<form action="/upload/{{.CurrentPath}}" method="POST" enctype="multipart/form-data">
			<div><input type="file" name="file"></div>
			<div><input type="submit" value="Upload"></div>
		</form>

		<h2>{{range $folder := .PathParts}}{{$folder}} > {{end}}</h2>

		<table>
			{{range $item := .Items}}
			<tr>
				<td>{{$item.Size}}</td>

				<td>
				{{if $item.IsDir}}
				<a href="/browse/{{$item.RelPath}}">{{$item.Name}}/</a>
				{{else}}
				<a href="/download/{{$item.RelPath}}">{{$item.Name}}</a>
				{{end}}
				</td>
			</tr>
			{{end}}
		</table>
	</body>
</html>`

var suffixes = []string{"bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// This was borrowed from https://stackoverflow.com/a/25613067
func humanizeFileSize(size int64) string {
	if size == 0 {
		return "0 bytes"
	} else if size == 1 {
		return "1 byte"
	}

	order := uint(math.Log2(float64(size)) / 10.0)
	denom := 1 << (order * 10)
	realSize := float32(size) / float32(denom)
	return fmt.Sprintf("%0.1f %s", realSize, suffixes[order])
}

type SideGate struct {
	// Directory that will be served
	Root string

	// Port to listen on
	Port int

	// Template for the index page
	indexTemplate *template.Template
}

func NewSideGate(root string, listenPort int) (*SideGate, error) {
	indexTemplate, err := template.New("index-page").Parse(TEMPLATE_INDEX)
	if err != nil {
		return nil, fmt.Errorf("Unable to build index page template: %w", err)
	}

	app := SideGate{
		Root: root,
		Port: listenPort,

		indexTemplate: indexTemplate,
	}

	return &app, nil
}

func (s SideGate) downloadHandler(w http.ResponseWriter, r *http.Request) {
	relPath := strings.Replace(r.URL.Path, "/download", "", 1)
	relPath = strings.TrimLeft(relPath, "/")

	var fullPath strings.Builder
	fullPath.WriteString(s.Root)
	fullPath.WriteRune(os.PathSeparator)
	fullPath.WriteString(relPath)

	http.ServeFile(w, r, fullPath.String())
}

func (s SideGate) uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1024 * 1024 * 10)

	fin, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Unable to get `file` parameter: %w", err)
		http.Error(w, "Unable to get file data from request", http.StatusInternalServerError)
		return
	}
	defer fin.Close()

	name := header.Filename

	// Create upload destination
	relPath := strings.Replace(r.URL.Path, "/upload", "", 1)
	relPath = strings.TrimLeft(relPath, "/")

	var outputFile strings.Builder
	outputFile.WriteString(s.Root)
	outputFile.WriteRune(os.PathSeparator)
	if relPath != "" {
		outputFile.WriteString(relPath)
		outputFile.WriteRune(os.PathSeparator)
	}
	outputFile.WriteString(name)

	filePath := outputFile.String()

	fout, err := os.Create(filePath)
	if err != nil {
		log.Printf("Unable to create file %s: %w", filePath, err)
		http.Error(w, "Unable to create file on disk", http.StatusInternalServerError)
		return
	}
	defer fout.Close()

	// Stream file to disk
	bytes, err := io.Copy(fout, fin)
	if err != nil {
		log.Printf("Failed to save file to path %s: %w", filePath, err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	log.Printf("File uploaded to: %s (%s)", filePath, humanizeFileSize(bytes))

	// Redirect back to the directory index
	var redirectPath strings.Builder
	redirectPath.WriteString("/browse/")
	redirectPath.WriteString(relPath)
	http.Redirect(w, r, redirectPath.String(), http.StatusFound)
}

type Node struct {
	// The base name of the file
	Name string

	// Is it a directory?
	IsDir bool

	// Human-readable file size
	Size string

	// The path to the file, relative to the root directory.
	RelPath string
}

type Directory struct {
	// The current path being served, relative to the root directory.
	CurrentPath string

	// Path being served by this request, with the root directory removed, and
	// each folder as a separate item in the array.
	// E.g. If we're serving /tmp, and the path being served is /tmp/foo/bar,
	// then CurrentPath will be: []string{"foo", "bar"}
	// This is used to show the path context, for friendlier browsing.
	PathParts []string

	// Files/directories and the metadata needed for rendering
	Items []Node
}

func (s SideGate) indexHandler(w http.ResponseWriter, r *http.Request) {
	relPath := strings.Replace(r.URL.Path, "/browse", "", 1)
	relPath = strings.TrimLeft(relPath, "/")

	var fullPath strings.Builder
	fullPath.WriteString(s.Root)
	fullPath.WriteRune(os.PathSeparator)
	fullPath.WriteString(relPath)

	dirObjects, err := ioutil.ReadDir(fullPath.String())
	if err != nil {
		log.Printf("Unable to read contents of directory %s: %w", fullPath.String(), err)
		s.indexTemplate.Execute(w, nil)
		return
	}

	numObjects := len(dirObjects)
	dirContents := make([]Node, numObjects)
	for i, obj := range dirObjects {
		var fileSize string
		if obj.IsDir() {
			fileSize = ""
		} else {
			fileSize = humanizeFileSize(obj.Size())
		}

		var fileRelPath strings.Builder
		if relPath != "" {
			fileRelPath.WriteString(relPath)
			fileRelPath.WriteRune(os.PathSeparator)
		}
		fileRelPath.WriteString(obj.Name())

		dirContents[i] = Node{
			Name:    obj.Name(),
			Size:    fileSize,
			IsDir:   obj.IsDir(),
			RelPath: fileRelPath.String(),
		}
	}

	sort.Slice(dirContents, func(i, j int) bool {
		if dirContents[i].IsDir && !dirContents[j].IsDir {
			return true
		}

		if dirContents[i].IsDir && dirContents[j].IsDir {
			return dirContents[i].Name < dirContents[j].Name
		}

		return false
	})

	s.indexTemplate.Execute(w, Directory{
		CurrentPath: relPath,
		PathParts:   strings.Split(relPath, "/"),
		Items:       dirContents,
	})
}

func (s SideGate) OpenTheGate() error {
	var listenAddrStr strings.Builder
	listenAddrStr.WriteString(":")
	listenAddrStr.WriteString(strconv.Itoa(s.Port))
	listenAddress := listenAddrStr.String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/browse", http.StatusFound)
	})
	mux.HandleFunc("/browse/", s.indexHandler)
	mux.HandleFunc("/upload/", s.uploadHandler)
	mux.HandleFunc("/download/", s.downloadHandler)

	server := http.Server{
		Addr:    listenAddress,
		Handler: mux,
	}

	log.Printf("Serving local directory %s", s.Root)

	// Best-effort attempt to get the local IP address, which can be used to
	// share directly with the person with whom we are sharing files.
	//
	// If we're unable to get any IP addresses from the local network
	// interfaces at all, we don't hold up the start-up of the system; we just
	// ignore and move on and let the user figure it out instead.
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Unable to get local network interfaces. Ignoring.", err)
		ifaces = []net.Interface{}
	}

	localIpFound := false
	for _, i := range ifaces {
		addrs, err := i.Addrs()

		if err != nil {
			log.Printf("Unable to get addresses from local interface %w. Ignoring.", i)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// There's a ton of interfaces that we're not interested in, so we
			// do our best to find the local IP address that's not a loopback
			// or multicast or unicast IP.
			if !ip.IsLoopback() &&
				!ip.IsLinkLocalUnicast() &&
				!ip.IsLinkLocalMulticast() &&
				!ip.IsMulticast() &&
				!ip.IsInterfaceLocalMulticast() {

				log.Printf("Listening on http://%s:%d", ip, s.Port)
				localIpFound = true
			}
		}
	}

	if !localIpFound {
		log.Printf("Listening on port %d", s.Port)
	}

	return server.ListenAndServe()
}

func main() {
	cwd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Unable to get current working directory: %w", err)
	}

	destinationDir := flag.String("destDir", cwd, "destination folder")
	listenPort := flag.Int("port", DEFAULT_LISTEN_PORT, "port to serve HTTP endpoint")
	flag.Parse()

	app, err := NewSideGate(*destinationDir, *listenPort)
	if err != nil {
		log.Fatalf("Unable to initialise: %w", err)
	}

	app.OpenTheGate()
}
