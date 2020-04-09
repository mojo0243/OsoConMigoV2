# Go Parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_SERVER=server
BINARY_CLIENT=client
BINARY_SHELL=shell
FLAGS="-s -w"

# Client Settings
URL=https://172.16.30.120:8443/tienda/peluche
SECRET=superChief
COMMS=10
FLEX=2
NODE=A100

LDFLAGS="-s -w -X main.name=$(NODE) -X main.url=$(URL) -X main.comms=$(COMMS) -X main.flex=$(FLEX) -X main.secret=$(SECRET)"

all: build
build:
	cd server; $(GOBUILD) -o $(BINARY_SERVER) -v -ldflags $(FLAGS); cd ../
	cd client; $(GOBUILD) -o $(BINARY_CLIENT) -v -ldflags $(LDFLAGS); cd ../
	cd shell; $(GOBUILD) -o $(BINARY_SHELL) -v -ldflags $(FLAGS); cd ../
clean:
	$(GOCLEAN)
	rm -f server/$(BINARY_SERVER)
	rm -f client/$(BINARY_CLIENT)
	rm -f shell/$(BINARY_SHELL)
	rm -f client/$(BINARY_CLIENT)_arm
	rm -f client/$(BINARY_CLIENT)_ppc
	rm -f client/$(BINARY_CLIENT)_arm
	rm -f client/$(BINARY_CLIENT)_linux64
build_server:
	cd server; $(GOBUILD) -o $(BINARY_SERVER) -v -ldflags $(FLAGS); cd ../
build_shell:
	cd shell; $(GOBUILD) -o $(BINARY_SHELL) -v -ldflags $(FLAGS); cd ../
build_linux64:
	cd client; GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(NODE)_linux64 -v -ldflags $(LDFLAGS); cd ../
build_mips:
	cd client; GOOS=linux GOARCH=mips $(GOBUILD) -o $(NODE)_mips -v -ldflags $(LDFLAGS); cd ../
build_ppc:
	cd client; GOOS=linux GOARCH=ppc64 $(GOBUILD) -o $(NODE)_ppc -v -ldflags $(LDFLAGS); cd ../
build_arm:
	cd client; GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(NODE)_arm -v -ldflags $(LDFLAGS); cd ../
build_linux32:
	cd client; GOOS=linux GOARCH=386 $(GOBUILD) -o $(NODE)_linux32 -v -ldflags $(LDFLAGS); cd ../
build_netgear:
	cd client; GOOS=linux GOARCH=arm $(GOBUILD) -o $(NODE)_netgear -v -ldflags $(LDFLAGS); cd ../

deps:
	$(GOGET) gopkg.in/yaml.v2
	$(GOGET) golang.org/x/net/http2
	$(GOGET) github.com/lib/pq
	$(GOGET) github.com/c-bata/go-prompt
	$(GOGET) github.com/common-nighthawk/go-figure
	$(GOGET) github.com/jedib0t/go-pretty/table
