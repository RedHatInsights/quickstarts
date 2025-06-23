# Resource Selector - MCP Tools

This package implements Model Context Protocol (MCP) tools for the quickstarts application. It provides an interface for AI models to interact with the quickstarts database and perform various operations.

## Available Tools

### 1. greetUser
A simple greeting tool for testing purposes.

**Parameters:**
- `name` (string, required): Name of the person to greet

**Example:**
```json
{
  "name": "John"
}
```

### 2. addNumbers
A mathematical tool that adds two numbers together.

**Parameters:**
- `a` (number, required): First number to add
- `b` (number, required): Second number to add  

**Example:**
```json
{
  "a": 5.5,
  "b": 3.2
}
```

### 3. searchQuickstarts
**NEW**: Search for quickstarts in the database using **Levenshtein distance** for superior fuzzy matching.

This tool performs advanced fuzzy searches using separate optional parameters with **intelligent scoring**:
- **name**: Exact match on quickstart name (database column) - **Best Score (0)**
- **displayName**: Fuzzy match on display name (from JSONB `spec.displayName`) - **High Priority (score × 10)**
- **description**: Fuzzy match on description (from JSONB `spec.description`) - **Medium Priority (score × 20)**

**Parameters (at least one required):**
- `name` (string, optional): Search by exact quickstart name or fuzzy match
- `displayName` (string, optional): Search by display name (fuzzy match with Levenshtein)
- `description` (string, optional): Search by description (fuzzy match with Levenshtein)
- `limit` (integer, optional): Maximum number of results to return (default: 10, max: 50)
- `maxDistance` (integer, optional): Maximum Levenshtein distance for fuzzy matching (default: 3)

**Example:**
```json
{
  "displayName": "API",
  "description": "documentation", 
  "maxDistance": 2,
  "limit": 5
}
```

**Response:**
Results are **automatically ordered by relevance** (best matches first):
```json
{
  "results": [
    {
      "name": "insights-apis",
      "displayName": "APIs", 
      "description": "Explore the API catalog and documentation."
    }
  ],
  "count": 1
}
```

## Features

### **Advanced Hybrid Search**
- **🎯 Partial Matching**: Finds substrings within attributes - "Create RHEL" matches "Create RHEL images for deployment"
- **🔤 Typo Tolerance**: Handles common typos like "blueprnt" → "blueprint", "secruity" → "security"  
- **📊 Intelligent Ranking**: Longer partial matches ranked higher than shorter ones
- **⚡ Performance**: Uses PostgreSQL `fuzzystrmatch` extension for optimal speed
- **🔄 Automatic Fallback**: Falls back to `ILIKE` pattern matching if extension unavailable

### **Enhanced Response Format**
- **🏆 Single Best Match**: Returns only the most relevant quickstart (no noise)
- **📦 Complete Object**: Includes full quickstart object with all database fields
- **🏷️ Match Metadata**: Provides match type, score, and search parameters
- **📋 Structured JSON**: Clean, consistent response format for easy consumption
- **🔍 Rich Context**: Extracted displayName and description for quick reference

### **Multi-Parameter Search**
- **`name`**: Exact quickstart identifier matching
- **`displayName`**: Partial and fuzzy matching for user-friendly titles  
- **`description`**: Partial and fuzzy matching within quickstart content
- **`limit`**: Control result count (1-50, default 10)
- **`maxDistance`**: Adjust typo tolerance (default 7, optimal balance of tolerance vs precision)

### **Partial Matching Examples**
Advanced substring matching finds relevant content within larger text:
- **Query**: `"Create RHEL"` **→** **Matches**: `"Create RHEL images for my deployment"`
- **Query**: `"security scanning"` **→** **Matches**: `"Advanced security scanning and vulnerability assessment"`
- **Query**: `"API documentation"` **→** **Matches**: `"Complete API documentation and examples"`
- **Query**: `"getting started"` **→** **Matches**: `"Getting started with OpenShift deployment"`

