package email

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/mvdan/xurls"
)

// Email represents an email
type email struct {
	locker  *sync.Mutex
	file    *os.File
	TempDir string
}

// New returns a new email
func New(tempdir ...string) *email {
	e := new(email)
	e.locker = new(sync.Mutex)
	if len(tempdir) != 0 {
		e.TempDir = tempdir[0]
	}
	return e
}

// ReadMessage read new email from io.Reader
func (m *email) ReadMessage(reader io.Reader) (err error) {
	if m.TempDir == "" {
		m.TempDir, err = ioutil.TempDir("", "emailpkg")
		if err != nil {
			return
		}
	}
	if m.file, err = ioutil.TempFile(m.TempDir, ""); err != nil {
		return
	}
	_, err = io.Copy(m.file, reader)
	m.file.Seek(0, 0)
	return
}

// Close is an explicit finalizer
// it close Email.reader and remove temporary files
func (m *email) Close() error {
	// TODO don't return on return
	if err := m.file.Close(); err != nil {
		return err
	}

	if err := os.Remove(m.file.Name()); err != nil {
		return err
	}

	// others stuff
	return nil
}

// Raw return email as raw []byte
func (m *email) Raw() (raw []byte, err error) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, err = m.file.Seek(0, 0); err != nil {
		return
	}
	raw, err = ioutil.ReadAll(m.file)
	return
}

// GetRawHeaders returns headers as []byte
func (m *email) GetRawheaders() ([]byte, error) {
	var err error
	var prev byte
	var headers []byte
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, err = m.file.Seek(0, 0); err != nil {
		return headers, err
	}
	buf := make([]byte, 1)
	for {
		if _, err = m.file.Read(buf); err != nil {
			return []byte{}, err
		}
		if prev == 10 && buf[0] == 10 {
			break
		}
		headers = append(headers, buf[0])
		prev = buf[0]
	}
	return headers[:len(headers)-1], nil
}

// GetRawBody returns body as []byte
func (m *email) GetRawBody() (body []byte, err error) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, err = m.file.Seek(0, 0); err != nil {
		return body, err
	}
	var prev byte
	buf := make([]byte, 1)

	// skip headers
	for {
		if _, err = m.file.Read(buf); err != nil {
			return body, err
		}
		if prev == 10 && buf[0] == 10 {
			break
		}
		prev = buf[0]
	}
	body, err = ioutil.ReadAll(m.file)
	return
}

// GetDomains returns un slice of domains names found in email src
func (m *email) GetDomains() (domains []string, err error) {
	raw, err := m.Raw()
	if err != nil {
		panic(err)
		return
	}
	found := xurls.Relaxed.FindAllString(string(raw), -1)

	for _, f := range found {
		f = strings.ToLower(f)
		if strings.Index(f, "@") != -1 {
			continue
		}
		domains = append(domains, f)
	}
	return
}
