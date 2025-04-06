package exchange

import (
	"bytes"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kyi000/upbit-auto-trading-bot/pkg/utils"
)

const (
	upbitAPIURL         = "https://api.upbit.com/v1"
	upbitWebSocketURL   = "wss://api.upbit.com/websocket/v1"
	defaultHTTPTimeout  = 10 * time.Second
	initialBackoff      = 1
	maxBackoff          = 30
)

// UpbitClient 업비트 API 클라이언트
type UpbitClient struct {
	accessKey   string
	secretKey   string
	httpClient  *http.Client
	logger      *utils.Logger
}

// Market 마켓 정보
type Market struct {
	MarketID    string `json:"market"`
	KoreanName  string `json:"korean_name"`
	EnglishName string `json:"english_name"`
	MarketType  string
}

// Ticker 현재가 정보
type Ticker struct {
	MarketID        string  `json:"market"`
	TradePrice      float64 `json:"trade_price"`
	OpeningPrice    float64 `json:"opening_price"`
	HighPrice       float64 `json:"high_price"`
	LowPrice        float64 `json:"low_price"`
	PrevClosingPrice float64 `json:"prev_closing_price"`
	SignedChangeRate float64 `json:"signed_change_rate"`
	AccTradeVolume24h float64 `json:"acc_trade_volume_24h"`
	Timestamp       int64   `json:"timestamp"`
}

// Candle 캔들스틱 정보
type Candle struct {
	MarketID           string  `json:"market"`
	CandleDateTimeUTC  string  `json:"candle_date_time_utc"`
	CandleDateTimeKST  string  `json:"candle_date_time_kst"`
	OpeningPrice       float64 `json:"opening_price"`
	HighPrice          float64 `json:"high_price"`
	LowPrice           float64 `json:"low_price"`
	TradePrice         float64 `json:"trade_price"`
	Timestamp          int64   `json:"timestamp"`
	CandleAccTradeVolume float64 `json:"candle_acc_trade_volume"`
	CandleAccTradePrice  float64 `json:"candle_acc_trade_price"`
}

// Account 계정 정보
type Account struct {
	Currency      string  `json:"currency"`
	Balance       string  `json:"balance"`
	Locked        string  `json:"locked"`
	AvgBuyPrice   string  `json:"avg_buy_price"`
	AvgBuyPriceModified bool    `json:"avg_buy_price_modified"`
}

// OrderRequest 주문 요청
type OrderRequest struct {
	MarketID    string  `json:"market"`
	Side        string  `json:"side"`        // bid(매수), ask(매도)
	Volume      float64 `json:"volume,omitempty,string"`
	Price       float64 `json:"price,omitempty,string"`
	OrderType   string  `json:"ord_type"`    // limit(지정가), market(시장가), price(매수총액)
	Identifier  string  `json:"identifier,omitempty"`
}

// OrderResponse 주문 응답
type OrderResponse struct {
	UUID            string   `json:"uuid"`
	Side            string   `json:"side"`
	OrderType       string   `json:"ord_type"`
	Price           string   `json:"price"`
	State           string   `json:"state"`
	MarketID        string   `json:"market"`
	CreatedAt       string   `json:"created_at"`
	Volume          string   `json:"volume"`
	RemainingVolume string   `json:"remaining_volume"`
	ExecutedVolume  string   `json:"executed_volume"`
}

// OrderTrade 체결 내역
type OrderTrade struct {
	UUID       string  `json:"uuid"`
	Price      string  `json:"price"`
	Volume     string  `json:"volume"`
	Fee        string  `json:"fee"`
	CreatedAt  string  `json:"created_at"`
}

// MarketData 시장 데이터
type MarketData struct {
	Type       string  `json:"type"`
	MarketID   string  `json:"code"`
	Timestamp  int64   `json:"timestamp"`
	TradePrice float64 `json:"trade_price,omitempty"`
	TradeVolume float64 `json:"trade_volume,omitempty"`
	Bid        float64 `json:"bid,omitempty"`
	Ask        float64 `json:"ask,omitempty"`
	BidVolume  float64 `json:"bid_volume,omitempty"`
	AskVolume  float64 `json:"ask_volume,omitempty"`
}

// NewUpbitClient 새로운 업비트 클라이언트 생성
func NewUpbitClient(accessKey, secretKey string) *UpbitClient {
	return &UpbitClient{
		accessKey:  accessKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		logger:     utils.NewLogger("upbit"),
	}
}

