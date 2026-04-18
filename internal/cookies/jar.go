package cookies

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	Expires  int64  `json:"expires"` // Unix timestamp, 0 means session cookie
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

	if err := cj.cleanExpired(); err != nil {
		return nil, fmt.Errorf("failed to clean expired cookies: %w", err)
	}

	return cj, nil
}

// cleanExpired removes expired cookies from the database
func (c *CookieJar) cleanExpired() error {
	now := time.Now().Unix()
	entries, err := c.db.GetAllCookies()
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Expires > 0 && e.Expires < now {
			if err := c.db.DeleteCookie(e.Domain, e.Name); err != nil {
				return err
			}
		}
	}

	return nil
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
		var expires int64
		if !cookie.Expires.IsZero() {
			expires = cookie.Expires.Unix()
		}
		entries[i] = &cookieJarEntry{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   domain,
			Path:     path,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
			Expires:  expires,
		}
	}

	c.db.SetCookies(entries)
}

func (c *CookieJar) Cookies(u *url.URL) []*http.Cookie {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Filter out expired cookies
	now := time.Now().Unix()
	var validCookies []*http.Cookie
	for _, cookie := range c.jar.Cookies(u) {
		if !cookie.Expires.IsZero() && cookie.Expires.Unix() < now {
			continue // Skip expired
		}
		validCookies = append(validCookies, cookie)
	}
	return validCookies
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
		var expires time.Time
		if e.Expires > 0 {
			expires = time.Unix(e.Expires, 0)
		}
		newJar.SetCookies(u, []*http.Cookie{
			{
				Name:     e.Name,
				Value:    e.Value,
				Domain:   e.Domain,
				Path:     e.Path,
				Secure:   e.Secure,
				HttpOnly: e.HttpOnly,
				Expires:  expires,
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
	return c.db.Write(batch, &opt.WriteOptions{Sync: true})
}

func (c *LMDBCookieDB) GetAllCookies() ([]*cookieJarEntry, error) {
	return c.GetAllCookiesWithPrefix(cookieKeyPrefix)
}

// GetAllCookiesWithPrefix returns all cookies with keys having the given prefix
func (c *LMDBCookieDB) GetAllCookiesWithPrefix(prefix string) ([]*cookieJarEntry, error) {
	var result []*cookieJarEntry
	iter := c.db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := string(iter.Key())
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
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

// GetAllCookiesForDomain returns all cookies for a specific domain using prefix iteration
func (c *LMDBCookieDB) GetAllCookiesForDomain(domain string) ([]*cookieJarEntry, error) {
	prefix := fmt.Sprintf("%s%s:", cookieKeyPrefix, domain)
	return c.GetAllCookiesWithPrefix(prefix)
}

func (c *LMDBCookieDB) DeleteCookie(host, name string) error {
	prefix := fmt.Sprintf("%s%s:", cookieKeyPrefix, host)
	iter := c.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Seek([]byte(prefix)); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		// Check if key still has our prefix (might be at a different key)
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			break
		}
		// Parse path:name from key: cookie:{host}:{path}:{name}
		// The prefix is cookie:{host}:, remaining is {path}:{name}
		remaining := key[len(prefix):]
		// Find the last colon which separates path from name
		colonIdx := strings.LastIndex(remaining, ":")
		if colonIdx == -1 {
			continue
		}
		cookieName := remaining[colonIdx+1:]
		if cookieName == name {
			// Found exact match - delete it directly
			return c.db.Delete(iter.Key(), &opt.WriteOptions{Sync: true})
		}
	}
	return nil // Cookie not found
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
	return c.db.Write(batch, &opt.WriteOptions{Sync: true})
}
