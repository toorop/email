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