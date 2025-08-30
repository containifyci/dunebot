package version

import "fmt"

type Version struct {
	version string
	commit  string
	date    string
}

// Initialize with defaults
var instance = &Version{
	version: "dev",
	commit:  "none",
	date:    "unknown",
}

func Init(version, commit, date string) {
	instance = &Version{
		version: version,
		commit:  commit,
		date:    date,
	}
}

func GetVersion() string {
	return instance.version
}

func GetCommit() string {
	return instance.commit
}

func GetDate() string {
	return instance.date
}

func String() string {
	return fmt.Sprintf("Version: %s\nCommit:  %s\nDate:    %s", instance.version, instance.commit, instance.date)
}
