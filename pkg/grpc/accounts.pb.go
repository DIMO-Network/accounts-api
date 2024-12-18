// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        v5.29.1
// source: pkg/grpc/accounts.proto

package grpc

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Email struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       string                 `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Email) Reset() {
	*x = Email{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Email) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Email) ProtoMessage() {}

func (x *Email) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Email.ProtoReflect.Descriptor instead.
func (*Email) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{0}
}

func (x *Email) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

type Wallet struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       []byte                 `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Wallet) Reset() {
	*x = Wallet{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Wallet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Wallet) ProtoMessage() {}

func (x *Wallet) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Wallet.ProtoReflect.Descriptor instead.
func (*Wallet) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{1}
}

func (x *Wallet) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

type Account struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	CountryCode   string                 `protobuf:"bytes,2,opt,name=country_code,json=countryCode,proto3" json:"country_code,omitempty"`
	CreatedAt     *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	Email         *Email                 `protobuf:"bytes,4,opt,name=email,proto3" json:"email,omitempty"`
	Wallet        *Wallet                `protobuf:"bytes,5,opt,name=wallet,proto3" json:"wallet,omitempty"`
	Referral      *Referral              `protobuf:"bytes,6,opt,name=referral,proto3" json:"referral,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Account) Reset() {
	*x = Account{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Account) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Account) ProtoMessage() {}

func (x *Account) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Account.ProtoReflect.Descriptor instead.
func (*Account) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{2}
}

func (x *Account) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Account) GetCountryCode() string {
	if x != nil {
		return x.CountryCode
	}
	return ""
}

func (x *Account) GetCreatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *Account) GetEmail() *Email {
	if x != nil {
		return x.Email
	}
	return nil
}

func (x *Account) GetWallet() *Wallet {
	if x != nil {
		return x.Wallet
	}
	return nil
}

func (x *Account) GetReferral() *Referral {
	if x != nil {
		return x.Referral
	}
	return nil
}

type Referral struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          string                 `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	ReferredBy    string                 `protobuf:"bytes,2,opt,name=referred_by,json=referredBy,proto3" json:"referred_by,omitempty"`
	ReferredAt    *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=referred_at,json=referredAt,proto3" json:"referred_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Referral) Reset() {
	*x = Referral{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Referral) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Referral) ProtoMessage() {}

func (x *Referral) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Referral.ProtoReflect.Descriptor instead.
func (*Referral) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{3}
}

func (x *Referral) GetCode() string {
	if x != nil {
		return x.Code
	}
	return ""
}

func (x *Referral) GetReferredBy() string {
	if x != nil {
		return x.ReferredBy
	}
	return ""
}

func (x *Referral) GetReferredAt() *timestamppb.Timestamp {
	if x != nil {
		return x.ReferredAt
	}
	return nil
}

