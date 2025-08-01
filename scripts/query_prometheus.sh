#!/bin/bash

# Prometheus metrics extraction script for gespann
# Queries all gespann metrics and formats them for analysis

PROMETHEUS_URL="http://localhost:9090"
GESPANN_METRICS_URL="http://localhost:8081"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

echo "========================================"
echo "GESPANN PROMETHEUS METRICS REPORT"
echo "Generated at: $TIMESTAMP"
echo "========================================"

# Function to query Prometheus and format output
query_metric() {
    local metric_name="$1"
    local description="$2"
    
    echo ""
    echo "--- $description ---"
    
    # Query the metric
    result=$(curl -s "${PROMETHEUS_URL}/api/v1/query?query=${metric_name}" | \
        python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if data['status'] == 'success' and data['data']['result']:
        for item in data['data']['result']:
            labels = item.get('metric', {})
            value = item['value'][1]
            if labels:
                label_str = ', '.join([f'{k}={v}' for k, v in labels.items() if k != '__name__'])
                print(f'{value} ({label_str})')
            else:
                print(f'{value}')
    else:
        print('No data available')
except:
    print('Error querying metric')
")
    
    echo "$result"
}

# Function to get all gespann metrics at once
get_all_metrics() {
    echo ""
    echo "--- ALL GESPANN METRICS (Raw) ---"
    curl -s "${PROMETHEUS_URL}/api/v1/label/__name__/values" | \
        python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    gespann_metrics = [m for m in data['data'] if m.startswith('gespann_')]
    if gespann_metrics:
        print('Available gespann metrics:')
        for metric in sorted(gespann_metrics):
            print(f'  - {metric}')
    else:
        print('No gespann metrics found')
except:
    print('Error retrieving metrics list')
"
}

# Check if Prometheus is accessible
echo "Checking Prometheus connectivity..."
if ! curl -s "${PROMETHEUS_URL}/api/v1/query?query=up" > /dev/null; then
    echo "❌ ERROR: Cannot connect to Prometheus at $PROMETHEUS_URL"
    echo "Make sure docker-compose is running: docker-compose ps"
    exit 1
fi
echo "✅ Prometheus is accessible"

# Also check direct gespann metrics (bypassing Prometheus)
echo ""
echo "--- DIRECT GESPANN METRICS (Current State) ---"
if docker exec gespann-gespann-1 wget -q -O - http://localhost:8081/metrics 2>/dev/null | grep -q "gespann_"; then
    echo "✅ Direct metrics endpoint accessible"
    echo ""
    echo "Current metrics from gespann:"
    docker exec gespann-gespann-1 wget -q -O - http://localhost:8081/metrics 2>/dev/null | grep "gespann_" | grep -v "^#" | head -20
else
    echo "❌ Cannot access direct metrics endpoint"
fi

# Get list of all available gespann metrics
get_all_metrics

# Query specific gespann metrics with descriptions
query_metric "gespann_total_connections" "Total Connections"
query_metric "gespann_open_connections" "Currently Open Connections"
query_metric "gespann_closed_connections_total" "Total Closed Connections"
query_metric "gespann_idle_connections" "Idle Connections"
query_metric "gespann_reset_connections_total" "Reset Connections"
query_metric "gespann_failed_connections_total" "Failed Connections"

query_metric "gespann_bytes_sent_total" "Total Bytes Sent"
query_metric "gespann_bytes_received_total" "Total Bytes Received"
query_metric "gespann_avg_connection_duration_ms" "Average Connection Duration (ms)"
query_metric "gespann_avg_rtt_microseconds" "Average RTT (microseconds)"

query_metric "gespann_tcp_connections_total" "TCP Connections"
query_metric "gespann_udp_connections_total" "UDP Connections"

query_metric "gespann_connection_events_total" "Connection Events by Type"
query_metric "gespann_connection_bandwidth_bytes_total" "Connection Bandwidth by Direction"

echo ""
echo "========================================"
echo "SUMMARY"
echo "========================================"

# Calculate some summary stats
curl -s "${PROMETHEUS_URL}/api/v1/query?query=gespann_connection_events_total" | \
python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if data['status'] == 'success' and data['data']['result']:
        total_events = 0
        event_types = {}
        for item in data['data']['result']:
            value = float(item['value'][1])
            total_events += value
            event_type = item['metric'].get('event_type', 'unknown')
            event_types[event_type] = event_types.get(event_type, 0) + value
        
        print(f'Total network events captured: {int(total_events)}')
        print('Events by type:')
        for event_type, count in event_types.items():
            print(f'  - {event_type}: {int(count)}')
    else:
        print('No connection events data available')
        print('This might indicate:')
        print('  1. eBPF program is not capturing events')
        print('  2. No network activity has occurred')
        print('  3. Metrics are not being exported properly')
except Exception as e:
    print('Error processing connection events data')
"

echo ""
echo "Next steps:"
echo "1. Check Grafana dashboard: http://localhost:3000 (admin/admin)"
echo "2. Generate more traffic: ./scripts/generate-traffic.sh"
echo "3. Check gespann logs: docker-compose logs gespann"
echo "4. View raw metrics: curl http://localhost:9090/api/v1/label/__name__/values"