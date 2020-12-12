package interview_accountapi

import "errors"

type Account struct {
	Attributes     *AccountAttributes `json:"attributes"`
	Id             string             `json:"id"`              // UUID ex: 7826c3cb-d6fd-41d0-b187-dc23ba928772
	OrganisationId string             `json:"organisation_id"` // UUID ex: ee2fb143-6dfe-4787-b183-ca8ddd4164d2
	//Relationships   AccountRelationships `json:"relationships"`
	Type    string `json:"type,omitempty"`    // name of resource type ^[A-Za-z_]*$ ex: accounts
	Version uint   `json:"version,omitempty"` // version >= 0 ex: 0
}

// Validates Account and also sets default values //
func (account *Account) Validate() error {
	if account.Id == "" {
		return errors.New("Account.Id can not be empty")
	}

	if account.OrganisationId == "" {
		return errors.New("Account.OrganisationId can not be empty")
	}

	switch account.Type {
	case "":
		account.Type = "accounts"
	case "accounts":
		// pass //
	default:
		return errors.New("Account.Type should be one of [accounts]")
	}

	if account.Attributes == nil {
		return errors.New("Account.Attributes can not be empty")
	} else if err := account.Attributes.Validate(); err != nil {
		return err
	}

	return nil
}

type AccountAttributes struct {
	/*	account_classification         string // enum {Personal, Business} def: Personal
		account_matching_opt_out       bool
		account_number                 string */
	AlternativeBankAccountNames []string `json:"alternative_bank_account_names,omitempty"`
	/*	alternative_names              []string
		bank_account_name              string
		bank_id                        string
		bank_id_code                   string
		bic                            string */
	Country string `json:"country"` // ISO 3166-1 alpha-2 country code ^[A-Z]{2}$ //
	/*	customer_id                 string
		first_name                  string
		iban                        string
		joint_account               bool
		name                        [string]
		organisation_identification AccountAttributesOrganisationIdentification
		private_identification      AccountAttributesPrivateIdentification
		secondary_identification    string
		status                      string // enum {"pending", "failed", "confirmed"}
		switched                    bool
		title                       string */
}

// Validates AccountAttributes and could also set defaults //
func (attr *AccountAttributes) Validate() error {
	if attr.Country == "" {
		return errors.New("AccountAttributes.Country can not be empty")
	}
	return nil
}

/* type AccountRelationships struct {
	AccountEvents []RelationshipData  `json:"account_events"`
	MasterAccount []RelationshipData  `json:"master_account"`
}

type RelationshipData struct {
	Id   string `json:"id"` // uuid
	Type string `json:"type"`
} */

type AccountDetailsListResponse struct {
	Data  []*Account `json:"data"`
	Links *Links     `json:"links"`
}

type Links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next"`
	Prev  string `json:"prev"`
	Self  string `json:"self"`
}

type AccountCreation struct {
	Data *Account `json:"data"`
}

type AccountAmendment struct {
	// by schema it's AccountUpdate but in this exercise properties & requirements are same as for Account //
	Data *Account `json:"data"`
}

type AccountDetailsResponse struct {
	Data  *Account `json:"data"`
	Links *Links   `json:"links"`
}

type AccountCreationResponse AccountDetailsResponse

var accountListFilters = map[string]bool{
	"bank_id_code":   true,
	"bank_id":        true,
	"account_number": true,
	"iban":           true,
	"customer_id":    true,
	"country":        true,
}
