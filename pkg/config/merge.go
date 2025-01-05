package config

import (
	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v2"
)

func merge(cnt ...[]byte) ([]byte, error) {
	var mergedData interface{}

	for _, c := range cnt {
		data, err := read(c)
		if err != nil {
			log.Error().Err(err).Msgf("Error unmarshal %s", err)
			return nil, err
		}

		mergeDeep(&mergedData, &data)
	}

	mergedYAML, err := yaml.Marshal(mergedData)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling merged data to YAML: %s", err)
		return nil, err
	}

	return mergedYAML, nil
}

func read(data []byte) (interface{}, error) {
	var result interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func mergeDeep(dest, src *interface{}) {
	switch destVal := (*dest).(type) {
	case nil:
		// If destination is nil, copy from source
		*dest = *src
	case map[interface{}]interface{}:
		srcMap, srcMapOk := (*src).(map[interface{}]interface{})
		if srcMapOk {
			// Both values are maps, recursively merge
			for key, srcValue := range srcMap {
				destValue, destValueOk := destVal[key]
				if destValueOk {
					// Key exists in dest, merge recursively
					mergeDeep(&destValue, &srcValue)
					destVal[key] = destValue
				} else {
					// Key doesn't exist in dest, simply copy from src
					destVal[key] = srcValue
				}
			}
		} else {
			// If src is not a map, overwrite dest
			*dest = *src
		}
	default:
		// If dest is not a map, overwrite dest with src
		*dest = *src
	}
}
