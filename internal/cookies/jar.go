package cookies

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const cookieKeyPrefix = "cookie:"

type cookieJarEntry struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Secure   bool   `json:"secure"`
	HttpOnly bool   `json:"http_only"`
}

type CookieDB interface {
	Open() error
	Close() error
	SetCookies(cookies []*cookieJarEntry) error
	GetAllCookies() ([]*cookieJarEntry, error)
	DeleteCookie(host, name string) error
	ClearAllCookies() error
}

type CookieJar struct {
	jar *cookiejar.Jar
	db  CookieDB
	mu  sync.Mutex
}

func NewCookieJar(db CookieDB) (*CookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	cj := &CookieJar{
		jar: jar,
		db:  db,
	}

	if err := db.Open(); err != nil {
		return nil, fmt.Errorf("failed to open cookie db: %w", err)
	}

	if err := cj.loadFromDB(); err != nil {
		return nil, fmt.Errorf("failed to load cookies from db: %w", err)
	}

	return cj, nil
}

func (c *CookieJar) loadFromDB() error {
	entries, err := c.db.GetAllCookies()
	if err != nil {
		return fmt.Errorf("failed to get cookies from db: %w", err)
	}

	for _, e := range entries {
		u := &url.URL{Scheme: "https", Host: e.Domain, Path: e.Path}
		cookies := []*http.Cookie{
			{
				Name:     e.Name,
				Value:    e.Value,
				Domain:   e.Domain,
				Path:     e.Path,
				Secure:   e.Secure,
				HttpOnly: e.HttpOnly,
			},
		}
		c.jar.SetCookies(u, cookies)
	}

	return nil
}

func (c *CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.jar.SetCookies(u, cookies)

	entries := make([]*cookieJarEntry, len(cookies))
	for i, cookie := range cookies {
		domain := cookie.Domain
		if domain == "" {
			domain = u.Host
		}
		path := cookie.Path
		if path == "" {
			path = u.Path
		}
		if path == "" {
			path = "/"
		}
		entries[i] = &cookieJarEntry{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   domain,
			Path:     path,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
		}
	}

	c.db.SetCookies(entries)
}

func (c *CookieJar) Cookies(u *url.URL) []*http.Cookie {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.jar.Cookies(u)
}

func (c *CookieJar) ClearAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.db.ClearAllCookies(); err != nil {
		return err
	}

	newJar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to recreate jar: %w", err)
	}
	c.jar = newJar
	return nil
}

func (c *CookieJar) DeleteCookie(host, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.db.DeleteCookie(host, name); err != nil {
		return err
	}

	entries, err := c.db.GetAllCookies()
	if err != nil {
		return err
	}

	newJar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to recreate jar: %w", err)
	}

	for _, e := range entries {
		u := &url.URL{Scheme: "https", Host: e.Domain, Path: e.Path}
		newJar.SetCookies(u, []*http.Cookie{
			{
				Name:     e.Name,
				Value:    e.Value,
				Domain:   e.Domain,
				Path:     e.Path,
				Secure:   e.Secure,
				HttpOnly: e.HttpOnly,
			},
		})
	}

	c.jar = newJar
	return nil
}

func (c *CookieJar) List() ([]*cookieJarEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db.GetAllCookies()
}

type LMDBCookieDB struct {
	db     *leveldb.DB
	dbPath string
}

func NewLMDBCookieDB() (*LMDBCookieDB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".local", "share", "gurl")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "gurl-cookies.db")
	if envPath := os.Getenv("GURL_COOKIE_DB_PATH"); envPath != "" {
		dbPath = envPath
	}

	return &LMDBCookieDB{dbPath: dbPath}, nil
}

func (c *LMDBCookieDB) Open() error {
	var err error
	c.db, err = leveldb.OpenFile(c.dbPath, &opt.Options{
		WriteBuffer: 4 * 1024 * 1024,
	})
	return err
}

func (c *LMDBCookieDB) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *LMDBCookieDB) SetCookies(cookies []*cookieJarEntry) error {
	batch := new(leveldb.Batch)
	for _, cookie := range cookies {
		data, err := json.Marshal(cookie)
		if err != nil {
			return fmt.Errorf("failed to marshal cookie: %w", err)
		}
		key := fmt.Sprintf("%s%s:%s:%s", cookieKeyPrefix, cookie.Domain, cookie.Path, cookie.Name)
		batch.Put([]byte(key), data)
	}
	return c.db.Write(batch, nil)
}

func (c *LMDBCookieDB) GetAllCookies() ([]*cookieJarEntry, error) {
	var result []*cookieJarEntry
	iter := c.db.NewIterator(nil, nil)
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

func (c *LMDBCookieDB) DeleteCookie(host, name string) error {
	iter := c.db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := string(iter.Key())
		var entry cookieJarEntry
		if json.Unmarshal(iter.Value(), &entry) == nil && entry.Domain == host && entry.Name == name {
			c.db.Delete([]byte(key), nil)
		}
	}
	return nil
}

func (c *LMDBCookieDB) ClearAllCookies() error {
	iter := c.db.NewIterator(nil, nil)
	defer iter.Release()
	batch := new(leveldb.Batch)
	for iter.Next() {
		key := string(iter.Key())
		if len(key) >= len(cookieKeyPrefix) && key[:len(cookieKeyPrefix)] == cookieKeyPrefix {
			batch.Delete(iter.Key())
		}
	}
	return c.db.Write(batch, nil)
}