### **Scoring Priorities** (Lower score = better match)
1. **Exact Match** (score: 0): Perfect text match
2. **Partial Match** (score: 1-199): Substring coverage-based scoring  
3. **Fuzzy Match** (score: 200-699): Levenshtein distance-based matching
4. **Fallback Match** (score: 700+): Pattern matching without advanced scoring

## Algorithm Details

### **Scoring System**
The search uses a sophisticated scoring system with weighted priorities:

1. **Exact name match**: Score = 0 (perfect match)
2. **DisplayName fuzzy match**: Score = `levenshtein_distance × 10`
3. **Name fuzzy match**: Score = `levenshtein_distance × 15`  
4. **Description fuzzy match**: Score = `levenshtein_distance × 20`
5. **No match**: Score = 999 (filtered out)

### **SQL Implementation**
The search uses a subquery approach to handle calculated scoring:

```sql
-- Main query structure
SELECT * FROM (
    SELECT quickstarts.*, 
           CASE 
               WHEN name = 'exact_match' THEN 0
               WHEN levenshtein(displayName, 'query') <= 3 THEN levenshtein(...) * 10
               WHEN levenshtein(description, 'query') <= 3 THEN levenshtein(...) * 20
               WHEN levenshtein(name, 'query') <= 3 THEN levenshtein(...) * 15
               ELSE 999
           END as score
    FROM quickstarts 
    WHERE (matching_conditions)
) as scored_quickstarts
WHERE score < 999
ORDER BY score ASC
LIMIT 10;
```

**Why Subquery?** PostgreSQL doesn't allow WHERE clauses to reference SELECT aliases directly. The subquery calculates the score first, then the outer query can filter and sort by it.

### **Query Examples**

**1. Typo-Tolerant Search:**
```bash
# Query: "create a blueprnt" (missing 'i')
# Matches: "Create a blueprint" with Levenshtein distance = 1
curl -X POST http://localhost:8000/api/quickstarts/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"query": "create a blueprnt"}'
```

**2. Multi-word Fuzzy Matching:**
```bash  
# Query: "getting startd" 
# Matches: "Getting started" guides with distance = 1
curl -X POST http://localhost:8000/api/quickstarts/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"query": "getting startd"}'
```

**3. Adjusting Typo Tolerance:**
```bash
# More permissive: allows up to 10 character differences
curl -X POST http://localhost:8000/api/quickstarts/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"query": "searchQuickstarts", "maxDistance": 10}'
```

**Default Values:**
- `maxDistance`: **7** (optimal balance of typo tolerance vs precision)
- `limit`: **10** (maximum 50)
- Scoring weights: displayName×10, name×15, description×20

## Database Query Details

The `searchQuickstarts` tool uses **PostgreSQL Levenshtein distance** with intelligent scoring for superior fuzzy matching:

### **Levenshtein Distance Implementation**

**Scoring Algorithm:**
```sql
CASE 
  WHEN name = 'exact_match' THEN 0                    -- Perfect match (best score)
  WHEN levenshtein(LOWER(displayName), LOWER('query')) <= 3 THEN 
    levenshtein(...) * 10                             -- DisplayName priority (high weight)
  WHEN levenshtein(LOWER(description), LOWER('query')) <= 3 THEN 
    levenshtein(...) * 20                             -- Description priority (medium weight)
  WHEN levenshtein(LOWER(name), LOWER('query')) <= 3 THEN 
    levenshtein(...) * 15                             -- Name fuzzy (medium-high weight)
  ELSE 999                                            -- No match
END as score
```

**Key Features:**
- **Smart Prioritization**: DisplayName matches score highest (most relevant)
- **Distance Control**: `maxDistance` parameter controls fuzziness (default: 3)
- **Automatic Ordering**: Results sorted by score (best matches first)  
- **Case Insensitive**: All comparisons use `LOWER()` for better matching

### **Query Building Logic:**
- **Exact name match**: `name = ?` (score: 0 - best possible)
- **DisplayName fuzzy**: `levenshtein(LOWER(displayName), LOWER(?)) <= maxDistance`
- **Description fuzzy**: `levenshtein(LOWER(description), LOWER(?)) <= maxDistance` 
- **Name fuzzy**: `levenshtein(LOWER(name), LOWER(?)) <= maxDistance`
- **Combined with OR**: Multiple parameters use OR logic
- **Filtered by score**: Only results with score < 999 are returned

