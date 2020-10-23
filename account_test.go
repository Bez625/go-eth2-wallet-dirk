// Copyright © 2020 Weald Technology Trading
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dirk_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/attestantio/dirk/testing/daemon"
	"github.com/attestantio/dirk/testing/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	e2types "github.com/wealdtech/go-eth2-types/v2"
	dirk "github.com/wealdtech/go-eth2-wallet-dirk"
	e2wtypes "github.com/wealdtech/go-eth2-wallet-types/v2"
)

func TestCreateAccount(t *testing.T) {
	err := e2types.InitBLS()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	port := uint32(12000 + rand.Intn(4000))
	_, path, err := daemon.New(ctx, "", 1, port,
		map[uint64]string{
			1: fmt.Sprintf("signer-test01:%d", port),
		})
	defer os.RemoveAll(path)
	require.NoError(t, err)

	endpoints := []*dirk.Endpoint{
		dirk.NewEndpoint("signer-test01", port),
	}

	credentials, err := dirk.Credentials(ctx,
		resources.ClientTest01Crt,
		resources.ClientTest01Key,
		resources.CACrt,
	)
	require.NoError(t, err)

	wallet, err := dirk.OpenWallet(ctx, "Wallet 1", credentials, endpoints)
	require.NoError(t, err)

	accountCreator, isAccountCreator := wallet.(e2wtypes.WalletAccountCreator)
	require.True(t, isAccountCreator)

	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	accountName := fmt.Sprintf("Test account %d", rand.Uint32())
	_, err = accountCreator.CreateAccount(context.Background(), accountName, []byte("pass"))
	require.NoError(t, err)

	require.NoError(t, wallet.(e2wtypes.WalletLocker).Lock(ctx))

	// Fetch the account to ensure it exists.
	account, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, accountName)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, account.ID())
	require.NotNil(t, account.PublicKey())
	require.NotNil(t, account.(e2wtypes.AccountWalletProvider).Wallet())
}

func TestUnlockAccount(t *testing.T) {
	err := e2types.InitBLS()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	port := uint32(12000 + rand.Intn(4000))
	capture, path, err := daemon.New(ctx, "", 1, port,
		map[uint64]string{
			1: fmt.Sprintf("signer-test01:%d", port),
		})
	defer os.RemoveAll(path)
	require.NoError(t, err)

	endpoints := []*dirk.Endpoint{
		dirk.NewEndpoint("signer-test01", port),
	}

	credentials, err := dirk.Credentials(ctx,
		resources.ClientTest01Crt,
		resources.ClientTest01Key,
		resources.CACrt,
	)
	require.NoError(t, err)

	wallet, err := dirk.OpenWallet(ctx, "Wallet 1", credentials, endpoints)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	account, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, "Account 1")
	fmt.Printf("%s", strings.Join(capture.Entries(), "\n"))
	require.NoError(t, err)

	// Unlock with incorrect passphrase.
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = account.(e2wtypes.AccountLocker).Unlock(ctx, []byte("bad"))
	assert.EqualError(t, err, "unlock attempt failed")

	// Unlock with correct passphrase.
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = account.(e2wtypes.AccountLocker).Unlock(ctx, []byte("pass"))
	require.NoError(t, err)

	unlocked, err := account.(e2wtypes.AccountLocker).IsUnlocked(ctx)
	require.NoError(t, err)
	require.True(t, unlocked)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assert.NoError(t, account.(e2wtypes.AccountLocker).Lock(ctx))
}

