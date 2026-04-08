package cookies

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type mockCookieDB struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMockCookieDB() *mockCookieDB {
	return &mockCookieDB{data: make(map[string][]byte)}
}

func (m *mockCookieDB) Open() error  { return nil }
func (m *mockCookieDB) Close() error { return nil }
func (m *mockCookieDB) SetCookies(cookies []*cookieJarEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range cookies {
		data, err := json.Marshal(c)
		if err != nil {
			return err
		}
		key := cookieKeyPrefix + c.Domain + ":" + c.Path + ":" + c.Name
		m.data[key] = data
	}
	return nil
}
func (m *mockCookieDB) GetAllCookies() ([]*cookieJarEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*cookieJarEntry
	for _, data := range m.data {
		var c cookieJarEntry
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		result = append(result, &c)
	}
	return result, nil
}
func (m *mockCookieDB) DeleteCookie(host, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, data := range m.data {
		var c cookieJarEntry
		if json.Unmarshal(data, &c) == nil && c.Domain == host && c.Name == name {
			delete(m.data, k)
		}
	}
	return nil
}
func (m *mockCookieDB) ClearAllCookies() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	return nil
}

type leveldbCookieDB struct {
	db     *leveldb.DB
	dbPath string
}

func newLeveldbCookieDB(t *testing.T) *leveldbCookieDB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cookies.db")
	db, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		t.Fatalf("failed to open leveldb: %v", err)
	}
	return &leveldbCookieDB{db: db, dbPath: dbPath}
}

func (d *leveldbCookieDB) Open() error { return nil }
func (d *leveldbCookieDB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
func (d *leveldbCookieDB) SetCookies(cookies []*cookieJarEntry) error {
	batch := new(leveldb.Batch)
	for _, c := range cookies {
		data, err := json.Marshal(c)
		if err != nil {
			return err
		}
		key := cookieKeyPrefix + c.Domain + ":" + c.Path + ":" + c.Name
		batch.Put([]byte(key), data)
	}
	return d.db.Write(batch, nil)
}
func (d *leveldbCookieDB) GetAllCookies() ([]*cookieJarEntry, error) {
	var result []*cookieJarEntry
	iter := d.db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := string(iter.Key())
		if len(key) < len(cookieKeyPrefix) || key[:len(cookieKeyPrefix)] != cookieKeyPrefix {
			continue
		}
		var entry cookieJarEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		result = append(result, &entry)
	}
	return result, nil
}
func (d *leveldbCookieDB) DeleteCookie(host, name string) error {
	iter := d.db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := string(iter.Key())
		var entry cookieJarEntry
		if json.Unmarshal(iter.Value(), &entry) == nil && entry.Domain == host && entry.Name == name {
			d.db.Delete([]byte(key), nil)
		}
	}
	return nil
}
func (d *leveldbCookieDB) ClearAllCookies() error {
	iter := d.db.NewIterator(nil, nil)
	defer iter.Release()
	batch := new(leveldb.Batch)
	for iter.Next() {
		key := string(iter.Key())
		if len(key) >= len(cookieKeyPrefix) && key[:len(cookieKeyPrefix)] == cookieKeyPrefix {
			batch.Delete(iter.Key())
		}
	}
	return d.db.Write(batch, nil)
}

func TestCookieJar_SavesCookiesFromResponse(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "session", Value: "abc123"},
	})

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(saved))
	}
	if saved[0].Name != "session" || saved[0].Value != "abc123" {
		t.Errorf("unexpected cookie: %+v", saved[0])
	}
}

func TestCookieJar_SendsCookiesOnSubsequentRequests(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u1, _ := url.Parse("https://example.com/")
	jar.SetCookies(u1, []*http.Cookie{
		{Name: "session", Value: "abc123"},
	})

	u2, _ := url.Parse("https://example.com/api")
	got := jar.Cookies(u2)
	if len(got) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(got))
	}
	if got[0].Name != "session" || got[0].Value != "abc123" {
		t.Errorf("unexpected cookie: %+v", got[0])
	}
}

func TestCookieJar_PersistsAcrossSessions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cookies.db")

	db1, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		t.Fatalf("failed to open leveldb: %v", err)
	}

	persistDB := &leveldbCookieDB{db: db1}
	jar1, err := NewCookieJar(persistDB)
	if err != nil {
		t.Fatalf("failed to create jar1: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar1.SetCookies(u, []*http.Cookie{
		{Name: "persist", Value: "yes"},
	})
	db1.Close()

	db2, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		t.Fatalf("failed to open leveldb: %v", err)
	}
	persistDB2 := &leveldbCookieDB{db: db2}
	jar2, err := NewCookieJar(persistDB2)
	if err != nil {
		t.Fatalf("failed to create jar2: %v", err)
	}
	defer db2.Close()

	u2, _ := url.Parse("https://example.com/")
	got := jar2.Cookies(u2)
	if len(got) != 1 {
		t.Fatalf("expected 1 cookie after reload, got %d", len(got))
	}
	if got[0].Value != "yes" {
		t.Errorf("expected persist=yes, got %s", got[0].Value)
	}
}

