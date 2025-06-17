# Test Data Structure

The testdata directory contains integration test cases with a simplified structure using a shared template.

## Structure

```text
testdata/
├── shared_template.md       # Single template used by all tests
├── complex_test/
│   ├── input.md
│   ├── expected_output.md
│   └── expected_input_after.md
├── simple_test/
│   ├── input.md
│   ├── expected_output.md
│   └── expected_input_after.md
├── simple_test_bullet_entries/
│   ├── input.md
│   ├── expected_output.md
│   └── expected_input_after.md
└── template/
    ├── input.md
    ├── expected_output.md
    └── expected_input_after.md
```

## Shared Template

All integration tests use `testdata/shared_template.md` as the template for generating the uncompleted tasks file. This eliminates duplication and makes it easier to add new test cases.

## Adding New Test Cases

To add a new test case:

1. Create a new directory under `testdata/` (e.g., `new_test/`)
2. Add `input.md` with the test input journal file
3. Add `expected_output.md` with the expected output using the shared template
4. Add `expected_input_after.md` with the expected modified input file
5. The test will automatically use `testdata/shared_template.md` as the template

## Test File Formats

- **input.md**: Input journal file with TODO items to be processed
- **expected_output.md**: Expected content of the output file with uncompleted tasks formatted using the shared template
- **expected_input_after.md**: Expected content of the input file after processing (completed tasks marked with date tags)
- **shared_template.md**: Template used for all tests with `{{.Date}}` and `{{.TODOS}}` placeholders
