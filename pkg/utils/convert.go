package utils

import (
	"encoding/json"
	"log"
)

// ObjectToJSON converts any object to json-string
func ObjectToJSON(v interface{}) string {
	json, err := json.Marshal(v)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	return string(json)
}
