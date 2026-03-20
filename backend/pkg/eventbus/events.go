package eventbus

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`           // e.g. "order.created", "payment.succeeded"
	AggregateID   string    `json:"aggregate_id"`   // e.g. the order ID or payment ID
	AggregateType string    `json:"aggregate_type"` // e.g. "order", "payment"
	Timestamp     time.Time `json:"timestamp"`
	Version       int       `json:"version"` // for schema evolution â€” increment when payload changes
	Data          []byte    `json:"data"`    // JSON-encoded domain payload (OrderCreatedData, etc.)
	Metadata      Metadata  `json:"metadata"`
}

type Metadata struct {
	CorrelationID string `json:"correlation_id"` // groups all events from a single user action
	CausationID   string `json:"causation_id"`   // the event ID that caused this event
	UserID        string `json:"user_id,omitempty"`
	Email         string `json:"email,omitempty"`    // user email for notification delivery
	TraceID       string `json:"trace_id,omitempty"` // OpenTelemetry trace ID
}

// NormalizePaymentMethod coerces any unrecognised value to "card".
// The canonical payment methods are "card" and "wallet".
func NormalizePaymentMethod(method string) string {
	if method == "wallet" {
		return "wallet"
	}
	return "card"
}

func NewEvent(eventType, aggregateID, aggregateType string, data []byte, meta Metadata) Event {
	return Event{
		ID:            uuid.New().String(),
		Type:          eventType,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Timestamp:     time.Now().UTC(),
		Version:       1,
		Data:          data,
		Metadata:      meta,
	}
}

type OrderCreatedData struct {
	OrderID        string      `json:"order_id"`
	UserID         string      `json:"user_id"`
	Items          []OrderItem `json:"items"`
	TotalAmount    int64       `json:"total_amount"` // in cents â€” always store money as integers
	Currency       string      `json:"currency"`
	PaymentMethod  string      `json:"payment_method"`
	IdempotencyKey string      `json:"idempotency_key"`
	StoreID        *string     `json:"store_id,omitempty"` // nil = platform order
}

type OrderItem struct {
	ProductID    string  `json:"product_id"`
	VariantID    *string `json:"variant_id,omitempty"`
	VariantSKU   string  `json:"variant_sku,omitempty"`
	VariantLabel string  `json:"variant_label,omitempty"`
	Quantity     int     `json:"quantity"`
	Price        int64   `json:"price"` // unit price in cents
}

type OrderConfirmedData struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

type OrderCancelledData struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
	Reason  string `json:"reason"`
}

type PaymentInitiatedData struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Method    string `json:"method"` // "card" or "wallet"
}

type PaymentSucceededData struct {
	PaymentID     string  `json:"payment_id"`
	OrderID       string  `json:"order_id"`
	UserID        string  `json:"user_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        int64   `json:"amount"`
	Currency      string  `json:"currency"`
	Method        string  `json:"method"`                   // "card" | "wallet"
	StoreID       *string `json:"store_id,omitempty"`       // nil = platform order
	StoreOwnerID  string  `json:"store_owner_id,omitempty"` // seller's user ID
	Commission    int     `json:"commission,omitempty"`     // platform cut in %; e.g. 10 = 10%
}

type PaymentFailedData struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
}

type PaymentRefundedData struct {
	PaymentID string `json:"payment_id"`
	RefundID  string `json:"refund_id"`
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
}

type WalletCreditedData struct {
	WalletID      string `json:"wallet_id"`
	UserID        string `json:"user_id"`
	Amount        int64  `json:"amount"`
	TransactionID string `json:"transaction_id"`
	Source        string `json:"source"` // "refund", "deposit", "transfer"
}

type WalletDebitedData struct {
	WalletID      string `json:"wallet_id"`
	UserID        string `json:"user_id"`
	Amount        int64  `json:"amount"`
	TransactionID string `json:"transaction_id"`
}

type FraudCheckResultData struct {
	CheckID   string   `json:"check_id"`
	PaymentID string   `json:"payment_id"`
	OrderID   string   `json:"order_id"`
	RiskScore float64  `json:"risk_score"`
	Decision  string   `json:"decision"` // "approved", "rejected", "review"
	Rules     []string `json:"rules_triggered"`
}

type UserRegisteredData struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type ProductCreatedData struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price"`
	Currency  string `json:"currency"`
	Stock     int    `json:"stock"`
}

type StockItem struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
}

type StockReservedData struct {
	OrderID string      `json:"order_id"`
	Items   []StockItem `json:"items"`
}

type StockReleasedData struct {
	OrderID string      `json:"order_id"`
	Items   []StockItem `json:"items"`
	Reason  string      `json:"reason"` // "payment_failed" | "order_cancelled"
}

type WithdrawalRequestedData struct {
	WithdrawalID string `json:"withdrawal_id"`
	UserID       string `json:"user_id"`
	StoreID      string `json:"store_id"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
	Method       string `json:"method"` // e.g. "bank_transfer"
}

type WithdrawalApprovedData struct {
	WithdrawalID string `json:"withdrawal_id"`
	UserID       string `json:"user_id"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
}

type WithdrawalRejectedData struct {
	WithdrawalID string `json:"withdrawal_id"`
	UserID       string `json:"user_id"`
	Amount       int64  `json:"amount"`
	Reason       string `json:"reason"`
}

type SettlementCompletedData struct {
	PaymentID    string `json:"payment_id"`
	OrderID      string `json:"order_id"`
	StoreID      string `json:"store_id"`
	StoreOwnerID string `json:"store_owner_id"`
	GrossAmount  int64  `json:"gross_amount"`  // full order total
	Commission   int    `json:"commission"`    // platform % taken
	SellerAmount int64  `json:"seller_amount"` // credited to seller wallet
	Currency     string `json:"currency"`
}

type StoreCreatedData struct {
	StoreID string `json:"store_id"`
	OwnerID string `json:"owner_id"`
	Name    string `json:"name"`
}

type StoreStatusChangedData struct {
	StoreID string `json:"store_id"`
	OwnerID string `json:"owner_id"`
	Status  string `json:"status"`
}

const (
	SubjectUserRegistered    = "users.registered"
	SubjectOrderCreated      = "orders.created"
	SubjectOrderConfirmed    = "orders.confirmed"
	SubjectOrderCancelled    = "orders.cancelled"
	SubjectPaymentInitiated  = "payments.initiated"
	SubjectPaymentSucceeded  = "payments.succeeded"
	SubjectPaymentFailed     = "payments.failed"
	SubjectPaymentRefunded   = "payments.refunded"
	SubjectWalletCredited    = "wallets.credited"
	SubjectWalletDebited     = "wallets.debited"
	SubjectFraudApproved     = "fraud.approved"
	SubjectFraudRejected     = "fraud.rejected"
	SubjectProductCreated    = "products.created"
	SubjectStockReserved     = "products.stock_reserved"
	SubjectStockReleased     = "products.stock_released"
	SubjectProductOutOfStock = "products.out_of_stock"

	SubjectStoreCreated        = "stores.created"
	SubjectStoreApproved       = "stores.approved"
	SubjectStoreSuspended      = "stores.suspended"
	SubjectStoreReactivated    = "stores.reactivated"
	SubjectStoreClosed         = "stores.closed"
	SubjectSettlementCompleted = "settlements.completed"
	SubjectWithdrawalRequested = "withdrawals.requested"
	SubjectWithdrawalApproved  = "withdrawals.approved"
	SubjectWithdrawalRejected  = "withdrawals.rejected"
)
