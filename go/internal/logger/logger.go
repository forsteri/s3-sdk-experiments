package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"s3-uploader/internal/models"
)

var (
	// グローバルロガー（Python版と同じアプローチ）
	globalLogger *Logger
)

// Logger wraps slog.Logger
type Logger struct {
	*slog.Logger
}

// ロガーの初期化
func Setup(config models.LoggingConfig) (*Logger, error) {
	// ログレベルをパース
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// ハンドラーのオプション
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}
			}
			return a
		},
	}

	// ハンドラーを作成
	var handler slog.Handler

	// 複数の出力先を設定
	var writers []io.Writer

	// コンソール出力は常に有効
	writers = append(writers, os.Stdout)

	// ファイル出力（設定されている場合）
	if config.File != nil && *config.File != "" {
		// ディレクトリが存在しない場合は作成
		dir := filepath.Dir(*config.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// ログファイルを開く
		file, err := os.OpenFile(*config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// 注意: このファイルハンドルはアプリケーションの終了時まで開いたままにする必要があります
		// 必要に応じて、graceful shutdownの実装を検討してください

		writers = append(writers, file)
	}

	// マルチライターを作成
	multiWriter := io.MultiWriter(writers...)

	// フォーマットに応じてハンドラーを選択
	// TODO: format文字列に"asctime"が含まれるかでハンドラーを判定するのは適切でない
	//       将来的には専用のhandlerフィールドを追加して明示的に指定できるようにすべき
	//       例: handler: "text" | "json"
	if strings.Contains(config.Format, "asctime") {
		// テキストハンドラー（人間が読みやすい形式）
		handler = slog.NewTextHandler(multiWriter, opts)
	} else {
		// JSONハンドラー（構造化ログ）
		handler = slog.NewJSONHandler(multiWriter, opts)
	}

	// ロガーを作成
	logger := &Logger{
		Logger: slog.New(handler),
	}

	// グローバルロガーとして設定
	globalLogger = logger
	slog.SetDefault(logger.Logger)

	return logger, nil
}

// GetLogger returns the global logger
func GetLogger() *Logger {
	if globalLogger == nil {
		panic("Logger not initialized. Call Setup() first.")
	}
	return globalLogger
}

// parseLogLevel converts string level to slog.Level
func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARNING", "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}

// Python版と同じメソッドを提供
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *Logger) Warning(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Fatalf はPython版にはないが、Goでは便利
func (l *Logger) Fatalf(format string, args ...any) {
	l.Logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
