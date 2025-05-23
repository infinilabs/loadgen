SHELL=/bin/bash

# APP info
APP_NAME := loadgen
APP_VERSION := 1.0.0_SNAPSHOT
APP_CONFIG := $(APP_NAME).yml $(APP_NAME).dsl
APP_EOLDate ?= "2025-12-31T10:10:10Z"
APP_STATIC_FOLDER := .public
APP_STATIC_PACKAGE := public
APP_UI_FOLDER := ui
APP_PLUGIN_FOLDER := proxy
GOMODULE := false

include ../framework/Makefile
