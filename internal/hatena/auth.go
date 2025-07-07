package hatena

import (
	"encoding/base64"
	"fmt"
)

func BasicAuth(hatenaID, apiKey string) string {
	credentials := fmt.Sprintf("%s:%s", hatenaID, apiKey)
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return fmt.Sprintf("Basic %s", encoded)
}