package email

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew New
/*func TestNew(t *testing.T) {
	t.Error("il y a une erreur")
}*/

func getMailReader(file string) *os.File {
	fd, err := os.Open("samples/" + file)
	if err != nil {
		log.Fatal(err)
	}
	return fd
}

func TestReadMessage(t *testing.T) {
	reader := getMailReader("testgetbody.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
}

func TestRaw(t *testing.T) {
	reader := getMailReader("raw.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	raw, err := email.Raw()
	require.NoError(t, err)
	require.Equal(t, "rawmessage\n", string(raw))

}

//
func TestGetRawHeaders(t *testing.T) {
	reader := getMailReader("rawheaders.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	raw, err := email.GetRawheaders()
	require.NoError(t, err)
	require.Equal(t, "header\nheader", string(raw))

}

func TestGetHeaders(t *testing.T) {
	reader := getMailReader("test-headers.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	hdr, err := email.GetHeaders("x-test-multi")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "a"}, hdr)
	h, err := email.GetHeader("x-test-multi")
	require.NoError(t, err)
	assert.Equal(t, h, "a")
}

// TestGetContentType
func TestGetContentType(t *testing.T) {
	reader := getMailReader("multipart-text-html.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	contentType, _, err := email.GetContentType()
	assert.NoError(t, err)
	assert.Equal(t, "multipart/related", contentType)
}

func TestGetPayloads(t *testing.T) {
	reader := getMailReader("base64.eml")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	err := email.GetPayloads()
	require.NoError(t, err)
}

// TestGetBody test GetBody
func TestGetRawBody(t *testing.T) {
	var err error
	reader := getMailReader("testgetbody.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	body, err := email.GetRawBody()
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
	reader := getMailReader("html.txt")
	defer reader.Close()
	email := New()
	defer email.Close()
	require.NoError(t, email.ReadMessage(reader))
	domains, err := email.GetDomains()
	require.NoError(t, err)
	//log.Println(domains)
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
