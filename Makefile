.PHONY: build run

# Variables
BIN_DIR := ./bin
HTTP_SERVER_BIN := $(BIN_DIR)/http-server
DEBUG_BIN := $(BIN_DIR)/debug
SERVICE_NAME := low-level-server

# VM deployment variables
VM_USER ?= aoi
VM_IP ?= 192.168.1.30
VM_PATH ?= /home/$(VM_USER)/low-level-server
VM_PORT ?= 8080

# Build variables
GO := go
GOFLAGS := -v
LDFLAGS := -s -w

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

## Build Commands

build:
	@echo "$(GREEN)Build$(NC)"
	GOOS=linux GOARCH=amd64 go build -o ./bin ./cmd/debug

run:
	@echo "$(GREEN)Run$(NC)"
	scp $(BIN_DIR)/debug mina-ubuntu-server-00:$(VM_PATH)/bin/debug
	ssh mina-ubuntu-server-00 "$(VM_PATH)/bin/debug"
