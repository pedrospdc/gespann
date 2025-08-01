#!/bin/bash

# Script to open the Gespann Network Monitoring dashboard

echo "🚀 Opening Gespann Network Monitoring Dashboard..."
echo ""
echo "Dashboard Access Information:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Grafana Dashboard: http://localhost:3000/d/gespann-network-monitoring"
echo "🔐 Username: admin"
echo "🔐 Password: admin"
echo ""
echo "Other Monitoring URLs:"
echo "📈 Prometheus: http://localhost:9090"
echo "🔧 gespann Metrics: http://localhost:8081/metrics"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if services are running
echo "🔍 Checking service status..."
if docker-compose ps | grep -q "Up"; then
    echo "✅ Services are running"
    
    # Check if we can access Grafana
    if curl -s http://localhost:3000/api/health > /dev/null 2>&1; then
        echo "✅ Grafana is accessible"
        
        # Check if Prometheus is accessible
        if curl -s http://localhost:9090/api/v1/targets > /dev/null 2>&1; then
            echo "✅ Prometheus is accessible"
            
            # Check if gespann metrics are available
            if curl -s http://localhost:8081/metrics | grep -q "gespann_" 2>/dev/null; then
                echo "✅ gespann metrics are available"
            else
                echo "⚠️  gespann metrics endpoint not responding"
            fi
        else
            echo "❌ Prometheus is not accessible"
        fi
    else
        echo "❌ Grafana is not accessible"
    fi
else
    echo "❌ Services are not running. Please start with: docker-compose up -d"
    exit 1
fi

echo ""
echo "🎯 Direct Dashboard Link:"
echo "   http://localhost:3000/d/gespann-network-monitoring/gespann-network-monitoring"
echo ""
echo "💡 Tips:"
echo "   • Refresh rate: 5 seconds (adjustable in top-right corner)"
echo "   • Time range: Last 15 minutes (adjustable in top-right corner)"
echo "   • Generate test traffic: ./scripts/generate-traffic.sh"
echo "   • View metrics data: ./scripts/query_prometheus.sh"
echo ""

# Attempt to open the dashboard in the default browser (macOS/Linux)
if command -v open > /dev/null 2>&1; then
    echo "🌐 Opening dashboard in your default browser..."
    open "http://localhost:3000/d/gespann-network-monitoring/gespann-network-monitoring"
elif command -v xdg-open > /dev/null 2>&1; then
    echo "🌐 Opening dashboard in your default browser..."
    xdg-open "http://localhost:3000/d/gespann-network-monitoring/gespann-network-monitoring"
else
    echo "📋 Please manually open the URL above in your browser"
fi