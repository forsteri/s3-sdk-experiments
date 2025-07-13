package progress

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DisplayMode 表示モード
type DisplayMode int

const (
	DisplayModeLog      DisplayMode = iota // ログ出力（定期的な進捗レポート）- デフォルト
	DisplayModeTerminal                    // ターミナル表示（プログレスバー）
	DisplayModeSilent                      // 表示なし（プログラムから利用）
)

// ProgressTracker アップロードの進捗を追跡する構造体
type ProgressTracker struct {
	mu              sync.RWMutex
	totalFiles      int64
	processedFiles  atomic.Int64
	failedFiles     atomic.Int64
	skippedFiles    atomic.Int64
	totalBytes      int64
	processedBytes  atomic.Int64
	startTime       time.Time
	activeWorkers   map[int]string // worker ID -> current file
	lastUpdateTime  time.Time
	updateInterval  time.Duration
}

// NewProgressTracker 新しいProgressTrackerを作成
func NewProgressTracker(totalFiles int, totalBytes int64) *ProgressTracker {
	return &ProgressTracker{
		totalFiles:     int64(totalFiles),
		totalBytes:     totalBytes,
		startTime:      time.Now(),
		activeWorkers:  make(map[int]string),
		updateInterval: 100 * time.Millisecond, // 100ms間隔で更新
		lastUpdateTime: time.Now(),
	}
}

// UpdateWorkerStatus ワーカーの状態を更新
func (pt *ProgressTracker) UpdateWorkerStatus(workerID int, fileName string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if fileName == "" {
		delete(pt.activeWorkers, workerID)
	} else {
		pt.activeWorkers[workerID] = fileName
	}
}

// IncrementProcessed 処理済みファイル数をインクリメント
func (pt *ProgressTracker) IncrementProcessed(bytes int64) {
	pt.processedFiles.Add(1)
	pt.processedBytes.Add(bytes)
}

// IncrementFailed 失敗ファイル数をインクリメント
func (pt *ProgressTracker) IncrementFailed() {
	pt.failedFiles.Add(1)
}

// IncrementSkipped スキップファイル数をインクリメント
func (pt *ProgressTracker) IncrementSkipped() {
	pt.skippedFiles.Add(1)
}

// ShouldUpdate 更新すべきかどうかを判定
func (pt *ProgressTracker) ShouldUpdate() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	return time.Since(pt.lastUpdateTime) >= pt.updateInterval
}

// GetProgressBar プログレスバーの文字列を生成
func (pt *ProgressTracker) GetProgressBar(width int) string {
	processed := pt.processedFiles.Load()
	failed := pt.failedFiles.Load()
	skipped := pt.skippedFiles.Load()
	completed := processed + failed + skipped
	
	if pt.totalFiles == 0 {
		return ""
	}
	
	percentage := float64(completed) / float64(pt.totalFiles) * 100
	filled := int(float64(width) * percentage / 100)
	
	bar := strings.Builder{}
	bar.WriteByte('[')
	
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteByte('=')
		} else if i == filled {
			bar.WriteByte('>')
		} else {
			bar.WriteByte(' ')
		}
	}
	
	bar.WriteByte(']')
	
	return fmt.Sprintf("%s %.1f%%", bar.String(), percentage)
}

// GetStats 現在の統計情報を取得
func (pt *ProgressTracker) GetStats() Stats {
	processed := pt.processedFiles.Load()
	failed := pt.failedFiles.Load()
	skipped := pt.skippedFiles.Load()
	processedBytes := pt.processedBytes.Load()
	
	elapsed := time.Since(pt.startTime)
	var speed float64
	if elapsed.Seconds() > 0 {
		speed = float64(processedBytes) / elapsed.Seconds()
	}
	
	// 残り時間の推定
	completed := processed + failed + skipped
	if completed > 0 && pt.totalFiles > 0 {
		avgTimePerFile := elapsed / time.Duration(completed)
		remainingFiles := pt.totalFiles - completed
		eta := avgTimePerFile * time.Duration(remainingFiles)
		
		return Stats{
			TotalFiles:     pt.totalFiles,
			ProcessedFiles: processed,
			FailedFiles:    failed,
			SkippedFiles:   skipped,
			TotalBytes:     pt.totalBytes,
			ProcessedBytes: processedBytes,
			Speed:          speed,
			Elapsed:        elapsed,
			ETA:            eta,
		}
	}
	
	return Stats{
		TotalFiles:     pt.totalFiles,
		ProcessedFiles: processed,
		FailedFiles:    failed,
		SkippedFiles:   skipped,
		TotalBytes:     pt.totalBytes,
		ProcessedBytes: processedBytes,
		Speed:          speed,
		Elapsed:        elapsed,
		ETA:            -1, // 計算不可
	}
}

// GetActiveWorkers アクティブなワーカーの情報を取得
func (pt *ProgressTracker) GetActiveWorkers() map[int]string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	// マップのコピーを返す
	workers := make(map[int]string)
	for k, v := range pt.activeWorkers {
		workers[k] = v
	}
	return workers
}

// UpdateLastTime 最終更新時刻を更新
func (pt *ProgressTracker) UpdateLastTime() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.lastUpdateTime = time.Now()
}

