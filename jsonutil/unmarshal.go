package jsonutil

import (
	"encoding/json"
	"fmt"
	"os"
)

// Unmarshals the given type from the given JSON file.
func UnmarshalFromFile[T any](filePath string) (T, error) {
	f, err := os.Open(filePath)
	if err != nil {
		var empty T
		return empty, fmt.Errorf("unable to open %q: %w", filePath, err)
	}
	defer f.Close()

	var result T
	err = json.NewDecoder(f).Decode(&result)
	if err != nil {
		var empty T
		return empty, fmt.Errorf(
			"unable to deserialize JSON from %q: %w",
			filePath,
			err,
		)
	}

	return result, nil
}
