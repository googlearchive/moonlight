# Copyright 2016 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PRJ       ?= google.com:moonlight
IMG       ?= moonlight
IMG_GCR   ?= gcr.io/google.com/moonlight/server

# GCS bucket of where the latest compiled headless_shell binary files
# can be found
MOONLIGHT_BUCKET = moonlight-files
HEADLESS_TAR = headless-shell.tar.gz

moonlight: .moonlight
.moonlight: Dockerfile bin/server bin/headless-shell.tar.gz
	docker build --rm -t $(IMG) .
	touch .moonlight

ARGS ?=
run: .moonlight
	docker run -ti --rm -p 8080:8080 $(IMG) $(ARGS)

deploy: .moonlight
	docker tag $(IMG) $(IMG_GCR)
	gcloud docker -- push $(IMG_GCR)
	gcloud app deploy --project=$(PRJ) server/app.yaml --image-url=$(IMG_GCR)

bin/server: $(wildcard server/*.go)
	GOOS=linux GOARCH=amd64 go build -o $@ $^

bin/headless-shell.tar.gz:
	curl -sSL -o $@ https://storage.googleapis.com/$(MOONLIGHT_BUCKET)/$(HEADLESS_TAR)

# The headless_shell is compiled with the following args.gn:
#
#     import("//build/args/headless.gn")
#     use_goma = true
#     is_debug = false
#     enable_nacl = false
#     remove_webcore_debug_symbols = true
#
# build command:
#
#     ninja -C out/Release -j 100 headless_shell
#
HEADLESS_OUT   ?= $(HOME)/chr/chromium/src/out/Release
HEADLESS_FILES  = headless_shell \
	headless_lib.pak \
	libtracing_library.so \
	libui_library.so \
	libosmesa.so \
	locales \
	mus_app_resources_100.pak \
	mus_app_resources_200.pak \
	mus_app_resources_strings.pak \
	natives_blob.bin \
	snapshot_blob.bin
bin/.headless-shell.tar.gz: $(addprefix $(HEADLESS_OUT)/,$(HEADLESS_FILES))
	tar -czf $@ --owner=0 --group=0 -C $(HEADLESS_OUT) $(HEADLESS_FILES)
# upload bin/.headless-shell.tar.gz to GCS for others to use
.up-headless-shell-tar: bin/.headless-shell.tar.gz
	gsutil cp $< gs://$(MOONLIGHT_BUCKET)/$(HEADLESS_TAR)
	gsutil acl ch -g all:R gs://$(MOONLIGHT_BUCKET)/$(HEADLESS_TAR)
