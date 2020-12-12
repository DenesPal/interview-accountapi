package interview_accountapi

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func (test *TestContext) ListAccounts(filters map[string]string) map[string]uint {
	test.T.Logf("ListAccounts(%s)", filters)

	channel, err := test.Client.ListAccounts(filters)
	if err != nil {
		test.T.Log("Failed listing accounts: ", err)
		return nil
	}

	accountVersionMap := make(map[string]uint)
	for account := range channel {
		accountVersionMap[account.Id] = account.Version
	}

	return accountVersionMap
}

func (test *TestContext) FetchAccount(id string) *Account {
	test.T.Logf("FetchAccount(%s)", id)
	account, err := test.Client.FetchAccount(id)
	if err != nil {
		test.T.Logf("Failed fetching account %s: %s", id, err)
		return nil
	}
	return account
}

func (test *TestContext) DeleteAccount(id string, version uint) bool {
	test.T.Logf("DeleteAccount(%s)", id)
	err := test.Client.DeleteAccount(id, version)
	if err != nil {
		test.T.Logf("Failed deleting account %s: %s", id, err)
		return false
	}
	return true
}

func (test *TestContext) CreateAccount(accountBud *Account) *Account {
	test.T.Logf("CreateAccount(%s)", fmt.Sprint(accountBud))

	if accountBud == nil {
		rand.Seed(time.Now().UnixNano())
		accountBud = &Account{
			Id:             uuid4s(),
			OrganisationId: uuid4s(),
			Attributes:     &AccountAttributes{Country: alpha2()},
		}
	}

	account, err := test.Client.CreateAccount(accountBud)

	if err != nil {
		test.T.Log("Failed to create account: ", err)
		return nil
	} else {
		if err = printJson(account); err != nil {
			test.T.Log(err)
		}
	}

	return account
}

func (test *TestContext) UpdateAccount(id string, updates *Account) *Account {
	test.T.Logf("UpdateAccount(%s)", fmt.Sprint(updates))

	if updates == nil {
		rand.Seed(time.Now().UnixNano())
		updates = &Account{
			Id:             id,
			OrganisationId: uuid4s(),
			Attributes:     &AccountAttributes{Country: alpha2()},
		}
	}

	account, err := test.Client.UpdateAccount(id, updates)

	if err != nil {
		test.T.Logf("Failed to update account %s: %s", id, err)
		return nil
	} else {
		if err = printJson(account); err != nil {
			test.T.Log(err)
		}
	}

	return account
}

func TestListAccounts(t *testing.T) {
	t.Log("TestListAccounts()")
	test := NewTestContext(t)
	accountVersionMap := test.ListAccounts(nil)
	if accountVersionMap == nil {
		t.Fail()
	} else {
		t.Logf("Seen %d accounts", len(accountVersionMap))
	}
}

func TestListAccountsFiltered(t *testing.T) {
	filters := map[string]string{"country": "GB"}
	t.Logf("TestListAccountsFiltered(%s)", fmt.Sprint(filters))
	test := NewTestContext(t)
	accountVersionMap := test.ListAccounts(filters)
	if accountVersionMap == nil {
		t.Fail()
	} else {
		t.Logf("Seen %d accounts for filter:", len(accountVersionMap), fmt.Sprint(filters))
	}
}

// Tests Create & Fetch //
func TestCreateAccount(t *testing.T) {
	t.Log("TestCreateAccount()")
	test := NewTestContext(t)

	account := test.CreateAccount(nil)
	if account == nil {
		t.Fail()
		return
	}

	accountVersionMap := test.ListAccounts(nil)
	if accountVersionMap == nil {
		t.Fail()
	} else if _, found := accountVersionMap[account.Id]; !found {
		t.Logf("Account %s was not seen (has %d accounts)", account.Id, len(accountVersionMap))
		t.Fail()
	}

	account2 := test.FetchAccount(account.Id)
	if account2 == nil {
		t.Logf("Failed to fetch account %s", account.Id)
		t.Fail()
		return
	}
	if account.Id != account2.Id {
		t.Logf("Fetched account id mismatch %s %s", account.Id, account2.Id)
		t.Fail()
	}
}

