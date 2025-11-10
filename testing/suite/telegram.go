package suite

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func ParseRequestBody(t *testing.T, request *http.Request) map[string]string {
	reader, err := request.MultipartReader()
	require.NoError(t, err)

	form := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		value, _ := io.ReadAll(part)
		form[part.FormName()] = string(value)
	}

	return form
}
