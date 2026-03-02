#!/bin/bash

# Test script for CoreDNS API plugin

echo "Testing CoreDNS API plugin..."

# Wait for CoreDNS to start (if running)
sleep 2

# Test endpoints
BASE_URL="http://localhost:8080"

# Test getting domains
echo "1. Testing GET /domains..."
curl -s $BASE_URL/domains | jq .

echo ""
echo "2. Testing GET /domains/example.com/records..."
curl -s $BASE_URL/domains/example.com/records | jq .

echo ""
echo "3. Testing POST /domains/example.com/records..."
curl -s -X POST $BASE_URL/domains/example.com/records \
    -H "Content-Type: application/json" \
    -d '{
        "name": "test",
        "type": "A", 
        "value": "192.168.1.200",
        "ttl": 3600
    }' | jq .

echo ""
echo "API test completed!"