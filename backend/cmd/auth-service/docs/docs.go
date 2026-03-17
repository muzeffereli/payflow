package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/v1/orders": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Returns paginated orders for the authenticated user.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "orders"
                ],
                "summary": "List orders",
                "parameters": [
                    {
                        "type": "integer",
                        "default": 20,
                        "description": "Max results (default 20, max 100)",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "default": 0,
                        "description": "Pagination offset",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.ListOrdersResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Create a new order. Idempotent: same Idempotency-Key returns the existing order.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "orders"
                ],
                "summary": "Create order",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Client-generated unique key for idempotency",
                        "name": "Idempotency-Key",
                        "in": "header",
                        "required": true
                    },
                    {
                        "description": "Order details",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.CreateOrderBody"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.Order"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/orders/{id}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Returns an order. Only the owner can access it.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "orders"
                ],
                "summary": "Get order by ID",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Order ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Order"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Cancels a pending or confirmed order. Only the owner can cancel. Paid or refunded orders cannot be cancelled.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "orders"
                ],
                "summary": "Cancel order",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Order ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.CancelOrderResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/payments/order/{order_id}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Returns the payment associated with a specific order ID.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "payments"
                ],
                "summary": "Get payment by order",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Order ID",
                        "name": "order_id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/payments/{id}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Returns payment details including status, amount, and transaction ID.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "payments"
                ],
                "summary": "Get payment",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Payment ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/payments/{id}/refund": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Refunds a succeeded payment. Only the payment owner can request a refund. Publishes payments.refunded which credits the user's wallet.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "payments"
                ],
                "summary": "Refund payment",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Payment ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.RefundResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.PaymentErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/products": {
            "get": {
                "description": "Returns paginated products. Filter by category, status, or a comma-separated list of IDs (used by order service for batch price lookup).",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "products"
                ],
                "summary": "List products",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Comma-separated product IDs for batch lookup",
                        "name": "ids",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Filter by category",
                        "name": "category",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Filter by status: active, inactive, out_of_stock",
                        "name": "status",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "default": 20,
                        "description": "Max results (default 20, max 100)",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "default": 0,
                        "description": "Pagination offset",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.ListProductsResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Adds a new product to the catalog with an authoritative price and initial stock.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "products"
                ],
                "summary": "Create product",
                "parameters": [
                    {
                        "description": "Product details",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.CreateProductRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.Product"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    },
                    "409": {
                        "description": "SKU already exists",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/products/{id}": {
            "get": {
                "description": "Returns product details including current price and stock level.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "products"
                ],
                "summary": "Get product",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Product ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Product"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Soft-deletes a product by setting its status to inactive. It remains in the DB.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "products"
                ],
                "summary": "Deactivate product",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Product ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    }
                }
            },
            "patch": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Partially updates a product. Only provided fields are changed.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "products"
                ],
                "summary": "Update product",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Product ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Fields to update",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.UpdateProductRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Product"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.ProductErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/wallet": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Returns the authenticated user's wallet balance and details.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "wallet"
                ],
                "summary": "Get wallet",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Wallet"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.WalletErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handler.WalletErrorResponse"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Creates a new wallet for the authenticated user. Each user can have only one wallet.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "wallet"
                ],
                "summary": "Create wallet",
                "parameters": [
                    {
                        "description": "Wallet details",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.CreateWalletRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.Wallet"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.WalletErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.WalletErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handler.WalletErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/change-password": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Changes the authenticated user's password and revokes all existing refresh tokens.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Change password",
                "parameters": [
                    {
                        "description": "Old and new passwords",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayChangePasswordBody"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/login": {
            "post": {
                "description": "Validates credentials and returns a token pair. Use the access_token as \"Bearer \u003ctoken\u003e\" in the Authorize dialog.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Login",
                "parameters": [
                    {
                        "description": "Credentials",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayLoginBody"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayTokenPairResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/logout": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Revokes the given refresh token. Access tokens expire naturally after 15 min.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Logout",
                "parameters": [
                    {
                        "description": "Refresh token to revoke",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayLogoutBody"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    }
                }
            }
        },
        "/auth/me": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Returns the authenticated user's profile.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Get current user",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayUserResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/refresh": {
            "post": {
                "description": "Exchanges a refresh token for a new token pair. The old refresh token is revoked immediately.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Refresh tokens",
                "parameters": [
                    {
                        "description": "Refresh token",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayRefreshBody"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayTokenPairResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/register": {
            "post": {
                "description": "Creates a new user account and returns an access token (15 min) + refresh token (7 days).",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Register",
                "parameters": [
                    {
                        "description": "Registration details",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayRegisterBody"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayTokenPairResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayErrorResponse"
                        }
                    },
                    "409": {
                        "description": "Email already taken",
                        "schema": {
                            "$ref": "#/definitions/handler.GatewayErrorResponse"
                        }
                    }
                }
            }
        },
        "/healthz": {
            "get": {
                "description": "Returns 200 if the API Gateway is alive.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "ops"
                ],
                "summary": "Health check",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handler.HealthResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "domain.Order": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string"
                },
                "currency": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "idempotency_key": {
                    "type": "string"
                },
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.OrderItem"
                    }
                },
                "shipping_address": {
                    "$ref": "#/definitions/domain.ShippingAddress"
                },
                "status": {
                    "$ref": "#/definitions/domain.OrderStatus"
                },
                "total_amount": {
                    "description": "cents",
                    "type": "integer"
                },
                "updated_at": {
                    "type": "string"
                },
                "user_id": {
                    "type": "string"
                }
            }
        },
        "domain.OrderItem": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string"
                },
                "order_id": {
                    "type": "string"
                },
                "price": {
                    "description": "unit price in cents",
                    "type": "integer"
                },
                "product_id": {
                    "type": "string"
                },
                "quantity": {
                    "type": "integer"
                }
            }
        },
        "domain.OrderStatus": {
            "type": "string",
            "enum": [
                "pending",
                "confirmed",
                "paid",
                "cancelled",
                "refunded"
            ],
            "x-enum-varnames": [
                "StatusPending",
                "StatusConfirmed",
                "StatusPaid",
                "StatusCancelled",
                "StatusRefunded"
            ]
        },
        "domain.PaymentStatus": {
            "type": "string",
            "enum": [
                "pending",
                "processing",
                "succeeded",
                "failed",
                "refunded"
            ],
            "x-enum-varnames": [
                "PaymentPending",
                "PaymentProcessing",
                "PaymentSucceeded",
                "PaymentFailed",
                "PaymentRefunded"
            ]
        },
        "domain.Product": {
            "type": "object",
            "properties": {
                "category": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "currency": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "price": {
                    "description": "unit price in cents",
                    "type": "integer"
                },
                "sku": {
                    "type": "string"
                },
                "status": {
                    "$ref": "#/definitions/domain.ProductStatus"
                },
                "stock": {
                    "type": "integer"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "domain.ProductStatus": {
            "type": "string",
            "enum": [
                "active",
                "inactive",
                "out_of_stock"
            ],
            "x-enum-varnames": [
                "StatusActive",
                "StatusInactive",
                "StatusOutOfStock"
            ]
        },
        "domain.ShippingAddress": {
            "type": "object",
            "properties": {
                "city": {
                    "type": "string"
                },
                "country": {
                    "description": "ISO 3166-1 alpha-2, e.g. \"US\"",
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "postal_code": {
                    "type": "string"
                },
                "state": {
                    "type": "string"
                },
                "street": {
                    "type": "string"
                }
            }
        },
        "domain.Wallet": {
            "type": "object",
            "properties": {
                "balance": {
                    "description": "cents",
                    "type": "integer"
                },
                "created_at": {
                    "type": "string"
                },
                "currency": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "user_id": {
                    "type": "string"
                }
            }
        },
        "handler.AuthErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "invalid email or password"
                }
            }
        },
        "handler.CancelOrderResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "order cancelled"
                },
                "order_id": {
                    "type": "string",
                    "example": "ord-abc-123"
                }
            }
        },
        "handler.ChangePasswordRequest": {
            "type": "object",
            "required": [
                "new_password",
                "old_password"
            ],
            "properties": {
                "new_password": {
                    "type": "string",
                    "minLength": 8,
                    "example": "newS3cret!"
                },
                "old_password": {
                    "type": "string",
                    "example": "s3cret!Pass"
                }
            }
        },
        "handler.CreateOrderBody": {
            "type": "object",
            "required": [
                "currency",
                "items"
            ],
            "properties": {
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "items": {
                    "type": "array",
                    "minItems": 1,
                    "items": {
                        "$ref": "#/definitions/handler.CreateOrderItemBody"
                    }
                },
                "shipping_address": {
                    "$ref": "#/definitions/handler.ShippingAddressBody"
                }
            }
        },
        "handler.CreateOrderItemBody": {
            "type": "object",
            "required": [
                "product_id",
                "quantity"
            ],
            "properties": {
                "product_id": {
                    "type": "string",
                    "example": "prod-abc-123"
                },
                "quantity": {
                    "type": "integer",
                    "minimum": 1,
                    "example": 2
                }
            }
        },
        "handler.CreateProductRequest": {
            "type": "object",
            "required": [
                "name",
                "price",
                "sku"
            ],
            "properties": {
                "category": {
                    "type": "string",
                    "example": "electronics"
                },
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "description": {
                    "type": "string",
                    "example": "Noise-cancelling over-ear headphones"
                },
                "name": {
                    "type": "string",
                    "example": "Wireless Headphones"
                },
                "price": {
                    "description": "cents",
                    "type": "integer",
                    "example": 9999
                },
                "sku": {
                    "type": "string",
                    "example": "HDPH-001"
                },
                "stock": {
                    "type": "integer",
                    "minimum": 0,
                    "example": 100
                }
            }
        },
        "handler.CreateWalletRequest": {
            "type": "object",
            "required": [
                "currency"
            ],
            "properties": {
                "currency": {
                    "type": "string",
                    "example": "USD"
                }
            }
        },
        "handler.ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "something went wrong"
                }
            }
        },
        "handler.GatewayChangePasswordBody": {
            "type": "object",
            "properties": {
                "new_password": {
                    "type": "string",
                    "example": "newS3cret!"
                },
                "old_password": {
                    "type": "string",
                    "example": "s3cret!Pass"
                }
            }
        },
        "handler.GatewayCreateOrderBody": {
            "type": "object",
            "properties": {
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/handler.GatewayOrderItemBody"
                    }
                }
            }
        },
        "handler.GatewayCreateWalletBody": {
            "type": "object",
            "properties": {
                "currency": {
                    "type": "string",
                    "example": "USD"
                }
            }
        },
        "handler.GatewayErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "service unavailable"
                }
            }
        },
        "handler.GatewayListOrdersResponse": {
            "type": "object",
            "properties": {
                "limit": {
                    "type": "integer",
                    "example": 20
                },
                "offset": {
                    "type": "integer",
                    "example": 0
                },
                "orders": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/handler.GatewayOrder"
                    }
                }
            }
        },
        "handler.GatewayLoginBody": {
            "type": "object",
            "properties": {
                "email": {
                    "type": "string",
                    "example": "alice@example.com"
                },
                "password": {
                    "type": "string",
                    "example": "s3cret!Pass"
                }
            }
        },
        "handler.GatewayLogoutBody": {
            "type": "object",
            "properties": {
                "refresh_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                }
            }
        },
        "handler.GatewayOrder": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string",
                    "example": "2026-03-13T00:00:00Z"
                },
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440000"
                },
                "idempotency_key": {
                    "type": "string",
                    "example": "order-req-abc"
                },
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/handler.GatewayOrderItemBody"
                    }
                },
                "status": {
                    "type": "string",
                    "example": "pending"
                },
                "total_amount": {
                    "type": "integer",
                    "example": 5000
                },
                "user_id": {
                    "type": "string",
                    "example": "user-123"
                }
            }
        },
        "handler.GatewayOrderItemBody": {
            "type": "object",
            "properties": {
                "price": {
                    "type": "integer",
                    "example": 2500
                },
                "product_id": {
                    "type": "string",
                    "example": "prod-abc-123"
                },
                "quantity": {
                    "type": "integer",
                    "example": 2
                }
            }
        },
        "handler.GatewayPayment": {
            "type": "object",
            "properties": {
                "amount": {
                    "type": "integer",
                    "example": 5000
                },
                "created_at": {
                    "type": "string",
                    "example": "2026-03-13T00:00:00Z"
                },
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440001"
                },
                "method": {
                    "type": "string",
                    "example": "card"
                },
                "order_id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440000"
                },
                "status": {
                    "type": "string",
                    "example": "succeeded"
                },
                "transaction_id": {
                    "type": "string",
                    "example": "txn_abc123"
                },
                "user_id": {
                    "type": "string",
                    "example": "user-123"
                }
            }
        },
        "handler.GatewayRefreshBody": {
            "type": "object",
            "properties": {
                "refresh_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                }
            }
        },
        "handler.GatewayRegisterBody": {
            "type": "object",
            "properties": {
                "email": {
                    "type": "string",
                    "example": "alice@example.com"
                },
                "name": {
                    "type": "string",
                    "example": "Alice Smith"
                },
                "password": {
                    "type": "string",
                    "example": "s3cret!Pass"
                }
            }
        },
        "handler.GatewayTokenPairResponse": {
            "type": "object",
            "properties": {
                "access_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                },
                "expires_in": {
                    "type": "integer",
                    "example": 900
                },
                "refresh_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                }
            }
        },
        "handler.GatewayUserResponse": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string",
                    "example": "2026-03-13T00:00:00Z"
                },
                "email": {
                    "type": "string",
                    "example": "alice@example.com"
                },
                "id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440000"
                },
                "name": {
                    "type": "string",
                    "example": "Alice Smith"
                }
            }
        },
        "handler.GatewayWallet": {
            "type": "object",
            "properties": {
                "balance": {
                    "type": "integer",
                    "example": 10000
                },
                "created_at": {
                    "type": "string",
                    "example": "2026-03-13T00:00:00Z"
                },
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440002"
                },
                "user_id": {
                    "type": "string",
                    "example": "user-123"
                }
            }
        },
        "handler.HealthResponse": {
            "type": "object",
            "properties": {
                "service": {
                    "type": "string",
                    "example": "api-gateway"
                },
                "status": {
                    "type": "string",
                    "example": "ok"
                }
            }
        },
        "handler.ListOrdersResponse": {
            "type": "object",
            "properties": {
                "limit": {
                    "type": "integer",
                    "example": 20
                },
                "offset": {
                    "type": "integer",
                    "example": 0
                },
                "orders": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Order"
                    }
                }
            }
        },
        "handler.ListProductsResponse": {
            "type": "object",
            "properties": {
                "limit": {
                    "type": "integer",
                    "example": 20
                },
                "offset": {
                    "type": "integer",
                    "example": 0
                },
                "products": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Product"
                    }
                },
                "total": {
                    "type": "integer",
                    "example": 42
                }
            }
        },
        "handler.LoginRequest": {
            "type": "object",
            "required": [
                "email",
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string",
                    "example": "alice@example.com"
                },
                "password": {
                    "type": "string",
                    "example": "s3cret!Pass"
                }
            }
        },
        "handler.LogoutRequest": {
            "type": "object",
            "required": [
                "refresh_token"
            ],
            "properties": {
                "refresh_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                }
            }
        },
        "handler.PaymentErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "payment not found"
                }
            }
        },
        "handler.PaymentResponse": {
            "type": "object",
            "properties": {
                "amount": {
                    "type": "integer",
                    "example": 5000
                },
                "created_at": {
                    "type": "string",
                    "example": "2026-01-01T00:00:00Z"
                },
                "currency": {
                    "type": "string",
                    "example": "USD"
                },
                "failure_reason": {
                    "type": "string",
                    "example": "card_declined"
                },
                "id": {
                    "type": "string",
                    "example": "pay-uuid"
                },
                "method": {
                    "type": "string",
                    "example": "card"
                },
                "order_id": {
                    "type": "string",
                    "example": "order-uuid"
                },
                "status": {
                    "allOf": [
                        {
                            "$ref": "#/definitions/domain.PaymentStatus"
                        }
                    ],
                    "example": "succeeded"
                },
                "transaction_id": {
                    "type": "string",
                    "example": "txn_abc123"
                },
                "updated_at": {
                    "type": "string",
                    "example": "2026-01-01T00:01:00Z"
                },
                "user_id": {
                    "type": "string",
                    "example": "user-uuid"
                }
            }
        },
        "handler.ProductErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "product not found"
                }
            }
        },
        "handler.RefreshRequest": {
            "type": "object",
            "required": [
                "refresh_token"
            ],
            "properties": {
                "refresh_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                }
            }
        },
        "handler.RefundResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "payment refunded"
                },
                "order_id": {
                    "type": "string",
                    "example": "ord-uuid"
                },
                "payment_id": {
                    "type": "string",
                    "example": "pay-uuid"
                }
            }
        },
        "handler.RegisterRequest": {
            "type": "object",
            "required": [
                "email",
                "name",
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string",
                    "example": "alice@example.com"
                },
                "name": {
                    "type": "string",
                    "minLength": 2,
                    "example": "Alice Smith"
                },
                "password": {
                    "type": "string",
                    "minLength": 8,
                    "example": "s3cret!Pass"
                }
            }
        },
        "handler.ShippingAddressBody": {
            "type": "object",
            "required": [
                "city",
                "country",
                "name",
                "postal_code",
                "street"
            ],
            "properties": {
                "city": {
                    "type": "string",
                    "example": "New York"
                },
                "country": {
                    "type": "string",
                    "example": "US"
                },
                "name": {
                    "type": "string",
                    "example": "Alice Smith"
                },
                "postal_code": {
                    "type": "string",
                    "example": "10001"
                },
                "state": {
                    "type": "string",
                    "example": "NY"
                },
                "street": {
                    "type": "string",
                    "example": "123 Main St"
                }
            }
        },
        "handler.TokenPairResponse": {
            "type": "object",
            "properties": {
                "access_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                },
                "expires_in": {
                    "type": "integer",
                    "example": 900
                },
                "refresh_token": {
                    "type": "string",
                    "example": "eyJhbGci..."
                }
            }
        },
        "handler.UpdateProductRequest": {
            "type": "object",
            "properties": {
                "category": {
                    "type": "string",
                    "example": "electronics"
                },
                "description": {
                    "type": "string",
                    "example": "Updated description"
                },
                "name": {
                    "type": "string",
                    "example": "Wireless Headphones Pro"
                },
                "price": {
                    "type": "integer",
                    "example": 11999
                },
                "stock": {
                    "type": "integer",
                    "example": 200
                }
            }
        },
        "handler.UserResponse": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string",
                    "example": "2026-03-13T00:00:00Z"
                },
                "email": {
                    "type": "string",
                    "example": "alice@example.com"
                },
                "id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440000"
                },
                "name": {
                    "type": "string",
                    "example": "Alice Smith"
                },
                "role": {
                    "type": "string",
                    "example": "customer"
                }
            }
        },
        "handler.WalletErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "wallet not found"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "description": "Access token (15 min). Obtain from POST /auth/login or POST /auth/register.",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8086",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Auth Service",
	Description:      "Manages user accounts and JWT token issuance. Provides register, login, refresh, and logout endpoints.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
