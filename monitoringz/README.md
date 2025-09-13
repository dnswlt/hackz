# Local Monitoring Stack with Docker Compose

This setup provides a complete, self-contained monitoring stack for a local machine
using Prometheus, Grafana, and Node Exporter, all managed via Docker Compose.


## How to Run

1.  Ensure Docker and Docker Compose are installed.
2.  Clone the repository or place both `docker-compose.yml` and `prometheus.yml` in the same directory.
3.  From that directory, start the entire stack in the background:
   
      ```bash
      docker compose up -d
      ```

Once running, the services will be available at:

- **Grafana:** `http://localhost:3000` (Login: `admin` / `admin`)
- **Prometheus:** `http://localhost:9091`
- **Node Exporter:** `http://localhost:9100`


## Initial Grafana Setup

On the first run, you need to configure Grafana to connect to Prometheus and import a dashboard to visualize the data.

### 1. Configure the Prometheus Data Source

1.  Open Grafana in your browser: `http://localhost:3000`.
2.  Log in with the credentials `admin` / `admin`.
3.  Navigate to **Connections** > **Data Sources** > **Add data source**.
4.  Select **Prometheus**.
5.  Set the "Prometheus server URL" to `http://localhost:9091`.
6.  Click **Save & Test**. You should see a green success message.

### 2. Import the "Node Exporter Full" Dashboard

1.  In the Grafana UI, go to **Dashboards** > **New** > **Import**.
2.  In the "Import via grafana.com" field, enter the ID **`1860`**.
3.  Click **Load**.
4.  On the next screen, ensure your Prometheus data source is selected at the bottom.
5.  Click **Import**.

You will now have a complete and detailed dashboard monitoring your machine's metrics.

## Key Configuration Notes

* **Host Networking:** All services are configured with `network_mode: "host"`. This means they share the host machine's network directly, simplifying access via `localhost`.
* **Prometheus Port:** The Prometheus server is configured to listen on port `9091` to avoid conflicts with other development tools that often use the default port `9090`.
* **Auto-Restart:** The `node-exporter` service is configured with `restart: unless-stopped`. This ensures that it will automatically start again after a system reboot.


## Troubleshooting

### Forgotten Grafana Password

If you change the Grafana admin password and forget it, you can reset it by running

```bash
docker exec grafana grafana-cli admin reset-admin-password <NEW_PASSWORD>
```
