package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"golang.org/x/net/http2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Database connection
var db *sql.DB

// Configuration file
var cfg Config

// Configuration file struct
type Config struct {
	Server struct {
		Ip	string `yaml:"ip"`
		Port 	string `yaml:"port"`
		Uri 	string `yaml:"uri"`
		Secret 	string `yaml:"secret"`
		Cert 	string `yaml:"cert"`
		Key 	string `yaml:"key"`
		In	string `yaml:"in"`
		Out	string `yaml:"out"`
	} `yaml:"server"`
	Database struct {
		Dbhost 	string 	`yaml:"host"`
		Dbport 	int	`yaml:"port"`
		Dbuser 	string	`yaml:"user"`
		Dbpass 	string 	`yaml:"pass"`
		Dbname 	string 	`yaml:"name"`
		Dbmode 	string 	`yaml:"mode"`
	} `yaml:"database"`
}

// Client task object
type ClientIdentity struct {
	Id 		int 	`db:"id"`
	Node 		string 	`db:"node"`
	Arch 		string 	`db:"arch"`
	Os 		string 	`db:"os"`
	Secret          string  `db:"secret"`
	Comms        	int	`db:"comms"`
	Flex         	int   	`db:"flex"`
	FirstSeen 	int64 	`db:"firstSeen"`
	LastSeen 	int64 	`db:"lastSeen"`
}

// Client task object
type ClientTask struct {
	Id 		int 	`db:"id"`
	Node 		string 	`db:"node"`
	Job             int     `db:"job"`
	Command 	string 	`db:"command"`
	Status 		string 	`db:"status"`
	TaskDate 	int64 	`db:"taskDate"`
	CompleteDate 	int64 	`db:"completeDate"`
	Complete 	bool 	`db:"complete"`
}

// Validate client token struct
type ClientToken struct {
	Node 	string `db:"node"`
	Secret 	string `db:"secret"`
	Token  	string `db:"token"`
}

// Client
type Client struct {
	Node 	string `json:"node"`
	Secret 	string `json:"secret"`
	Job     string `json:"job"`
	Results	[]Result
}

// Client struct for posting results
type Result struct {
	Node   string `json:"node"`
	JobId  int    `json:"jobId"`
	Output string `json:"output"`
}

// Client struct for tasks
type Task struct {
	Id 	int 	`json:"id"`
	Command string 	`json:"command"`
}

type TaskList struct {
	ClientTasking []Task
}

func main() {
	// Required: server and database configuration file
	var config = flag.String("c", "config.yml", "Configuration file")
	flag.Parse()

	// Parse the configuration file
	initServer(*config)

	// Start the server
	StartServer(cfg.Server.Ip, cfg.Server.Port, cfg.Server.Uri)
}

func initServer(c string) {

	// Read server configuration
	f, err := os.Open(c)
	if err != nil {
		fmt.Println("[!] Missing configuration file.")
		os.Exit(3)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		fmt.Println("[!] Error reading configuration file.")
		os.Exit(3)
	}

	// Connect to database
	connect()

	// Create db schema if not exist
	CreateSchemas()

	// Get working path and make directories
	working := getWorkingDirectory()
	makeDirectories(working)
}

