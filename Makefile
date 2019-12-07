.PHONY: clean build test build_test run

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY=ph
VERSION=$(shell git describe --long --always --dirty)
CFG_FILE=cfg.json
TEST_BINARY=test_process
TEST_BINARY1=test_process1
TEST_BINARY2=test_process2
BALANCE_JSON=balance.json

clean:
ifeq ($(OS), Windows_NT)
	del /Q bin\*
else
	rm -f bin/*
endif


build: 
ifeq ($(OS), Windows_NT)
	cd cmd & $(GOBUILD) -v -o ..\bin\$(BINARY).exe -ldflags="-X main.version=$(VERSION)"
	copy testdata\$(CFG_FILE) bin
else
	cd cmd; $(GOBUILD) -v -o ../bin/$(BINARY) -ldflags="-X main.version=$(VERSION)"
	cp testdata/$(CFG_FILE) bin
endif

build_test: 
ifeq ($(OS), Windows_NT)
	cd test_process & $(GOBUILD) -v -o ..\bin\$(TEST_BINARY).exe
	copy bin\$(TEST_BINARY).exe bin\$(TEST_BINARY1).exe
	copy bin\$(TEST_BINARY).exe bin\$(TEST_BINARY2).exe
	copy testdata\$(CFG_FILE) bin
else
	cd test_process; $(GOBUILD) -v -o ../bin/$(TEST_BINARY)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY1)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY2)
	cp testdata/$(CFG_FILE) bin
endif

test: clean build_test
ifeq ($(OS), Windows_NT)
	copy testdata\$(CFG_FILE) bin
	cd engine & $(GOTEST) -v
else
	cp testdata/$(CFG_FILE) bin
	cd engine ; $(GOTEST) -v
endif

run: clean build build_test
ifeq ($(OS), Windows_NT)
	copy testdata\$(CFG_FILE) bin\ 
	cd bin & start $(TEST_BINARY)
	cd bin & start $(TEST_BINARY1)	
	cd bin & start $(TEST_BINARY2)
	cd bin & $(BINARY)
else
	cp testdata/$(CFG_FILE) bin/
	cd bin; (./$(TEST_BINARY) &)
	cd bin; (./$(TEST_BINARY1) &)	
	cd bin; (./$(TEST_BINARY2) &)
	cd bin; ./$(BINARY)
endif
