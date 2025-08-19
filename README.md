# Todo List API

A simple RESTful Todo List API built with Go and deployable to Kubernetes.

## Features

- RESTful API endpoints for todo management
- Persistent storage using BoltDB
- Docker containerization
- Kubernetes deployment support
- Health check endpoints
- Configuration via ConfigMaps

## Prerequisites

- Go 1.25 or later
- Docker Desktop with Kubernetes enabled
- kubectl command-line tool

## Project Structure

```
todo-list/
├── main.go           # Main application code
├── go.mod           # Go module definition
├── go.sum           # Go module checksums
├── Dockerfile       # Docker build instructions
└── k8s/             # Kubernetes manifests
    ├── configmap.yaml
    ├── deployment.yaml
    ├── pvc.yaml
    └── service.yaml
```

## Local Development

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Run the application:
   ```bash
   go run main.go
   ```
   The server will start on port 8080.

## Docker Build and Run

1. Build the Docker image:
   ```bash
   docker build -t todo-app .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 todo-app
   ```

## Kubernetes Deployment

1. Start a local Docker registry:
   ```bash
   docker run -d -p 5000:5000 --restart=always --name registry registry:2
   ```

2. Tag and push the image to local registry:
   ```bash
   docker tag todo-app localhost:5000/todo-app
   docker push localhost:5000/todo-app
   ```

3. Apply Kubernetes manifests:
   ```bash
   kubectl apply -f k8s/
   ```

4. Verify the deployment:
   ```bash
   kubectl get pods
   kubectl get services
   ```

The application will be available at `http://localhost`

## API Endpoints

### GET /todos
List all todo items

### POST /todos
Create a new todo item
```json
{
    "title": "Task name",
    "completed": false
}
```

### PUT /todos/{id}
Update an existing todo item
```json
{
    "title": "Updated task",
    "completed": true
}
```

### DELETE /todos/{id}
Delete a todo item

### GET /health
Health check endpoint

## Environment Variables

- `PORT`: Server port (default: 8080)

## Persistence

The application uses BoltDB for data storage. In Kubernetes, the data is persisted using a PersistentVolumeClaim.

## Health Checks

The application includes readiness and liveness probes configured in the Kubernetes deployment. The `/health` endpoint is used to verify the application's health.
