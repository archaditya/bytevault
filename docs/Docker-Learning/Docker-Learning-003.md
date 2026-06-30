# Docker Learning 003

## Docker Compose

1. Create a `docker-compose.yml` in the root directory of the project.

```yaml
version: '3.8'
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: win-win-api-api
    ports:
      - "3000:3000"
    volumes:
      - ./:/app
      - /app/node_modules
    environment:
      - NODE_ENV=development
    networks:
      - win-win-api-network
  worker:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: win-win-api-worker
    volumes:
      - ./:/app
      - /app/node_modules
    environment:
      - NODE_ENV=development
    networks:
      - win-win-api-network
  scheduler:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: win-win-api-scheduler
    volumes:
      - ./:/app
      - /app/node_modules
    environment:
      - NODE_ENV=development
    networks:
      - win-win-api-network

networks:
  win-win-api-network:
    driver: bridge
```

3. Run the application.

```bash
docker-compose up --build
```

This command will build the images and start the containers.