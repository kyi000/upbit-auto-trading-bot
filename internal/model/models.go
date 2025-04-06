package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Candlestick 캔들스틱 데이터
type Candlestick struct {
	gorm.Model
	MarketID  string    `gorm:"column:market_id;not null;index:idx_market_timeframe_timestamp,priority:1"`
	Timeframe string    `gorm:"column:timeframe;not null;index:idx_market_timeframe_timestamp,priority:2"`
	Timestamp time.Time `gorm:"column:timestamp;not null;index:idx_market_timeframe_timestamp,priority:3"`
	Open      float64   `gorm:"column:open;not null"`
	High      float64   `gorm:"column:high;not null"`
	Low       float64   `gorm:"column:low;not null"`
	Close     float64   `gorm:"column:close;not null"`
	Volume    float64   `gorm:"column:volume;not null"`
}

// TableName Candlestick 테이블 이름 설정
func (Candlestick) TableName() string {
	return "candlesticks"
}

// Parameters JSONB 타입 정의
type Parameters map[string]interface{}

// Value JSONB 데이터베이스 인코딩
func (p Parameters) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan JSONB 데이터베이스 디코딩
func (p *Parameters) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	return json.Unmarshal(bytes, &p)
}

// Signal 매매 신호
type Signal struct {
	gorm.Model
	MarketID     string     `gorm:"column:market_id;not null;index:idx_market_timestamp,priority:1"`
	StrategyName string     `gorm:"column:strategy_name;not null"`
	SignalType   string     `gorm:"column:signal_type;not null"` // BUY, SELL
	Price        float64    `gorm:"column:price;not null"`
	Confidence   float64    `gorm:"column:confidence;not null"`
	Timestamp    time.Time  `gorm:"column:timestamp;not null;index:idx_market_timestamp,priority:2"`
	Parameters   Parameters `gorm:"column:parameters;type:jsonb"`
}

// TableName Signal 테이블 이름 설정
func (Signal) TableName() string {
	return "signals"
}

// Order 주문 정보
type Order struct {
	gorm.Model
	MarketID       string    `gorm:"column:market_id;not null;index:idx_market_status,priority:1"`
	OrderID        string    `gorm:"column:order_id;not null;unique;index"`
	Side           string    `gorm:"column:side;not null"` // BUY, SELL
	OrderType      string    `gorm:"column:order_type;not null"`
	Price          float64   `gorm:"column:price"`
	Volume         float64   `gorm:"column:volume;not null"`
	ExecutedVolume float64   `gorm:"column:executed_volume;default:0"`
	Status         string    `gorm:"column:status;not null;index:idx_market_status,priority:2"` // WAIT, DONE, CANCEL
	SignalID       uint      `gorm:"column:signal_id"`
	LastUpdated    time.Time `gorm:"column:last_updated"`
}

// TableName Order 테이블 이름 설정
func (Order) TableName() string {
	return "orders"
}

// Trade 체결 내역
type Trade struct {
	gorm.Model
	MarketID  string    `gorm:"column:market_id;not null;index:idx_market_timestamp,priority:1"`
	OrderID   string    `gorm:"column:order_id;not null;index"`
	Price     float64   `gorm:"column:price;not null"`
	Volume    float64   `gorm:"column:volume;not null"`
	Side      string    `gorm:"column:side;not null"` // BUY, SELL
	Fee       float64   `gorm:"column:fee;not null"`
	Timestamp time.Time `gorm:"column:timestamp;not null;index:idx_market_timestamp,priority:2"`
}

// TableName Trade 테이블 이름 설정
func (Trade) TableName() string {
	return "trades"
}

// Position 포지션 정보
type Position struct {
	gorm.Model
	MarketID      string    `gorm:"column:market_id;not null;uniqueIndex"`
	EntryPrice    float64   `gorm:"column:entry_price;not null"`
	EntryTime     time.Time `gorm:"column:entry_time;not null"`
	Quantity      float64   `gorm:"column:quantity;not null"`
	Status        string    `gorm:"column:status;not null"` // OPEN, CLOSED
	ProfitTarget  float64   `gorm:"column:profit_target;not null"`
	StopLoss      float64   `gorm:"column:stop_loss;not null"`
	LastPrice     float64   `gorm:"column:last_price;not null"`
	CurrentProfit float64   `gorm:"column:current_profit"`
	ExitPrice     float64   `gorm:"column:exit_price"`
	ExitTime      time.Time `gorm:"column:exit_time"`
	ExitReason    string    `gorm:"column:exit_reason"` // TARGET, STOP, MANUAL, SIGNAL
}

// TableName Position 테이블 이름 설정
func (Position) TableName() string {
	return "positions"
}

// StrategyConfig 전략 설정
type StrategyConfig struct {
	gorm.Model
	MarketID     string     `gorm:"column:market_id;not null;uniqueIndex"`
	StrategyName string     `gorm:"column:strategy_name;not null"`
	Timeframe    string     `gorm:"column:timeframe;not null"`
	ProfitTarget float64    `gorm:"column:profit_target;not null"`
	StopLoss     float64    `gorm:"column:stop_loss;not null"`
	Enabled      bool       `gorm:"column:enabled;not null;default:true"`
	Parameters   Parameters `gorm:"column:parameters;type:jsonb"`
}

// TableName StrategyConfig 테이블 이름 설정
func (StrategyConfig) TableName() string {
	return "strategy_configs"
}

// PerformanceMetric 성능 지표
type PerformanceMetric struct {
	gorm.Model
	StrategyName     string    `gorm:"column:strategy_name;not null;index:idx_strategy_market,priority:1"`
	MarketID         string    `gorm:"column:market_id;not null;index:idx_strategy_market,priority:2"`
	StartTime        time.Time `gorm:"column:start_time;not null"`
	EndTime          time.Time `gorm:"column:end_time;not null"`
	TotalTrades      int       `gorm:"column:total_trades;not null"`
	WinningTrades    int       `gorm:"column:winning_trades;not null"`
	ProfitPercentage float64   `gorm:"column:profit_percentage;not null"`
	MaxDrawdown      float64   `gorm:"column:max_drawdown;not null"`
	SharpeRatio      float64   `gorm:"column:sharpe_ratio"`
}

// TableName PerformanceMetric 테이블 이름 설정
func (PerformanceMetric) TableName() string {
	return "performance_metrics"
}

// DailyPerformance 일별 성과
type DailyPerformance struct {
	gorm.Model
	Date             time.Time `gorm:"column:date;not null;uniqueIndex:idx_date_market"`
	MarketID         string    `gorm:"column:market_id;not null;uniqueIndex:idx_date_market"`
	ProfitPercentage float64   `gorm:"column:profit_percentage;not null"`
	ProfitAmount     float64   `gorm:"column:profit_amount;not null"`
	TradeCount       int       `gorm:"column:trade_count;not null"`
	WinCount         int       `gorm:"column:win_count;not null"`
}

// TableName DailyPerformance 테이블 이름 설정
func (DailyPerformance) TableName() string {
	return "daily_performances"
}
