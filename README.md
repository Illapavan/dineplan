# Simple REST API Framework in Go

This project implements a lightweight REST API framework in Go, demonstrating basic routing, request handling, and response management.

## Features

- Simple routing system supporting All HTTP methods
- JSON request body parsing
- Customizable response handling with support for JSON responses
- Basic in-memory user management

## Getting Started

### Prerequisites

- Go (version 1.16 or later recommended)

### Running the Server

To start the server, run the following command in the project root directory:

```bash
go run main.go restFramework.go
```

## API END POINTS

Get All Users

retrives all users in the system.

```bash
curl --location 'localhost:8000/get-all-users'
```

Get User by ID

Retrives a specific user by their ID.

```bash
curl --location 'localhost:8000/get-user/1'
```

Add New student:

adds a new student to the system

```bash
curl --location 'localhost:8000/add-student' \
--header 'Content-Type: application/json' \
--data '{
    "name": "Teja Illa",
    "phone": "98998899090",
    "age": 30
}'
```