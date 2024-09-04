package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var (
	values []string
)

type Pair struct {
	Key   string
	Value interface{}
}

// func printAllValues(pairs []Pair) {
// 	for _, pair := range pairs {
// 		switch v := pair.Value.(type) {
// 		case map[string]interface{}:
// 			// Mengonversi nested map menjadi slice pairs
// 			subpairs := mapToPairs(v)
// 			printAllValues(subpairs)
// 		default:
// 			data := fmt.Sprintf("%v", pair.Value)
// 			values = append(values, data)
// 		}
// 	}
// }

// func mapToPairs(m map[string]interface{}) []Pair {
// 	pairs := make([]Pair, 0, len(m))
// 	for key, value := range m {
// 		pairs = append(pairs, Pair{Key: key, Value: value})
// 	}
// 	return pairs
// }

func extractSubTokens(m map[string]interface{}) []string {
	var subTokens []string
	for _, v := range m {
		switch val := v.(type) {
		case string:
			subTokens = append(subTokens, val)
		case map[string]interface{}:
			subTokens = append(subTokens, extractSubTokens(val)...)
		case []interface{}:
			for _, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					subTokens = append(subTokens, extractSubTokens(itemMap)...)
				}
			}
		}
	}
	return subTokens
}

func parseJSONToPairs(data []byte) (string, error) {
	// Use json.Decoder to read JSON elements in the original order
	decoder := json.NewDecoder(bytes.NewReader(data))
	var tokens []string
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}

		switch t.(type) {
		case json.Delim: // Start or end of an object/array
			continue
		case string: // Key name
			// Read the value after the key
			if decoder.More() {
				val, _ := decoder.Token()
				switch v := val.(type) {
				case string:
					tokens = append(tokens, v)
				case float64:
					tokens = append(tokens, fmt.Sprintf("%.0f", v))
				case map[string]interface{}:
					subTokens := extractSubTokens(v)
					tokens = append(tokens, subTokens...)
				case []interface{}:
					for _, item := range v {
						if itemMap, ok := item.(map[string]interface{}); ok {
							subTokens := extractSubTokens(itemMap)
							tokens = append(tokens, subTokens...)
						}
					}
				}
			}
		}
	}

	// Join the result with slashes
	return strings.Join(tokens, "/"), nil
}

func generatePayload(payload []byte) (resp string, err error) {

	var compactedJSON bytes.Buffer

	// Gunakan json.Compact untuk menghilangkan spasi dan newline
	err = json.Compact(&compactedJSON, payload)
	if err != nil {
		fmt.Println("Error compacting JSON:", err)
		return
	}

	resp, err = parseJSONToPairs(compactedJSON.Bytes())
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func main() {
	app := fiber.New()

	app.Post("/generate-signature", func(c *fiber.Ctx) error {

		payload := c.BodyRaw()

		if !strings.EqualFold(string(payload), "") {
			generatedPayload, err := generatePayload(payload)
			if err != nil {
				return err
			}

			values = append(values, generatedPayload)
		}

		Xtimestamp := c.Get("X-TIMESTAMP")

		values = append(values, Xtimestamp)
		values = append(values, "TEST")

		generateMessage := strings.Join(values, "/")
		values = []string{}

		key := []byte("X-SIGNATURE")
		message := []byte(generateMessage)
		h := hmac.New(sha256.New, key)
		h.Write(message)
		mac := h.Sum(nil)

		encodedKey := hex.EncodeToString(mac)

		return c.Status(http.StatusOK).JSON(map[string]interface{}{
			"parsed_data": generateMessage,
			"signtature":  encodedKey,
		})
	})

	app.Listen(os.Getenv("APP_HOST"))
}