func TestUpdateAccount(t *testing.T) {
	t.Log("TestUpdateAccount()")
	test := NewTestContext(t)

	origAccount := test.CreateAccount(nil)
	if origAccount == nil {
		t.Fail()
		return
	}

	account := test.FetchAccount(origAccount.Id)
	if account == nil {
		t.Logf("Failed to fetch account %s", origAccount.Id)
		t.Fail()
		return
	}
	if origAccount.Id != account.Id {
		t.Logf("Fetched account id mismatch %s %s", origAccount.Id, account.Id)
		t.Fail()
	}

	updates := &Account{
		Id:             account.Id,
		OrganisationId: uuid4s(),
		Attributes:     &AccountAttributes{},
	}
	// make sure Country code is changed //
	if updates.Attributes.Country == "XX" {
		updates.Attributes.Country = "XY"
	} else {
		updates.Attributes.Country = "XX"
	}

	updAccount := test.UpdateAccount(origAccount.Id, updates)
	if updAccount == nil {
		t.Fail()
		return
	}

	if origAccount.Id != updAccount.Id {
		t.Logf("Account id %s mismatch %s", origAccount.Id, updAccount.Id)
		t.Fail()
		return
	}
	if updAccount.Version == origAccount.Version+1 {
		t.Logf("Account version %d is the same %s %s", origAccount.Version, origAccount.Id, updAccount.Id)
		t.Fail()
	}
	if origAccount.OrganisationId == updAccount.OrganisationId {
		t.Logf("Account.OrganisationId %s should differ %s %s", origAccount.OrganisationId, origAccount.Id, updAccount.Id)
		t.Fail()
	}
	if origAccount.Attributes.Country != updAccount.Attributes.Country {
		t.Logf("Account.Attributes.Country does not match %s %s", origAccount.Id, updAccount.Id)
		t.Fail()
	}

	accountVersionMap := test.ListAccounts(nil)
	if accountVersionMap == nil {
		t.Fail()
		return
	}

	version, found := accountVersionMap[updAccount.Id]
	if !found {
		t.Logf("Account %s was not seen (has %d accounts)", updAccount.Id, len(accountVersionMap))
		t.Fail()
	} else if version != updAccount.Version {
		t.Logf("Seen account version %d in list Vs %d %s", version, updAccount.Version, updAccount.Id)
		t.Fail()
	}
}

func TestDeleteAccount(t *testing.T) {
	t.Log("TestDeleteAccount()")
	test := NewTestContext(t)

	account := test.CreateAccount(nil)
	if account == nil {
		t.Fail()
		return
	}

	accountVersionMap := test.ListAccounts(nil)
	if accountVersionMap == nil {
		t.Fail()
	} else if _, found := accountVersionMap[account.Id]; !found {
		t.Logf("Account %s was not seen (has %d accounts)", account.Id, len(accountVersionMap))
		t.Fail()
	}

	account2 := test.FetchAccount(account.Id)
	if account2 == nil {
		t.Logf("Failed to fetch account %s", account.Id)
		t.Fail()
		return
	}

	if !test.DeleteAccount(account.Id, account2.Version) {
		t.Logf("Failed to delete account %s version %d", account.Id, account2.Version)
		t.Fail()
		return
	}

	account2 = test.FetchAccount(account.Id)
	if account2 != nil {
		t.Logf("Account was fetched after delete %s", account.Id)
		t.Fail()
	}

	accountVersionMap = test.ListAccounts(nil)
	if accountVersionMap == nil {
		t.Fail()
	} else if _, found := accountVersionMap[account.Id]; found {
		t.Logf("Account %s was seen after delete (has %d accounts)", account.Id, len(accountVersionMap))
		t.Fail()
	}
}

func TestFetchAccountPagination(t *testing.T) {
	t.Log("TestCreateAccount()")
	test := NewTestContext(t)
	test.Client.PageSize = 5

	accountVersionMap := test.ListAccounts(nil)
	if accountVersionMap == nil {
		t.Fail()
		return
	}
	t.Logf("Seen %d accounts", len(accountVersionMap))

	var c int
	for i := len(accountVersionMap); i < int(test.Client.PageSize)*3+1; i++ {
		account := test.CreateAccount(nil)
		if account == nil {
			t.Fail()
			return
		}
		c++
	}

	if c > 0 {
		t.Logf("Created %d accounts", c)

		accountVersionMap = test.ListAccounts(nil)
		if accountVersionMap == nil {
			t.Fail()
			return
		}
		t.Logf("Seen %d accounts", len(accountVersionMap))
	}
}