// createJWT JWT 생성
func (c *UpbitClient) createJWT(params map[string]string) (string, error) {
	claims := jwt.MapClaims{
		"access_key": c.accessKey,
		"nonce":      uuid.New().String(),
	}

	if len(params) > 0 {
		query := url.Values{}
		for key, value := range params {
			query.Add(key, value)
		}
		queryString := query.Encode()
		
		h := sha512.New()
		h.Write([]byte(queryString))
		queryHash := fmt.Sprintf("%x", h.Sum(nil))
		
		claims["query_hash"] = queryHash
		claims["query_hash_alg"] = "SHA512"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	tokenString, err := token.SignedString([]byte(c.secretKey))
	if err != nil {
		return "", err
	}
	
	return "Bearer " + tokenString, nil
}

// GetMarkets 마켓 코드 조회
func (c *UpbitClient) GetMarkets() ([]Market, error) {
	url := fmt.Sprintf("%s/market/all", upbitAPIURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var markets []Market
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, err
	}
	
	// 마켓 타입 분류 (KRW, BTC, USDT)
	for i := range markets {
		parts := strings.Split(markets[i].MarketID, "-")
		if len(parts) > 0 {
			markets[i].MarketType = parts[0]
		}
	}
	
	// KRW 마켓만 필터링, USDT/USDC 제외
	var filteredMarkets []Market
	for _, market := range markets {
		if market.MarketType == "KRW" && 
		   !strings.Contains(market.MarketID, "USDT") && 
		   !strings.Contains(market.MarketID, "USDC") {
			filteredMarkets = append(filteredMarkets, market)
		}
	}
	
	return filteredMarkets, nil
}

// GetTicker 현재가 정보 조회
func (c *UpbitClient) GetTicker(marketID string) (*Ticker, error) {
	url := fmt.Sprintf("%s/ticker?markets=%s", upbitAPIURL, marketID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var tickers []Ticker
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return nil, err
	}
	
	if len(tickers) == 0 {
		return nil, fmt.Errorf("티커 정보가 없습니다: %s", marketID)
	}
	
	return &tickers[0], nil
}

// GetCandles 캔들스틱 정보 조회
func (c *UpbitClient) GetCandles(marketID, timeframe string, count int) ([]Candle, error) {
	var url string
	
	// 타임프레임에 따른 엔드포인트 선택
	switch {
	case strings.HasPrefix(timeframe, "minutes"):
		parts := strings.Split(timeframe, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("잘못된 타임프레임 형식: %s", timeframe)
		}
		unit := parts[1]
		url = fmt.Sprintf("%s/candles/minutes/%s?market=%s&count=%d", upbitAPIURL, unit, marketID, count)
	case timeframe == "days":
		url = fmt.Sprintf("%s/candles/days?market=%s&count=%d", upbitAPIURL, marketID, count)
	case timeframe == "weeks":
		url = fmt.Sprintf("%s/candles/weeks?market=%s&count=%d", upbitAPIURL, marketID, count)
	case timeframe == "months":
		url = fmt.Sprintf("%s/candles/months?market=%s&count=%d", upbitAPIURL, marketID, count)
	default:
		return nil, fmt.Errorf("지원되지 않는 타임프레임: %s", timeframe)
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var candles []Candle
	if err := json.NewDecoder(resp.Body).Decode(&candles); err != nil {
		return nil, err
	}
	
	return candles, nil
}

// GetAccounts 계정 정보 조회
func (c *UpbitClient) GetAccounts() ([]Account, error) {
	url := fmt.Sprintf("%s/accounts", upbitAPIURL)
	
	token, err := c.createJWT(nil)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Authorization", token)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var accounts []Account
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, err
	}
	
	return accounts, nil
}

// CreateOrder 주문 생성
func (c *UpbitClient) CreateOrder(marketID, side, orderType string, volume, price float64) (*OrderResponse, error) {
	url := fmt.Sprintf("%s/orders", upbitAPIURL)
	
	orderRequest := OrderRequest{
		MarketID:   marketID,
		Side:       side,
		OrderType:  orderType,
		Identifier: uuid.New().String(),
	}
	
	// 주문 유형에 따른 필드 설정
	if orderType == "limit" {
		orderRequest.Price = price
		orderRequest.Volume = volume
	} else if orderType == "market" {
		if side == "bid" {
			orderRequest.Price = price
		} else {
			orderRequest.Volume = volume
		}
	}
	
	jsonData, err := json.Marshal(orderRequest)
	if err != nil {
		return nil, err
	}
	
	params := map[string]string{
		"market":     marketID,
		"side":       side,
		"ord_type":   orderType,
		"identifier": orderRequest.Identifier,
	}
	
	if orderType == "limit" {
		params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
		params["volume"] = strconv.FormatFloat(volume, 'f', -1, 64)
	} else if orderType == "market" {
		if side == "bid" {
			params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
		} else {
			params["volume"] = strconv.FormatFloat(volume, 'f', -1, 64)
		}
	}
	
	token, err := c.createJWT(params)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", token)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var orderResponse OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResponse); err != nil {
		return nil, err
	}
	
	return &orderResponse, nil
}

