#!/bin/bash

dir=$(dirname "$0")
grep version: $dir/../builder-config.yaml | head -n 1 | awk '{ print $2 }'
