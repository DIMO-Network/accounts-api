package rpc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/DIMO-Network/accounts-api/models"
	pb "github.com/DIMO-Network/accounts-api/pkg/grpc"
	"github.com/DIMO-Network/shared/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedAccountsServer
	DBS db.Store
}

var emailJoin = fmt.Sprintf("%s ON %s = %s", models.TableNames.Emails, models.EmailTableColumns.AccountID, models.AccountTableColumns.ID)
var walletJoin = fmt.Sprintf("%s ON %s = %s", models.TableNames.Wallets, models.WalletTableColumns.AccountID, models.AccountTableColumns.ID)

var emailHas = fmt.Sprintf("position(? in %s) > 0", models.EmailTableColumns.Address)
var walletHas = fmt.Sprintf("position(? in %s) > 0", models.WalletTableColumns.Address)

func (s *Server) ListAccounts(ctx context.Context, in *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	var mods = []qm.QueryMod{
		qm.Load(models.AccountRels.Email),
		qm.Load(models.AccountRels.Wallet),
		qm.LeftOuterJoin(emailJoin), // TODO(elffjs): This seems a bit wasteful.
		qm.LeftOuterJoin(walletJoin),
		qm.OrderBy(models.AccountColumns.CreatedAt + " DESC"),
		qm.Limit(100), // TODO(elffjs): Revisit.
	}

	if in.PartialEmailAddress != "" {
		mods = append(mods, qm.Where(emailHas, strings.ToLower(in.PartialEmailAddress)))
	}
	if addrLen := len(in.PartialWalletAddress); addrLen != 0 {
		if addrLen > common.AddressLength {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Partial wallet address, at %d bytes, is too long.", addrLen))
		}
		mods = append(mods, qm.Where(walletHas, in.PartialWalletAddress))
	}

	accs, err := models.Accounts(mods...).All(ctx, s.DBS.DBS().Reader)
	if err != nil {
		return nil, err
	}

	out := &pb.ListAccountsResponse{
		Accounts: make([]*pb.Account, len(accs)),
	}

	for i, a := range accs {
		out.Accounts[i] = dbToRPC(a)
	}

	return out, nil
}

func (s *Server) TempReferral(ctx context.Context, req *pb.TempReferralRequest) (*pb.TempReferralResponse, error) {
	if len(req.WalletAddress) != common.AddressLength {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Address must have length %d.", common.AddressLength))
	}

	wallet, err := models.Wallets(
		models.WalletWhere.Address.EQ(req.WalletAddress),
		qm.Load(qm.Rels(models.WalletRels.Account, models.AccountRels.ReferredByAccount, models.AccountRels.Wallet)),
	).One(ctx, s.DBS.DBS().Reader)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("No account found with wallet %s.", common.BytesToAddress(req.WalletAddress)))
		}
	}
	out := &pb.TempReferralResponse{
		AccountId:   wallet.R.Account.ID,
		WasReferred: wallet.R.Account.ReferredAt.Valid,
	}

	if wallet.R.Account.R.ReferredByAccount != nil && wallet.R.Account.R.ReferredByAccount.R.Wallet != nil {
		out.ReferrerAccountId = wallet.R.Account.R.ReferredByAccount.ID
		out.ReferrerWalletAddress = wallet.R.Account.R.ReferredByAccount.R.Wallet.Address
	}

	return out, nil
}

func dbToRPC(acc *models.Account) *pb.Account {
	out := &pb.Account{
		Id:        acc.ID,
		CreatedAt: timestamppb.New(acc.CreatedAt),
	}

	if acc.CountryCode.Valid {
		out.CountryCode = acc.CountryCode.String
	}

	if acc.R.Email != nil {
		out.Email = &pb.Email{
			Address: acc.R.Email.Address,
		}
	}
	if acc.R.Wallet != nil {
		out.Wallet = &pb.Wallet{
			Address: acc.R.Wallet.Address,
		}
	}

	return out
}
