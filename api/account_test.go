package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/qwerqy/mock_bank/db/mock"
	db "github.com/qwerqy/mock_bank/db/sqlc"
	"github.com/qwerqy/mock_bank/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAccountAPI(t *testing.T) {
	params := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: util.RandomCurrency(),
		Balance:  0,
	}

	invalidParams := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: "A",
		Balance:  0,
	}

	testCases := []struct {
		name          string
		params        db.CreateAccountParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			params: params,
			buildStubs: func(store *mockdb.MockStore) {

				//build stubs
				store.EXPECT().CreateAccount(gomock.Any(), params).Times(1).Return(db.Account{Owner: params.Owner, Currency: params.Currency}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name:   "InternalServerError",
			params: params,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

			},
		},
		{
			name:   "BadRequest",
			params: invalidParams,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), invalidParams).Times(0).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusBadRequest, recorder.Code)

			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			args := createAccountRequest{
				Owner:    tc.params.Owner,
				Currency: tc.params.Currency,
			}

			json, err := json.Marshal(args)
			require.NoError(t, err)

			body := []byte(json)

			url := "/accounts"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetAccountAPI(t *testing.T) {
	account := randomAccount()

	testCases := []struct {
		name          string
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccount(t, recorder.Body, account)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusNotFound, recorder.Code)

			},
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

			},
		},
		{
			name:      "InvalidId",
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusBadRequest, recorder.Code)

			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountsAPI(t *testing.T) {
	var accounts []db.Account
	for i := 0; i < 5; i++ {
		accounts = append(accounts, randomAccount())
	}

	req := listAccountsRequest{
		PageID:   1,
		PageSize: 5,
	}
	testCases := []struct {
		name          string
		req           listAccountsRequest
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			req:  req,
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListAccountsParams{
					Limit:  req.PageSize,
					Offset: (req.PageID - 1) * req.PageSize,
				}
				//build stubs
				store.EXPECT().ListAccounts(gomock.Any(), arg).Times(1).Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccounts(t, recorder.Body, accounts)
			},
		},
		{
			name: "NotFound",
			req:  req,
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListAccountsParams{
					Limit:  req.PageSize,
					Offset: (req.PageID - 1) * req.PageSize,
				}
				//build stubs
				store.EXPECT().ListAccounts(gomock.Any(), arg).Times(1).Return([]db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			req:  req,
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.ListAccountsParams{
					Limit:  req.PageSize,
					Offset: (req.PageID - 1) * req.PageSize,
				}
				//build stubs
				store.EXPECT().ListAccounts(gomock.Any(), arg).Times(1).Return([]db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

			},
		},
		{
			name: "InvalidPageId",
			req: listAccountsRequest{
				PageID:   0,
				PageSize: 5,
			},
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusBadRequest, recorder.Code)

			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			// start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts?page_id=%[1]d&page_size=%[2]d", tc.req.PageID, tc.req.PageSize)

			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

//TODO: Complete update account test
func TestUpdateAccountAPI(t *testing.T) {
	params := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: util.RandomCurrency(),
		Balance:  0,
	}

	invalidParams := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: "A",
		Balance:  0,
	}

	testCases := []struct {
		name          string
		params        db.CreateAccountParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			params: params,
			buildStubs: func(store *mockdb.MockStore) {

				//build stubs
				store.EXPECT().CreateAccount(gomock.Any(), params).Times(1).Return(db.Account{Owner: params.Owner, Currency: params.Currency}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name:   "InternalServerError",
			params: params,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

			},
		},
		{
			name:   "BadRequest",
			params: invalidParams,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), invalidParams).Times(0).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusBadRequest, recorder.Code)

			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			args := createAccountRequest{
				Owner:    tc.params.Owner,
				Currency: tc.params.Currency,
			}

			json, err := json.Marshal(args)
			require.NoError(t, err)

			body := []byte(json)

			url := "/accounts"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

//TODO: Complete delete account test
func TestDeleteAccountAPI(t *testing.T) {
	params := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: util.RandomCurrency(),
		Balance:  0,
	}

	invalidParams := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: "A",
		Balance:  0,
	}

	testCases := []struct {
		name          string
		params        db.CreateAccountParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			params: params,
			buildStubs: func(store *mockdb.MockStore) {

				//build stubs
				store.EXPECT().CreateAccount(gomock.Any(), params).Times(1).Return(db.Account{Owner: params.Owner, Currency: params.Currency}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name:   "InternalServerError",
			params: params,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

			},
		},
		{
			name:   "BadRequest",
			params: invalidParams,
			buildStubs: func(store *mockdb.MockStore) {
				//build stubs
				store.EXPECT().GetAccount(gomock.Any(), invalidParams).Times(0).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check responses
				require.Equal(t, http.StatusBadRequest, recorder.Code)

			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			args := createAccountRequest{
				Owner:    tc.params.Owner,
				Currency: tc.params.Currency,
			}

			json, err := json.Marshal(args)
			require.NoError(t, err)

			body := []byte(json)

			url := "/accounts"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomAccount() db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, account db.Account) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotAccount db.Account
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account, gotAccount)
}

func requireBodyMatchAccounts(t *testing.T, body *bytes.Buffer, accounts []db.Account) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotAccounts []db.Account
	err = json.Unmarshal(data, &gotAccounts)
	require.NoError(t, err)
	require.Equal(t, accounts, gotAccounts)
}