func StartServer(ip string, port string, uri string) {

	mux := http.NewServeMux()
	mux.HandleFunc(uri, taskHandler)

	// TLS server configuration
	s := fmt.Sprintf("%s:%s", ip, port)
	server := &http.Server{
		Addr: s,
		Handler: mux,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig: tlsConfig(),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// Configure TLS server
	var http2Server = http2.Server{}
	if err := http2.ConfigureServer(server, &http2Server); err != nil {
		processError(err)
	}

	d := fmt.Sprintf("[+] Go Backend: { HTTPVersion = 2 }\n[+] Server started")
	log.Print(d)

	// Start TLS Server
	if err := server.ListenAndServeTLS("", ""); err != nil {
		processError(err)
	}
}

func taskHandler(w http.ResponseWriter, req *http.Request) {

	if req.Method != "POST" {
		http.Error(w, "page not found", 404)
		return
	}

	// Instantiate client
	var a Client

	// Parse JSON client body
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&a)
	if err != nil {
		http.Error(w, "page not found", 404)
		return
	}

	aExists := clientExist(a.Node)
	if aExists {
		x := GetClient(a.Node)

		if x.Secret != a.Secret {
			http.Error(w, "page not found", 404)
			return
		}

		if a.Job == "reboot" || a.Job == "pulse" {
			if a.Job == "reboot" {
				err := AddRebootTask(a.Node)
				if err != nil {
					http.Error(w, err.Error(), 404)
					return
				}
				fmt.Println("[*] Reboot settings requested from ", a.Node)
			} else {
				fmt.Println("[*] Observed pulse from", a.Node)
			}

			// Update the client check in time
			UpdateClientStatus(a.Node)

			// Query client tasks based on node
			t := TaskList{}
			err := GetClientJobs(a.Node, &t)
			if err != nil {
				http.Error(w, err.Error(), 404)
				return
			}

			// Check for tasks, if none return 404
			if len(t.ClientTasking) < 1 {
				http.Error(w, "page not found", 404)
				return
			}

			// Tasks to json
			out, err := json.Marshal(t)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			_,_ = fmt.Fprint(w, string(out))
		} else if a.Job == "post" {
			if len(a.Results) > 0 {
				fmt.Println("[*] Receiving data from", a.Node)
			}
			for i := 0; i < len(a.Results); i++ {
				UpdateClientJobs(a.Results[i].JobId, strings.TrimSpace(a.Results[i].Output), a.Node)
			}
		} else {
			http.Error(w, "page not found", 404)
			return
		}
	} else if a.Node != "" && a.Secret != "" && a.Job != "" {
		if a.Job == "pulse" {
			token := generateRandomString()
			cmd := fmt.Sprintf("update %s", token)
			AddToken(a.Node, a.Secret, token)

			// Create a single task
			tList := TaskList{}
			t := Task{}
			t.Id = 0
			t.Command = cmd
			tList.ClientTasking = append(tList.ClientTasking, t)

			// Create JSON task
			out, err := json.Marshal(tList)

			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			fmt.Println("[+] New client check in from", a.Node)

			_,_ = fmt.Fprint(w, string(out))
		} else if a.Job == "update" {

			//t := b64Decode(strings.TrimSpace(a.Results[0].Output))
			t := strings.TrimSpace(a.Results[0].Output)
			c := strings.Split(t, ",")
			x := GetToken(a.Node, a.Secret, strings.TrimSpace(c[0]))

			if x {
				// Add the new client to the db
				w, _ := strconv.Atoi(strings.TrimSpace(c[5]))
				z, _ := strconv.Atoi(strings.TrimSpace(c[6]))
				AddNewClient(strings.TrimSpace(c[1]), strings.TrimSpace(c[2]), strings.TrimSpace(c[3]),
					strings.TrimSpace(c[4]), w, z)

				fmt.Println("[+] New client pulse from", a.Node)
			} else {
				http.Error(w, "page not found", 404)
				return
			}
		} else {
			http.Error(w, "page not found", 404)
			return
		}
	} else {
		http.Error(w, "page not found", 404)
		return
	}
}

// Error function
func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func b64Encode(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}

func b64Decode(s string) string {
	x,_ := b64.StdEncoding.DecodeString(s)
	return string(x)
}

func generateRandomString() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	length := 12
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			processError(err)
		}
	}
}

func getWorkingDirectory() string {
	// Get working directory
	path, err := os.Getwd()
	if err != nil {
		processError(err)
	}
	return path
}

func makeDirectories(path string) {
	// Create inbox and outbox paths
	outPath := fmt.Sprintf("%s/%s", path, cfg.Server.In)
	inPath := fmt.Sprintf("%s/%s", path, cfg.Server.Out)
	CreateDirIfNotExist(outPath)
	CreateDirIfNotExist(inPath)
}

