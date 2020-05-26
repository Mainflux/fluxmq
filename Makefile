# Copyright (c) Mainflux
# SPDX-License-Identifier: Apache-2.0

PROGRAM = fluxmq
SOURCES = $(wildcard *.go) cmd/main.go

all: $(PROGRAM)

.PHONY: all clean $(PROGRAM)

$(PROGRAM): $(SOURCES)
	go build -ldflags "-s -w" -o $@ cmd/main.go

clean:
	rm -rf $(PROGRAM)
