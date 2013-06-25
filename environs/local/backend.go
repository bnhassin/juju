// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package local

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// storageBackend provides HTTP access to a defined path. The local
// provider otimally would use a much simpler Storage, but this
// code may be useful in storage-free environs. Here it requires
// additional authentication work before it's viable.
type storageBackend struct {
	path string
}

// ServeHTTP handles the HTTP requests to the container.
func (s *storageBackend) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		if strings.HasSuffix(req.URL.Path, "*") {
			s.handleList(w, req)
		} else {
			s.handleGet(w, req)
		}
	case "PUT":
		s.handlePut(w, req)
	case "DELETE":
		s.handleDelete(w, req)
	default:
		http.Error(w, "method "+req.Method+" is not supported", http.StatusMethodNotAllowed)
	}
}

// handleGet returns a storage file to the client.
func (s *storageBackend) handleGet(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadFile(filepath.Join(s.path, req.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("404 %v", err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

// handleList returns the file names in the storage to the client.
func (s *storageBackend) handleList(w http.ResponseWriter, req *http.Request) {
	fp := filepath.Join(s.path, req.URL.Path)
	dir, prefix := filepath.Split(fp)
	names, err := readDirs(dir, prefix[:len(prefix)-1], len(s.path)+1)
	if err != nil {
		http.Error(w, fmt.Sprintf("404 %v", err), http.StatusNotFound)
		return
	}
	sort.Strings(names)
	data := []byte(strings.Join(names, "\n"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

// readDirs reads the directory hierarchy and compares the found
// names with the given prefix.
func readDirs(dir, prefix string, start int) ([]string, error) {
	names := []string{}
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		name := fi.Name()
		if strings.HasPrefix(name, prefix) {
			if fi.IsDir() {
				dnames, err := readDirs(filepath.Join(dir, name), prefix, start)
				if err != nil {
					return nil, err
				}
				names = append(names, dnames...)
				continue
			}
			fullname := filepath.Join(dir, name)[start:]
			names = append(names, fullname)
		}
	}
	return names, nil
}

// handlePut stores data from the client in the storage.
func (s *storageBackend) handlePut(w http.ResponseWriter, req *http.Request) {
	fp := filepath.Join(s.path, req.URL.Path)
	dir, _ := filepath.Split(fp)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		http.Error(w, fmt.Sprintf("500 %v", err), http.StatusInternalServerError)
		return
	}
	out, err := os.Create(fp)
	if err != nil {
		http.Error(w, fmt.Sprintf("500 %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, req.Body); err != nil {
		http.Error(w, fmt.Sprintf("500 %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// handleDelete removes a file from the storage.
func (s *storageBackend) handleDelete(w http.ResponseWriter, req *http.Request) {
	fp := filepath.Join(s.path, req.URL.Path)
	if err := os.Remove(fp); err != nil && !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("500 %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// listen starts an HTTP listener to serve the
// provider storage.
func listen(dataPath, ip string, port int) (net.Listener, error) {
	backend := &storageBackend{
		path: dataPath,
	}
	if err := os.MkdirAll(backend.path, 0777); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, fmt.Errorf("cannot start listener: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", backend)

	go http.Serve(listener, mux)

	return listener, nil
}
