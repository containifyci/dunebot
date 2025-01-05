package compare

func Complement(actual []string, required []string) []string {
	actualSet := make(map[string]struct{})
	for _, s := range actual {
		actualSet[s] = struct{}{}
	}

	seen := make(map[string]struct{})
	var result []string
	for _, s := range required {
		if _, ok := actualSet[s]; !ok {
			if _, alreadySeen := seen[s]; !alreadySeen {
				result = append(result, s)
				seen[s] = struct{}{}
			}
		}
	}
	return result
}
