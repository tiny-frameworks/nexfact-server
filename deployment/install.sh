podman build -t ghcr.io/tiny-frameworks/nexfact-server:latest .

podman run -d \
  --name nexfact-server \
  -p 8080:8080 \
  -v ./data/watch:/app/watch \
  -v ./data/processed:/app/processed \
  -v ./data/error:/app/error \
  -v ./data/env:/app/env \
  -v ./config.yaml:/app/config.yaml:ro \
  ghcr.io/tiny-frameworks/nexfact-server:latest
