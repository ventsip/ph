.PHONY: build test build_test run

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY=ph
TEST_BINARY=test_process
TEST_BINARY1=test_process1
TEST_BINARY2=test_process2


build:
	cd main; $(GOBUILD) -v -o ../bin/$(BINARY)

test:
	cd lib; $(GOTEST) -v

build_test:
	cd test_process; $(GOBUILD) -v -o ../bin/$(TEST_BINARY)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY1)
	cp bin/$(TEST_BINARY) bin/$(TEST_BINARY2)

run: build build_test
	cd bin; (./$(TEST_BINARY) &)
	cd bin; (./$(TEST_BINARY1) &)	
	cd bin; (./$(TEST_BINARY2) &)
	cd bin; ./$(BINARY)