type ListAccountsRequest struct {
	state                protoimpl.MessageState `protogen:"open.v1"`
	PartialEmailAddress  string                 `protobuf:"bytes,1,opt,name=partial_email_address,json=partialEmailAddress,proto3" json:"partial_email_address,omitempty"`
	PartialWalletAddress []byte                 `protobuf:"bytes,2,opt,name=partial_wallet_address,json=partialWalletAddress,proto3" json:"partial_wallet_address,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *ListAccountsRequest) Reset() {
	*x = ListAccountsRequest{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListAccountsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListAccountsRequest) ProtoMessage() {}

func (x *ListAccountsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListAccountsRequest.ProtoReflect.Descriptor instead.
func (*ListAccountsRequest) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{4}
}

func (x *ListAccountsRequest) GetPartialEmailAddress() string {
	if x != nil {
		return x.PartialEmailAddress
	}
	return ""
}

func (x *ListAccountsRequest) GetPartialWalletAddress() []byte {
	if x != nil {
		return x.PartialWalletAddress
	}
	return nil
}

type GetAccountRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	EmailAddress  string                 `protobuf:"bytes,2,opt,name=email_address,json=emailAddress,proto3" json:"email_address,omitempty"`
	WalletAddress []byte                 `protobuf:"bytes,3,opt,name=wallet_address,json=walletAddress,proto3" json:"wallet_address,omitempty"`
	ReferralCode  string                 `protobuf:"bytes,4,opt,name=referral_code,json=referralCode,proto3" json:"referral_code,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetAccountRequest) Reset() {
	*x = GetAccountRequest{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetAccountRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAccountRequest) ProtoMessage() {}

func (x *GetAccountRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAccountRequest.ProtoReflect.Descriptor instead.
func (*GetAccountRequest) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{5}
}

func (x *GetAccountRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *GetAccountRequest) GetEmailAddress() string {
	if x != nil {
		return x.EmailAddress
	}
	return ""
}

func (x *GetAccountRequest) GetWalletAddress() []byte {
	if x != nil {
		return x.WalletAddress
	}
	return nil
}

func (x *GetAccountRequest) GetReferralCode() string {
	if x != nil {
		return x.ReferralCode
	}
	return ""
}

type ListAccountsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Accounts      []*Account             `protobuf:"bytes,1,rep,name=accounts,proto3" json:"accounts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListAccountsResponse) Reset() {
	*x = ListAccountsResponse{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListAccountsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListAccountsResponse) ProtoMessage() {}

func (x *ListAccountsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListAccountsResponse.ProtoReflect.Descriptor instead.
func (*ListAccountsResponse) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{6}
}

func (x *ListAccountsResponse) GetAccounts() []*Account {
	if x != nil {
		return x.Accounts
	}
	return nil
}

type TempReferralRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	WalletAddress []byte                 `protobuf:"bytes,1,opt,name=wallet_address,json=walletAddress,proto3" json:"wallet_address,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TempReferralRequest) Reset() {
	*x = TempReferralRequest{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TempReferralRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TempReferralRequest) ProtoMessage() {}

func (x *TempReferralRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TempReferralRequest.ProtoReflect.Descriptor instead.
func (*TempReferralRequest) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{7}
}

func (x *TempReferralRequest) GetWalletAddress() []byte {
	if x != nil {
		return x.WalletAddress
	}
	return nil
}

type TempReferralResponse struct {
	state                 protoimpl.MessageState `protogen:"open.v1"`
	AccountId             string                 `protobuf:"bytes,1,opt,name=account_id,json=accountId,proto3" json:"account_id,omitempty"`
	WasReferred           bool                   `protobuf:"varint,2,opt,name=was_referred,json=wasReferred,proto3" json:"was_referred,omitempty"`
	ReferrerAccountId     string                 `protobuf:"bytes,3,opt,name=referrer_account_id,json=referrerAccountId,proto3" json:"referrer_account_id,omitempty"`
	ReferrerWalletAddress []byte                 `protobuf:"bytes,4,opt,name=referrer_wallet_address,json=referrerWalletAddress,proto3" json:"referrer_wallet_address,omitempty"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *TempReferralResponse) Reset() {
	*x = TempReferralResponse{}
	mi := &file_pkg_grpc_accounts_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TempReferralResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TempReferralResponse) ProtoMessage() {}

func (x *TempReferralResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_grpc_accounts_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TempReferralResponse.ProtoReflect.Descriptor instead.
func (*TempReferralResponse) Descriptor() ([]byte, []int) {
	return file_pkg_grpc_accounts_proto_rawDescGZIP(), []int{8}
}

func (x *TempReferralResponse) GetAccountId() string {
	if x != nil {
		return x.AccountId
	}
	return ""
}

func (x *TempReferralResponse) GetWasReferred() bool {
	if x != nil {
		return x.WasReferred
	}
	return false
}

func (x *TempReferralResponse) GetReferrerAccountId() string {
	if x != nil {
		return x.ReferrerAccountId
	}
	return ""
}

func (x *TempReferralResponse) GetReferrerWalletAddress() []byte {
	if x != nil {
		return x.ReferrerWalletAddress
	}
	return nil
}

var File_pkg_grpc_accounts_proto protoreflect.FileDescriptor

var file_pkg_grpc_accounts_proto_rawDesc = []byte{
	0x0a, 0x17, 0x70, 0x6b, 0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x21, 0x0a, 0x05, 0x45, 0x6d,
	0x61, 0x69, 0x6c, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x22, 0x0a,
	0x06, 0x57, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x22, 0xdd, 0x01, 0x0a, 0x07, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x21, 0x0a,
	0x0c, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x72, 0x79, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x72, 0x79, 0x43, 0x6f, 0x64, 0x65,
	0x12, 0x39, 0x0a, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x1c, 0x0a, 0x05, 0x65,
	0x6d, 0x61, 0x69, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x06, 0x2e, 0x45, 0x6d, 0x61,
	0x69, 0x6c, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x1f, 0x0a, 0x06, 0x77, 0x61, 0x6c,
	0x6c, 0x65, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x07, 0x2e, 0x57, 0x61, 0x6c, 0x6c,
	0x65, 0x74, 0x52, 0x06, 0x77, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x12, 0x25, 0x0a, 0x08, 0x72, 0x65,
	0x66, 0x65, 0x72, 0x72, 0x61, 0x6c, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x52,
	0x65, 0x66, 0x65, 0x72, 0x72, 0x61, 0x6c, 0x52, 0x08, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x61,
	0x6c, 0x22, 0x7c, 0x0a, 0x08, 0x52, 0x65, 0x66, 0x65, 0x72, 0x72, 0x61, 0x6c, 0x12, 0x12, 0x0a,
	0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x63, 0x6f, 0x64,
	0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x79,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64,
	0x42, 0x79, 0x12, 0x3b, 0x0a, 0x0b, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x61,
	0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x52, 0x0a, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x41, 0x74, 0x22,
	0x7f, 0x0a, 0x13, 0x4c, 0x69, 0x73, 0x74, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x32, 0x0a, 0x15, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61,
	0x6c, 0x5f, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x45, 0x6d,
	0x61, 0x69, 0x6c, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x34, 0x0a, 0x16, 0x70, 0x61,
	0x72, 0x74, 0x69, 0x61, 0x6c, 0x5f, 0x77, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x5f, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x14, 0x70, 0x61, 0x72, 0x74,
	0x69, 0x61, 0x6c, 0x57, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x22, 0x94, 0x01, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x23, 0x0a, 0x0d, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x5f,
	0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x65,
	0x6d, 0x61, 0x69, 0x6c, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x77,
	0x61, 0x6c, 0x6c, 0x65, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x0d, 0x77, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x61, 0x6c, 0x5f, 0x63,
	0x6f, 0x64, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72, 0x65, 0x66, 0x65, 0x72,
	0x72, 0x61, 0x6c, 0x43, 0x6f, 0x64, 0x65, 0x22, 0x3c, 0x0a, 0x14, 0x4c, 0x69, 0x73, 0x74, 0x41,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x24, 0x0a, 0x08, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x08, 0x2e, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x08, 0x61, 0x63, 0x63,
	0x6f, 0x75, 0x6e, 0x74, 0x73, 0x22, 0x3c, 0x0a, 0x13, 0x54, 0x65, 0x6d, 0x70, 0x52, 0x65, 0x66,
	0x65, 0x72, 0x72, 0x61, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x25, 0x0a, 0x0e,
	0x77, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x77, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x41, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x22, 0xc0, 0x01, 0x0a, 0x14, 0x54, 0x65, 0x6d, 0x70, 0x52, 0x65, 0x66, 0x65,
	0x72, 0x72, 0x61, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1d, 0x0a, 0x0a,
	0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x77,
	0x61, 0x73, 0x5f, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x0b, 0x77, 0x61, 0x73, 0x52, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x12, 0x2e,
	0x0a, 0x13, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x72, 0x5f, 0x61, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x72, 0x65, 0x66,
	0x65, 0x72, 0x72, 0x65, 0x72, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x36,
	0x0a, 0x17, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x72, 0x5f, 0x77, 0x61, 0x6c, 0x6c, 0x65,
	0x74, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x15, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x72, 0x57, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x41,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x32, 0xb0, 0x01, 0x0a, 0x08, 0x41, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x73, 0x12, 0x3b, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x41, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x73, 0x12, 0x14, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e, 0x4c, 0x69, 0x73, 0x74,
	0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x2a, 0x0a, 0x0a, 0x47, 0x65, 0x74, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x12,
	0x2e, 0x47, 0x65, 0x74, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x08, 0x2e, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x3b, 0x0a, 0x0c,
	0x54, 0x65, 0x6d, 0x70, 0x52, 0x65, 0x66, 0x65, 0x72, 0x72, 0x61, 0x6c, 0x12, 0x14, 0x2e, 0x54,
	0x65, 0x6d, 0x70, 0x52, 0x65, 0x66, 0x65, 0x72, 0x72, 0x61, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x15, 0x2e, 0x54, 0x65, 0x6d, 0x70, 0x52, 0x65, 0x66, 0x65, 0x72, 0x72, 0x61,
	0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x2f, 0x5a, 0x2d, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x44, 0x49, 0x4d, 0x4f, 0x2d, 0x4e, 0x65, 0x74,
	0x77, 0x6f, 0x72, 0x6b, 0x2f, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x2d, 0x61, 0x70,
	0x69, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_pkg_grpc_accounts_proto_rawDescOnce sync.Once
	file_pkg_grpc_accounts_proto_rawDescData = file_pkg_grpc_accounts_proto_rawDesc
)

func file_pkg_grpc_accounts_proto_rawDescGZIP() []byte {
	file_pkg_grpc_accounts_proto_rawDescOnce.Do(func() {
		file_pkg_grpc_accounts_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_grpc_accounts_proto_rawDescData)
	})
	return file_pkg_grpc_accounts_proto_rawDescData
}

var file_pkg_grpc_accounts_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_pkg_grpc_accounts_proto_goTypes = []any{
	(*Email)(nil),                 // 0: Email
	(*Wallet)(nil),                // 1: Wallet
	(*Account)(nil),               // 2: Account
	(*Referral)(nil),              // 3: Referral
	(*ListAccountsRequest)(nil),   // 4: ListAccountsRequest
	(*GetAccountRequest)(nil),     // 5: GetAccountRequest
	(*ListAccountsResponse)(nil),  // 6: ListAccountsResponse
	(*TempReferralRequest)(nil),   // 7: TempReferralRequest
	(*TempReferralResponse)(nil),  // 8: TempReferralResponse
	(*timestamppb.Timestamp)(nil), // 9: google.protobuf.Timestamp
}
var file_pkg_grpc_accounts_proto_depIdxs = []int32{
	9, // 0: Account.created_at:type_name -> google.protobuf.Timestamp
	0, // 1: Account.email:type_name -> Email
	1, // 2: Account.wallet:type_name -> Wallet
	3, // 3: Account.referral:type_name -> Referral
	9, // 4: Referral.referred_at:type_name -> google.protobuf.Timestamp
	2, // 5: ListAccountsResponse.accounts:type_name -> Account
	4, // 6: Accounts.ListAccounts:input_type -> ListAccountsRequest
	5, // 7: Accounts.GetAccount:input_type -> GetAccountRequest
	7, // 8: Accounts.TempReferral:input_type -> TempReferralRequest
	6, // 9: Accounts.ListAccounts:output_type -> ListAccountsResponse
	2, // 10: Accounts.GetAccount:output_type -> Account
	8, // 11: Accounts.TempReferral:output_type -> TempReferralResponse
	9, // [9:12] is the sub-list for method output_type
	6, // [6:9] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_pkg_grpc_accounts_proto_init() }
func file_pkg_grpc_accounts_proto_init() {
	if File_pkg_grpc_accounts_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pkg_grpc_accounts_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pkg_grpc_accounts_proto_goTypes,
		DependencyIndexes: file_pkg_grpc_accounts_proto_depIdxs,
		MessageInfos:      file_pkg_grpc_accounts_proto_msgTypes,
	}.Build()
	File_pkg_grpc_accounts_proto = out.File
	file_pkg_grpc_accounts_proto_rawDesc = nil
	file_pkg_grpc_accounts_proto_goTypes = nil
	file_pkg_grpc_accounts_proto_depIdxs = nil
}
