APP := telegram-dsm-bot
BIN := ./bin/$(APP)
PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin
CONFIG_DIR ?= /etc/lifeline
SERVICE_FILE := configs/lifeline.service

.PHONY: build test fmt run install install-service clean

build: fmt
	GO111MODULE=on go build -o $(BIN) ./cmd/bot

test:
	go test ./...

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

run: build
	CONFIG_PATH=configs/config.yaml $(BIN)

install: build
	install -d $(BINDIR)
	install -m 0755 $(BIN) $(BINDIR)/$(APP)
	install -d $(CONFIG_DIR)
	[ -f $(CONFIG_DIR)/config.yaml ] || install -m 0640 configs/config.yaml $(CONFIG_DIR)/config.yaml

install-service: install
	install -m 0644 $(SERVICE_FILE) /etc/systemd/system/lifeline.service
	systemctl daemon-reload
	systemctl enable --now lifeline

clean:
	rm -rf ./bin
