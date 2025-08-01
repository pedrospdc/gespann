# Gespann Network Monitoring Dashboard

This Grafana dashboard provides comprehensive visualization of network connection metrics captured by the gespann eBPF monitoring system.

## Dashboard Overview

The dashboard is organized into several sections that provide different perspectives on network activity:

### 1. Real-time Connection Activity
- **Connection Rate**: Shows the rate of new connections and closed connections per second
- **Currently Open Connections**: Gauge displaying the current number of active connections

### 2. Network Bandwidth Monitoring
- **Network Bandwidth**: Time series showing bytes sent and received per second
- Provides real-time visibility into network throughput

### 3. Connection Analysis
- **Connection Events Distribution**: Pie chart showing the breakdown of different event types (open, close, reset, etc.)
- **Protocol Distribution**: Pie chart showing TCP vs UDP connection distribution

### 4. Performance Metrics
- **Average Connection Duration**: Shows how long connections typically stay open (in seconds)
- **Average RTT**: Round-trip time performance metrics (in milliseconds)

### 5. Volume Statistics
- **Total Bytes Sent**: Cumulative bytes transmitted across all connections
- **Total Bytes Received**: Cumulative bytes received across all connections  
- **Total Connections Tracked**: Total number of connections monitored since startup

### 6. Event Rates
- **Connection Events Rate by Type**: Time series showing the rate of different connection events
- Useful for identifying patterns in connection behavior

### 7. Health Monitoring
- **Failed Connections**: Number of connection attempts that failed
- **Reset Connections**: Number of connections that were reset abnormally
- **Idle Connections**: Number of connections currently idle

## Key Metrics Explained

| Metric | Description | Normal Range |
|--------|-------------|--------------|
| Connection Rate | New connections per second | Depends on application load |
| Open Connections | Currently active connections | Should be reasonable for your workload |
| Bandwidth | Bytes per second transmitted/received | Application dependent |
| RTT | Round-trip time in milliseconds | < 100ms for local/fast networks |
| Duration | Average connection lifetime | Application dependent |
| Failed Connections | Should be 0 or very low | Close to 0 |
| Reset Connections | Abnormal connection terminations | Close to 0 |

## Using the Dashboard

### Time Range
- Default: Last 15 minutes with 5-second refresh
- Adjust the time range using the time picker in the top-right corner
- Use refresh intervals from 5s to 1m depending on your monitoring needs

### Filtering
- All panels automatically filter to show only `gespann_*` metrics
- Metrics are labeled with instance and job information for multi-instance deployments

### Alerting
You can set up alerts on key metrics such as:
- High connection failure rates
- Unusual bandwidth spikes
- High RTT values
- Excessive connection reset rates

## Accessing the Dashboard

1. **Start the monitoring stack:**
   ```bash
   docker-compose up -d
   ```

2. **Access Grafana:**
   - URL: http://localhost:3000
   - Username: `admin`
   - Password: `admin`

3. **Navigate to the dashboard:**
   - Go to "Dashboards" in the sidebar
   - Look for "Gespann eBPF Monitoring" folder
   - Click on "Gespann Network Monitoring"

## Troubleshooting

### No Data Showing
1. Check that gespann is running: `docker-compose ps`
2. Verify Prometheus is scraping: http://localhost:9090/targets
3. Generate test traffic: `./scripts/generate-traffic.sh`
4. Check if metrics are available: `./scripts/query_prometheus.sh`

### Metrics Not Updating
1. Verify the refresh interval is set appropriately
2. Check that network activity is generating new events
3. Ensure Prometheus scrape interval is reasonable (default: 5s)

### Performance Issues
1. Reduce the refresh rate if dashboard is slow
2. Adjust time range to show less historical data
3. Consider using longer scrape intervals for high-volume environments

## Customization

The dashboard JSON can be modified to:
- Add new panels for additional metrics
- Change visualization types (line graphs, bar charts, heatmaps)
- Adjust thresholds and color schemes
- Add custom alerts and annotations

## Dashboard File Location

- **Dashboard JSON**: `/monitoring/grafana/dashboards/gespann-network-monitoring.json`
- **Provisioning Config**: `/monitoring/grafana/provisioning/dashboards/dashboards.yml`
- **Datasource Config**: `/monitoring/grafana/provisioning/datasources/prometheus.yml`

The dashboard is automatically provisioned when the Grafana container starts, so no manual import is required.