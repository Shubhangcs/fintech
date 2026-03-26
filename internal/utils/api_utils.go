package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	PaysprintAPI        = os.Getenv("PAYSPRINT_API")
	RechargeKitAPI1     = os.Getenv("RECHARGE_KIT_API_1")
	RechargeKitAPI2     = os.Getenv("RECHARGE_KIT_API_2")
	RechargeKitAPIToken = os.Getenv("RECHARGE_KIT_API_TOKEN")
)

const (
	PennyDrop      = "/api/v1/verification/penny_drop_v2"
	Payout         = "/rkitpayout/payoutTransfer"
	PayoutStatus   = "/recharge/statusCheck"
	MobileRecharge         = "/recharge/prepaid"
	PostpaidMobileRecharge = "/recharge/postpaid"
	PrepaidPlanFetch       = "/recharge/prepaidPlanFetch"
	PostpaidBillFetch      = "/recharge/postPaidBillFetch"
	DTHRecharge            = "/recharge/dth"
	ElectricityBill        = "/recharge/electricity"
)

var apiHTTPClient = &http.Client{Timeout: 30 * time.Second}

func PostRequest(url, authKey, authValue string, body map[string]any, res any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("PostRequest marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("PostRequest build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(authKey, authValue)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("PostRequest do: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		return fmt.Errorf("PostRequest decode (status %d): %w", resp.StatusCode, err)
	}
	return nil
}

func GetRequest(url, authKey, authValue string, res any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("GetRequest build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authKey, authValue)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("GetRequest do: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		return fmt.Errorf("GetRequest decode (status %d): %w", resp.StatusCode, err)
	}
	return nil
}
