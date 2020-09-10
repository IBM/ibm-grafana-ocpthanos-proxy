# Copyright 2020 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Specify whether this repo is build locally or not, default values is '1';
# If set to 1, then you need to also set 'DOCKER_USERNAME' and 'DOCKER_PASSWORD'
# environment variables before build the repo.
BUILD_LOCALLY ?= 1

# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the
# IMAGE_REPO, IMAGE_NAME and RELEASE_TAG environment variable.
IMAGE_REPO ?= "hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom"
IMAGE_NAME ?= grafana-ocpthanos-proxy

# Github host to use for checking the source tree;
# Override this variable ue with your own value if you're working on forked repo.
GIT_HOST ?= github.com/IBM

PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))

# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
VERSION ?= $(shell cat ./version/version.go | grep "Version =" | awk '{ print $$3}' | tr -d '"')

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
    TARGET_OS ?= linux
    XARGS_FLAGS="-r"
else ifeq ($(LOCAL_OS),Darwin)
    TARGET_OS ?= darwin
    XARGS_FLAGS=
else
    $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
endif

ARCH := $(shell uname -m)
LOCAL_ARCH := "amd64"
ifeq ($(ARCH),x86_64)
    LOCAL_ARCH="amd64"
else ifeq ($(ARCH),ppc64le)
    LOCAL_ARCH="ppc64le"
else ifeq ($(ARCH),s390x)
    LOCAL_ARCH="s390x"
else
    $(error "This system's ARCH $(ARCH) isn't recognized/supported")
endif

## Setup docker image build options
IMAGE_DISPLAY_NAME=IBM Grafana Thanos Proxy
IMAGE_MAINTAINER=dybo@cn.ibm.com
IMAGE_VENDOR=IBM
IMAGE_VERSION=$(VERSION)
IMAGE_DESCRIPTION=A simple proxy between Grafana and OCP thanos-querier service
IMAGE_SUMMARY=$(IMAGE_DESCRIPTION)
IMAGE_OPENSHIFT_TAGS=monitoring
$(eval WORKING_CHANGES := $(shell git status --porcelain))
$(eval BUILD_DATE := $(shell date +%m/%d@%H:%M:%S))
$(eval GIT_COMMIT := $(shell git rev-parse --short HEAD))
$(eval VCS_REF := $(GIT_COMMIT))
IMAGE_RELEASE=$(VERSION)
GIT_REMOTE_URL = $(shell git config --get remote.origin.url)
$(eval DOCKER_BUILD_OPTS := --build-arg "IMAGE_NAME=$(IMAGE_NAME)" --build-arg "IMAGE_DISPLAY_NAME=$(IMAGE_DISPLAY_NAME)" --build-arg "IMAGE_MAINTAINER=$(IMAGE_MAINTAINER)" --build-arg "IMAGE_VENDOR=$(IMAGE_VENDOR)" --build-arg "IMAGE_VERSION=$(IMAGE_VERSION)" --build-arg "IMAGE_RELEASE=$(IMAGE_RELEASE)" --build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" --build-arg "IMAGE_SUMMARY=$(IMAGE_SUMMARY)" --build-arg "IMAGE_OPENSHIFT_TAGS=$(IMAGE_OPENSHIFT_TAGS)" --build-arg "VCS_REF=$(VCS_REF)" --build-arg "VCS_URL=$(GIT_REMOTE_URL)")

#####
MARKDOWN_LINT_WHITELIST="http://thanos-proxy:9096,https://thanos-querier.openshift-monitoring.svc:9091"

all: fmt check test coverage build images

ifeq (,$(wildcard go.mod))
ifneq ("$(realpath $(DEST))", "$(realpath $(PWD))")
    $(error Please run 'make' from $(DEST). Current directory is $(PWD))
endif
endif

include common/Makefile.common.mk

############################################################
# format section
############################################################

# All available format: format-go format-python
# Default value will run all formats, override these make target with your requirements:
#    eg: fmt: format-go format-protos
fmt: format-go format-python

############################################################
# check section
############################################################

check: lint

# All available linters: lint-dockerfiles lint-scripts lint-yaml lint-copyright-banner lint-go lint-python lint-helm lint-markdown
# Default value will run all linters, override these make target with your requirements:
#    eg: lint: lint-go lint-yaml
# The MARKDOWN_LINT_WHITELIST variable can be set with comma separated urls you want to whitelist
lint: lint-all

############################################################
# test section
############################################################

test:
	@echo "Running the tests for $(IMAGE_NAME) on $(LOCAL_ARCH)..."
	@go test $(TESTARGS) ./...

############################################################
# coverage section
############################################################

coverage:
	@common/scripts/codecov.sh $(BUILD_LOCALLY)

############################################################
# build section
############################################################

build:
	@echo "Building the $(IMAGE_NAME) $(LOCAL_ARCH) binary..."
	@GOARCH=$(LOCAL_ARCH) common/scripts/gobuild.sh $(IMAGE_NAME) ./cmd

############################################################
# image section
############################################################

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
endif

build-image: build
	@echo "Building the $(IMAGE_NAME) docker image for $(LOCAL_ARCH)"
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION) $(DOCKER_BUILD_OPTS) --build-arg "IMAGE_NAME_ARCH=$(IMAGE_NAME)-amd64" -f Dockerfile .

build-push-image: $(CONFIG_DOCKER_TARGET) build-image
	@echo "Pushing the $(IMAGE_NAME) docker image for $(LOCAL_ARCH)"
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME)-$(LOCAL_ARCH):$(VERSION)

images: build-push-image multiarch-image

############################################################
# multiarch-image section
############################################################

multiarch-image: $(CONFIG_DOCKER_TARGET)
	@common/scripts/multiarch_image.sh $(IMAGE_REPO) $(IMAGE_NAME) $(VERSION)

.PHONY: all work fmt check coverage lint test build images build-push-image multiarch-image
