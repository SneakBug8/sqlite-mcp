package handlers

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rvarun11/sqlite-mcp/internal/logger"
	"github.com/rvarun11/sqlite-mcp/internal/repository"
)

func setupTestMCPHandler(t *testing.T) (*MCPHandler, string, func()) {
	// Create temporary database file
	tmpfile, err := os.CreateTemp("", "test_mcp_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()

	log := logger.NewTestLogger()
	repo, err := repository.NewSQLiteDB(tmpfile.Name(), log)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Create test tables with various schema elements
	_, err = repo.Execute(`
        CREATE TABLE users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            age INTEGER,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		repo.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to create users table: %v", err)
	}

	_, err = repo.Execute(`
        CREATE TABLE orders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            product_name TEXT NOT NULL,
            quantity INTEGER DEFAULT 1,
            price DECIMAL(10,2),
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)
	if err != nil {
		repo.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to create orders table: %v", err)
	}

	// Create indexes
	_, err = repo.Execute(`CREATE INDEX idx_users_email ON users(email)`)
	if err != nil {
		repo.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to create index: %v", err)
	}

	// Insert test data
	_, err = repo.Execute(`
        INSERT INTO users (name, email, age) VALUES 
        ('John Doe', 'john@example.com', 30),
        ('Jane Smith', 'jane@example.com', 25),
        ('Bob Wilson', 'bob@example.com', 35)
    `)
	if err != nil {
		repo.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Release the setup connection; the handler manager opens its own.
	repo.Close()

	manager := repository.NewManager(log)
	handler := NewMCPHandler(manager, log)

	cleanup := func() {
		manager.Close()
		os.Remove(tmpfile.Name())
	}

	return handler, tmpfile.Name(), cleanup
}

func TestMCPHandler_GetSchema(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_schema",
			Arguments: map[string]any{
				"database": dbPath,
			},
		},
	}

	result, err := handler.GetSchema(ctx, request)
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	// Verify result structure
	if result.IsError {
		t.Error("Expected successful result, got error")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Verify response contains expected table information
	response := textContent.Text
	if response == "" {
		t.Error("Expected non-empty response")
	}

	// Check that both tables are mentioned
	if !containsString(response, "users") {
		t.Error("Expected response to contain 'users' table")
	}

	if !containsString(response, "orders") {
		t.Error("Expected response to contain 'orders' table")
	}

	// Check for column information
	if !containsString(response, "email") {
		t.Error("Expected response to contain column information")
	}

	// Check for foreign key information
	if !containsString(response, "Foreign Keys") {
		t.Error("Expected response to contain foreign key information")
	}
}

func TestMCPHandler_GetSchema_MissingDatabase(t *testing.T) {
	handler, _, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_schema",
			Arguments: map[string]any{},
		},
	}

	result, err := handler.GetSchema(ctx, request)
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing database parameter")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Database parameter is required") {
		t.Error("Expected error message about missing database parameter")
	}
}

func TestMCPHandler_GetSchema_NonexistentDatabase(t *testing.T) {
	handler, _, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_schema",
			Arguments: map[string]any{
				"database": "does_not_exist.db",
			},
		},
	}

	result, err := handler.GetSchema(ctx, request)
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for nonexistent database")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Failed to open database") {
		t.Error("Expected error message about opening the database")
	}
}

func TestMCPHandler_Query_Success(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "SELECT name, email FROM users WHERE age > 25 ORDER BY name",
			},
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Verify result structure
	if result.IsError {
		t.Error("Expected successful result, got error")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	response := textContent.Text
	if response == "" {
		t.Error("Expected non-empty response")
	}

	// Verify response contains query results
	if !containsString(response, "Query Results") {
		t.Error("Expected response to contain 'Query Results'")
	}

	if !containsString(response, "Row Count") {
		t.Error("Expected response to contain row count")
	}

	// Should contain data for users with age > 25
	if !containsString(response, "john@example.com") || !containsString(response, "bob@example.com") {
		t.Error("Expected response to contain filtered user data")
	}
}

func TestMCPHandler_Query_MissingSQL(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"database": dbPath,
			}, // Missing sql parameter
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Should return error result
	if !result.IsError {
		t.Error("Expected error result for missing SQL parameter")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "SQL query parameter is required") {
		t.Error("Expected error message about missing SQL parameter")
	}
}

