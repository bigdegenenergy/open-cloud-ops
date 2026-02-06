# Add Tests

Add comprehensive tests for the specified code.

## Testing Strategy

1. **Analyze the Code**
   - Understand the function/module to test
   - Identify inputs, outputs, and side effects
   - Map code branches and edge cases
   - Note dependencies that may need mocking

2. **Identify Test Cases**

   **Happy Path Tests**
   - Standard use cases
   - Typical inputs producing expected outputs

   **Edge Cases**
   - Empty inputs (null, undefined, empty string, empty array)
   - Boundary values (0, -1, MAX_INT, etc.)
   - Single item vs many items
   - Special characters and unicode

   **Error Cases**
   - Invalid inputs
   - Missing required parameters
   - Malformed data
   - Network/IO failures (if applicable)

   **Integration Points**
   - Dependencies working correctly
   - Database interactions
   - External API calls

3. **Write Tests**
   - Follow existing test patterns in the codebase
   - Use descriptive test names
   - One assertion per concept
   - Arrange-Act-Assert pattern

4. **Verify Coverage**
   - Run tests to ensure they pass
   - Check coverage if available
   - Aim for meaningful coverage, not just numbers

## Test Naming Convention

```
describe('[UnitName]', () => {
  describe('[methodName]', () => {
    it('should [expected behavior] when [condition]', () => {
      // test
    });
  });
});
```

## Output

Create test files following the project's testing conventions:

- Jest: `*.test.ts` or `*.spec.ts`
- Pytest: `test_*.py`
- Go: `*_test.go`

Include:

- Unit tests for individual functions
- Integration tests for component interactions
- Mock external dependencies appropriately

Run the tests after creating them to verify they work.
