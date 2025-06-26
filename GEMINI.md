
# Kyotaidoshin Gemini Assistant Configuration

This file provides context to the Gemini code assistant, helping it understand the project structure, conventions, and key commands.

## Project Overview

This is a monorepo for the Kyotaidoshin application, a web app for managing building expenses and related tasks. The project is built with a combination of technologies, including:

-   **Frontend:** Vite, Alpine.js, htmx, Tailwind CSS
-   **Backend:** Serverless functions (Node.js and Go)
-   **Infrastructure:** SST (Serverless Stack)
-   **Database:** Turso

## Monorepo Structure

The project is organized as a monorepo with the following packages:

-   `packages/frontend`: The main web application.
-   `packages/backend`: Contains the backend services, including:
    -   `kyo-repo`: A Go application that provides the core API.
    -   `html-to-pdf`: A serverless function for converting HTML to PDF.
    -   `process-bcv-file-v2`: A serverless function for processing BCV files.
-   `packages/auth`: Handles authentication and authorization.

## Key Commands

### Frontend

-   `npm run dev`: Starts the development server.
-   `npm run build`: Builds the frontend for production.
-   `npm run preview`: Previews the production build.

### Backend (Go)

The Go application in `packages/backend/kyo-repo` is the core of the backend. To run it, you'll need to have Go installed.

-   `go run ./cmd/app`: Starts the main application.
-   `go test ./...`: Runs the tests.

### SST (Serverless Stack)

SST is used to deploy the serverless application.

-   `sst dev`: Starts the SST development environment.
-   `sst deploy`: Deploys the application to production.

## Development Conventions

-   **Code Style:** Please maintain the existing code style.
-   **Commit Messages:** Follow the conventional commit format.
-   **Dependencies:** Use `bun` to manage dependencies.
