package interview_accountapi

import (
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

func (test *TestContext) ListAccounts(filters map[string]string) (map[string]uint, *ApiError) {
	test.T.Logf("ListAccounts(%s)", filters)

	results := test.Client.ListAccounts(filters)
	accountVersionMap := make(map[string]uint)
	for account := range results.Channel {
		accountVersionMap[account.Id] = account.Version
	}
	results.Close()

	return accountVersionMap, results.Error
}

func (test *TestContext) FetchAccount(id string) (*Account, *ApiError) {
	test.T.Logf("FetchAccount(%s)", id)
	return test.Client.FetchAccount(id)
}

func (test *TestContext) DeleteAccount(id string, version uint) *ApiError {
	test.T.Logf("DeleteAccount(%s, version=%d)", id, version)
	return test.Client.DeleteAccount(id, version)
}

func (test *TestContext) CreateAccount(accountBud *Account) (*Account, *ApiError) {
	test.T.Logf("CreateAccount(%s)", fmt.Sprint(accountBud))

	if accountBud == nil {
		accountBud = test.NewAccountBud()
	}

	account, apiErr := test.Client.CreateAccount(accountBud)

	if apiErr == nil {
		if e := printJson(account); e != nil {
			test.T.Log(e)
		}
	}

	return account, apiErr
}

func (test *TestContext) NewAccountBud() *Account {
	accountBud := &Account{
		Id:             uuid4s(),
		OrganisationId: uuid4s(),
		Attributes:     &AccountAttributes{Country: alpha2()},
	}
	return accountBud
}

func (test *TestContext) UpdateAccount(id string, updates *Account) (*Account, *ApiError) {
	test.T.Logf("UpdateAccount(%s)", id)

	if updates == nil {
		rand.Seed(time.Now().UnixNano())
		updates = &Account{
			Id:             id,
			OrganisationId: uuid4s(),
			Attributes:     &AccountAttributes{Country: alpha2()},
		}
	}

	account, apiErr := test.Client.UpdateAccount(id, updates)

	if apiErr == nil {
		if e := printJson(account); e != nil {
			test.T.Log(e)
		}
	}

	return account, apiErr
}

func (test *TestContext) CompareAccounts(acc *Account, bud *Account) {
	if acc.Id == "" {
		test.T.Error("Account id is empty")
	}
	if bud.Id != "" && acc.Id != bud.Id {
		test.T.Error("Account.Id mismatch")
	}
	if acc.OrganisationId != bud.OrganisationId {
		test.T.Error("Account.OrganisationId mismatch")
	}
	if bud.Attributes != nil && acc.Attributes == nil {
		test.T.Error("Account.Attributes missing")
	} else if acc.Attributes.Country != bud.Attributes.Country {
		test.T.Error("Account.Attributes.Country mismatch")
	}
}

func TestListAccounts(t *testing.T) {
	t.Log("TestListAccounts()")
	test := NewTestContext(t)
	accountVersionMap, err := test.ListAccounts(nil)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Seen %d accounts", len(accountVersionMap))
	}
}

func TestListAccountsFiltered(t *testing.T) {
	filters := map[string]string{"country": "GB"}
	t.Logf("TestListAccountsFiltered(%s)", fmt.Sprint(filters))
	test := NewTestContext(t)
	accountVersionMap, err := test.ListAccounts(filters)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Seen %d accounts for filter: %s", len(accountVersionMap), fmt.Sprint(filters))
	}
}

func TestCreateFetchAccount(t *testing.T) {
	t.Log("TestCreateFetchAccount()")
	test := NewTestContext(t)

	var account *Account
	var err *ApiError
	var accountBud = test.NewAccountBud()

	// Also checks for idempotence
	for i := 0; i < 3; i++ {
		account, err = test.CreateAccount(accountBud)
		if err != nil && (i < 1 || err.StatusCode != http.StatusConflict) {
			t.Fatal(err)
		}
		if account == nil {
			test.T.Fatal("Account is nil")
		}
	}

	test.CompareAccounts(account, accountBud)

	accountVersionMap, err := test.ListAccounts(nil)
	if err != nil {
		t.Fatal(err)
	} else if _, found := accountVersionMap[account.Id]; !found {
		t.Errorf("Account %s was not seen (has %d accounts)", account.Id, len(accountVersionMap))
	}

	account2, err := test.FetchAccount(account.Id)
	if err != nil {
		t.Fatalf("Failed to fetch account %s : %s", account.Id, err)
	}
	if account.Id != account2.Id {
		t.Errorf("Fetched account id mismatch %s %s", account.Id, account2.Id)
	}
}

