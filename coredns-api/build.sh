#!/bin/bash

# Build CoreDNS with API plugin
echo "Building CoreDNS with API plugin..."

# Create necessary directories
mkdir -p build

# First, let's test if our plugin compiles correctly
echo "Testing plugin compilation..."
go build -buildmode=plugin -o build/api.so plugin-1.go

if [ $? -eq 0 ]; then
    echo "Plugin compiled successfully!"
    echo ""
    echo "To use this plugin with CoreDNS:"
    echo "1. Copy api.so to your CoreDNS plugins directory"
    echo "2. Add 'api' directive to your Corefile"
    echo "3. Start CoreDNS normally"
    echo ""
    echo "Example Corefile configuration:"
    cat Corefile
else
    echo "Plugin compilation failed!"
    exit 1
fi