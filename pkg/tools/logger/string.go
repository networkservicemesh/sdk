package logger

import "strings"

func format(v ...interface{}) string {
	return strings.Trim(strings.Repeat("%+v"+separator, len(v)), separator)
}
func limitString(s string) string {
	if len(s) > maxStringLength {
		return s[maxStringLength-dotCount:] + strings.Repeat(".", dotCount)
	}
	return s
}

func levelName(level loggerLevel) string {
	levelNames := map[loggerLevel]string{
		INFO:  "info",
		WARN:  "warn",
		ERROR: "error",
		FATAL: "fatal",
	}
	return levelNames[level]
}
