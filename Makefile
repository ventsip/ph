.PHONY: clean build test build_test run

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY=ph
BINARY_SVC=phsvc
VERSION=$(shell git describe --long --always --dirty)
CFG_FILE=cfg.json
TEST_BINARY=test_process
TEST_BINARY1=test_process1
TEST_BINARY2=test_process2
BALANCE_JSON=balance.json
COPY_WEB_FOLDER_WIN=xcopy server\web bin\web /i /e /y
COPY_WEB_FOLDER_LINUX=cp -r server/web bin

clean:
ifeq ($(OS), Windows_NT)
	rmdir /S /Q bin
else
	rm -f -r bin
endif

build:
ifeq ($(OS), Windows_NT)
	cd cmd\cli & $(GOBUILD) -v -o ..\..\bin\$(BINARY).exe -ldflags="-X main.version=$(VERSION)"
	cd cmd\winsvc & $(GOBUILD) -v -o ..\..\bin\$(BINARY_SVC).exe -ldflags="-X main.version=$(VERSION)"

	copy testdata\$(CFG_FILE) bin
	$(COPY_WEB_FOLDER_WIN)
else
	cd cmd/cli; $(GOBUILD) -v -o ../../bin/$(BINARY) -ldflags="-X main.version=$(VERSION)"
	cd cmd/winsvc; env GOOS=windows GOARCH=amd64 $(GOBUILD) -v -o ../../bin/$(BINARY_SVC).exe -ldflags="-X main.version=$(VERSION)"
	
	cp testdata/$(CFG_FILE) bin
	$(COPY_WEB_FOLDER_LINUX)
endif

build_test:
ifeq ($(OS), Windows_NT)
	cd test_process & $(GOBUILD) -v -o ..\bin\$(TEST_BINARY).exe
	copy bin\$(TEST_BINARY).exe bin\$(TEST_BINARY1).exe
	copy bin\$(TEST_BINARY).exe bin\$(TEST_BINARY2).exe
	copy testdata\$(CFG_FILE) bin
	$(COPY_WEB_FOLDER_WIN)
else
	cd test_process; $(GOBUILD) -v -o ../bin/$(TEST_BINARY)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY1)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY2)
	cp testdata/$(CFG_FILE) bin
	$(COPY_WEB_FOLDER_LINUX)
endif

test: clean build_test
ifeq ($(OS), Windows_NT)
	copy testdata\$(CFG_FILE) bin
	$(COPY_WEB_FOLDER_WIN)
	$(GOTEST) .\... -v
else
	cp testdata/$(CFG_FILE) bin
	$(COPY_WEB_FOLDER_LINUX)
	$(GOTEST) ./... -v
endif

run: clean build build_test
ifeq ($(OS), Windows_NT)
	copy testdata\$(CFG_FILE) bin\ 
	$(COPY_WEB_FOLDER_WIN)
	cd bin & start $(TEST_BINARY)
	cd bin & start $(TEST_BINARY1)	
	cd bin & start $(TEST_BINARY2)
	cd bin & $(BINARY)
else
	cp testdata/$(CFG_FILE) bin/
	$(COPY_WEB_FOLDER_LINUX)
	cd bin; (./$(TEST_BINARY) &)
	cd bin; (./$(TEST_BINARY1) &)	
	cd bin; (./$(TEST_BINARY2) &)
	cd bin; ./$(BINARY)
endif