func TestUpdateAccount(t *testing.T) {
	t.Log("TestUpdateAccount()")
	test := NewTestContext(t)

	origAccount, err := test.CreateAccount(nil)
	if err != nil {
		t.Fatal(err)
	}
	if origAccount == nil {
		test.T.Fatal("Account is nil")
	}

	account, err := test.FetchAccount(origAccount.Id)
	if account == nil {
		t.Fatalf("Failed to fetch account %s", origAccount.Id)
	}
	test.CompareAccounts(account, origAccount)

	updates := &Account{
		Id:             account.Id,
		OrganisationId: account.OrganisationId,
		Attributes:     &AccountAttributes{},
	}
	// make sure Country code is changed
	if updates.Attributes.Country == "XX" {
		updates.Attributes.Country = "XY"
	} else {
		updates.Attributes.Country = "XX"
	}

	updAccount, err := test.UpdateAccount(origAccount.Id, updates)
	if err != nil {
		if err.StatusCode == 404 {
			t.Skip("Update test is expected to fail here if PATCH is not implemented on mock backend.")
		} else {

			t.Fatal(err)
		}
	}

	if origAccount.Id != updAccount.Id {
		t.Fatalf("Account id %s mismatch %s", origAccount.Id, updAccount.Id)
	}
	if updAccount.Version == origAccount.Version+1 {
		t.Errorf("Account version %d is the same %s %s", origAccount.Version, origAccount.Id, updAccount.Id)
	}
	if origAccount.OrganisationId == updAccount.OrganisationId {
		t.Errorf("Account.OrganisationId %s should differ %s %s",
			origAccount.OrganisationId, origAccount.Id, updAccount.Id)
	}
	if origAccount.Attributes.Country != updAccount.Attributes.Country {
		t.Errorf("Account.Attributes.Country does not match %s %s", origAccount.Id, updAccount.Id)
	}

	accountVersionMap, err := test.ListAccounts(nil)
	if err != nil {
		t.Fatal(err)
	}

	version, found := accountVersionMap[updAccount.Id]
	if !found {
		t.Errorf("Account %s was not seen (has %d accounts)", updAccount.Id, len(accountVersionMap))
	} else if version != updAccount.Version {
		t.Errorf("Seen account version %d in list Vs %d %s", version, updAccount.Version, updAccount.Id)
	}
}

func TestDeleteAccount(t *testing.T) {
	t.Log("TestDeleteAccount()")
	test := NewTestContext(t)

	account, err := test.CreateAccount(nil)
	if err != nil {
		t.Fatal(err)
	}

	accountVersionMap, err := test.ListAccounts(nil)
	if err != nil {
		t.Error(err)
	} else if _, found := accountVersionMap[account.Id]; !found {
		t.Errorf("Account %s was not seen (has %d accounts)", account.Id, len(accountVersionMap))
	}

	account2, err := test.FetchAccount(account.Id)
	if account2 == nil {
		t.Fatal(err)
	}

	if err = test.DeleteAccount(account2.Id, account2.Version); err != nil {
		t.Fatal(err)
	}

	account, err = test.FetchAccount(account.Id)
	if err == nil || account != nil {
		t.Errorf("Account was fetched after delete %s", account2.Id)
	} else if err.StatusCode != 404 {
		t.Errorf("Received unexpected code %d while testing fetch-fail of deleted acount %s",
			err.StatusCode, account2.Id)
	}

	accountVersionMap, err = test.ListAccounts(nil)
	if err != nil {
		t.Error(err)
	} else if _, found := accountVersionMap[account2.Id]; found {
		t.Errorf("Account %s was seen after delete (has %d accounts)", account2.Id, len(accountVersionMap))
	}

	if err = test.DeleteAccount(account2.Id, account2.Version); err != nil {
		t.Fatalf("Replaying of Delete for account %s version %d failed: %s", account2.Id, account2.Version, err)
	}
}

func TestFetchAccountPagination(t *testing.T) {
	t.Log("TestFetchAccountPagination()")
	test := NewTestContext(t)
	test.Client.SetPageSize(1011)

	accountVersionMap, err := test.ListAccounts(nil)
	if err != nil {
		t.Fatal(err)
	}

	num := len(accountVersionMap)
	t.Logf("Has %d accounts", num)

	var c int
	for i := num; i < 10; i++ {
		_, err := test.CreateAccount(nil)
		if err != nil {
			t.Fatal(err)
		}
		c++
	}
	if c > 0 {
		t.Logf("Created %d accounts", c)
	}

	test.Client.SetPageSize((num + c) / 3)
	t.Logf("Paginating with page size %d", test.Client.PageSize())

	accountVersionMap, err = test.ListAccounts(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Seen %d accounts", len(accountVersionMap))
}

func TestDeleteAccount_no_id(t *testing.T) {
	var id string
	t.Log("TestDeleteAccount_no_id()")
	test := NewTestContext(t)
	apiErr := test.Client.DeleteAccount(id, 0)
	if apiErr == nil {
		t.Errorf("DeleteAccount(id, version) returned no error for empty id")
	}
}

func TestFetchAccount_no_id(t *testing.T) {
	var id string
	t.Log("TestFetchAccount_no_id()")
	test := NewTestContext(t)
	acc, apiErr := test.Client.FetchAccount(id)
	if apiErr == nil {
		t.Errorf("FetchAccount(id) returned no error for empty id")
	}
	if acc != nil {
		t.Errorf("FetchAccount(id) returned data for empty id")
	}
}