func TestCookieJar_RespectsExpiry(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	expiredTime := time.Now().Add(-1 * time.Hour)
	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "expired", Value: "yes", Expires: expiredTime},
	})

	got := jar.Cookies(u)
	found := false
	for _, c := range got {
		if c.Name == "expired" {
			found = true
			break
		}
	}
	if found {
		t.Error("expected expired cookie not to be returned")
	}
}

func TestCookieJar_ClearAll(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
	})

	if err := jar.ClearAll(); err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	got := jar.Cookies(u)
	if len(got) != 0 {
		t.Errorf("expected 0 cookies after clear, got %d", len(got))
	}
}

func TestCookieJar_List(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "session", Value: "abc"},
	})

	u2, _ := url.Parse("https://other.com/")
	jar.SetCookies(u2, []*http.Cookie{
		{Name: "token", Value: "xyz"},
	})

	cookies, err := jar.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(cookies))
	}
}

func TestLMDBCookieDB_NewAndPersist(t *testing.T) {
	db := newLeveldbCookieDB(t)
	defer db.Close()

	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "lmdb", Value: "works"},
	})

	got := jar.Cookies(u)
	if len(got) != 1 || got[0].Value != "works" {
		t.Errorf("unexpected cookies: %+v", got)
	}
}

func TestCookieJar_ImplementsHTTPCookieJarInterface(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	var _ interface{} = jar
	_ = struct {
		Jar interface {
			SetCookies(u *url.URL, cookies []*http.Cookie)
			Cookies(u *url.URL) []*http.Cookie
		}
	}{jar}
}

func TestCookieJar_DeleteCookie(t *testing.T) {
	db := newLeveldbCookieDB(t)
	defer db.Close()

	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "todelete", Value: "yes"},
	})

	db.DeleteCookie("example.com", "todelete")

	cookies, err := jar.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(cookies) != 0 {
		t.Errorf("expected 0 cookies after delete, got %d", len(cookies))
	}
}

func TestCookieJar_DeleteCookieFromDifferentHost(t *testing.T) {
	db := newLeveldbCookieDB(t)
	defer db.Close()

	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create jar: %v", err)
	}

	u1, _ := url.Parse("https://example.com/")
	u2, _ := url.Parse("https://other.com/")

	jar.SetCookies(u1, []*http.Cookie{
		{Name: "session", Value: "abc123", Domain: "example.com"},
	})
	jar.SetCookies(u2, []*http.Cookie{
		{Name: "token", Value: "xyz789", Domain: "other.com"},
	})

	err = jar.DeleteCookie("example.com", "session")
	if err != nil {
		t.Fatalf("DeleteCookie failed: %v", err)
	}

	cookies, err := jar.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(cookies) != 1 {
		t.Errorf("expected 1 cookie after deleting from example.com, got %d", len(cookies))
	}

	remainingCookie := cookies[0]
	if remainingCookie.Name != "token" || remainingCookie.Value != "xyz789" {
		t.Errorf("expected remaining cookie to be token=xyz789, got %+v", remainingCookie)
	}
}

func TestCookieJar_DeleteNonexistentCookie(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "existing", Value: "value"},
	})

	err = jar.DeleteCookie("example.com", "nonexistent")
	if err != nil {
		t.Fatalf("DeleteCookie should not fail for nonexistent cookie: %v", err)
	}

	cookies, err := jar.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(cookies))
	}
}

func TestCookieJar_CookieAttributes(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://secure.example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "secure_cookie", Value: "secret", Secure: true, HttpOnly: true, Path: "/api"},
	})

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(saved))
	}

	if !saved[0].Secure {
		t.Error("expected cookie to be marked as Secure")
	}
	if !saved[0].HttpOnly {
		t.Error("expected cookie to be marked as HttpOnly")
	}
	if saved[0].Path != "/api" {
		t.Errorf("expected path=/api, got %s", saved[0].Path)
	}
}

func TestCookieJar_DomainMatching(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	jar.SetCookies(urlMustParse("https://example.com/"), []*http.Cookie{
		{Name: "domain_cookie", Value: "value", Domain: ".example.com"},
	})

	if got := jar.Cookies(urlMustParse("https://sub.example.com/")); len(got) == 0 {
		t.Error("expected cookie to be sent to subdomains")
	}
	if got := jar.Cookies(urlMustParse("https://other.com/")); len(got) != 0 {
		t.Error("expected cookie not to be sent to other domains")
	}
}

func urlMustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestLMDBCookieDB_SetCookies(t *testing.T) {
	db := newLeveldbCookieDB(t)
	defer db.Close()

	entries := []*cookieJarEntry{
		{Name: "cookie1", Value: "val1", Domain: "example.com", Path: "/", Secure: false, HttpOnly: false},
		{Name: "cookie2", Value: "val2", Domain: "example.com", Path: "/api", Secure: true, HttpOnly: true},
	}

	err := db.SetCookies(entries)
	if err != nil {
		t.Fatalf("SetCookies failed: %v", err)
	}

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies failed: %v", err)
	}
	if len(saved) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(saved))
	}
}

func TestLMDBCookieDB_GetAllCookies_EmptyDB(t *testing.T) {
	db := newLeveldbCookieDB(t)
	defer db.Close()

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies failed: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("expected 0 cookies, got %d", len(saved))
	}
}

func TestLMDBCookieDB_DeleteCookie(t *testing.T) {
	db := newLeveldbCookieDB(t)

	entries := []*cookieJarEntry{
		{Name: "todelete", Value: "val", Domain: "example.com", Path: "/"},
	}
	db.SetCookies(entries)

	err := db.DeleteCookie("example.com", "todelete")
	if err != nil {
		t.Fatalf("DeleteCookie failed: %v", err)
	}

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies failed: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("expected 0 cookies after delete, got %d", len(saved))
	}

	db.Close()
}

func TestLMDBCookieDB_ClearAllCookies(t *testing.T) {
	db := newLeveldbCookieDB(t)

	entries := []*cookieJarEntry{
		{Name: "cookie1", Value: "val1", Domain: "example.com", Path: "/"},
		{Name: "cookie2", Value: "val2", Domain: "other.com", Path: "/"},
	}
	db.SetCookies(entries)

	err := db.ClearAllCookies()
	if err != nil {
		t.Fatalf("ClearAllCookies failed: %v", err)
	}

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("GetAllCookies failed: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("expected 0 cookies after clear, got %d", len(saved))
	}

	db.Close()
}

func TestLMDBCookieDB_OpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_cookies.db")

	impl := &LMDBCookieDB{dbPath: dbPath}

	err := impl.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	err = impl.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestLMDBCookieDB_CloseWithoutOpen(t *testing.T) {
	impl := &LMDBCookieDB{dbPath: "/tmp/nonexistent/db/path"}

	err := impl.Close()
	if err != nil {
		t.Fatalf("Close should not fail: %v", err)
	}
}

func TestCookieJar_SetCookiesWithDefaults(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/api/v1")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "session", Value: "abc123"},
	})

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}

	if len(saved) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(saved))
	}

	if saved[0].Domain != "example.com" {
		t.Errorf("expected domain=example.com, got %s", saved[0].Domain)
	}

	if saved[0].Path != "/api/v1" {
		t.Errorf("expected path=/api/v1, got %s", saved[0].Path)
	}
}

func TestCookieJar_SetCookiesWithEmptyPath(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "test", Value: "value", Path: ""},
	})

	saved, err := db.GetAllCookies()
	if err != nil {
		t.Fatalf("failed to get cookies: %v", err)
	}

	if saved[0].Path != "/" {
		t.Errorf("expected default path=/, got %s", saved[0].Path)
	}
}

func TestCookieJar_EmptyCollection(t *testing.T) {
	db := newMockCookieDB()
	jar, err := NewCookieJar(db)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	u, _ := url.Parse("https://example.com/")
	got := jar.Cookies(u)
	if len(got) != 0 {
		t.Errorf("expected 0 cookies for empty jar, got %d", len(got))
	}
}

func TestCookieJar_ClearAllPersists(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "clear_test.db")

	db1, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		t.Fatalf("failed to open leveldb: %v", err)
	}
	persistDB := &leveldbCookieDB{db: db1}

	jar1, err := NewCookieJar(persistDB)
	if err != nil {
		t.Fatalf("failed to create jar1: %v", err)
	}

	jar1.SetCookies(urlMustParse("https://example.com/"), []*http.Cookie{
		{Name: "persistent", Value: "value"},
	})

	jar1.ClearAll()
	db1.Close()

	db2, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		t.Fatalf("failed to open leveldb: %v", err)
	}
	persistDB2 := &leveldbCookieDB{db: db2}

	jar2, err := NewCookieJar(persistDB2)
	if err != nil {
		t.Fatalf("failed to create jar2: %v", err)
	}
	defer db2.Close()

	got := jar2.Cookies(urlMustParse("https://example.com/"))
	if len(got) != 0 {
		t.Errorf("expected 0 cookies after clear and reopen, got %d", len(got))
	}
}
