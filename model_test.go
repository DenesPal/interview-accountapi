package interview_accountapi

import (
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestAccount_Validate(t *testing.T) {
	var account Account
	invalidAccounts := []Account{
		{},
		{OrganisationId: "abc", Type: "accounts", Attributes: &AccountAttributes{Country: "GB"}},
		{Id: "1234", Type: "accounts", Attributes: &AccountAttributes{Country: "GB"}},
		{Id: "1234", OrganisationId: "abc", Type: "accounts"},
		{Id: "1234", OrganisationId: "abc", Type: "accounts", Attributes: &AccountAttributes{}},
		{Id: "1234", OrganisationId: "abc", Type: "foobar", Attributes: &AccountAttributes{Country: "GB"}},
	}

	account = Account{Id: "1234", OrganisationId: "abc", Type: "accounts",
		Attributes: &AccountAttributes{Country: "GB"}}
	if account.Validate() != nil {
		t.Fatal("Mock Account #0 does not validate, fix the test.")
	}

	for i, account := range invalidAccounts {
		if account.Validate() == nil {
			jsonData, err := json.Marshal(account)
			if err != nil {
				t.Fatalf("json.Marshal failed: %s", err)
			}
			t.Errorf("Account #%d must not validate: %s", i, string(jsonData))
		}
	}

	account = Account{Id: "ad27e265-9605-4b4b-a0e5-3003ea9cc4dc",
		OrganisationId: "eb0bd6f5-c3f5-44b2-b677-acd23cdde73c",
		Attributes:     &AccountAttributes{Country: "SP"}}
	if account.Validate() != nil {
		t.Fatal("Mock Account #1 does not validate.")
	}
}

func TestAccount_Marshal(t *testing.T) {
	account := Account{Id: "ad27e265-9605-4b4b-a0e5-3003ea9cc4dc",
		OrganisationId: "eb0bd6f5-c3f5-44b2-b677-acd23cdde73c",
		Attributes:     &AccountAttributes{Country: "GB"},
		Type:           "accounts"}
	marshalled := `{"attributes":{"country":"GB"},"id":"ad27e265-9605-4b4b-a0e5-3003ea9cc4dc","organisation_id":"eb0bd6f5-c3f5-44b2-b677-acd23cdde73c","type":"accounts"}`

	jsonData, err := json.Marshal(account)
	if err != nil {
		log.Fatalf("Account marshalling failed: %s", err)
	}

	jsonStr := string(jsonData)
	if jsonStr != marshalled {
		t.Errorf("Incorrect account marshalling: %s", jsonStr)
	}
}

func TestAccount_Decode(t *testing.T) {
	const jsonString = `{
	   "data": {
		 "type": "accounts",
		 "id": "ad27e265-9605-4b4b-a0e5-3003ea9cc4dc",
		 "organisation_id": "eb0bd6f5-c3f5-44b2-b677-acd23cdde73c",
		 "attributes": {
		   "country": "GB",
		   "base_currency": "GBP",
		   "bank_id": "400300",
		   "bank_id_code": "GBDSC",
		   "bic": "NWBKGB22"
		 }
	   }
	 }`

	decoder := json.NewDecoder(strings.NewReader(jsonString))
	var ad AccountDetailsResponse
	err := decoder.Decode(&ad)
	if err != nil {
		t.Fatalf("Failed decoding AccountDetailsResponse: %s", err)
	}
	if ad.Data == nil {
		t.Fatalf("Failed decoding Account: %s", err)
	}
	account := ad.Data

	if account.Id != "ad27e265-9605-4b4b-a0e5-3003ea9cc4dc" {
		t.Errorf("Account.Id mismatch: %s", account.Id)
	}
	if account.OrganisationId != "eb0bd6f5-c3f5-44b2-b677-acd23cdde73c" {
		t.Errorf("Account.OrganisationId mismatch: %s", account.OrganisationId)
	}
	if account.Type != "accounts" {
		t.Errorf("Account.Type mismatch: %s", account.Type)
	}
	if account.Attributes == nil {
		t.Fatal("Missing Account.Attributes")
	}
	if account.Attributes.Country != "GB" {
		t.Errorf("Account.Attributes.Country mismatch: %s", account.Attributes.Country)
	}
}