// Server TLS Configuration
func tlsConfig() *tls.Config {

	crt, err := ioutil.ReadFile(cfg.Server.Cert)
	if err != nil {
		processError(err)
	}

	key, err := ioutil.ReadFile(cfg.Server.Key)
	if err != nil {
		processError(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		processError(err)
	}

	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		Certificates: []tls.Certificate{cert},
		ServerName:   "localhost",
	}
}

// Connect to postgres database
func connect() {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Dbhost, cfg.Database.Dbport, cfg.Database.Dbuser, cfg.Database.Dbpass, cfg.Database.Dbname,
		cfg.Database.Dbmode)

	var err error
	db, err = sql.Open("postgres", connectionString)
	if err != nil {
		processError(err)
	}

	err = db.Ping()
	if err != nil {
		processError(err)
	}
}

// Execute postgres commands
func exec(command string) {
	_, err := db.Exec(command)
	if err != nil {
		processError(err)
	}
}

// Get current epoch time
func GetCurrentEpoch() int64 {
	return time.Now().Unix()
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
		return "Error"
	}
}

// Add client to Postgres
func AddNewClient(a string, s string, ar string, o string, c int, j int) {
	t := "INSERT INTO clients (node,arch,os,secret,comms,flex,firstSeen,lastSeen) VALUES ('%s', '%s', '%s', '%s', %d, %d, %d, %d)"
	command := fmt.Sprintf(t, a, ar, o, s, c, j, GetCurrentEpoch(), GetCurrentEpoch())
	exec(command)
	fmt.Printf("[+] New client checked in: %s", a)
}

// Get client tasks
func GetClientJobs(n string, tList *TaskList) error {
	q := "SELECT job,command FROM tasks WHERE node='%s' AND complete=false AND status='Deployed' ORDER BY taskDate ASC"
	command := fmt.Sprintf(q, n)
	rows, err := db.Query(command)

	if err != nil {
		processError(err)
	}

	defer rows.Close()
	for rows.Next() {
		t := Task{}
		err = rows.Scan(&t.Id, &t.Command)
		if err != nil {
			return err
		}
		c := strings.Split(t.Command, " ")
		if c[0] == "push" {
			x := fmt.Sprintf("%s %s %s", c[0], readFile(c[1]), c[2])
			t.Command = x
		}
		UpdateClientJobStatus(t.Id)
		tList.ClientTasking = append(tList.ClientTasking, t)
	}

	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}

// Get client tasks
func AddRebootTask(n string) error {
	q := "SELECT comms,flex FROM clients WHERE node='%s' LIMIT 1"
	command := fmt.Sprintf(q, n)
	row := db.QueryRow(command)

	j := []string{"comms", "flex"}

	var a ClientIdentity
	switch err := row.Scan(&a.Comms, &a.Flex); err  {
	case sql.ErrNoRows:
		return err
	case nil:
		for _, s := range j {
			at := ClientTask{}
			at.Node = n
			at.Status = "Deployed"
			at.TaskDate = GetCurrentEpoch()
			at.CompleteDate = 0
			at.Complete = false

			if s == "comms" {
				// Make command string
				x := fmt.Sprintf("set comms %d", a.Comms)
				at.Command = x
			} else {
				x := fmt.Sprintf("set flex %d", a.Flex)
				at.Command = x
			}
			AddClientTask(at)
		}
		return  nil
	default:
		return err
	}
}

// Add task to Postgres
func AddClientTask(a ClientTask) {
	t := "INSERT INTO tasks (node, command, status, taskDate, completeDate, complete) VALUES ('%s', '%s', '%s', %d, %d, %t);"
	command := fmt.Sprintf(t, a.Node, a.Command, a.Status, a.TaskDate, a.CompleteDate, a.Complete)
	exec(command)
}

func clientExist(n string) bool {
	q := "SELECT id,node,secret FROM clients WHERE node='%s'"
	command := fmt.Sprintf(q, n)
	row := db.QueryRow(command)

	var a ClientIdentity
	switch err := row.Scan(&a.Id, &a.Node, &a.Secret); err  {
	case sql.ErrNoRows:
		return false
	case nil:
		return true
	default:
		return false
	}
}

