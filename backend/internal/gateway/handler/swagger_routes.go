package handler

type HealthResponse struct {
	Status  string `json:"status"  example:"ok"`
	Service string `json:"service" example:"api-gateway"`
}

type GatewayErrorResponse struct {
	Error string `json:"error" example:"service unavailable"`
}

type GatewayTokenPairResponse struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGci..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
	ExpiresIn    int64  `json:"expires_in"    example:"900"`
}

type GatewayRegisterBody struct {
	Email    string `json:"email"    example:"alice@example.com"`
	Name     string `json:"name"     example:"Alice Smith"`
	Password string `json:"password" example:"s3cret!Pass"`
}

type GatewayLoginBody struct {
	Email    string `json:"email"    example:"alice@example.com"`
	Password string `json:"password" example:"s3cret!Pass"`
}

type GatewayRefreshBody struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type GatewayLogoutBody struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type GatewayUserResponse struct {
	ID        string `json:"id"         example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email"      example:"alice@example.com"`
	Name      string `json:"name"       example:"Alice Smith"`
	CreatedAt string `json:"created_at" example:"2026-03-13T00:00:00Z"`
}

type GatewayChangePasswordBody struct {
	OldPassword string `json:"old_password" example:"s3cret!Pass"`
	NewPassword string `json:"new_password" example:"newS3cret!"`
}

func docHealthz() {}

func docRegister() {}

func docLogin() {}

func docRefresh() {}

func docLogout() {}

func docMe() {}

func docChangePassword() {}

type GatewayCreateOrderBody struct {
	Currency string                 `json:"currency" example:"USD"`
	Items    []GatewayOrderItemBody `json:"items"`
}

type GatewayOrderItemBody struct {
	ProductID string `json:"product_id" example:"prod-abc-123"`
	Quantity  int    `json:"quantity"   example:"2"`
	Price     int64  `json:"price"      example:"2500"`
}

type GatewayOrder struct {
	ID             string                 `json:"id"              example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID         string                 `json:"user_id"         example:"user-123"`
	Status         string                 `json:"status"          example:"pending"`
	Items          []GatewayOrderItemBody `json:"items"`
	TotalAmount    int64                  `json:"total_amount"    example:"5000"`
	Currency       string                 `json:"currency"        example:"USD"`
	IdempotencyKey string                 `json:"idempotency_key" example:"order-req-abc"`
	CreatedAt      string                 `json:"created_at"      example:"2026-03-13T00:00:00Z"`
}

type GatewayListOrdersResponse struct {
	Orders []GatewayOrder `json:"orders"`
	Limit  int            `json:"limit"  example:"20"`
	Offset int            `json:"offset" example:"0"`
}

func docCreateOrder() {}

func docListOrders() {}

func docGetOrder() {}

type GatewayPayment struct {
	ID            string `json:"id"                       example:"550e8400-e29b-41d4-a716-446655440001"`
	OrderID       string `json:"order_id"                 example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID        string `json:"user_id"                  example:"user-123"`
	Amount        int64  `json:"amount"                   example:"5000"`
	Currency      string `json:"currency"                 example:"USD"`
	Status        string `json:"status"                   example:"succeeded"`
	Method        string `json:"method"                   example:"card"`
	TransactionID string `json:"transaction_id,omitempty" example:"txn_abc123"`
	CreatedAt     string `json:"created_at"               example:"2026-03-13T00:00:00Z"`
}

func docGetPayment() {}

type GatewayWallet struct {
	ID        string `json:"id"         example:"550e8400-e29b-41d4-a716-446655440002"`
	UserID    string `json:"user_id"    example:"user-123"`
	Balance   int64  `json:"balance"    example:"10000"`
	Currency  string `json:"currency"   example:"USD"`
	CreatedAt string `json:"created_at" example:"2026-03-13T00:00:00Z"`
}

type GatewayCreateWalletBody struct {
	Currency string `json:"currency" example:"USD"`
}

func docGetWallet() {}

func docCreateWallet() {}
