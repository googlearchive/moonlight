// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package provides frontend server as described in the design section
// of README.md
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
)

// defaultFmt is the default output format for lighthouse-cli.
// It has to be one of outFmt keys.
const defaultFmt = "html"

// outFmt maps lighthouse output formats to their corresponding mime types.
var outFmt = map[string]string{
	"pretty": "text/plain; charset=utf-8",
	"html":   "text/html; charset=utf-8",
	"json":   "application/json; charset=utf-8",
}

// main is the program entry point.
func main() {
	http.HandleFunc("/audit", handleAudit)
	http.HandleFunc("/_ah/start", ok)
	http.HandleFunc("/_ah/stop", ok)
	http.HandleFunc("/_ah/health", ok)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

// ok responds with HTTP 200 / OK and empty body
func ok(w http.ResponseWriter, r *http.Request) {}

// handleAudit responds to the in-flight audit requests.
// It runs lighthouse-cli and responds with the command stdout.
func handleAudit(w http.ResponseWriter, r *http.Request) {
	if o := r.Header.Get("origin"); o != "" {
		w.Header().Set("access-control-allow-origin", o)
		w.Header().Set("access-control-allow-credentials", "true")
		w.Header().Set("access-control-allow-methods", "GET, OPTIONS")
		w.Header().Set("access-control-max-age", "3600")
	}

	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "url param is empty", http.StatusBadRequest)
		return
	}

	ofmt := r.FormValue("fmt")
	if _, ok := outFmt[ofmt]; !ok {
		ofmt = defaultFmt
	}

	res, err := lighthouse(url, ofmt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", outFmt[ofmt])
	w.Write(res)
}

// lighthouse executes lighthouse-cli and returns its stdout.
// It passes the url and ofmt as --output arguments to the cli.
func lighthouse(url, ofmt string) ([]byte, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("unable to find free port for headless: %v", err)
	}
	h, err := startHeadless(port)
	if err != nil {
		return nil, fmt.Errorf("headless_shell: %v", err)
	}
	defer func() { go h.Process.Kill() }()

	cmd := exec.Command("node_modules/.bin/lighthouse", "--skip-autolaunch", "--verbose", "--output="+ofmt, url)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("PORT=%d", port),
		"DEBUG_FD=2",     // ensure debug package logs to stderr
		"DEBUG_COLORS=0", // no color-escape chars in logs
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// start executiong and read all stdout/err
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	bout, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, err
	}
	berr, err := ioutil.ReadAll(stderr)
	if err != nil {
		return nil, err
	}

	// wait until command exits
	err = cmd.Wait()
	log.Printf("lighthouse stderr on %s (%s)\n%s", url, ofmt, berr)
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, berr)
	}
	return bout, nil
}

// startHeadless starts a new instance of headless_shell with remote debugging enabled
// on the given port number.
func startHeadless(port int) (*exec.Cmd, error) {
	// TODO: remove dir when cmd.Process is killed
	dir, err := ioutil.TempDir("", "headless")
	if err != nil {
		return nil, err
	}
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", dir),
		"--no-sandbox",
		"about:blank",
	}
	cmd := exec.Command("./bin/headless_shell", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("headless_shell %q", args)
	return cmd, cmd.Start()
}

// freePort returns an unused tcp port number.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "")
	if err != nil {
		return -1, err
	}
	a := l.Addr().String()
	l.Close()
	_, p, err := net.SplitHostPort(a)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(p)
}
