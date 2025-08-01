#!/bin/bash

echo "Generating network traffic to test gespann..."

# Function to make HTTP requests
generate_http_traffic() {
    echo "Making HTTP requests..."
    for i in {1..10}; do
        curl -s http://httpbin.org/get > /dev/null &
        curl -s https://api.github.com > /dev/null &
        sleep 0.5
    done
    wait
}

# Function to create TCP connections
generate_tcp_traffic() {
    echo "Creating TCP connections..."
    for i in {1..5}; do
        nc -z google.com 80 &
        nc -z github.com 443 &
        sleep 1
    done
    wait
}

# Function to create persistent connections
generate_persistent_connections() {
    echo "Creating persistent connections..."
    # Keep some connections open for a while
    for i in {1..3}; do
        (sleep 30 | nc google.com 80) &
    done
}

echo "Starting traffic generation..."
generate_http_traffic
generate_tcp_traffic
generate_persistent_connections

echo "Traffic generation complete. Check your metrics!"
echo "Prometheus: http://localhost:9090"
echo "Grafana: http://localhost:3000 (admin/admin)"