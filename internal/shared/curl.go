// Utilities for parsing cURL commands.
package shared

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CurlHeaders represents parsed headers and cookies from a cURL command.
type CurlHeaders struct {
	Headers map[string]string
	Cookie  string
}

// ParseCurlFile reads a .sh file containing a cURL command and extracts headers.
func ParseCurlFile(filepath string) (*CurlHeaders, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read curl file: %w", err)
	}

	return ParseCurlCommand(content)
}

// ParseCurlCommand parses a cURL command string and extracts headers.
func ParseCurlCommand(data []byte) (*CurlHeaders, error) {
	curlCmd := string(data)
	curlCmd = strings.ReplaceAll(curlCmd, "\\\n", " ")
	curlCmd = strings.ReplaceAll(curlCmd, "\\", "")

	headers := make(map[string]string)
	var cookie string

	headerRegex := regexp.MustCompile(`-H\s+'([^']+)'|-H\s+"([^"]+)"`)
	matches := headerRegex.FindAllStringSubmatch(curlCmd, -1)

	for _, match := range matches {
		var headerLine string
		if match[1] != "" {
			headerLine = match[1]
		} else {
			headerLine = match[2]
		}

		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if strings.ToLower(key) != "cookie" {
				headers[key] = value
			}
		}
	}

	cookieRegex := regexp.MustCompile(`-b\s+'([^']+)'|-b\s+"([^"]+)"`)
	cookieMatches := cookieRegex.FindStringSubmatch(curlCmd)
	if len(cookieMatches) > 1 {
		if cookieMatches[1] != "" {
			cookie = cookieMatches[1]
		} else {
			cookie = cookieMatches[2]
		}
	}

	if cookie == "" {
		for _, match := range matches {
			var headerLine string
			if match[1] != "" {
				headerLine = match[1]
			} else {
				headerLine = match[2]
			}

			if strings.HasPrefix(strings.ToLower(headerLine), "cookie:") {
				parts := strings.SplitN(headerLine, ":", 2)
				if len(parts) == 2 {
					cookie = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}

	if len(headers) == 0 && cookie == "" {
		return nil, fmt.Errorf("no headers found in curl command")
	}

	return &CurlHeaders{
		Headers: headers,
		Cookie:  cookie,
	}, nil
}

// ToHeadersRaw converts parsed headers to headers_raw format for ytmusicapi.
//
// Format is newline-separated "Key: Value" pairs.
func (c *CurlHeaders) ToHeadersRaw() string {
	var lines []string

	for key, value := range c.Headers {
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}

	if c.Cookie != "" {
		lines = append(lines, fmt.Sprintf("cookie: %s", c.Cookie))
	}

	return strings.Join(lines, "\n")
}
