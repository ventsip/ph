.PHONY: clean build test build_test run

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY=ph
TEST_BINARY=test_process
TEST_BINARY1=test_process1
TEST_BINARY2=test_process2
BALANCE_JSON=balance.json

clean:
ifeq ($(OS), Windows_NT)
	del /Q bin\*
	del /Q testdata\$(BALANCE_JSON)
else
	rm bin/*
	rm bin/$(BALANCE_JSON)
endif


build: 
ifeq ($(OS), Windows_NT)
	cd main & $(GOBUILD) -v -o ..\bin\$(BINARY).exe
else
	cd main; $(GOBUILD) -v -o ../bin/$(BINARY)
endif

build_test: 
ifeq ($(OS), Windows_NT)
	cd test_process & $(GOBUILD) -v -o ..\bin\$(TEST_BINARY).exe
	copy bin\$(TEST_BINARY).exe bin\$(TEST_BINARY1).exe
	copy bin\$(TEST_BINARY).exe bin\$(TEST_BINARY2).exe
else
	cd test_process; $(GOBUILD) -v -o ../bin/$(TEST_BINARY)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY1)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY2)
endif

test: clean build_test
ifeq ($(OS), Windows_NT)
	cd lib & $(GOTEST) -v
else
	cd lib ; $(GOTEST) -v
endif

run: clean build build_test
ifeq ($(OS), Windows_NT)
	cd bin & start $(TEST_BINARY)
	cd bin & start $(TEST_BINARY1)	
	cd bin & start $(TEST_BINARY2)
	cd bin & $(BINARY)
else
	cd bin; (./$(TEST_BINARY) &)
	cd bin; (./$(TEST_BINARY1) &)	
	cd bin; (./$(TEST_BINARY2) &)
	cd bin; ./$(BINARY)
endif