func TestMCPHandler_Query_MissingDatabase(t *testing.T) {
	handler, _, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"sql": "SELECT 1",
			},
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing database parameter")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Database parameter is required") {
		t.Error("Expected error message about missing database parameter")
	}
}

func TestMCPHandler_Query_EmptySQL(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "", // Empty SQL
			},
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Should return error result
	if !result.IsError {
		t.Error("Expected error result for empty SQL")
	}
}

func TestMCPHandler_Query_InvalidSQL(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "SELECT * FROM nonexistent_table",
			},
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Should return error result
	if !result.IsError {
		t.Error("Expected error result for invalid SQL")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Query execution failed") {
		t.Error("Expected error message about query execution failure")
	}
}

func TestMCPHandler_Execute_Success(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "execute",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "INSERT INTO users (name, email, age) VALUES ('Test User', 'test@example.com', 28)",
			},
		},
	}

	result, err := handler.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result structure
	if result.IsError {
		t.Error("Expected successful result, got error")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	response := textContent.Text
	if response == "" {
		t.Error("Expected non-empty response")
	}

	// Verify response contains execution results
	if !containsString(response, "Execution Result") {
		t.Error("Expected response to contain 'Execution Result'")
	}

	if !containsString(response, "Rows Affected: 1") {
		t.Error("Expected response to show 1 row affected")
	}

	if !containsString(response, "Last Insert ID") {
		t.Error("Expected response to contain last insert ID")
	}
}

func TestMCPHandler_Execute_Update(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "execute",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "UPDATE users SET age = 31 WHERE name = 'John Doe'",
			},
		},
	}

	result, err := handler.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected successful result, got error")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	response := textContent.Text
	if !containsString(response, "Rows Affected: 1") {
		t.Error("Expected response to show 1 row affected")
	}
}

func TestMCPHandler_Execute_MissingSQL(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "execute",
			Arguments: map[string]any{
				"database": dbPath,
			}, // Missing sql parameter
		},
	}

	result, err := handler.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should return error result
	if !result.IsError {
		t.Error("Expected error result for missing SQL parameter")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Missing or invalid 'sql' argument") {
		t.Error("Expected error message about missing SQL argument")
	}
}

func TestMCPHandler_Execute_MissingDatabase(t *testing.T) {
	handler, _, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "execute",
			Arguments: map[string]any{
				"sql": "CREATE TABLE t (id INTEGER)",
			},
		},
	}

	result, err := handler.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for missing database parameter")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Database parameter is required") {
		t.Error("Expected error message about missing database parameter")
	}
}

func TestMCPHandler_Execute_InvalidSQL(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "execute",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "INSERT INTO nonexistent_table (name) VALUES ('test')",
			},
		},
	}

	result, err := handler.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should return error result
	if !result.IsError {
		t.Error("Expected error result for invalid SQL")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !containsString(textContent.Text, "Statement execution failed") {
		t.Error("Expected error message about statement execution failure")
	}
}

func TestMCPHandler_Execute_DDL(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "execute",
			Arguments: map[string]any{
				"database": dbPath,
				"sql": `CREATE TABLE test_table (
                    id INTEGER PRIMARY KEY,
                    name TEXT NOT NULL
                )`,
			},
		},
	}

	result, err := handler.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected successful result for CREATE TABLE")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	response := textContent.Text
	if !containsString(response, "Statement executed successfully") {
		t.Error("Expected success message for DDL operation")
	}
}

func TestMCPHandler_Query_WithClause(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"database": dbPath,
				"sql": `WITH adult_users AS (
                    SELECT name, email FROM users WHERE age >= 30
                ) SELECT * FROM adult_users ORDER BY name`,
			},
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected successful result for WITH clause query")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	response := textContent.Text
	if !containsString(response, "Query Results") {
		t.Error("Expected query results for WITH clause")
	}
}

func TestMCPHandler_Query_Explain(t *testing.T) {
	handler, dbPath, cleanup := setupTestMCPHandler(t)
	defer cleanup()

	ctx := context.Background()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query",
			Arguments: map[string]any{
				"database": dbPath,
				"sql":      "EXPLAIN SELECT * FROM users WHERE age > 25",
			},
		},
	}

	result, err := handler.Query(ctx, request)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected successful result for EXPLAIN query")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	response := textContent.Text
	if !containsString(response, "Query Results") {
		t.Error("Expected query results for EXPLAIN")
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
