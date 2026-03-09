package domain

import "time"

type SubscriptionType string

const (
	SubscriptionTypeFree    SubscriptionType = "free"
	SubscriptionTypeBasic   SubscriptionType = "basic"
	SubscriptionTypePremium SubscriptionType = "premium"
)

type User struct {
	ID               int64            `json:"id" db:"id"`
	Age              int              `json:"age" db:"age"`
	Country          string           `json:"country" db:"country"`
	SubscriptionType SubscriptionType `json:"subscription_type" db:"subscription_type"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}