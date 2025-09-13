# Local Monitoring Stack with Docker Compose

This setup provides a complete, self-contained monitoring stack for a local machine
using Prometheus, Grafana, and Node Exporter, all managed via Docker Compose.


## How to Run

1.  Ensure Docker and Docker Compose are installed.
2.  Clone the repository or place both `docker-compose.yml` and `prometheus.yml` in the same directory.
3.  From that directory, start the entire stack in the background:
   
      ```bash
      docker-compose up -d
      ```

Once running, the services will be available at:

- **Grafana:** `http://localhost:3000` (Login: `admin` / `admin`)
- **Prometheus:** `http://localhost:9091`
- **Node Exporter:** `http://localhost:9100`


## Key Configuration Notes

* **Host Networking:** All services are configured with `network_mode: "host"`. This means they share the host machine's network directly, simplifying access via `localhost`.
* **Prometheus Port:** The Prometheus server is configured to listen on port `9091` to avoid conflicts with other development tools that often use the default port `9090`.
* **Grafana Data Source:** On the first run, you must manually configure the Prometheus data source in Grafana. Go to **Connections > Data Sources > Prometheus** and set the URL to `http://localhost:9091`.
* **Auto-Restart:** The `node-exporter` service is configured with `restart: unless-stopped`. This ensures that it will automatically start again after a system reboot.


## Troubleshooting

### Forgotten Grafana Password

If you change the Grafana admin password and forget it, you can reset it by running

```bash
docker exec grafana grafana-cli admin reset-admin-password <NEW_PASSWORD>
```
