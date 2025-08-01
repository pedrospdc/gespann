#!/bin/bash

set -e

echo "🚀 Testing gespann locally..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "❌ This script must be run as root for eBPF access"
   echo "Run: sudo ./scripts/test-local.sh"
   exit 1
fi

# Build if needed
if [[ ! -f "bin/gespann" ]]; then
    echo "📦 Building gespann..."
    make build
fi

# Start gespann in background
echo "🔄 Starting gespann..."
./bin/gespann -config config.yaml &
GESPANN_PID=$!

# Wait for startup
sleep 3

# Check if process is running
if ! kill -0 $GESPANN_PID 2>/dev/null; then
    echo "❌ gespann failed to start"
    exit 1
fi

echo "✅ gespann started (PID: $GESPANN_PID)"

# Test metrics endpoint
echo "🔍 Testing metrics endpoint..."
if curl -s http://localhost:8080/metrics | grep -q "gespann"; then
    echo "✅ Metrics endpoint working"
else
    echo "❌ Metrics endpoint not responding"
    kill $GESPANN_PID
    exit 1
fi

# Generate some traffic
echo "🌐 Generating network traffic..."
./scripts/generate-traffic.sh &
TRAFFIC_PID=$!

# Wait a bit for metrics to be collected
sleep 10

# Check metrics again
echo "📊 Checking for connection metrics..."
METRICS=$(curl -s http://localhost:8080/metrics)

if echo "$METRICS" | grep -q "gespann_connection_events_total"; then
    echo "✅ Connection events detected!"
    echo "📈 Metrics sample:"
    echo "$METRICS" | grep "gespann_" | head -5
else
    echo "⚠️  No connection events yet (this might be normal for short tests)"
fi

# Cleanup
echo "🧹 Cleaning up..."
kill $GESPANN_PID 2>/dev/null || true
kill $TRAFFIC_PID 2>/dev/null || true

echo "✅ Local test complete!"
echo ""
echo "Next steps:"
echo "1. Check full metrics: curl http://localhost:8080/metrics"
echo "2. Use docker-compose for visualization: make dev-up"
echo "3. Access Grafana: http://localhost:3000 (admin/admin)"