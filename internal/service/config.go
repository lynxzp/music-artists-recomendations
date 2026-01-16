package service

import (
	"fmt"
	"os"
	"reflect"
)

func loadConfig(config any) error {
	//filename := os.Args[1]
	//
	//// Read and decode TOML file
	//_, err := toml.DecodeFile(filename, config)
	//if err != nil {
	//	return fmt.Errorf("failed to decode TOML: %w", err)
	//}

	// Get API_KEY from environment
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil // No API key set, skip setting field
	}

	// Use reflection to set APIKey field
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to struct")
	}

	// Find and set APIKey field
	field := v.FieldByName("APIKey")
	if !field.IsValid() {
		return fmt.Errorf("APIKey field not found in struct")
	}

	if !field.CanSet() {
		return fmt.Errorf("APIKey field cannot be set (not exported?)")
	}

	if field.Kind() != reflect.String {
		return fmt.Errorf("APIKey field must be string type")
	}

	field.SetString(apiKey)

	return nil
}
