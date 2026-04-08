MANUAL QA REPORT
================
Binary: /tmp/gurl-test
Date: Wed Apr 08 2026
Environment: GURL_DATA_DIR=/tmp/gurl-qa-test

SCENARIOS:

1. Save+Run: PASS
   Command: /tmp/gurl-test save "users" https://jsonplaceholder.typicode.com/users
   Exit: 0
   Output: ✓ Saved request 'users'

   Command: /tmp/gurl-test run "users"
   Exit: 0
   Output: JSON array with 10 users returned

2. Curl import: PASS
   Command: /tmp/gurl-test save "create-post" --curl "curl -X POST ..."
   Exit: 0
   Output: ✓ Saved request 'create-post'

   Command: /tmp/gurl-test run "create-post"
   Exit: 0
   Output: {"id":101} - POST returns 201

3. Environment: FAIL
   Command: /tmp/gurl-test env create dev --var "BASE_URL=..." --secret "API_KEY=..."
   Exit: 1
   Error: "flag provided but not defined: -var"
   Notes: The 'env' CLI command is NOT registered in main.go. Only internal storage exists.
          Environment storage is used by RunCommand and CollectionCommand but not exposed as CLI.

4. List and search: PASS
   Command: /tmp/gurl-test list
   Exit: 0
   Output: Table showing 2 requests (users, create-post)

   Command: /tmp/gurl-test list --json
   Exit: 0
   Output: JSON array with full request metadata

5. Edit: PASS
   Command: /tmp/gurl-test edit "users" --tag "v1"
   Exit: 0
   Output: ✓ Updated request 'users', added tag 'v1'

   Command: /tmp/gurl-test list --tag "v1"
   Exit: 0
   Output: Filtered list showing only users with v1 tag

6. History: PASS
   Command: /tmp/gurl-test run "users"
   Exit: 0

   Command: /tmp/gurl-test history "users"
   Exit: 0 (had to retry due to DB lock on first attempt)
   Output: Table with 2 history entries showing status, duration, size, timestamp

7. Diff: PASS
   Command: /tmp/gurl-test run "users"
   Exit: 0

   Command: /tmp/gurl-test diff "users"
   Exit: 0
   Output: "No differences found (JSONs are semantically identical)"

8. Export+Import: FAIL
   Command: /tmp/gurl-test export --all --output /tmp/gurl-qa-export.json
   Exit: 0
   Output: ✓ Exported 2 request(s) to /tmp/gurl-qa-export.json

   Command: /tmp/gurl-test delete "users"
   Exit: 0

   Command: /tmp/gurl-test import /tmp/gurl-qa-export.json
   Exit: 1
   Error: "import failed: unsupported file format: ..json"
   Notes: The import command does not support gurl's native export format.
          It only supports OpenAPI, Insomnia, Postman, Bruno, and HAR formats.
          Round-trip export/import not supported.

9. Codegen: PASS
   Command: /tmp/gurl-test codegen "users" --lang curl
   Exit: 0
   Output: curl -X GET 'https://jsonplaceholder.typicode.com/users'

   Command: /tmp/gurl-test codegen "users" --lang python
   Exit: 0
   Output: Valid Python requests code

   Command: /tmp/gurl-test codegen "users" --lang go
   Exit: 0
   Output: Valid Go http code

10. Collection: FAIL
    Command: /tmp/gurl-test save "posts" https://jsonplaceholder.typicode.com/posts -c "blog-api"
    Exit: 0

    Command: /tmp/gurl-test collection list
    Exit: 0
    Output: "No collections found."

    Command: /tmp/gurl-test list --collection "blog-api"
    Exit: 0
    Output: "No saved requests found."
    Notes: Collection creation/save/filter appears broken. Collection is not persisted correctly.

11. Assertions: PASS
    Command: /tmp/gurl-test run "users" --assert "status=200"
    Exit: 0
    Output: "=== Assertions: 1 passed, 0 failed ===" followed by JSON response

12. Format options: FAIL
    Command: /tmp/gurl-test run "users" --format json
    Exit: 0
    Output: JSON (works)

    Command: /tmp/gurl-test run "users" --format table
    Exit: 0
    Output: Still returns JSON, not table format
    Notes: The --format table flag does not change output to table format

13. GraphQL: PASS
    Command: /tmp/gurl-test graphql "https://countries.trevorblades.com" --query '{ countries { name code } }'
    Exit: 0
    Output: {"countries":[{"code":"AD","name":"Andorra"},...]}

14. Help and version: PASS
    Command: /tmp/gurl-test --help
    Exit: 0
    Output: Full help text with command list

    Command: /tmp/gurl-test --version
    Exit: 0
    Output: "gurl version dev"

INTEGRATION: 10/14 pass

EDGE CASES TESTED:
- Database locking (handled gracefully with retry)
- Empty state (clean database works)
- Invalid flags (proper error messages)

VERDICT: FAIL

CRITICAL ISSUES:
1. env CLI command not registered (missing feature)
2. Collection CRUD completely broken (cannot list collections, cannot filter by collection)
3. Export format not importable (gurl native format not supported for re-import)
4. --format table flag does not work (outputs JSON regardless)
