# Testdata Structure

The testdata directory contains integration test cases with a simplified structure using a shared template.

## Structure

```
testdata/
├── shared_template.md       # Single template used by all tests
├── complex_test/
│   ├── original.md
│   ├── expected.md
│   └── expected_modified_original.md
├── simple_test/
│   ├── original.md
│   ├── expected.md
│   └── expected_modified_original.md
├── simple_test_bullet_entries/
│   ├── original.md
│   ├── expected.md
│   └── expected_modified_original.md
└── template/
    ├── original.md
    ├── expected.md
    └── expected_modified_original.md
```

## Shared Template

All integration tests use `testdata/shared_template.md` as the template for generating the uncompleted tasks file. This eliminates duplication and makes it easier to add new test cases.

## Adding New Test Cases

To add a new test case:

1. Create a new directory under `testdata/` (e.g., `new_test/`)
2. Add `original.md` with the test input
3. Add `expected.md` with the expected output using the shared template
4. Add `expected_modified_original.md` with the expected modified original file
5. The test will automatically use `testdata/shared_template.md` as the template

## Test File Formats

- **original.md**: Input journal file with TODO items
- **expected.md**: Expected output file with uncompleted tasks formatted using the shared template
- **expected_modified_original.md**: Expected modified original file with completed tasks marked with date tags
- **shared_template.md**: Template used for all tests with `{{.Date}}` and `{{.TODOS}}` placeholders