func TestSignGeneric(t *testing.T) {
	err := e2types.InitBLS()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	port := uint32(12000 + rand.Intn(4000))
	_, path, err := daemon.New(ctx, "", 1, port,
		map[uint64]string{
			1: fmt.Sprintf("signer-test01:%d", port),
		})
	defer os.RemoveAll(path)
	require.NoError(t, err)

	endpoints := []*dirk.Endpoint{
		dirk.NewEndpoint("signer-test01", port),
	}

	credentials, err := dirk.Credentials(ctx,
		resources.ClientTest01Crt,
		resources.ClientTest01Key,
		resources.CACrt,
	)
	require.NoError(t, err)

	wallet, err := dirk.OpenWallet(ctx, "Wallet 1", credentials, endpoints)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	account, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, "Account 1")
	require.NoError(t, err)

	tests := []struct {
		name   string
		data   []byte
		domain []byte
		err    string
		sig    []byte
	}{
		{
			name: "ProposerDomain",
			data: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			err: "request to obtain signature denied",
		},
		{
			name: "AttesterDomain",
			data: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			err: "request to obtain signature denied",
		},
		{
			name: "DataLengthIncorrect",
			data: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			err: "data must be 32 bytes in length",
		},
		{
			name: "Good",
			data: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sig: []byte{
				0x8e, 0x16, 0x21, 0x7c, 0xfb, 0x18, 0xe2, 0xf2, 0xb2, 0xc8, 0x88, 0x5b, 0x02, 0xd7, 0x34, 0x36,
				0x00, 0x69, 0x03, 0xba, 0x77, 0x32, 0x0b, 0x43, 0xa8, 0xcd, 0x7b, 0x60, 0x30, 0xbe, 0x67, 0x94,
				0x95, 0x46, 0x38, 0x1e, 0xfb, 0xd0, 0x9e, 0x8d, 0x21, 0x47, 0x85, 0x5b, 0x05, 0xad, 0x8c, 0xc9,
				0x11, 0x93, 0x33, 0xf4, 0x28, 0x99, 0xaa, 0xf7, 0x45, 0xa7, 0x61, 0x1e, 0x4f, 0xad, 0x52, 0xaa,
				0x08, 0xe6, 0xa2, 0x80, 0xe1, 0xef, 0x4e, 0xf9, 0xc5, 0x3c, 0x42, 0x60, 0x28, 0xca, 0xbf, 0x5b,
				0x45, 0xd7, 0x3c, 0xb5, 0xbc, 0x8c, 0x34, 0x3c, 0xd9, 0x44, 0xa9, 0x99, 0xda, 0x1e, 0x6f, 0x4e,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sig, err := account.(e2wtypes.AccountProtectingSigner).SignGeneric(ctx, test.data, test.domain)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.sig, sig.Marshal())
			}
		})
	}
}

func TestSignBeaconProposal(t *testing.T) {
	err := e2types.InitBLS()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	port := uint32(12000 + rand.Intn(4000))
	_, path, err := daemon.New(ctx, "", 1, port,
		map[uint64]string{
			1: fmt.Sprintf("signer-test01:%d", port),
		})
	defer os.RemoveAll(path)
	require.NoError(t, err)

	endpoints := []*dirk.Endpoint{
		dirk.NewEndpoint("signer-test01", port),
	}

	credentials, err := dirk.Credentials(ctx,
		resources.ClientTest01Crt,
		resources.ClientTest01Key,
		resources.CACrt,
	)
	require.NoError(t, err)

	wallet, err := dirk.OpenWallet(ctx, "Wallet 1", credentials, endpoints)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	account, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, "Account 1")
	require.NoError(t, err)

	tests := []struct {
		name          string
		slot          uint64
		proposerIndex uint64
		parentRoot    []byte
		stateRoot     []byte
		bodyRoot      []byte
		domain        []byte
		err           string
		sig           []byte
	}{
		{
			name:          "Good",
			slot:          1,
			proposerIndex: 1,
			parentRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			stateRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			bodyRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sig: []byte{
				0x91, 0x76, 0xac, 0x06, 0xd4, 0x26, 0xbe, 0xb7, 0x4b, 0xfb, 0x39, 0x69, 0xa6, 0x19, 0xd3, 0x64,
				0xa5, 0xe2, 0x14, 0xf0, 0xda, 0x51, 0x14, 0x33, 0xb5, 0x6f, 0x8d, 0xca, 0x47, 0xb9, 0x37, 0x25,
				0xc4, 0xdd, 0xd2, 0x25, 0xc1, 0xcb, 0x81, 0xdd, 0x5a, 0x26, 0xfc, 0x32, 0x18, 0x01, 0xa1, 0xd7,
				0x0b, 0x1c, 0x28, 0xc6, 0xcf, 0x70, 0x5c, 0x01, 0x0f, 0xad, 0xf9, 0xc4, 0xf1, 0x39, 0xc0, 0x44,
				0x22, 0x46, 0x80, 0xd8, 0xc6, 0x0e, 0x42, 0x5c, 0x9d, 0x6e, 0x6b, 0x8e, 0x8a, 0xf2, 0x1f, 0x12,
				0xcf, 0xfa, 0xb3, 0x8b, 0x39, 0x3a, 0x9b, 0x75, 0x29, 0x52, 0x7b, 0x92, 0xcf, 0x20, 0xe8, 0x0a,
			},
		},
		{
			name:          "Repeat",
			slot:          1,
			proposerIndex: 1,
			parentRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			stateRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			bodyRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			err: "request to obtain signature denied",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sig, err := account.(e2wtypes.AccountProtectingSigner).SignBeaconProposal(ctx, test.slot, test.proposerIndex, test.parentRoot, test.stateRoot, test.bodyRoot, test.domain)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.sig, sig.Marshal())
			}
		})
	}
}

