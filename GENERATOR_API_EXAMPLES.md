# Generator API Examples

## New Options-Based API (Recommended)

### Basic Usage
```go
gen, err := generator.NewGeneratorWithOptions(templateContent, "2024-01-15")
if err != nil {
    log.Fatal(err)
}
```

### With Previous Date
```go
gen, err := generator.NewGeneratorWithOptions(
    templateContent, 
    "2024-01-15",
    generator.WithPreviousDate("2024-01-14"),
)
```

### With Custom Variables
```go
customVars := map[string]interface{}{
    "projectName": "My Project",
    "version":     "1.0.0",
    "author":      "John Doe",
}

gen, err := generator.NewGeneratorWithOptions(
    templateContent, 
    "2024-01-15",
    generator.WithCustomVariables(customVars),
)
```

### Complete Configuration
```go
gen, err := generator.NewGeneratorWithOptions(
    templateContent, 
    "2024-01-15",
    generator.WithPreviousDate("2024-01-14"),
    generator.WithCustomVariables(customVars),
)
```

### From File
```go
gen, err := generator.NewGeneratorFromFileWithOptions(
    "template.md", 
    "2024-01-15",
    generator.WithPreviousDate("2024-01-14"),
)
```

### Reconfiguring Existing Generator
```go
// Start with basic generator
gen1, err := generator.NewGeneratorWithOptions(templateContent, "2024-01-15")

// Create a new generator with additional options
gen2, err := gen1.WithOptions(
    generator.WithPreviousDate("2024-01-14"),
    generator.WithCustomVariables(customVars),
)
```

## Benefits of New API

1. **Template Validation**: Templates are validated at creation time, catching syntax errors early
2. **Flexibility**: Easy to add new configuration options without breaking existing code
3. **Composability**: Options can be combined in any order
4. **Thread Safety**: Generators are explicitly documented as thread-safe
5. **Reconfiguration**: Existing generators can be reconfigured without rebuilding from scratch

## Migration Guide

### Old API
```go
// Old way - multiple constructors
gen, err := generator.NewGeneratorWithPreviousAndCustom(
    templateContent, 
    "2024-01-15", 
    "2024-01-14", 
    customVars,
)
```

### New API
```go
// New way - options pattern
gen, err := generator.NewGeneratorWithOptions(
    templateContent, 
    "2024-01-15",
    generator.WithPreviousDate("2024-01-14"),
    generator.WithCustomVariables(customVars),
)
```

The old constructors are still available but deprecated. They will continue to work but new code should use the options-based API.
