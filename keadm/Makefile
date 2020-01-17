GO_LDFLAGS=-ldflags '$(shell ../hack/lib/version.sh)'
KEADM_BINARY=keadm
BINDIR=/usr/local/bin

# make keadm
.PHONY: default keadm

default: keadm

keadm: verify 
	go build $(GO_LDFLAGS) -o ${KEADM_BINARY} keadm.go

.PHONY: verify
verify:
	bash ../hack/verify-golang.sh

lint:
	golangci-lint run --deadline '10m' --disable-all -E golint ./...
	go vet ./...

.PHONY: install
install: default
	install ${KEADM_BINARY} ${BINDIR}

.PHONY: uninstall
uninstall:
	rm -f ${BINDIR}/${KEADM_BINARY}

.PHONY: clean
clean:
	rm -f ${KEADM_BINARY}

