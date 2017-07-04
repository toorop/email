package email

// todo
// - parse on init

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"

	"github.com/mvdan/xurls"
)

const (
	crlf = "\r\n"
)

// TempDir is the working dir (where we save email as file)
//var TempDir string

// Email represents an email
type Email struct {
	locker *sync.Mutex
	file   *os.File

	//Raw     []byte
	Headers Header

	// todo slice -> map
	Parts []multipart.Part

	ContentType       string
	ContentTypeParams map[string]string
	IsMultipart       bool
}

// NewFromReader TODO

// NewFromFile returns email from file
func NewFromFile(path string) (m Email, err error) {
	m = Email{
		locker: new(sync.Mutex),
	}
	fd, err := os.Open(path)
	if err != nil {
		return
	}
	defer fd.Close()
	if m.file, err = ioutil.TempFile("", ""); err != nil {
		return
	}
	r := lf2crlf(fd)
	_, err = io.Copy(m.file, &r)
	m.file.Seek(0, 0)
	err = m.parse()
	return
}

// NewFromByte returns email from []byte
func NewFromByte(messageBytes []byte) (m Email, err error) {
	m = Email{
		locker: new(sync.Mutex),
	}
	r := lf2crlf(bytes.NewReader(messageBytes))
	if m.file, err = ioutil.TempFile("", ""); err != nil {
		return
	}
	_, err = io.Copy(m.file, &r)
	m.file.Seek(0, 0)
	err = m.parse()
	return
}

// NewFromString retuns email from a string
func NewFromString(messageStr string) (m Email, err error) {
	return NewFromByte([]byte(messageStr))
}

// parse
func (m *Email) parse() (err error) {
	// headers
	if err = m.parseHeader(); err != nil {
		return
	}

	// Content-type
	hdrCt, err := m.GetHeader("Content-Type")
	if err != nil {
		return
	}

	if hdrCt == "" {
		body, err := m.GetRawBody()
		if err != nil {
			return err
		}
		hdrCt = http.DetectContentType(body)
		println(hdrCt)
	}

	m.ContentType, m.ContentTypeParams, err = mime.ParseMediaType(hdrCt)
	if err != nil {
		return
	}
	m.IsMultipart = strings.HasPrefix(m.ContentType, "multipart/")

	// parts
	m.Parts = []multipart.Part{}
	if m.IsMultipart {
		var body []byte
		body, err = m.GetRawBody()
		if err != nil {
			return
		}
		mr := multipart.NewReader(bytes.NewReader(body), m.ContentTypeParams["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				err = nil
				break
			}
			if err != nil {
				return err
			}
			m.Parts = append(m.Parts, *part)
		}
	}
	return
}

// Close is an explicit finalizer
// it close Email.reader and remove temporary files
func (m *Email) Close() error {
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

// RawFromFile return email as raw []byte
func (m *Email) RawFromFile() (raw []byte, err error) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, err = m.file.Seek(0, 0); err != nil {
		return
	}
	raw, err = ioutil.ReadAll(m.file)
	return
}

// RawFromStruct return email as raw []byte
func (m *Email) RawFromStruct() (raw []byte, err error) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, err = m.file.Seek(0, 0); err != nil {
		return
	}
	raw, err = ioutil.ReadAll(m.file)
	return
}

// parseHeader parse headers
func (m *Email) parseHeader() (err error) {
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
	m.Headers = Header(hdr)
	return nil
}

// GetRawHeaders returns headers as []byte
func (m *Email) GetRawHeaders() ([]byte, error) {
	var err error
	var prev byte
	var lfcrSeen bool
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
		if prev == 13 && buf[0] == 10 {
			if lfcrSeen {
				break
			}
			lfcrSeen = true
		} else if prev != 10 {
			lfcrSeen = false
		}
		headers = append(headers, buf[0])
		prev = buf[0]
	}
	return headers[:len(headers)-3], nil
}

// GetHeaders returns valueS for header key key
func (m *Email) GetHeaders(key string) (headers []string, err error) {
	headers, _ = m.Headers[textproto.CanonicalMIMEHeaderKey(key)]
	return
}

// GetHeader returns first value for header key key
func (m *Email) GetHeader(key string) (string, error) {
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
func (m *Email) GetRawBody() (body []byte, err error) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if _, err = m.file.Seek(0, 0); err != nil {
		return body, err
	}
	tp := textproto.NewReader(bufio.NewReader(m.file))
	if _, err = tp.ReadMIMEHeader(); err != nil {
		return
	}
	return ioutil.ReadAll(tp.R)
}

// replace LF by CRLF
// todo -> utils.go
func lf2crlf(in io.Reader) (out bytes.Buffer) {
	var prev byte
	buf := make([]byte, 1)

	for {
		_, err := in.Read(buf)
		// EOF
		if err != nil {
			break
		}
		if buf[0] == 10 && prev != 13 {
			out.Write([]byte{13, 10})
		} else {
			out.Write(buf)
		}
		prev = buf[0]
	}
	return out
}

// GetDomains returns un slice of domains names found in email src
func (m *Email) GetDomains() (domains map[string]int, err error) {
	domains = make(map[string]int)
	var parts []string
	raw, err := m.RawFromFile()
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

/*
// GetContentType returns content-type of the message
func (m *Email) GetContentType() (contentType string, params map[string]string, err error) {
	hdrCt, err := m.GetHeader("Content-Type")
	if err != nil {
		return
	}
	return mime.ParseMediaType(hdrCt)
}*/

/*
// IsMultipart checks if mail is multipart
func (m *Email) isMultipart() (bool, error) {
	contentType, _, err := m.GetContentType()
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(contentType, "multipart"), nil
}
*/

/*
// GetParts returns messages parts
func (m *Email) GetParts() (parts []multipart.Part, err error) {
	var body []byte
	body, err = m.GetRawBody()
	if err != nil {
		return parts, err
	}

	contentType, params, err := m.GetContentType()
	if err != nil {
		return
	}

	// todo use IsMultipart
	if strings.HasPrefix(contentType, "multipart") {
		mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				err = nil
				break
			}
			if err != nil {
				return parts, err
			}
			parts = append(parts, *part)
		}
	} else {
		// fake multipart to have "part" type
		var hdr, ct, te string
		if hdr, err = m.GetHeader("Content-Type"); err != nil {
			return parts, err
		}
		ct = hdr
		if hdr, err = m.GetHeader("Content-transfer-encoding"); err != nil {
			return parts, err
		}
		te = hdr

		msgStr := "Content-Type: multipart/mixed; boundary=foo\r\n\r\n"
		msgStr += "--foo\r\n"
		msgStr += "Content-Type: " + ct + "\r\n"
		msgStr += "Content-transfer-encoding: " + te + "\r\n\r\n"
		msgStr += string(body) + "\r\n--foo--"
		msg, err := NewFromString(msgStr)
		if err != nil {
			return parts, err
		}
		return msg.GetParts()
	}
	return
}
*/
