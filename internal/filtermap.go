package internal

func filterMap(m map[string]interface{}, filter ...string) {
	for key := range m {
		contained, subfilter := contains(filter, key)
		if contained {
			switch t := m[key].(type) {
			case map[string]interface{}:
				if len(subfilter) > 0 {
					filterMap(t, subfilter)
				}
			default:
				// can"t apply subfilter
			}
		}

		if !contained {
			delete(m, key)
		}
	}
}
