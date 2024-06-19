package shortener

import "math/rand"

// excludes zero, uppercase 'O', uppercase 'I', and lowercase 'l'
var readerFriendlyCharset = []byte(
	"123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ_-",
)

func generateShortUrl(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = readerFriendlyCharset[rand.Int63()%int64(len(readerFriendlyCharset))]
	}
	return string(b)
}
