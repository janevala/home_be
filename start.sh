#!/bin/bash

ARC=$(uname -m)

if [ "$ARC" = "aarch64" ]; then
    if [ ! -f "home_be_arm64" ]; then
        /usr/bin/make clean
        /usr/bin/make release
        /usr/bin/make dep
        /usr/bin/make run
    else
        ./home_be_arm64
    fi
fi

if [ "$ARC" = "x86_64" ]; then
    if [ ! -f "home_be_amd64" ]; then
        /usr/bin/make clean
        /usr/bin/make release
        /usr/bin/make dep
        /usr/bin/make run
    else
        ./home_be_amd64
    fi
fi
