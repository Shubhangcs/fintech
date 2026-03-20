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

type CommisionModel struct {
	CommisionID                int64     `json:"commision_id,omitempty"`
	UserID                     string    `json:"user_id"`
	Service                    string    `json:"service"`
	TotalCommision             float64   `json:"total_commision"`
	AdminCommision             float64   `json:"admin_commision"`
	MasterDistributorCommision float64   `json:"master_distributor_commision"`
	DistributorCommision       float64   `json:"distributor_commision"`
	RetailerCommision          float64   `json:"retailer_commision"`
	CreatedAT                  time.Time `json:"created_at"`
	UpdatedAT                  time.Time `json:"updated_at"`
}

func (c *CommisionModel) Validate() error {
	if c.UserID == "" {
		return errors.New("user_id is required")
	}
	if !validServices[c.Service] {
		return errors.New("service must be one of PAYOUT, DMT, AEPS, BBPS")
	}
	if c.TotalCommision < 0 {
		return errors.New("total_commision must be >= 0")
	}
	if c.AdminCommision == 0 && c.MasterDistributorCommision == 0 && c.DistributorCommision == 0 && c.RetailerCommision == 0 {
		c.AdminCommision = 1
	}
	return nil
}
