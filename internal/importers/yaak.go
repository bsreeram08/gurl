package importers

import (
"encoding/base64"
"encoding/json"
"fmt"
"os"

"github.com/sreeram/gurl/pkg/types"
)

type YaakImporter struct{}

func (i *YaakImporter) Name() string { return "yaak" }

func (i *YaakImporter) Extensions() []string { return []string{".json"} }

type YaakExport struct {
Version      string            `json:"version"`
Collections  []YaakCollection  `json:"collections"`
Environments []YaakEnvironment `json:"environments"`
}

type YaakCollection struct {
ID       string        `json:"id"`
Name     string        `json:"name"`
Folders  []YaakFolder  `json:"folders"`
Requests []YaakRequest `json:"requests"`
}

type YaakFolder struct {
ID   string `json:"id"`
Name string `json:"name"`
}

type YaakRequest struct {
ID       string         `json:"id"`
Name     string         `json:"name"`
Method   string         `json:"method"`
URL      string         `json:"url"`
Headers  []YaakHeader   `json:"headers"`
Body     *YaakBody      `json:"body"`
FolderID string         `json:"folderId"`
Auth     *YaakAuth      `json:"auth"`
}

type YaakHeader struct {
Key   string `json:"key"`
Value string `json:"value"`
}

type YaakBody struct {
Mode string `json:"mode"`
Raw  string `json:"raw"`
}

type YaakAuth struct {
Type   string      `json:"type"`
Basic  *YaakBasic  `json:"basic"`
Bearer *YaakBearer `json:"bearer"`
APIKey *YaakAPIKey `json:"apiKey"`
}

type YaakBasic struct {
Username string `json:"username"`
Password string `json:"password"`
}

type YaakBearer struct {
Token string `json:"token"`
}

type YaakAPIKey struct {
Key      string `json:"key"`
Value    string `json:"value"`
Location string `json:"location"`
}

type YaakEnvironment struct {
ID        string            `json:"id"`
Name      string            `json:"name"`
Variables map[string]string `json:"variables"`
}

func (i *YaakImporter) Parse(path string) ([]*types.SavedRequest, error) {
data, err := os.ReadFile(path)
if err != nil {
return nil, fmt.Errorf("read file: %w", err)
}

var export YaakExport
if err := json.Unmarshal(data, &export); err != nil {
return nil, fmt.Errorf("parse yaak export: %w", err)
}

return i.convertToRequests(&export), nil
}

func (i *YaakImporter) convertToRequests(export *YaakExport) []*types.SavedRequest {
var requests []*types.SavedRequest
folders := make(map[string]string)
for _, col := range export.Collections {
for _, fld := range col.Folders {
folders[fld.ID] = fld.Name
}
}

for _, col := range export.Collections {
for _, req := range col.Requests {
savedReq := i.requestToSavedRequest(&req, folders, col.Name)
requests = append(requests, savedReq)
}
}

return requests
}

func (i *YaakImporter) requestToSavedRequest(req *YaakRequest, folders map[string]string, collectionName string) *types.SavedRequest {
var headers []types.Header
for _, h := range req.Headers {
headers = append(headers, types.Header{Key: h.Key, Value: h.Value})
}

if req.Auth != nil {
switch req.Auth.Type {
case "basic":
if req.Auth.Basic != nil {
encoded := base64.StdEncoding.EncodeToString([]byte(req.Auth.Basic.Username + ":" + req.Auth.Basic.Password))
headers = append(headers, types.Header{Key: "Authorization", Value: "Basic " + encoded})
}
case "bearer":
if req.Auth.Bearer != nil {
headers = append(headers, types.Header{Key: "Authorization", Value: "Bearer " + req.Auth.Bearer.Token})
}
case "apiKey":
if req.Auth.APIKey != nil && req.Auth.APIKey.Location == "header" {
headers = append(headers, types.Header{Key: req.Auth.APIKey.Key, Value: req.Auth.APIKey.Value})
}
}
}

body := ""
if req.Body != nil {
body = req.Body.Raw
}

collection := collectionName
if req.FolderID != "" {
if folderName, ok := folders[req.FolderID]; ok {
collection = folderName
}
}

var tags []string
if collection != "" {
tags = append(tags, collection)
}

method := req.Method
if method == "" {
method = "GET"
}

return &types.SavedRequest{
Name:       req.Name,
URL:        req.URL,
Method:     method,
Headers:    headers,
Body:       body,
Collection: collection,
Tags:       tags,
}
}
