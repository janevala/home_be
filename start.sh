#!/bin/bash

# Build and run ARM for Raspberry Pi

if [ ! -f "home_be_arm64" ]; then
    /usr/bin/make production_build_and_run
else
    ./home_be_arm64
fi
