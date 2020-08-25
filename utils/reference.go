package utils

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// GenerateReferenceNumber produces a random reference number in the format of [A-Z]{2}[0-9]{8}
func GenerateReferenceNumber() string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 2)
	for i := 0; i < 2; i++ {
		b[i] = chars[rand.Intn(len(chars))]
	}

	return string(b) + fmt.Sprintf("%08d", rand.Intn(99999999))
}

// GenerateEtag generates a random etag which is generated on every write action
func GenerateEtag() (string, error) {
	// Get a random number and the time in seconds and milliseconds
	timeNow := time.Now()
	rand.Seed(timeNow.UTC().UnixNano())
	randomNumber := fmt.Sprintf("%07d", rand.Intn(9999999))
	timeInMillis := strconv.FormatInt(timeNow.UnixNano()/int64(time.Millisecond), 10)
	timeInSeconds := strconv.FormatInt(timeNow.UnixNano()/int64(time.Second), 10)
	// Calculate a SHA-512 truncated digest
	shaDigest := sha512.New512_224()
	_, err := shaDigest.Write([]byte(randomNumber + timeInMillis + timeInSeconds))
	if err != nil {
		return "", fmt.Errorf("error writing sha digest: [%s]", err)
	}
	sha1Hash := hex.EncodeToString(shaDigest.Sum(nil))
	return sha1Hash, nil
}
