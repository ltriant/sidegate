//                       _     _                  _
//                      (_)   | |                | |
//                   ___ _  __| | ___  __ _  __ _| |_ ___
//                  / __| |/ _` |/ _ \/ _` |/ _` | __/ _ \
//                  \__ \ | (_| |  __/ (_| | (_| | ||  __/
//                  |___/_|\__,_|\___|\__, |\__,_|\__\___|
//                                     __/ |
//                                    |___/
//
//   Share files with friends by leaving the sidegate open... or something
//
//
//                                   {} {}
//                             !  !  II II  !  !
//                          !  I__I__II II__I__I  !
//                          I_/|--|--|| ||--|--|\_I
//         .-'"'-.       ! /|_/|  |  || ||  |  |\_|\ !       .-'"'-.
//        /===    \      I//|  |  |  || ||  |  |  |\\I      /===    \
//        \==     /   ! /|/ |  |  |  || ||  |  |  | \|\ !   \==     /
//         \__  _/    I//|  |  |  |  || ||  |  |  |  |\\I    \__  _/
//          _} {_  ! /|/ |  |  |  |  || ||  |  |  |  | \|\ !  _} {_
//         {_____} I//|  |  |  |  |  || ||  |  |  |  |  |\\I {_____}
//    !  !  |=  |=/|/ |  |  |  |  |  || ||  |  |  |  |  | \|\=|-  |  !  !
//   _I__I__|=  ||/|  |  |  |  |  |  || ||  |  |  |  |  |  |\||   |__I__I_
//   -|--|--|-  || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||=  |--|--|-
//   _|__|__|   ||_|__|__|__|__|__|__|| ||__|__|__|__|__|__|_||-  |__|__|_
//   -|--|--|   ||-|--|--|--|--|--|--|| ||--|--|--|--|--|--|-||   |--|--|-
//    |  |  |=  || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||   |  |  |
//    |  |  |   || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||=  |  |  |
//    |  |  |-  || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||   |  |  |
//    |  |  |   || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||=  |  |  |
//    |  |  |=  || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||   |  |  |
//    |  |  |   || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||   |  |  |
//    |  |  |   || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||-  |  |  |
//   _|__|__|   || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||=  |__|__|_
//   -|--|--|=  || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||   |--|--|-
//   _|__|__|   ||_|__|__|__|__|__|__|| ||__|__|__|__|__|__|_||-  |__|__|_
//   -|--|--|=  ||-|--|--|--|--|--|--|| ||--|--|--|--|--|--|-||=  |--|--|-
//   jgs |  |-  || |  |  |  |  |  |  || ||  |  |  |  |  |  | ||-  |  |  |
//  ~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^~~~~~~~~~~~

