package models

// PaddleWebhookEvent represents the top-level Paddle webhook payload
type PaddleWebhookEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	OccurredAt string               `json:"occurred_at"`
	Data      PaddleTransactionData  `json:"data"`
}

// PaddleTransactionData contains the transaction details
type PaddleTransactionData struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	CustomerID string          `json:"customer_id"`
	Items      []PaddleItem    `json:"items"`
	Customer   PaddleCustomer  `json:"customer"`
}

// PaddleItem represents a line item in the transaction
type PaddleItem struct {
	Price PaddlePrice `json:"price"`
}

// PaddlePrice holds the price ID for plan mapping
type PaddlePrice struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
}

// PaddleCustomer contains the customer info from the webhook
type PaddleCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