func TestSignBeaconAttestation(t *testing.T) {
	err := e2types.InitBLS()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	port := uint32(12000 + rand.Intn(4000))
	_, path, err := daemon.New(ctx, "", 1, port,
		map[uint64]string{
			1: fmt.Sprintf("signer-test01:%d", port),
		})
	defer os.RemoveAll(path)
	require.NoError(t, err)

	endpoints := []*dirk.Endpoint{
		dirk.NewEndpoint("signer-test01", port),
	}

	credentials, err := dirk.Credentials(ctx,
		resources.ClientTest01Crt,
		resources.ClientTest01Key,
		resources.CACrt,
	)
	require.NoError(t, err)

	wallet, err := dirk.OpenWallet(ctx, "Wallet 1", credentials, endpoints)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	account, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, "Account 1")
	require.NoError(t, err)

	tests := []struct {
		name           string
		slot           uint64
		committeeIndex uint64
		blockRoot      []byte
		sourceEpoch    uint64
		sourceRoot     []byte
		targetEpoch    uint64
		targetRoot     []byte
		domain         []byte
		err            string
		sig            []byte
	}{
		{
			name:           "Good",
			slot:           1,
			committeeIndex: 1,
			blockRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sourceEpoch: 0,
			sourceRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			targetEpoch: 1,
			targetRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sig: []byte{
				0x84, 0xa1, 0xc3, 0xc5, 0x0d, 0x09, 0x39, 0x01, 0xea, 0x1b, 0x02, 0x7b, 0x18, 0x59, 0x8e, 0x4b,
				0x9c, 0xf0, 0xf8, 0x48, 0xf6, 0xbd, 0x49, 0xf2, 0x80, 0x7a, 0x3f, 0x6e, 0xa3, 0x7c, 0x0c, 0xbf,
				0x37, 0x94, 0x55, 0x67, 0x05, 0x86, 0x3d, 0xe0, 0xae, 0x8e, 0xa7, 0xdd, 0x2d, 0xa4, 0xd4, 0xd9,
				0x14, 0x36, 0xe2, 0xca, 0x96, 0xfa, 0x1e, 0xb0, 0x45, 0xa2, 0x2f, 0xb7, 0x70, 0x4c, 0xed, 0xf8,
				0xa8, 0x42, 0xfa, 0x88, 0x1a, 0x41, 0x6e, 0xaa, 0x02, 0x44, 0x44, 0x54, 0xd9, 0xf7, 0xf8, 0x04,
				0x0b, 0x84, 0xfc, 0x3c, 0xd3, 0xd4, 0x28, 0x17, 0xf6, 0x99, 0x2c, 0x3c, 0x29, 0xe1, 0x60, 0x07,
			},
		},
		{
			name:           "Repeat",
			slot:           1,
			committeeIndex: 1,
			blockRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sourceEpoch: 0,
			sourceRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			targetEpoch: 1,
			targetRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			err: "request to obtain signature denied",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sig, err := account.(e2wtypes.AccountProtectingSigner).SignBeaconAttestation(ctx, test.slot, test.committeeIndex, test.blockRoot, test.sourceEpoch, test.sourceRoot, test.targetEpoch, test.targetRoot, test.domain)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.sig, sig.Marshal())
			}
		})
	}
}