### **Fallback Mechanism:**
If PostgreSQL `fuzzystrmatch` extension is not available, automatically falls back to `ILIKE` pattern matching:
```sql
-- Fallback queries
WHERE name ILIKE '%query%' 
   OR content->'spec'->>'displayName' ILIKE '%query%' 
   OR content->'spec'->>'description' ILIKE '%query%'
```

### **Example Scoring:**
- `"API"` searching `"APIs"` → Distance: 1, DisplayName Score: 10
- `"blueprint"` searching `"Creating blueprint images"` → Distance: 2, Description Score: 40
- Exact name match → Score: 0 (always wins)

## PostgreSQL Requirements

The `searchQuickstarts` tool uses the PostgreSQL `fuzzystrmatch` extension for Levenshtein distance calculations.

**✅ Automatic Setup:**
The extension is **automatically created** during:
- Database initialization (`database.Init()`)
- Migration runs (`cmd/migrate/migrate.go`)

```sql
-- This is handled automatically by the application
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
```

**Extension Status:**
- **✅ Available**: Uses advanced Levenshtein distance with intelligent scoring
- **⚠️ Unavailable**: Automatically falls back to `ILIKE` pattern matching with warning logs

**Manual Verification:**
```sql
-- Test if extension is working
SELECT levenshtein('test', 'test');  -- Should return 0
SELECT levenshtein('API', 'APIs');  -- Should return 1
```

**If Manual Installation Needed:**
- **Ubuntu/Debian**: `sudo apt-get install postgresql-contrib`
- **CentOS/RHEL**: `sudo yum install postgresql-contrib`  
- **Docker**: Most PostgreSQL images include contrib extensions
- **Cloud**: Usually pre-installed (AWS RDS, Google Cloud SQL, etc.)

## HTTP Endpoint

**URL**: `POST /api/quickstarts/v1/mcp`

**Request Body**:
```json
{
  "query": "create a blueprint"
}
```

**Response**:
```json
{
  "status": "success",
  "query": "create a blueprint",
  "tool_result": {
    "tool": "searchQuickstarts",
    "data": {
      "tool": "searchQuickstarts",
      "searchParams": {
        "displayName": "create a blueprint",
        "maxDistance": 7
      },
      "result": {
        "score": 0,
        "matchType": "display_name",
        "quickstart": {
          "id": 123,
          "name": "insights-blueprints",
          "content": {...},
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T00:00:00Z"
        },
        "displayName": "Create a blueprint",
        "description": "Learn how to create and manage blueprints for system images"
      }
    }
  }
}
```

### **Response Fields**
- **`status`**: "success", "no_tool_call", or "error"
- **`query`**: Original user query
- **`tool_result.tool`**: Name of the tool that was executed (for programmatic use)
- **`tool_result.data`**: Complete structured response from the tool
  - **`tool`**: Tool name (repeated for clarity)
  - **`searchParams`**: Parameters used for the search
  - **`result`**: **Single best match** with complete quickstart object
    - **`score`**: Hybrid scoring (lower = better match)
    - **`matchType`**: Type of match ("exact_match", "partial_match", "fuzzy_match", "fallback_match")
    - **`quickstart`**: **Complete quickstart object** from database
    - **`displayName`**: Extracted display name from quickstart content
    - **`description`**: Extracted description from quickstart content

## Usage Examples

### **1. Via HTTP API**
```bash
curl -X POST http://localhost:8000/api/quickstarts/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"query": "create a blueprint"}'
```

### **2. Typo-Tolerant Search**
```bash
curl -X POST http://localhost:8000/api/quickstarts/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"query": "create a blueprnt"}'  # Missing 'i' in blueprint
```

### **3. Security-Related Search**
```bash
curl -X POST http://localhost:8000/api/quickstarts/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{"query": "security scanning"}'
```

## Usage Example

