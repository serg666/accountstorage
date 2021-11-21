package chaincode

import (
	"log"
	"fmt"
	"time"
	"encoding/json"
	"github.com/shomali11/util/xhashes"
	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const accountIndex = "account~email"
const docTypeIndex = "doc~type"

type AccountStorage struct {
	contractapi.Contract
}

type Participant struct {
	DocType string
	Email   string
	Name    string
	Surname string
	Phone   string
	Passwd  string
}

type Account struct {
	ID       string
	Currency string
	Balance  int
	Email    string
}

type HistoryQueryResult struct {
	Record    *Account
	TxId      string
	Timestamp time.Time
	IsDelete  bool
}

// ParticipantExists returns true when participant with given Email exists in the ledger
func (t *AccountStorage) ParticipantExists(ctx contractapi.TransactionContextInterface, email string) (bool, error) {
	participantBytes, err := ctx.GetStub().GetState(email)
	if err != nil {
		return false, fmt.Errorf("failed to read participant %s from world state. %v", email, err)
	}

	return participantBytes != nil, nil
}

// CreatePatricipant initializes a new participant in the ledger
func (t *AccountStorage) CreateParticipant(ctx contractapi.TransactionContextInterface, email, name, surname, phone, passwd string) error {
	exists, err := t.ParticipantExists(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get participant: %v", err)
	}
	if exists {
		return fmt.Errorf("participan already exists: %s", email)
	}

	participant := &Participant{
		DocType: "participant",
		Email:   email,
		Name:    name,
		Surname: surname,
		Phone:   phone,
		Passwd:  xhashes.MD5(passwd),
	}
	participantBytes, err := json.Marshal(participant)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(email, participantBytes)
	if err != nil {
		return err
	}

	docTypeIndexKey, err := ctx.GetStub().CreateCompositeKey(docTypeIndex, []string{participant.DocType, participant.Email})
	if err != nil {
		return err
	}

	value := []byte{0x00}
	return ctx.GetStub().PutState(docTypeIndexKey, value)
}

// AccountExists returns true when account with given ID exists in the ledger.
func (t *AccountStorage) AccountExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	accountBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read account %s from world state. %v", id, err)
	}

	return accountBytes != nil, nil
}

// CreateAccount initializes a new account in the ledger
func (t *AccountStorage) CreateAccount(ctx contractapi.TransactionContextInterface, id, currency string, balance int, email string) error {
	log.Println("Init account")
	exists, err := t.AccountExists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get account: %v", err)
	}
	if exists {
		return fmt.Errorf("account already exists: %s", id)
	}

	account := &Account{
		ID:       id,
		Currency: currency,
		Balance:  balance,
		Email:    email,
	}
	accountBytes, err := json.Marshal(account)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, accountBytes)
	if err != nil {
		return err
	}

	accountEmailIndexKey, err := ctx.GetStub().CreateCompositeKey(accountIndex, []string{account.Email, account.ID})
	if err != nil {
		return err
	}

	value := []byte{0x00}
	return ctx.GetStub().PutState(accountEmailIndexKey, value)
}

// ReadParticipant retrieves an participant from the ledger
func (t *AccountStorage) ReadParticipant(ctx contractapi.TransactionContextInterface, email string) (*Participant, error) {
	participantBytes, err := ctx.GetStub().GetState(email)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant %s: %v", email, err)
	}
	if participantBytes == nil {
		return nil, fmt.Errorf("participant %s does not exist", email)
	}

	var participant Participant
	err = json.Unmarshal(participantBytes, &participant)
	if err != nil {
		return nil, err
	}

	return &participant, nil
}

// ReadAccount retrieves an account from the ledger
func (t *AccountStorage) ReadAccount(ctx contractapi.TransactionContextInterface, id string) (*Account, error) {
	accountBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %v", id, err)
	}
	if accountBytes == nil {
		return nil, fmt.Errorf("account %s does not exists", id)
	}

	var account Account
	err = json.Unmarshal(accountBytes, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// Transaction makes payment of x units from a to b
func (t *AccountStorage) Transaction(ctx contractapi.TransactionContextInterface, a, b string, x int) error {
	var sender, recipient *Account
	var err error

	sender, err = t.ReadAccount(ctx, a)
	if err != nil {
		return err
	}

	recipient, err = t.ReadAccount(ctx, b)
	if err != nil {
		return err
	}

	if sender.Currency != recipient.Currency {
		return fmt.Errorf("currency mismatch %s != %s", sender.Currency, recipient.Currency)
	}

	// Perform the execution
	sender.Balance -= x
	recipient.Balance += x

        log.Printf("sender balance = %d, recipient balance = %d\n", sender.Balance, recipient.Balance)

	senderBytes, err := json.Marshal(sender)
	if err != nil {
		return err
	}

	recipientBytes, err := json.Marshal(recipient)
	if err != nil {
		return err
	}

	// Write the state back to the ledger
	err = ctx.GetStub().PutState(a, senderBytes)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(b, recipientBytes)
}

// GetAllParticipants returns all participants
func (t *AccountStorage) GetAllParticipants(ctx contractapi.TransactionContextInterface) ([]*Participant, error) {
	participantResultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(docTypeIndex, []string{"participant"})
	if err != nil {
		return nil, err
	}
	defer participantResultsIterator.Close()

	var participants []*Participant
	for participantResultsIterator.HasNext() {
		responseRange, err := participantResultsIterator.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}

		if len(compositeKeyParts) > 1 {
			returnedParticipantEmail := compositeKeyParts[1]
			participant, err := t.ReadParticipant(ctx, returnedParticipantEmail)
			if err != nil {
				return nil, err
			}
			participants = append(participants, participant)
		}
	}

	return participants, nil
}

// GetParticipantAccounts retrieves all accounts for particular participant
func (t *AccountStorage) GetParticipantAccounts(ctx contractapi.TransactionContextInterface, email string) ([]*Account, error) {
	participantAccountResultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(accountIndex, []string{email})
	if err != nil {
		return nil, err
	}
	defer participantAccountResultsIterator.Close()

	var accounts []*Account
	for participantAccountResultsIterator.HasNext() {
		responseRange, err := participantAccountResultsIterator.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}

		if len(compositeKeyParts) > 1 {
			returnedAccountID := compositeKeyParts[1]
			account, err := t.ReadAccount(ctx, returnedAccountID)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

// GetAccountHistory returns the chain of custody for an account since issuance.
func (t *AccountStorage) GetAccountHistory(ctx contractapi.TransactionContextInterface, id string) ([]HistoryQueryResult, error) {
	log.Printf("GetAccountHistory: ID %v", id)

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var account Account
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &account)
			if err != nil {
				return nil, err
			}
		} else {
			account = Account{
				ID: id,
			}
		}

		timestamp, err := ptypes.Timestamp(response.Timestamp)
		if err != nil {
			return nil, err
		}

		record := HistoryQueryResult{
			TxId:      response.TxId,
			Timestamp: timestamp,
			Record:    &account,
			IsDelete:  response.IsDelete,
		}
		records = append(records, record)
	}

	return records, nil
}
