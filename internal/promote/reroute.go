package promote

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

func alterWatch(src, dst, nextRelease string, overwrite bool) error {
	log.Printf("Rotating values from file %s to %s", src, dst)
	yamlFile, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	oldValues := make(map[string]interface{})
	unmarshalErr := yaml.Unmarshal(yamlFile, &oldValues)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	if global := oldValues["global"]; global != nil {
		managedRelease := global.(map[string]interface{})["managedRelease"]
		if !overwrite && managedRelease != nil {
			return fmt.Errorf("promote.Copy : existing value cannot be overwritten")
		}
		global.(map[string]interface{})["managedRelease"] = nextRelease
	}

	newValues, marshalErr := yaml.Marshal(&oldValues)
	if marshalErr != nil {
		return marshalErr
	}

	if writeErr := ioutil.WriteFile(dst, newValues, 0644); writeErr != nil {
		return writeErr
	}
	return nil
}

func Reroute(context *Context) {
	valuesName := "values.yaml"
	currentName := context.OldRelease + "-values.yaml"
	futureName := context.NewRelease + "-values.yaml"

	if ok := fileExists(&valuesName); ok {
		backup(valuesName, currentName, true)
		backup(valuesName, futureName, true)
		alterWatch(futureName, valuesName, context.NewRelease, true)
	} else {
		log.Fatal("values.yaml not found")
	}
}
