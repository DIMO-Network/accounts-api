package rpc

import (
	"context"
	"fmt"

	"github.com/DIMO-Network/accounts-api/models"
	pb "github.com/DIMO-Network/accounts-api/pkg/grpc"
	"github.com/DIMO-Network/shared/db"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
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
		mods = append(mods, qm.Where(emailHas, in.PartialEmailAddress))
	}
	if len(in.PartialWalletAddress) != 0 {
		mods = append(mods, qm.Where(walletHas, in.PartialEmailAddress))
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
