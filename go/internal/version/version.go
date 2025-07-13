package version

import (
	"fmt"
	"runtime"
)

// ビルド時に ldflags で設定される変数
var (
	// Version アプリケーションのバージョン
	Version = "dev"
	
	// Commit Git commit SHA
	Commit = "unknown"
	
	// Date ビルド日時
	Date = "unknown"
	
	// GoVersion Go version
	GoVersion = runtime.Version()
)

// Info バージョン情報を構造体で返す
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
}

// Get バージョン情報を取得
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: GoVersion,
	}
}

// String バージョン情報を文字列で返す
func String() string {
	return fmt.Sprintf("Version: %s\nCommit: %s\nBuilt: %s\nGo: %s",
		Version, Commit, Date, GoVersion)
}

// Short 短いバージョン情報を返す
func Short() string {
	if len(Commit) >= 7 {
		return fmt.Sprintf("%s (%s)", Version, Commit[:7])
	}
	return fmt.Sprintf("%s (%s)", Version, Commit)
}