// GetOrder 주문 조회
func (c *UpbitClient) GetOrder(uuid string) (*OrderResponse, error) {
	params := map[string]string{
		"uuid": uuid,
	}
	
	token, err := c.createJWT(params)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/order?uuid=%s", upbitAPIURL, uuid)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Authorization", token)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var orderResponse OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResponse); err != nil {
		return nil, err
	}
	
	return &orderResponse, nil
}

// GetOrderTrades 체결 내역 조회
func (c *UpbitClient) GetOrderTrades(uuid string) ([]OrderTrade, error) {
	params := map[string]string{
		"uuid": uuid,
	}
	
	token, err := c.createJWT(params)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/order/trades?uuid=%s", upbitAPIURL, uuid)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Authorization", token)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var trades []OrderTrade
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		return nil, err
	}
	
	return trades, nil
}

// CancelOrder 주문 취소
func (c *UpbitClient) CancelOrder(uuid string) (*OrderResponse, error) {
	params := map[string]string{
		"uuid": uuid,
	}
	
	token, err := c.createJWT(params)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/order", upbitAPIURL)
	
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", token)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 오류: %s %s", resp.Status, string(body))
	}
	
	var orderResponse OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResponse); err != nil {
		return nil, err
	}
	
	return &orderResponse, nil
}

// ConnectWebSocket 웹소켓 연결
func (c *UpbitClient) ConnectWebSocket(markets []string, types []string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}
	
	// 웹소켓 연결
	conn, _, err := dialer.Dial(upbitWebSocketURL, nil)
	if err != nil {
		return nil, err
	}
	
	// 구독 요청 생성
	type TickerRequest struct {
		Ticket string   `json:"ticket"`
		Type   string   `json:"type"`
		Codes  []string `json:"codes"`
	}
	
	for _, t := range types {
		request := TickerRequest{
			Ticket: uuid.New().String(),
			Type:   t,
			Codes:  markets,
		}
		
		if err := conn.WriteJSON(request); err != nil {
			conn.Close()
			return nil, err
		}
	}
	
	return conn, nil
}

// MaintainWebSocketConnection 웹소켓 연결 유지
func (c *UpbitClient) MaintainWebSocketConnection(markets []string, types []string, dataCh chan<- MarketData, done <-chan struct{}) {
	backoff := initialBackoff
	
	for {
		select {
		case <-done:
			return
		default:
			conn, err := c.ConnectWebSocket(markets, types)
			if err != nil {
				c.logger.Error("웹소켓 연결 실패:", err)
				time.Sleep(time.Duration(backoff) * time.Second)
				backoff = min(backoff*2, maxBackoff) // 지수 백오프
				continue
			}
			
			// 연결 성공 시 백오프 리셋
			backoff = initialBackoff
			
			// 웹소켓 데이터 처리
			c.handleWebSocketConnection(conn, dataCh, done)
		}
	}
}

// handleWebSocketConnection 웹소켓 연결 처리
func (c *UpbitClient) handleWebSocketConnection(conn *websocket.Conn, dataCh chan<- MarketData, done <-chan struct{}) {
	defer conn.Close()
	
	// 핑 처리를 위한 타이머
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()
	
	// 고루틴으로 데이터 수신
	dataDone := make(chan struct{})
	
	go func() {
		defer close(dataDone)
		
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				c.logger.Error("웹소켓 메시지 읽기 실패:", err)
				return
			}
			
			var data MarketData
			if err := json.Unmarshal(message, &data); err != nil {
				c.logger.Error("웹소켓 메시지 파싱 실패:", err)
				continue
			}
			
			dataCh <- data
		}
	}()
	
	// 메인 루프
	for {
		select {
		case <-done:
			// 클라이언트에서 종료 신호
			return
		case <-dataDone:
			// 데이터 수신 고루틴 종료 (연결 끊김)
			return
		case <-pingTicker.C:
			// 핑 전송
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				c.logger.Error("웹소켓 핑 전송 실패:", err)
				return
			}
		}
	}
}

// min 두 정수 중 작은 값 반환
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
