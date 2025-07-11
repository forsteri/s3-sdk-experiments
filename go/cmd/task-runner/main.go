package main

import (
	"context"
	"flag"
	"log"
	"os"

	awsclient "s3-uploader/internal/aws"
	"s3-uploader/internal/logger"
	"s3-uploader/internal/models"
	"s3-uploader/internal/uploader"
)

func main() {
	// コマンドライン引数
	var (
		configFile = flag.String("config", "config.json", "設定ファイルのパス")
		dryRun     = flag.Bool("dry-run", false, "ドライランモード")
		taskName   = flag.String("task", "", "実行する特定のタスク名（指定しない場合は全タスク実行）")
	)
	flag.Parse()

	// 設定を読み込み
	cfg, err := models.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("設定読み込みエラー: %v", err)
	}

	// ドライランモードの上書き
	if *dryRun {
		cfg.Options.DryRun = true
	}

	// ロガーをセットアップ
	_, err = logger.Setup(cfg.Logging)
	if err != nil {
		log.Fatalf("ロガーの初期化に失敗: %v", err)
	}

	lgr := logger.GetLogger()
	lgr.Info("🚀 Task Runner started",
		"config_file", *configFile,
		"dry_run", cfg.Options.DryRun,
		"task_filter", *taskName,
	)

	// S3クライアントマネージャーを作成
	clientManager, err := awsclient.NewClientManager(cfg.AWS)
	if err != nil {
		lgr.Fatalf("S3クライアント作成エラー: %v", err)
	}

	// 特定のタスクのみを実行する場合はフィルタリング
	if *taskName != "" {
		filteredTasks := []models.UploadTask{}
		for _, task := range cfg.UploadTasks {
			if task.Name == *taskName {
				filteredTasks = append(filteredTasks, task)
				break
			}
		}
		if len(filteredTasks) == 0 {
			lgr.Fatalf("指定されたタスク '%s' が見つかりません", *taskName)
		}
		cfg.UploadTasks = filteredTasks
		lgr.Info("Running specific task", "task", *taskName)
	}

	// タスクランナーを作成
	runner := uploader.NewTaskRunner(clientManager, *cfg)

	// タスクを実行
	ctx := context.Background()
	report, err := runner.RunAllTasks(ctx)
	if err != nil {
		lgr.Fatalf("タスク実行エラー: %v", err)
	}

	// レポートを表示
	runner.PrintReport(report)

	// 終了コードを決定
	exitCode := 0
	if report.FailedTasks > 0 {
		exitCode = 1
		lgr.Error("Some tasks failed", "failed_count", report.FailedTasks)
	} else {
		lgr.Info("✅ All tasks completed successfully!")
	}

	os.Exit(exitCode)
}
