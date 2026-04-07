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