func TestSignBeaconAttestations(t *testing.T) {
	err := e2types.InitBLS()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// #nosec G404
	port := uint32(12000 + rand.Intn(4000))
	_, path, err := daemon.New(ctx, "", 1, port,
		map[uint64]string{
			1: fmt.Sprintf("signer-test01:%d", port),
		})
	defer os.RemoveAll(path)
	require.NoError(t, err)

	endpoints := []*dirk.Endpoint{
		dirk.NewEndpoint("signer-test01", port),
	}

	credentials, err := dirk.Credentials(ctx,
		resources.ClientTest01Crt,
		resources.ClientTest01Key,
		resources.CACrt,
	)
	require.NoError(t, err)

	wallet, err := dirk.OpenWallet(ctx, "Wallet 1", credentials, endpoints)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	account1, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, "Account 1")
	require.NoError(t, err)
	account2, err := wallet.(e2wtypes.WalletAccountByNameProvider).AccountByName(ctx, "Account 2")
	require.NoError(t, err)

	tests := []struct {
		name             string
		slot             uint64
		accounts         []e2wtypes.Account
		committeeIndices []uint64
		blockRoot        []byte
		sourceEpoch      uint64
		sourceRoot       []byte
		targetEpoch      uint64
		targetRoot       []byte
		domain           []byte
		err              string
		sigs             [][]byte
	}{
		{
			name:             "Good",
			slot:             1,
			accounts:         []e2wtypes.Account{account1, account2},
			committeeIndices: []uint64{1, 2},
			blockRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sourceEpoch: 0,
			sourceRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			targetEpoch: 1,
			targetRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sigs: [][]byte{
				{
					0x84, 0xa1, 0xc3, 0xc5, 0x0d, 0x09, 0x39, 0x01, 0xea, 0x1b, 0x02, 0x7b, 0x18, 0x59, 0x8e, 0x4b,
					0x9c, 0xf0, 0xf8, 0x48, 0xf6, 0xbd, 0x49, 0xf2, 0x80, 0x7a, 0x3f, 0x6e, 0xa3, 0x7c, 0x0c, 0xbf,
					0x37, 0x94, 0x55, 0x67, 0x05, 0x86, 0x3d, 0xe0, 0xae, 0x8e, 0xa7, 0xdd, 0x2d, 0xa4, 0xd4, 0xd9,
					0x14, 0x36, 0xe2, 0xca, 0x96, 0xfa, 0x1e, 0xb0, 0x45, 0xa2, 0x2f, 0xb7, 0x70, 0x4c, 0xed, 0xf8,
					0xa8, 0x42, 0xfa, 0x88, 0x1a, 0x41, 0x6e, 0xaa, 0x02, 0x44, 0x44, 0x54, 0xd9, 0xf7, 0xf8, 0x04,
					0x0b, 0x84, 0xfc, 0x3c, 0xd3, 0xd4, 0x28, 0x17, 0xf6, 0x99, 0x2c, 0x3c, 0x29, 0xe1, 0x60, 0x07,
				},
				{
					0x80, 0xec, 0x2b, 0x5f, 0x8d, 0x35, 0x1e, 0x33, 0xe2, 0xbb, 0x6b, 0x2d, 0x2c, 0xfe, 0x56, 0x70,
					0xbc, 0xa1, 0xe1, 0x8a, 0x3b, 0xac, 0x94, 0x8d, 0xac, 0x81, 0x13, 0x78, 0x3b, 0x55, 0x01, 0xe8,
					0xb0, 0x7d, 0xde, 0x23, 0x95, 0x70, 0x34, 0x0d, 0x94, 0x3c, 0xe0, 0x2f, 0x90, 0x9a, 0x2e, 0xa2,
					0x03, 0xb3, 0xb6, 0xf5, 0xc4, 0xc9, 0xa5, 0x1e, 0xc9, 0x3e, 0xb9, 0xb8, 0xc6, 0x71, 0xeb, 0x5d,
					0xed, 0xa0, 0x90, 0x2c, 0x20, 0xa0, 0x4d, 0x82, 0x96, 0x25, 0xfc, 0x92, 0x4d, 0xef, 0x7d, 0xd0,
					0xbf, 0xca, 0xc1, 0xe1, 0x9e, 0x10, 0xf8, 0xf0, 0xb8, 0x07, 0x80, 0x5c, 0x44, 0x88, 0xca, 0xc0,
				},
			},
		},
		{
			name:             "Repeat2",
			slot:             1,
			accounts:         []e2wtypes.Account{account1, account2},
			committeeIndices: []uint64{1, 2},
			blockRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			sourceEpoch: 0,
			sourceRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			targetEpoch: 1,
			targetRoot: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			domain: []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			err: "request to obtain signatures denied",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sigs, err := account1.(e2wtypes.AccountProtectingMultiSigner).SignBeaconAttestations(ctx, test.slot, test.accounts, test.committeeIndices, test.blockRoot, test.sourceEpoch, test.sourceRoot, test.targetEpoch, test.targetRoot, test.domain)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(test.sigs), len(sigs))
				for i := range sigs {
					require.Equal(t, test.sigs[i], sigs[i].Marshal())
				}
			}
		})
	}
}
