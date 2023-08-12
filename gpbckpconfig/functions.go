package gpbckpconfig

import (
	"os"

	"gopkg.in/yaml.v2"
)

var execReadFile = os.ReadFile

// ReadHistoryFile Read history file.
func ReadHistoryFile(filename string) ([]byte, error) {
	data, err := execReadFile(filename)
	return data, err
}

// ParseResult Parse result to History struct.
func ParseResult(output []byte) (History, error) {
	var hData History
	err := yaml.Unmarshal(output, &hData)
	return hData, err
}
