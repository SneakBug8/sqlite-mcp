# SQLite MCP Server

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for SQLite database operations. This server provides a standardized interface for SQLite database interactions including schema inspection, read and write operations.

## Available Tools

The MCP server exposes three main tools:

#### get_schema

- Description: List all tables in the SQLite database with their schema information
- Parameters:
  - `database` (required): Path to the SQLite database file to explore
- Usage: Provides complete schema introspection including columns, types, constraints, and indexes

#### query

- Description: Execute read-only queries against the SQLite database.
- Parameters: 
  - `database` (required): Path to the SQLite database file to explore
  - `sql` (required): Read-only SQL query to execute
-Usage: Only SELECT, WITH, and EXPLAIN queries are allowed
- Example: `SELECT * FROM users WHERE age > 25`

#### execute

- Description: Execute write operations against the SQLite database
- Parameters:
  - `database` (required): Path to the SQLite database file to explore
  - `sql` (required): SQL statement that modifies the database
- Usage: INSERT, UPDATE, DELETE, CREATE, ALTER, DROP operations
- Example: `INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com')`

Each tool call selects the database to explore with the required `database` argument. The calling agent can work with multiple databases without restarting the server.


## Get Started

### Prerequisites

The required dependencies are Go, Taskfile, and SQLite3. The optional dependencies are golangci-lint and Docker. Make sure each dependency is available in the system `PATH`.

- **Go**: SQLite MCP Server requires Go 1.24.4 or later. Download from [go.dev/doc/install](https://go.dev/doc/install)
- **Taskfile**: Install [go-task](https://taskfile.dev/) to run automated development tasks. Install using Homebrew with `brew install go-task`
- **SQLite3**: For creating and managing SQLite databases locally
- **golangci-lint**: Go linter. Install using Homebrew with `brew install golangci-lint` or follow instructions at [golangci-lint.run](https://golangci-lint.run/) (optional)
- **Docker**: To run the server with Docker (optional)

#### Windows

On Windows, Go, Taskfile, and SQLite3 must be in the system `PATH`. Do these steps:

1. Install Go from [go.dev/doc/install](https://go.dev/doc/install).
2. Install Taskfile. Use `winget install Task.Task`, `choco install go-task`, or `scoop install task`.
3. Install SQLite3. Use `winget install SQLite.SQLite`, `choco install sqlite`, or `scoop install sqlite`.
4. Add the install folders to the `PATH` environment variable.
5. Open a new terminal.
6. Run `task --version` and `sqlite3 --version`. Both commands must show a version.

The `task build` command uses `sqlite3` to create the example database. Windows PowerShell uses `;` to separate commands and does not support `&&`. Use `task` tasks instead of the bash examples.

Note: The server uses `go-sqlite3`, which needs CGO and a C compiler. Install a GCC toolchain, such as MinGW-w64 or TDM-GCC, and add it to the `PATH` before you build on Windows.

### Installation

1. Clone the repository:
```bash
git clone https://github.com/rvarun11/sqlite-mcp.git
cd sqlite-mcp
```

2. Build:
```bash
# Builds the binary and the example database
task build

# Or if you prefer Docker:
task docker-build
```

### MCP Client Configuration

After installation, add the following configuration to your MCP client:

#### Using built binary:

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "/path/to/your/sqlite-mcp/build/sqlite-mcp",
      "args": []
    }
  }
}
```
Args:
- `--debug`: Enable debug mode for verbose logging (optional)

On Windows, use the full path to the `.exe` file with backslashes escaped in JSON:
```json
{
  "mcpServers": {
    "sqlite": {
      "command": "C:\\path\\to\\sqlite-mcp\\build\\sqlite-mcp.exe",
      "args": []
    }
  }
}
```

The database file path is not a command argument. The calling agent passes the `database` argument to each tool call instead.

#### Using Docker:

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "/path/to/your/.docker/bin/docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "sqlite-mcp-server"
      ]
    }
  }
}
```

The above uses `example.sql` to build the example database. To use a custom schema sql, [see the Docker commands below](#docker). The example database is created at `/app/database.db`. Pass this path as the `database` argument in tool calls.

**Important Note for GUI Applications**: When configuring MCP clients in GUI applications (like Claude Desktop), you must use absolute paths for both the command and database file paths. Do not use:
- Tilde (`~`) for home directory shortcuts
- Environment variables like `$HOME` or `$PATH`
- Relative paths like `./build/sqlite-mcp`
- Command shortcuts that rely on PATH resolution (like just `docker`)

## Development

To see a list of all available development tasks, run:
```bash
task --list
```

Available tasks include:
- `fmt`: Tidy modules and format code
- `lint`: Run `goclangci-lint` static analysis
- `test`: Run unit tests
- `check`: Run `fmt`, `lint` and `test`.
- `build-example-db`: Create example db from `example.sql`
- `run-dev`: Run from source with example db, includes all checks
- `build`: Build the binary with example db, including tests
- `docker-build`: Build Docker image


### Run the Build

#### Binary

- After building the binary with `task build` or `task build-with-db`, run:
```bash
./build/sqlite-mcp [--debug]
```
On Windows, run the binary with:
```powershell
.\build\sqlite-mcp.exe --debug
```
Then pass the path to the database file as the `database` argument in each tool call.

#### Docker 

- After building the docker image with `task docker-build`, run the Docker container:
```bash
# Run with default example.sql database
docker run -i --rm sqlite-mcp-server

# Or run with custom schema.sql file
docker run -i --rm -v "/path/to/your/schema.sql:/data/schema.sql" sqlite-mcp-server
```
