#!/bin/bash

echo "Integrating API plugin with CoreDNS..."

# Check if CoreDNS directory exists
if [ ! -d "coredns" ]; then
    echo "Error: CoreDNS directory not found. Please clone it first."
    exit 1
fi

cd coredns

# Ensure plugin files are in place
if [ ! -f "plugin/api/setup.go" ]; then
    echo "Error: setup.go not found in plugin/api directory"
    exit 1
fi

if [ ! -f "plugin/api/api.go" ]; then
    echo "Error: api.go not found in plugin/api directory"
    exit 1
fi

# Update plugin.cfg to include our API plugin
echo "api:github.com/coredns/coredns/plugin/api" >> plugin.cfg

# Build CoreDNS with our plugin
echo "Building CoreDNS with integrated API plugin..."
go build -o ../build/coredns-with-api

if [ $? -eq 0 ]; then
    echo "Integration successful!"
    echo "Built binary: build/coredns-with-api"
    echo ""
    echo "To run:"
    echo "  ./build/coredns-with-api -conf ../Corefile"
else
    echo "Build failed!"
    exit 1
fi