1. **Natural Language Query**: "Find quickstarts about creating blueprints for RHEL images"
2. **Ollama LLM Processing**: Determines to use `searchQuickstarts` tool with `displayName: "blueprint"` and `description: "RHEL images"`
3. **Tool Execution**: Searches database using Levenshtein distance with intelligent scoring
4. **Response**: Returns relevant quickstarts ordered by best match (displayName matches ranked highest)

### **Advanced Examples:**

**Fuzzy Matching Examples:**
- **"Find API docs"** → Matches `"APIs"`, `"API catalog"`, `"REST API"` (distance ≤ 3)
- **"blueprint images"** → Matches `"Creating blueprint images"`, `"Image blueprints"` 
- **"getting started"** → Matches `"Getting Started"`, `"Get started"`, `"Start guide"`

**Intelligent Scoring in Action:**
```json
Query: {"displayName": "API", "description": "documentation"}

Results (ordered by relevance):
1. name: "insights-apis", displayName: "APIs" (score: 10 - displayName match)
2. name: "api-docs", description: "API documentation guide" (score: 40 - description match)  
3. name: "rest-api-guide", displayName: "REST APIs" (score: 20 - displayName match, higher distance)
```

**Query Examples:**
- **"Find the insights-apis quickstart"** → `{"name": "insights-apis"}` (exact match, score: 0)
- **"Show API-related quickstarts"** → `{"displayName": "API", "maxDistance": 2}`
- **"Documentation about blueprints"** → `{"description": "blueprint", "displayName": "documentation"}`

## Error Handling

The search tool handles various error conditions:
- **Database Unavailable**: Returns error message if database connection is nil
- **Query Errors**: Returns formatted error message for database query failures
- **JSON Processing**: Gracefully handles JSONB unmarshaling errors
- **Limit Validation**: Enforces maximum limit of 50 results
- **Parameter Validation**: Ensures at least one search parameter is provided
- **Robust JSON Unmarshaling**: Handles both string and integer values for `limit` and `maxDistance` parameters
- **Extension Fallback**: Automatically falls back to ILIKE if PostgreSQL `fuzzystrmatch` extension is unavailable
- **Distance Validation**: Sets sensible defaults for `maxDistance` (default: 3) to prevent excessive fuzzy matching

## Testing

Run tests with:
```bash
go test ./pkg/resourceSelector/ -v
```

The tests cover:
- Argument serialization/deserialization
- Response structure validation
- Error handling (including database connection failures)

## Multi-Tool System Documentation

This package implements a flexible multi-tool system that can dynamically call different tool functions based on Ollama's tool selection. The system uses a tool registry to manage multiple tools with different argument schemas.

## Architecture

The system consists of several key components:

1. **ToolRegistry**: Manages tool handlers and their schemas
2. **ToolHandler**: Generic function signature for all tools
3. **Tool Functions**: Specific implementations of tools (e.g., GreetUserTool, AddNumbersTool)
4. **Ollama Integration**: Generates tool calls based on user queries
5. **MCP Integration**: Registers tools with the MCP server

## Current Tools

### 1. GreetUser Tool
- **Name**: `greetUser`
- **Description**: Greets users by their given name
- **Arguments**: `{name: string}`
- **Example**: "Hello, John!"

### 2. AddNumbers Tool
- **Name**: `addNumbers`
- **Description**: Adds two numbers together
- **Arguments**: `{a: number, b: number}`
- **Example**: "5.00 + 3.00 = 8.00"

## How to Add a New Tool

### Step 1: Define the Tool Arguments Structure

```go
type YourToolArgs struct {
    Param1 string  `json:"param1" jsonschema:"required,description=Description of param1"`
    Param2 int     `json:"param2" jsonschema:"required,description=Description of param2"`
}
```

### Step 2: Implement the Tool Function

```go
func YourTool(ctx context.Context, args YourToolArgs) (*mcp.ToolResponse, error) {
    logrus.Infoln("Your tool called with args:", args)
    
    // Your tool logic here
    result := fmt.Sprintf("Processed %s with %d", args.Param1, args.Param2)
    
    return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
}
```

### Step 3: Create a Tool Handler

