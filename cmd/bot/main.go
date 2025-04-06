package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyi000/upbit-auto-trading-bot/internal/api"
	"github.com/kyi000/upbit-auto-trading-bot/internal/config"
	"github.com/kyi000/upbit-auto-trading-bot/internal/exchange"
	"github.com/kyi000/upbit-auto-trading-bot/internal/risk"
	"github.com/kyi000/upbit-auto-trading-bot/internal/storage"
	"github.com/kyi000/upbit-auto-trading-bot/internal/strategy"
	"github.com/kyi000/upbit-auto-trading-bot/pkg/utils"
)

func main() {
	// 로거 초기화
	logger := utils.NewLogger("main")
	logger.Info("업비트 트레이딩 봇 시작")

	// 설정 로드
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		logger.Fatal("설정 로드 실패:", err)
	}

	// 컨텍스트 생성 (종료 시그널 처리용)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 데이터베이스 연결
	db, err := storage.NewDatabase(cfg.Database)
	if err != nil {
		logger.Fatal("데이터베이스 연결 실패:", err)
	}
	defer db.Close()

	// 업비트 클라이언트 생성
	upbitClient := exchange.NewUpbitClient(cfg.Upbit.AccessKey, cfg.Upbit.SecretKey)
	
	// 채널 생성
	marketDataCh := make(chan exchange.MarketData, 100)
	signalCh := make(chan strategy.Signal, 100)
	orderCh := make(chan exchange.Order, 100)

	// 위험 관리 모듈 초기화
	riskManager := risk.NewManager(db.GetDB(), upbitClient, nil)
	riskManager.Start()

	// 전략 관리자 초기화
	strategyManager := strategy.NewManager(db.GetDB(), upbitClient, marketDataCh, signalCh)
	if err := strategyManager.LoadStrategies(); err != nil {
		logger.Error("전략 로드 실패:", err)
	}
	strategyManager.Start()

	// 주문 실행기 초기화
	orderExecutor := exchange.NewOrderExecutor(db.GetDB(), upbitClient, riskManager, signalCh, orderCh)
	orderExecutor.Start()

	// 시장 데이터 수집기 초기화
	dataCollector := exchange.NewDataCollector(upbitClient, db.GetDB(), marketDataCh)
	dataCollector.Start(ctx)

	// API 서버 시작
	server := api.NewServer(db.GetDB(), upbitClient, strategyManager, riskManager, orderCh)
	go func() {
		if err := server.Start(cfg.Server.Host + ":" + cfg.Server.Port); err != nil {
			logger.Fatal("API 서버 시작 실패:", err)
		}
	}()

	// 종료 시그널 처리
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Info("종료 신호 수신. 정상 종료 진행...")

	// 정상 종료를 위한 컨텍스트 취소
	cancel()

	// 서버 종료 타임아웃 설정
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 모듈 정상 종료
	strategyManager.Stop()
	orderExecutor.Stop()
	server.Stop(shutdownCtx)

	logger.Info("업비트 트레이딩 봇 종료")
}