func GetClient(n string) ClientIdentity {
	q := "SELECT id,node,secret FROM clients WHERE node='%s'"
	command := fmt.Sprintf(q, n)
	row := db.QueryRow(command)

	var a ClientIdentity
	switch err := row.Scan(&a.Id, &a.Node, &a.Secret); err  {
	case sql.ErrNoRows:
		a.Secret = ""
		return a
	case nil:
		return a
	default:
		a.Secret = ""
		return a
	}
}

func UpdateClientStatus(n string) {
	u := "UPDATE clients SET lastSeen = %d WHERE node = '%s'"
	c := fmt.Sprintf(u, GetCurrentEpoch(), n)
	exec(c)
}

func UpdateClientJobStatus(i int) {
	u := "UPDATE tasks SET status = 'Sent' WHERE job = %d"
	c := fmt.Sprintf(u, i)
	exec(c)
}

// Get client tasks
func UpdateClientJobs(i int, s string, n string) {
	u := "UPDATE tasks SET status = 'Complete', completeDate = %d, complete = true WHERE job = %d"
	c1 := fmt.Sprintf(u,GetCurrentEpoch(),i)
	exec(c1)

	j := "INSERT INTO results (node,jobId,output,completeDate) VALUES ('%s', %d, '%s', %d)"
	c2 := fmt.Sprintf(j, n, i, s, GetCurrentEpoch())
	exec(c2)
}

// Add Client token
func AddToken(n string, s string, t string) {
	u := "INSERT INTO tokens (node,secret,token) VALUES ('%s','%s','%s')"
	c := fmt.Sprintf(u, n, s, t)
	exec(c)
}

// Add Client token
func GetToken(n string, s string, t string) bool {
	q := "SELECT node,secret,token FROM tokens WHERE node='%s'"
	command := fmt.Sprintf(q, n)
	row := db.QueryRow(command)

	var a ClientToken
	switch err := row.Scan(&a.Node, &a.Secret, &a.Token); err  {
	case sql.ErrNoRows:
		return false
	case nil:
		if s == a.Secret && t == a.Token {
			return true
		} else {
			fmt.Println(a.Secret)
			return false
		}
	default:
		return false
	}
}

// Create schemas
func CreateSchemas() {
	createClientSchema()
	createTaskSchema()
	createResultSchema()
	createTokenSchema()
}

// Create Postgres database schema for clients
func createClientSchema() {
	clientSchema := `
        CREATE TABLE IF NOT EXISTS clients (
          id SERIAL PRIMARY KEY,
          node TEXT UNIQUE,
          arch TEXT,
          os TEXT,
          secret TEXT,
          comms INTEGER,
          flex INTEGER,
          firstSeen INTEGER,
          lastSeen INTEGER
        );
    `
	exec(clientSchema)
}

// Create Postgres database schema for tasks
func createTaskSchema() {
	taskSchema := `
        CREATE TABLE IF NOT EXISTS tasks (
          id SERIAL PRIMARY KEY,
          node TEXT,
   	  job SERIAL UNIQUE,
          command TEXT,
          status TEXT,
          taskDate INTEGER,
          completeDate INTEGER,
          complete BOOLEAN
        );
    `
	exec(taskSchema)
}

// Create Postgres database schema for results
func createResultSchema() {
	resultSchema := `
        CREATE TABLE IF NOT EXISTS results (
          id SERIAL PRIMARY KEY,
          node TEXT,
          jobId INTEGER,
          output TEXT,
          completeDate INTEGER
        );
    `
	exec(resultSchema)
}

// Create Postgres database schema for results
func createTokenSchema() {
	tokenSchema := `
        CREATE TABLE IF NOT EXISTS tokens (
          id SERIAL PRIMARY KEY,
          node TEXT UNIQUE,
          secret TEXT,
          token TEXT
        );
    `
	exec(tokenSchema)
}
