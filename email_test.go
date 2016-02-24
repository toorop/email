package email

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMessage(path string) Email {
	message, err := NewFromFile(path)
	if err != nil {
		panic(err)
	}
	return message

}

//
func TestNewFromFile(t *testing.T) {
	message, err := NewFromFile("samples/testgetbody.txt")
	require.NoError(t, err)
	defer message.Close()
}

func TestNewFromByte(t *testing.T) {
	fd, err := os.Open("samples/testgetbody.txt")
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	mBytes, err := ioutil.ReadAll(fd)
	if err != nil {
		panic(err)
	}
	message, err := NewFromByte(mBytes)
	require.NoError(t, err)
	defer message.Close()
}

func TestRaw(t *testing.T) {
	message := newMessage("samples/raw.txt")
	defer message.Close()
	raw, err := message.Raw()
	require.NoError(t, err)
	require.Equal(t, "rawmessage\n", string(raw))
}

//
func TestGetRawHeaders(t *testing.T) {
	message := newMessage("samples/rawheaders.txt")
	defer message.Close()
	raw, err := message.GetRawHeaders()
	require.NoError(t, err)
	require.Equal(t, "header\nheader", string(raw))
}

func TestGetHeaders(t *testing.T) {
	message := newMessage("samples/test-headers.txt")
	defer message.Close()
	hdr, err := message.GetHeaders("x-test-multi")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "a"}, hdr)
	h, err := message.GetHeader("x-test-multi")
	require.NoError(t, err)
	assert.Equal(t, h, "a")
}

// TestGetContentType
func TestGetContentType(t *testing.T) {
	message := newMessage("samples/multipart-text-html.txt")
	defer message.Close()
	contentType, _, err := message.GetContentType()
	assert.NoError(t, err)
	assert.Equal(t, "multipart/related", contentType)
}

func TestGetPayloads(t *testing.T) {
	message := newMessage("samples/base64.eml")
	defer message.Close()
	err := message.GetPayloads()
	require.NoError(t, err)
}

// TestGetBody test GetBody
func TestGetRawBody(t *testing.T) {
	message := newMessage("samples/testgetbody.txt")
	defer message.Close()
	body, err := message.GetRawBody()
	require.NoError(t, err)
	bodyStr := string(body[:len(body)-1])
	assert.Equal(t, "The Best Play!", bodyStr)
}

func TestGetDomains(t *testing.T) {
	var err error
	expectedDomains := map[string]int{
		"protecmail.com": 5,
		"tedmailing1.fr": 1,
		"bacori1.fr":     8,
		"majuscul1.fr":   1,
	}
	message := newMessage("samples/html.txt")
	defer message.Close()

	domains, err := message.GetDomains()
	require.NoError(t, err)
	for d, o := range expectedDomains {
		_, found := domains[d]
		require.True(t, found)
		assert.Equal(t, o, domains[d])
	}
	for d, o := range domains {
		_, found := expectedDomains[d]
		require.True(t, found)
		assert.Equal(t, o, expectedDomains[d])
	}
}
