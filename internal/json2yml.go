package internal

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func json2yaml(str string) (string, error) {
	helper := &map[string]interface{}{}
	err := json.Unmarshal([]byte(str), helper)
	if err != nil {
		return "", err
	}
	bt, err := yaml.Marshal(helper)
	if err != nil {
		return "", err
	}
	return string(bt), nil
}
