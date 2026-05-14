#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Building..."
go build -o ewallet .

echo "Starting ewallet..."
./ewallet