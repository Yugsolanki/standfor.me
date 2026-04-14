---
name: golang-test-generator
description: Generate idiomatic, robust test cases for Golang backend projects using testify and manual mocks
license: MIT
compatibility: opencode
metadata:
  language: golang
  testing_style: table-driven
  ecosystem: backend
  assertions: testify
  mocking: manual-stubs
  scope: white-box
---

## What I do
- Analyze Go source files to identify functions, methods, and interfaces that require testing.
- Write **white-box tests** (using the exact same `package mypkg` as the source code) to allow testing of unexported fields, functions, and internal state.
- Generate idiomatic **table-driven tests** using Go's standard `testing` package combined with `github.com/stretchr/testify/assert` and `require`.
- Mock external dependencies manually by generating in-file struct stubs that implement the required interfaces (no third-party mocking frameworks).
- Write integration and unit tests for HTTP handlers using the `net/http/httptest` package.
- Implement context handling, timeout checks, and concurrency testing (via `t.Parallel()`).

## When to use me
- When a new Go package, function, struct, or HTTP handler is created.
- When you need to increase test coverage for an existing backend module.
- When establishing regression tests for a recently fixed bug to ensure it doesn't happen again.
- When refactoring code and you need baseline tests to ensure behavior remains consistent.

## Execution Guidelines (How I work)

1. **Package Scope (White-box):** 
   Always declare the test file in the exact same package as the code being tested (e.g., `package user` instead of `package user_test`).

2. **Assertions (`testify`):** 
   Always import `github.com/stretchr/testify/assert` and/or `github.com/stretchr/testify/require`. Use `require` for fatal checks (like asserting an error did/didn't happen before proceeding) and `assert` for value comparisons.

3. **Manual Mocking Pattern:** 
   When a function depends on an interface, create a local mock struct with function fields. This allows individual test cases to easily define custom behavior for the mock.
   ```go
   // Pattern I will use for mocking dependencies:
   type mockRepository struct {
       GetByIDFunc func(ctx context.Context, id string) (*User, error)
   }
   func (m *mockRepository) GetByID(ctx context.Context, id string) (*User, error) {
       if m.GetByIDFunc != nil {
           return m.GetByIDFunc(ctx, id)
       }
       return nil, nil // Default fallback
   }
   ```

4. **Table-Driven Structure:** 
   I will structure tests cleanly, injecting the manual mocks into the `struct` setup.
   ```go
   func TestFunctionName(t *testing.T) {
       t.Parallel() // Use parallel where safe

       tests := []struct {
           name      string
           mockSetup func(*mockRepository) // Setup manual mocks
           args      args
           want      string
           wantErr   bool
       }{
           {
               name: "happy path",
               mockSetup: func(m *mockRepository) {
                   m.GetByIDFunc = func(ctx context.Context, id string) (*User, error) {
                       return &User{Name: "Alice"}, nil
                   }
               },
               args:    args{id: "123"},
               want:    "Alice",
               wantErr: false,
           },
       }
       for _, tt := range tests {
           tt := tt // capture range variable
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()
               
               repo := &mockRepository{}
               if tt.mockSetup != nil {
                   tt.mockSetup(repo)
               }
               
               // Execute
               got, err := FunctionName(context.Background(), repo, tt.args.id)
               
               // Assert using testify
               if tt.wantErr {
                   require.Error(t, err)
               } else {
                   require.NoError(t, err)
                   assert.Equal(t, tt.want, got)
               }
           })
       }
   }
   ```

5. **HTTP Handlers:** 
   For API endpoints, I will use `httptest.NewRecorder()` and `httptest.NewRequest()`, parse the JSON response body, and validate the HTTP status codes using `assert.Equal(t, http.StatusOK, rr.Code)`.
