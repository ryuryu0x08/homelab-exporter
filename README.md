# homelab-exporter

`homelab-exporter` is the single Prometheus scrape gateway for a homelab host. It concurrently scrapes local Prometheus endpoints, merges their metric families, and adds a `homelab_source` label to every upstream sample so common runtime metrics remain distinguishable.

Only Windows is supported in the current release. Platform validation is explicit so future implementations can be added without leaking platform concerns into aggregation or HTTP layers.

## Behavior

- Required source failures return HTTP 503 while retaining diagnostic metrics in the response.
- Optional source failures return HTTP 200 and do not suppress healthy sources. This is intended for on-demand services such as `llama-server`.
- `homelab_exporter_source_up{source="..."}` reports whether each source was scraped, parsed, and merged successfully.
- `homelab_exporter_source_scrape_duration_seconds{source="..."}` reports local scrape duration.
- A metric-family type or help conflict fails that source instead of silently producing invalid exposition.
- On Windows, NVIDIA GPU metrics are collected directly through `nvidia-smi`; no separate NVIDIA exporter process is required.
- `/healthz` reports gateway process health; `/metrics` performs live fan-out.

## Run

The CLI follows the domain/resource/action convention:

```powershell
.\bin\homelab-exporter.exe services exporter serve --config .\config.windows.toml
```

The sample configuration listens on port `9836`. Configure Prometheus to scrape only `http://<host>:9836/metrics`; upstream ports remain local implementation details.

## Test and build

```powershell
go test ./...
go vet ./...
go build -o bin\homelab-exporter.exe ./cmd/homelab-exporter
```
