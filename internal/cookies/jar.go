package cookies

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"golang.org/x/net/publicsuffix"
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
	// DomainCookie is set when the cookie had a valid Domain attribute, and
	// then Domain is that attribute (no leading dot) and the cookie matches
	// subdomains. Otherwise the cookie is host-only and Domain is the
	// request host, port included. Entries written before this field
	// existed load as host-only, the narrower scope.
	DomainCookie bool `json:"domain_cookie,omitempty"`
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

// newMemoryJar returns the in-memory jar. The public suffix list makes it
// reject Domain attributes such as "com" or "co.uk".
func newMemoryJar() (*cookiejar.Jar, error) {
	return cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
}

func NewCookieJar(db CookieDB) (*CookieJar, error) {
	jar, err := newMemoryJar()
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
		u, cookie := storedCookie(e)
		c.jar.SetCookies(u, []*http.Cookie{cookie})
	}

	return nil
}

// storedCookie rebuilds the URL and cookie that replay a stored entry into
// the in-memory jar. A host-only cookie gets no Domain attribute, so the jar
// keeps it host-only for the stored host (the URL host, port allowed); a
// domain cookie gets its domain back.
func storedCookie(e *cookieJarEntry) (*url.URL, *http.Cookie) {
	u := &url.URL{Scheme: "https", Host: strings.TrimPrefix(e.Domain, "."), Path: e.Path}
	var expires time.Time
	if e.Expires > 0 {
		expires = time.Unix(e.Expires, 0)
	}
	cookie := &http.Cookie{
		Name:     e.Name,
		Value:    e.Value,
		Path:     e.Path,
		Secure:   e.Secure,
		HttpOnly: e.HttpOnly,
		Expires:  expires,
	}
	if e.DomainCookie {
		cookie.Domain = u.Hostname()
	}
	return u, cookie
}

// cookieDomain applies the Domain attribute rules of RFC 6265 section 5.3,
// as net/http/cookiejar does with a public suffix list, to a cookie set by
// a response from host. It returns the domain a domain cookie applies to,
// or hostOnly when the cookie only applies to host. ok is false when the
// jar rejects the cookie: a Domain the host does not domain-match (another
// site) or a public suffix such as "com".
func cookieDomain(host, attr string) (domain string, hostOnly, ok bool) {
	if attr == "" {
		return "", true, true
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if _, err := netip.ParseAddr(host); err == nil {
		// An IP host only accepts its own address as Domain, and the
		// cookie stays host-only.
		return "", true, attr == host
	}
	attr = strings.TrimPrefix(attr, ".")
	if attr == "" || attr[0] == '.' || attr[len(attr)-1] == '.' {
		return "", false, false
	}
	for i := 0; i < len(attr); i++ {
		if attr[i] >= 0x80 {
			return "", false, false
		}
	}
	attr = strings.ToLower(attr)
	if ps := publicsuffix.List.PublicSuffix(attr); ps != "" && !strings.HasSuffix(attr, "."+ps) {
		// attr is itself a public suffix: only allowed as the host itself,
		// and then the cookie is host-only.
		return "", true, host == attr
	}
	if host != attr && !strings.HasSuffix(host, "."+attr) {
		return "", false, false
	}
	return attr, false, true
}

// SetCookies stores cookies from a response to u in the in-memory jar and
// persists the ones the jar accepts, with the scope the jar gave them.
func (c *CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.jar.SetCookies(u, cookies)

	entries := make([]*cookieJarEntry, 0, len(cookies))
	for _, cookie := range cookies {
		attrDomain, hostOnly, ok := cookieDomain(u.Hostname(), cookie.Domain)
		if !ok {
			continue // rejected by the jar too: never persist it
		}
		domain := u.Host
		if !hostOnly {
			domain = attrDomain
		}
		path := cookie.Path
		if !strings.HasPrefix(path, "/") {
			path = defaultPath(u.Path)
		}
		var expires int64
		if !cookie.Expires.IsZero() {
			expires = cookie.Expires.Unix()
		}
		entries = append(entries, &cookieJarEntry{
			Name:         cookie.Name,
			Value:        cookie.Value,
			Domain:       domain,
			Path:         path,
			Secure:       cookie.Secure,
			HttpOnly:     cookie.HttpOnly,
			Expires:      expires,
			DomainCookie: !hostOnly,
		})
	}
	if len(entries) == 0 {
		return
	}

	c.db.SetCookies(entries)
}

// defaultPath is the RFC 6265 section 5.1.4 default-path of a cookie set
// without a Path attribute: the request path up to its last "/".
func defaultPath(p string) string {
	i := strings.LastIndex(p, "/")
	if i <= 0 {
		return "/"
	}
	return p[:i]
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

	newJar, err := newMemoryJar()
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

	newJar, err := newMemoryJar()
	if err != nil {
		return fmt.Errorf("failed to recreate jar: %w", err)
	}

	for _, e := range entries {
		u, cookie := storedCookie(e)
		newJar.SetCookies(u, []*http.Cookie{cookie})
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