```go
func YourToolHandler(ctx context.Context, args json.RawMessage) (*mcp.ToolResponse, error) {
    var toolArgs YourToolArgs
    if err := json.Unmarshal(args, &toolArgs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal your tool args: %w", err)
    }
    return YourTool(ctx, toolArgs)
}
```

### Step 4: Define the Ollama Tool Schema

```go
func CreateYourToolSchema() ollamaApi.ToolFunction {
    return ollamaApi.ToolFunction{
        Name:        "yourTool",
        Description: "Description of what your tool does",
        Parameters: struct {
            Type       string   `json:"type"`
            Defs       any      `json:"$defs,omitempty"`
            Items      any      `json:"items,omitempty"`
            Required   []string `json:"required"`
            Properties map[string]struct {
                Type        ollamaApi.PropertyType `json:"type"`
                Items       any                    `json:"items,omitempty"`
                Description string                 `json:"description"`
                Enum        []any                  `json:"enum,omitempty"`
            } `json:"properties"`
        }{
            Type:     "object",
            Required: []string{"param1", "param2"},
            Properties: map[string]struct {
                Type        ollamaApi.PropertyType `json:"type"`
                Items       any                    `json:"items,omitempty"`
                Description string                 `json:"description"`
                Enum        []any                  `json:"enum,omitempty"`
            }{
                "param1": {
                    Type:        ollamaApi.PropertyType{"string"},
                    Description: "Description of param1",
                },
                "param2": {
                    Type:        ollamaApi.PropertyType{"number"},
                    Description: "Description of param2",
                },
            },
        },
    }
}
```

### Step 5: Register the Tool

Add this to the `init()` function in `resourceSelector.go`:

```go
// Register with MCP server
err = mcpServer.RegisterTool(
    "yourTool",
    "Description of your tool",
    YourTool,
)
if err != nil {
    logrus.Fatalf("Failed to register yourTool: %v", err)
}

// Create schema and register with tool registry
yourToolSchema := CreateYourToolSchema()
toolRegistry.RegisterTool("yourTool", YourToolHandler, yourToolSchema)

// Update ollamaTools (this should be done after all tools are registered)
ollamaTools = toolRegistry.GetSchemas()
```

## Example Tools

See `example_tools.go` for complete examples including:

- **ReverseString Tool**: Reverses a given string
- **AnalyzeText Tool**: Analyzes text for word/character/line count with enum parameters

## Tool Parameter Types

### Supported JSON Schema Types

- `string`: Text input
- `number`: Numeric input (int or float)
- `boolean`: True/false values
- `array`: Lists of values
- `object`: Nested objects

### Special Features

- **Required Fields**: Use the `Required` slice to specify mandatory parameters
- **Enums**: Use the `Enum` field to restrict values to specific options
- **Descriptions**: Always provide clear descriptions for better LLM understanding

## Testing Your Tools

1. Build the application: `go build -o quickstarts-server .`
2. Run the server: `./quickstarts-server`
3. Send a POST request to `/mcp-query` with:
   ```json
   {
     "query": "Please greet John"
   }
   ```
4. The system will:
   - Send the query to Ollama
   - Ollama will generate a tool call
   - The system will execute the appropriate tool
   - Return the result

## Best Practices

1. **Clear Descriptions**: Write descriptive tool and parameter descriptions for better LLM understanding
2. **Error Handling**: Always handle unmarshaling errors in tool handlers
3. **Logging**: Use logrus to log tool calls for debugging
4. **Validation**: Validate input parameters in your tool functions
5. **Consistent Naming**: Use consistent naming conventions (camelCase for tool names)
6. **Documentation**: Document complex tools with examples

## Troubleshooting

### Tool Not Found Error
- Ensure the tool is registered in both the MCP server and tool registry
- Check that the tool name matches exactly between registration and Ollama schema

### Unmarshaling Errors
- Verify the JSON tags in your argument struct match the schema
- Ensure required fields are properly defined

### Tool Not Called by Ollama
- Check the tool description and parameter descriptions are clear
- Verify the tool schema is properly formatted
- Test with explicit tool-calling queries 