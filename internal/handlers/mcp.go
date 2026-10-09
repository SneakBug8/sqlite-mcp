package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/rvarun11/sqlite-mcp/internal/models"
	"github.com/rvarun11/sqlite-mcp/internal/repository"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

type MCPHandler struct {
	manager *repository.Manager
	logger  *zap.SugaredLogger
}

func NewMCPHandler(manager *repository.Manager, logger *zap.SugaredLogger) *MCPHandler {
	return &MCPHandler{
		manager: manager,
		logger:  logger,
	}
}

func (h *MCPHandler) GetSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info("Handling get_schema request")

	dbPath, ok := databaseArgument(request)
	if !ok {
		return errorResult("Database parameter is required"), nil
	}

	db, err := h.manager.Get(dbPath)
	if err != nil {
		h.logger.Error("Failed to open database: ", err)
		return errorResult(fmt.Sprintf("Failed to open database: %v", err)), nil
	}

	tables, err := db.GetSchema()
	if err != nil {
		h.logger.Error("Failed to list tables", err)
		return errorResult("Failed to retrieve table information. Please check your database connection."), nil
	}

	return textResult(formatTablesResponse(tables)), nil
}

func (h *MCPHandler) Query(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info("Handling query request")

	dbPath, ok := databaseArgument(request)
	if !ok {
		return errorResult("Database parameter is required"), nil
	}

	sql, ok := stringArgument(request, "sql")
	if !ok || sql == "" {
		return errorResult("SQL query parameter is required"), nil
	}

	db, err := h.manager.Get(dbPath)
	if err != nil {
		h.logger.Error("Failed to open database: ", err)
		return errorResult(fmt.Sprintf("Failed to open database: %v", err)), nil
	}

	result, err := db.Query(sql)
	if err != nil {
		h.logger.Error("Query execution failed: ", err)
		return errorResult("Query execution failed. Please check your SQL syntax and try again."), nil
	}

	return textResult(formatQueryResponse(result)), nil
}

func (h *MCPHandler) Execute(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info("Handling execute request")

	dbPath, ok := databaseArgument(request)
	if !ok {
		return errorResult("Database parameter is required"), nil
	}

	sql, ok := stringArgument(request, "sql")
	if !ok {
		return errorResult("Missing or invalid 'sql' argument"), nil
	}

	db, err := h.manager.Get(dbPath)
	if err != nil {
		h.logger.Error("Failed to open database: ", err)
		return errorResult(fmt.Sprintf("Failed to open database: %v", err)), nil
	}

	result, err := db.Execute(sql)
	if err != nil {
		h.logger.Error("Statement execution failed: ", err)
		return errorResult("Statement execution failed. Please check your SQL syntax and try again."), nil
	}

	return textResult(formatExecuteResponse(result)), nil
}

// databaseArgument reads the required "database" argument from the tool call.
func databaseArgument(request mcp.CallToolRequest) (string, bool) {
	return stringArgument(request, "database")
}

func stringArgument(request mcp.CallToolRequest, name string) (string, bool) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := args[name].(string)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

func errorResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// Helper functions for formatting responses
func formatTablesResponse(tables []models.Table) string {
	if len(tables) == 0 {
		return "No tables found in the database."
	}

	response := "Database Tables:\n\n"
	for _, table := range tables {
		response += "Table: " + table.Name + "\n"

		if len(table.Columns) > 0 {
			response += "Columns:\n"
			for _, col := range table.Columns {
				response += "  - " + col.Name + " (" + col.Type + ")"
				if col.NotNull {
					response += " NOT NULL"
				}
				if col.PrimaryKey {
					response += " PRIMARY KEY"
				}
				if col.DefaultValue != nil {
					response += " DEFAULT " + *col.DefaultValue
				}
				response += "\n"
			}
		}

		if len(table.Indexes) > 0 {
			response += "Indexes:\n"
			for _, index := range table.Indexes {
				response += "  - " + index + "\n"
			}
		}

		if len(table.ForeignKeys) > 0 {
			response += "Foreign Keys:\n"
			for _, fk := range table.ForeignKeys {
				response += "  - " + fk.From + " -> " + fk.Table + "(" + fk.To + ")"
				if fk.OnDelete != "NO ACTION" {
					response += " ON DELETE " + fk.OnDelete
				}
				if fk.OnUpdate != "NO ACTION" {
					response += " ON UPDATE " + fk.OnUpdate
				}
				response += "\n"
			}
		}
		response += "\n"
	}

	return response
}

func formatQueryResponse(result *models.QueryResult) string {
	response := fmt.Sprintf("Query Results:\nColumns: %s\nRow Count: %d\n\n",
		strings.Join(result.Columns, ", "),
		result.Count)

	if result.Count > 0 {
		response += "Data:\n"
		for i, row := range result.Rows {
			if i >= 10 { // Limit display to first 10 rows
				response += "... (showing first 10 rows)\n"
				break
			}

			response += fmt.Sprintf("Row %d: ", i+1)

			var pairs []string
			for _, col := range result.Columns {
				value := row[col]
				if value == nil {
					value = "<NULL>"
				}
				pairs = append(pairs, fmt.Sprintf("%s=%v", col, value))
			}
			response += strings.Join(pairs, ", ") + "\n"
		}
	}

	return response
}

func formatExecuteResponse(result *models.ExecuteResult) string {
	response := "Execution Result:\n"
	response += fmt.Sprintf("Rows Affected: %d\n", result.RowsAffected)
	if result.LastInsertId > 0 {
		response += fmt.Sprintf("Last Insert ID: %d\n", result.LastInsertId)
	}
	response += "Message: " + result.Message + "\n"
	return response
}
