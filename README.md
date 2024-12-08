# Claude Proxy Server

This is a Go-based desktop application that provides a proxy server for the Anthropic Claude API with API key rotation on rate limit.

## Features

- Proxy server for the Anthropic Claude API
- API key rotation on rate limit

## Prerequisites

- Go 1.16 or later

## Installation

1. Clone the repository:

   ```sh
   git clone https://github.com/lamtran2601/claude-proxy.git
   cd claude-proxy
   ```

2. Set your API keys in an environment variable:

   ```sh
   export API_KEYS="API_KEY_1,API_KEY_2,API_KEY_3"
   ```

3. Build the application:

   ```sh
   go build -o release/claude-proxy main.go
   ```

## Usage

1. Run the application:

   ```sh
   ./claude-proxy
   ```

2. The proxy server will start on the port specified by the `PORT` environment variable, or port `8080` if `PORT` is not set. You can now make requests to `http://localhost:<port>` and the proxy will forward them to the Anthropic Claude API, handling API key rotation on rate limit.

## Code Overview

### main.go

- `init()`: Loads API keys from the `API_KEYS` environment variable.
- `main()`: Starts the HTTP server and listens on the port specified by the `PORT` environment variable, or port `8080` if `PORT` is not set.
- `proxyHandler()`: Handles incoming requests, forwards them to the Anthropic Claude API, and handles API key rotation on rate limit.
- `rotateAPIKey()`: Rotates the API key when a rate limit is detected.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
