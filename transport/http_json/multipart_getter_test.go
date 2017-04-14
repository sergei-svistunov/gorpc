package http_json

import (
	"bytes"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func prepareRequest(t testing.TB, filename string, size int) (*http.Request, error) {
	var err error

	bodyBuff := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuff)
	contentType := writer.FormDataContentType()

	err = writer.WriteField("some field", filename)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	// Add file
	fileWriter, err := writer.CreateFormFile("file", filename)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	file := make([]byte, size)
	_, err = rand.Read(file)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	_, err = fileWriter.Write(file)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	request := httptest.NewRequest("POST", "/test", bodyBuff)
	request.Header.Set("Content-Type", contentType)

	return request, nil
}

func TestMultipartGetter_Parse(t *testing.T) {
	filename := "100.csv"

	request, err := prepareRequest(t, filename, 100000000)
	assert.NoError(t, err)
	t.Logf("Size: %v", request.ContentLength)

	getter, err := NewMultipartGetter(request)
	assert.NoError(t, err)

	err = getter.Parse()
	assert.NoError(t, err)
}

func bench(b *testing.B, size int) {
	filename := fmt.Sprintf("%d.csv", size)
	request, err := prepareRequest(b, filename, size)
	b.Logf("Size: %v", request.ContentLength)

	getter, err := NewMultipartGetter(request)
	assert.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = getter.Parse()
		assert.NoError(b, err)
	}
}

func BenchmarkMultipartGetter_Parse1Mb(b *testing.B)   { bench(b, 1000000) }
func BenchmarkMultipartGetter_Parse10Mb(b *testing.B)  { bench(b, 10000000) }
func BenchmarkMultipartGetter_Parse100Mb(b *testing.B) { bench(b, 100000000) }