// Stats 統計情報
type Stats struct {
	TotalFiles     int64
	ProcessedFiles int64
	FailedFiles    int64
	SkippedFiles   int64
	TotalBytes     int64
	ProcessedBytes int64
	Speed          float64 // bytes per second
	Elapsed        time.Duration
	ETA            time.Duration
}

// Logger ログ出力のインターフェース
type Logger interface {
	Info(msg string, args ...interface{})
}

// DisplayOption 表示オプション
type DisplayOption func(*ProgressDisplay)

// WithDisplayMode 表示モードを設定
func WithDisplayMode(mode DisplayMode) DisplayOption {
	return func(pd *ProgressDisplay) {
		pd.mode = mode
	}
}

// WithLogger ロガーを設定
func WithLogger(logger Logger) DisplayOption {
	return func(pd *ProgressDisplay) {
		pd.logger = logger
	}
}

// WithLogInterval ログ出力間隔を設定
func WithLogInterval(interval time.Duration) DisplayOption {
	return func(pd *ProgressDisplay) {
		pd.logInterval = interval
	}
}

// ProgressDisplay プログレス表示を管理
type ProgressDisplay struct {
	tracker      *ProgressTracker
	stop         chan struct{}
	wg           sync.WaitGroup
	mode         DisplayMode
	logger       Logger // ログ出力用のインターフェース
	logInterval  time.Duration
}

// NewProgressDisplay 新しいProgressDisplayを作成
func NewProgressDisplay(tracker *ProgressTracker, opts ...DisplayOption) *ProgressDisplay {
	pd := &ProgressDisplay{
		tracker:     tracker,
		stop:        make(chan struct{}),
		mode:        DisplayModeLog, // デフォルトはログモード（サーバー環境向け）
		logInterval: 30 * time.Second, // デフォルトは30秒間隔（サーバー向けに長めに）
	}
	
	for _, opt := range opts {
		opt(pd)
	}
	
	return pd
}

// Start プログレス表示を開始
func (pd *ProgressDisplay) Start() {
	pd.wg.Add(1)
	go pd.displayLoop()
}

// Stop プログレス表示を停止
func (pd *ProgressDisplay) Stop() {
	close(pd.stop)
	pd.wg.Wait()
	
	// 最終状態を表示
	switch pd.mode {
	case DisplayModeTerminal:
		pd.displayProgress()
		fmt.Println() // 改行
	case DisplayModeLog:
		pd.logProgress() // 最終状態をログ出力
	}
}

// displayLoop 表示ループ
func (pd *ProgressDisplay) displayLoop() {
	defer pd.wg.Done()
	
	var ticker *time.Ticker
	switch pd.mode {
	case DisplayModeTerminal:
		ticker = time.NewTicker(100 * time.Millisecond)
	case DisplayModeLog:
		ticker = time.NewTicker(pd.logInterval)
	case DisplayModeSilent:
		return // 何も表示しない
	}
	defer ticker.Stop()
	
	for {
		select {
		case <-pd.stop:
			return
		case <-ticker.C:
			switch pd.mode {
			case DisplayModeTerminal:
				if pd.tracker.ShouldUpdate() {
					pd.displayProgress()
					pd.tracker.UpdateLastTime()
				}
			case DisplayModeLog:
				pd.logProgress()
			}
		}
	}
}

// displayProgress 進捗を表示
func (pd *ProgressDisplay) displayProgress() {
	stats := pd.tracker.GetStats()
	workers := pd.tracker.GetActiveWorkers()
	
	// カーソルを行頭に移動してクリア
	fmt.Print("\r\033[K")
	
	// プログレスバーを表示
	bar := pd.tracker.GetProgressBar(40)
	fmt.Printf("%s ", bar)
	
	// 統計情報を表示
	fmt.Printf("[%d/%d files, %s, %s/s",
		stats.ProcessedFiles+stats.FailedFiles+stats.SkippedFiles,
		stats.TotalFiles,
		formatBytes(stats.ProcessedBytes),
		formatBytes(int64(stats.Speed)),
	)
	
	// ETAを表示
	if stats.ETA > 0 {
		fmt.Printf(", ETA: %s", formatDuration(stats.ETA))
	}
	
	fmt.Print("]")
	
	// アクティブワーカー数を表示
	if len(workers) > 0 {
		fmt.Printf(" [%d workers active]", len(workers))
	}
}

// logProgress ログに進捗を出力
func (pd *ProgressDisplay) logProgress() {
	if pd.logger == nil {
		return
	}
	
	stats := pd.tracker.GetStats()
	completed := stats.ProcessedFiles + stats.FailedFiles + stats.SkippedFiles
	
	pd.logger.Info("Upload progress",
		"completed", completed,
		"total", stats.TotalFiles,
		"processed", stats.ProcessedFiles,
		"failed", stats.FailedFiles,
		"skipped", stats.SkippedFiles,
		"bytes_processed", stats.ProcessedBytes,
		"speed_mbps", fmt.Sprintf("%.2f", stats.Speed/1024/1024),
		"elapsed", formatDuration(stats.Elapsed),
		"eta", formatDuration(stats.ETA),
		"percentage", fmt.Sprintf("%.1f%%", float64(completed)/float64(stats.TotalFiles)*100),
	)
}

// formatBytes バイト数を人間が読みやすい形式に変換
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatDuration 時間を人間が読みやすい形式に変換
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}
