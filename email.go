package email

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/textproto"
	"os"
	"strings"
	"sync"

	"github.com/mvdan/xurls"
)

// Email represents an email
type email struct {
	locker           *sync.Mutex
	file             *os.File
	TempDir          string
	Header           Header
	flagHeaderParsed bool
	contentType      string
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

// parseHeader parse headers
func (m *email) parseHeader() (err error) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if _, err = m.file.Seek(0, 0); err != nil {
		return err
	}

	tp := textproto.NewReader(bufio.NewReader(m.file))
	hdr, err := tp.ReadMIMEHeader()
	if err != nil {
		return err
	}
	m.Header = Header(hdr)
	m.flagHeaderParsed = true
	return nil
}

// GetHeaders returns valueS for header key key
func (m *email) GetHeaders(key string) (headers []string, err error) {
	// if not parsed
	if !m.flagHeaderParsed {
		if err = m.parseHeader(); err != nil {
			return headers, err
		}
	}
	headers, _ = m.Header[textproto.CanonicalMIMEHeaderKey(key)]
	return
}

// GetHeader returns first value for header key key
func (m *email) GetHeader(key string) (string, error) {
	var err error
	var hs []string
	if hs, err = m.GetHeaders(key); err != nil {
		return "", err
	}
	if len(hs) == 0 {
		return "", err
	}
	return hs[0], nil
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

// GetContentType returns content-type of the message
func (m *email) GetContentType() (contentType string, params map[string]string, err error) {
	/*if m.contentType != "" {
		return m.contentType, nil
	}*/
	hdrCt, err := m.GetHeader("Content-Type")
	if err != nil {
		return
	}
	return mime.ParseMediaType(hdrCt)
	/*if err != nil {
		m.contentType = contentType
	}
	return*/
}

// GetPayloads returns
func (m *email) GetPayloads() error {
	contentType, params, err := m.GetContentType()
	if err != nil {
		return err
	}
	println(contentType)
	for k, v := range params {
		println(k, v)
	}
	if strings.HasPrefix(contentType, "multipart") {
		body, err := m.GetRawBody()
		if err != nil {
			return err
		}
		mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			payload, err := ioutil.ReadAll(part)
			if err != nil {
				return err
			}
			println(string(payload))
		}
	}
	return nil
}

// GetDomains returns un slice of domains names found in email src
func (m *email) GetDomains() (domains map[string]int, err error) {
	domains = make(map[string]int)
	var parts []string
	raw, err := m.Raw()
	if err != nil {
		return
	}
	found := xurls.Relaxed.FindAllString(string(raw), -1)

	for _, f := range found {
		f = strings.ToLower(f)
		// email
		if strings.Index(f, "@") != -1 {
			continue
		}
		// Link http, ftp
		if i := strings.Index(f, "://"); i != -1 {
			f = f[i+3:]
		}
		// url style aka truc.com/foo/bar
		if parts = strings.Split(f, "/"); len(parts) != 1 {
			f = parts[0]
		}
		// IP ?
		if net.ParseIP(f) != nil {
			continue
		}
		// root
		parts = strings.Split(f, ".")
		lenParts := len(parts)
		if lenParts == 1 {
			continue
		}
		f = parts[lenParts-2] + "." + parts[lenParts-1]

		if _, found := domains[f]; found {
			domains[f]++
		} else {
			domains[f] = 1
		}
	}
	return
}
