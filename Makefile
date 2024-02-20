# Copyright 2017 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# Makefile for building a simple game-server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#
GIT_DESCRIBE = $(shell git describe --tags --always --dirty=d)
REPOSITORY ?=
PROD_REPO ?= echotools

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
ifeq ($(REPOSITORY),)
	server_tag := evr-game-wrapper:latest
else
	server_tag := $(REPOSITORY)/evr-game-wrapper:latest
endif


server_tag_linux_amd64 = $(server_tag)-amd64

push_server_manifest = $(server_tag_linux_amd64)
root_path = $(realpath $(project_path))

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

build: build-linux-image-amd64

# Builds all image artifacts and create a docker manifest that is used to inform the CRI (Docker or containerd usually)
# which image is the best fit for the host. See https://www.docker.com/blog/multi-arch-images/ for details.
push: push-linux-image-amd64

# Pushes all variants of the Windows images to the container image registry.
push-linux-image-amd64: build
	docker push $(server_tag_linux_amd64)

# Build a docker image for the server, and tag it
build-linux-image-amd64:
	cd $(root_path) && docker build -f $(project_path)Dockerfile --build-arg GIT_DESCRIBE=$(GIT_DESCRIBE) --tag=evr-game-wrapper:latest --tag=$(server_tag_linux_amd64) .

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(PROD_REPO)/$(server_tag)

#output the tag for this image
echo-image-tag:
	@echo $(PROD_REPO)/$(server_tag)

# build and push the evr-game-wrapper image with specified tag
cloud-build:
	cd $(root_path) && gcloud builds submit --config=evr-game-wrapper/cloudbuild.yaml
