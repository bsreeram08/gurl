package importers

import (
"os"
"testing"
)

func TestYaakImporterName(t *testing.T) {
i := &YaakImporter{}
if i.Name() != "yaak" {
t.Errorf("expected yaak, got %s", i.Name())
}
}

func TestYaakImporterExtensions(t *testing.T) {
i := &YaakImporter{}
exts := i.Extensions()
if len(exts) != 1 || exts[0] != ".json" {
t.Errorf("expected [.json], got %v", exts)
}
}

func TestYaakParse(t *testing.T) {
content := `{
"version": "1.0",
"collections": [{
"id": "col_1",
"name": "Test API",
"folders": [{"id": "fld_1", "name": "Users"}],
"requests": [{
"id": "req_1",
"name": "Get User",
"method": "GET",
"url": "https://api.example.com/users",
"headers": [{"key": "Accept", "value": "application/json"}],
"folderId": "fld_1"
}]
}]
}`

tmpfile, err := os.CreateTemp("", "yaak-*.json")
if err != nil {
t.Fatal(err)
}
defer os.Remove(tmpfile.Name())

if _, err := tmpfile.Write([]byte(content)); err != nil {
t.Fatal(err)
}
tmpfile.Close()

i := &YaakImporter{}
reqs, err := i.Parse(tmpfile.Name())
if err != nil {
t.Fatal(err)
}

if len(reqs) != 1 {
t.Fatalf("expected 1 request, got %d", len(reqs))
}

if reqs[0].Name != "Get User" {
t.Errorf("expected name 'Get User', got %s", reqs[0].Name)
}

if reqs[0].Method != "GET" {
t.Errorf("expected method GET, got %s", reqs[0].Method)
}

if reqs[0].Collection != "Users" {
t.Errorf("expected collection 'Users', got %s", reqs[0].Collection)
}
}