package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
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
			}
			@media (min-width: 850px) {
				body {
					margin: 0 auto;
					padding: 0.5em 1em;
				}
			}
			h1 { font-size: 14pt; font-weight: 700; }
			h2 { font-size: 12pt; font-weight: 700; font-family: monospace; }
			th { font-weight: 700; text-align: left; }
			td { padding-left: 5px; padding-right: 5px; font-family: monospace; }
			a:link    { color: #67ce2c; }
			a:visited { color: #67ce2c; }
			a:hover   { color: #97ee4c; text-decoration: none; }
		</style>

		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	</head>
	<body>
		<h1>Upload a File</h1>
		<form action="/upload/{{.CurrentPath}}" method="POST" enctype="multipart/form-data">
			<div><input type="file" name="file" multiple></div>
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

func fileIsReadable(fullPath string) bool {
	info, err := os.Lstat(fullPath)
	if err != nil {
		log.Printf("Unable to lstat %s: %s", fullPath, err)
		return false
	}

	// Only serve files that are world-readable.
	if (info.Mode() & fs.ModePerm & 0o004) == 0 {
		return false
	}

	return true
}

type SideGate struct {
	// Directory that will be served
	Root string

	// Port to listen on
	Port int

	// Template for the index page
	indexTemplate *template.Template

	// The basename of the root directory
	rootBasename string
}

func NewSideGate(root string, listenPort int) (*SideGate, error) {
	indexTemplate, err := template.New("index-page").Parse(TEMPLATE_INDEX)
	if err != nil {
		return nil, fmt.Errorf("Unable to build index page template: %s", err.Error())
	}

	var rootBasename string
	basename := strings.Split(root, "/")
	if len(basename) > 0 {
		rootBasename = basename[len(basename)-1]
	} else {
		rootBasename = ""
	}

	app := SideGate{
		Root: root,
		Port: listenPort,

		indexTemplate: indexTemplate,
		rootBasename:  rootBasename,
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

	if !fileIsReadable(fullPath.String()) {
		http.Error(w, "File is not readable", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, fullPath.String())
}

func (s SideGate) uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1024 * 1024 * 10)

	// The base path on the filesystem for the file upload(s)
	relPath := strings.Replace(r.URL.Path, "/upload", "", 1)
	relPath = strings.TrimLeft(relPath, "/")

	var outputFileBase strings.Builder
	outputFileBase.WriteString(s.Root)
	outputFileBase.WriteRune(os.PathSeparator)
	if relPath != "" {
		outputFileBase.WriteString(relPath)
		outputFileBase.WriteRune(os.PathSeparator)
	}

	m := r.MultipartForm
	files := m.File["file"]

	for i, _ := range files {
		fin, err := files[i].Open()

		if err != nil {
			log.Printf("Unable to get handle to submitted file: %w", err)
			http.Error(w, "Unable to get file data from request",
				http.StatusInternalServerError)
			return
		}

		defer fin.Close()

		name := files[i].Filename

		var outputFile strings.Builder
		outputFile.WriteString(outputFileBase.String())
		outputFile.WriteString(name)
		filePath := outputFile.String()

		fout, err := os.Create(filePath)
		if err != nil {
			log.Printf("Unable to create file %s: %w", filePath, err)
			http.Error(w, "Unable to create file on disk",
				http.StatusInternalServerError)
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
	}

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

	// Path being served by this request, with the root directory removed,
	// and each folder as a separate item in the array.
	// E.g. If we're serving /tmp, and the path being served is
	// /tmp/foo/bar, then PathParts will be: []string{"foo", "bar"}
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

	if relPath != "" {
		fullPath.WriteRune(os.PathSeparator)
		fullPath.WriteString(relPath)
	}

	dirObjects, err := ioutil.ReadDir(fullPath.String())
	if err != nil {
		log.Printf("Unable to read contents of directory %s: %s",
			fullPath.String(), err.Error())
		s.indexTemplate.Execute(w, nil)
		return
	}

	numObjects := len(dirObjects)
	dirContents := make([]Node, numObjects)
	for i, obj := range dirObjects {
		var absolutePath strings.Builder
		absolutePath.WriteString(fullPath.String())
		absolutePath.WriteRune(os.PathSeparator)
		absolutePath.WriteString(obj.Name())

		if !fileIsReadable(absolutePath.String()) {
			continue
		}

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
		// Sort directories to the top
		if dirContents[i].IsDir && !dirContents[j].IsDir {
			return true
		} else if !dirContents[i].IsDir && dirContents[j].IsDir {
			return false
		} else {
			// In this case, we're comparing either two directories
			// or two files.
			return dirContents[i].Name < dirContents[j].Name
		}
	})

	pathParts := []string{s.rootBasename}
	if parts := strings.Split(relPath, "/"); len(parts) > 0 && parts[0] != "" {
		pathParts = append(pathParts, parts...)
	}

	s.indexTemplate.Execute(w, Directory{
		CurrentPath: relPath,
		PathParts:   pathParts,
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
	// interfaces at all, we don't hold up the start-up of the system; we
	// just ignore and move on and let the user figure it out instead.
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Unable to get local network interfaces: %s. Ignoring.",
			err.Error())
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
	log.SetOutput(os.Stdout)
	cwd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Unable to get current working directory: %w", err)
	}

	servedDir := flag.String("dir", cwd, "folder to serve")
	listenPort := flag.Int("port", DEFAULT_LISTEN_PORT, "port to serve the HTTP endpoint")
	flag.Parse()

	app, err := NewSideGate(*servedDir, *listenPort)
	if err != nil {
		log.Fatalf("Unable to initialise: %w", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-signalChan:
			log.Printf("Got signal %s, exiting.", sig.String())
			os.Exit(1)
		}
	}()

	app.OpenTheGate()
}
