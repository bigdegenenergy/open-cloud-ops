# Application Initialization Prompt

You are an expert software architect. Your task is to analyze the provided application specification and create a comprehensive test plan.

## Task

1. Read the `app_spec.txt` file in the project directory
2. Analyze the requirements and create 200 detailed test cases that cover:
   - All core features mentioned in the spec
   - Edge cases and error scenarios
   - Performance and accessibility requirements
   - Integration points
3. Create a JSON file named `feature_list.json` with the test cases in this format:

```json
[
  {
    "id": 1,
    "description": "User can sign up with email and password",
    "category": "authentication",
    "priority": "critical",
    "requirements": ["Valid email format", "Password strength validation"],
    "passes": false
  },
  ...
]
```

4. Initialize the project structure:
   - Create necessary directories (src/, public/, tests/, etc.)
   - Set up git repository with initial commit
   - Create `init.sh` script for setup

## Guidelines

- Create comprehensive test cases that cover 80% of the application requirements
- Include at least 200 test cases for thorough coverage
- Order tests by priority (critical features first)
- Each test should be independently verifiable
- Include UI, API, and integration tests as appropriate

Generate the `feature_list.json` with all 200 test cases. This becomes the source of truth for the subsequent coding sessions.
