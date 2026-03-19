package models

import (
	"errors"
	"time"
)

var validServices = map[string]bool{
	"PAYOUT": true,
	"DMT":    true,
	"AEPS":   true,
	"BBPS":   true,
}

type CommissionModel struct {
	CommissionID                int64     `json:"commision_id,omitempty"`
	UserID                      string    `json:"user_id"`
	Service                     string    `json:"service"`
	TotalCommission             float64   `json:"total_commision"`
	AdminCommission             float64   `json:"admin_commision"`
	MasterDistributorCommission float64   `json:"master_distributor_commision"`
	DistributorCommission       float64   `json:"distributor_commision"`
	RetailerCommission          float64   `json:"retailer_commision"`
	CreatedAT                   time.Time `json:"created_at,omitempty"`
	UpdatedAT                   time.Time `json:"updated_at,omitempty"`
}

func (c *CommissionModel) Validate() error {
	if c.UserID == "" {
		return errors.New("user_id is required")
	}
	if !validServices[c.Service] {
		return errors.New("service must be one of PAYOUT, DMT, AEPS, BBPS")
	}
	if c.TotalCommission < 0 {
		return errors.New("total_commision must be >= 0")
	}
	return nil
}
