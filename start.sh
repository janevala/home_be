#!/bin/bash -x

if [ ! -f ".env" ]; then
    echo "Error: .env file not found"
    exit 1
fi

ENV=$(cat .env | grep ENV | cut -d '=' -f2)
ARC=$(uname -m)

if [ "$ARC" = "aarch64" ]; then
    if [ ! -f "home_be_arm64" ]; then
        /usr/bin/make ${ENV}
        /usr/bin/make dep
        ./home_be_arm64
    else
        ./home_be_arm64
    fi
fi

if [ "$ARC" = "x86_64" ]; then
    if [ ! -f "home_be_amd64" ]; then
        /usr/bin/make ${ENV}
        /usr/bin/make dep
        ./home_be_amd64
    else
        ./home_be_amd64
    fi
fi
