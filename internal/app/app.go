package app

import (
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/levionstudio/fintech/internal/handlers"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type Application struct {
	Logger                   *slog.Logger
	DB                       *sql.DB
	AdminHandler             *handlers.AdminHandler
	MasterDistributorHandler *handlers.MasterDistributorHandler
	DistributorHandler       *handlers.DistributorHandler
	RetailerHandler          *handlers.RetailerHandler
	WalletTransactionHandler *handlers.WalletTransactionHandler
	FundTransferHandler      *handlers.FundTransferHandler
	FundRequestHandler       *handlers.FundRequestHandler
	BankHandler              *handlers.BankHandler
	CommisionHandler         *handlers.CommisionHandler
	TransactionLimitHandler  *handlers.TransactionLimitHandler
	TicketHandler            *handlers.TicketHandler
	BeneficiaryHandler       *handlers.BeneficiaryHandler
	RevertTransactionHandler *handlers.RevertTransactionHandler
	PayoutHandler            *handlers.PayoutHandler
	MobileRechargeHandler    *handlers.MobileRechargeHandler
	DTHRechargeHandler       *handlers.DTHRechargeHandler
	ElectricityBillHandler   *handlers.ElectricityBillHandler
}

func NewApplication() (*Application, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	pgdb, err := store.Open()
	if err != nil {
		return nil, err
	}

	awss3, err := utils.Connect()
	if err != nil {
		return nil, err
	}

	// Stores
	adminStore := store.NewPostgresAdminStore(pgdb)
	mdStore := store.NewPostgresMasterDistributorStore(pgdb)
	distributorStore := store.NewPostgresDistributorStore(pgdb)
	retailerStore := store.NewPostgresRetailerStore(pgdb)
	walletTransactionStore := store.NewPostgresWalletTransactionStore(pgdb)
	fundTransferStore := store.NewPostgresFundTransferStore(pgdb, walletTransactionStore)
	fundRequestStore := store.NewPostgresFundRequestStore(pgdb, walletTransactionStore)
	bankStore := store.NewPostgresBankStore(pgdb)
	commisionStore := store.NewPostgresCommisionStore(pgdb)
	transactionLimitStore := store.NewPostgresTransactionLimitStore(pgdb)
	ticketStore := store.NewPostgresTicketStore(pgdb)
	beneficiaryStore := store.NewPostgresBeneficiaryStore(pgdb)
	revertTransactionStore := store.NewPostgresRevertTransactionStore(pgdb, walletTransactionStore)
	payoutTransactionStore := store.NewPostgresPayoutTransactionStore(pgdb, commisionStore, walletTransactionStore, transactionLimitStore)
	mobileRechargeStore := store.NewPostgresMobileRechargeStore(pgdb, walletTransactionStore)
	dthRechargeStore := store.NewPostgresDTHRechargeStore(pgdb, walletTransactionStore)
	electricityBillStore := store.NewPostgresElectricityBillStore(pgdb, walletTransactionStore)
	loginActivityStore := store.NewPostgresLoginActivityStore(pgdb)

	// Handlers
	adminHandler := handlers.NewAdminHandler(adminStore, walletTransactionStore, loginActivityStore, logger)
	mdHandler := handlers.NewMasterDistributorHandler(mdStore, loginActivityStore, logger, awss3)
	distributorHandler := handlers.NewDistributorHandler(distributorStore, loginActivityStore, logger, awss3)
	retailerHandler := handlers.NewRetailerHandler(retailerStore, loginActivityStore, logger, awss3)
	walletTransactionHandler := handlers.NewWalletTransactionHandler(walletTransactionStore, logger)
	fundTransferHandler := handlers.NewFundTransferHandler(fundTransferStore, logger)
	fundRequestHandler := handlers.NewFundRequestHandler(fundRequestStore, logger)
	bankHandler := handlers.NewBankHandler(bankStore, logger)
	commisionHandler := handlers.NewCommisionHandler(commisionStore, logger)
	transactionLimitHandler := handlers.NewTransactionLimitHandler(transactionLimitStore, logger)
	ticketHandler := handlers.NewTicketHandler(ticketStore, logger)
	beneficiaryHandler := handlers.NewBeneficiaryHandler(beneficiaryStore, logger)
	revertTransactionHandler := handlers.NewRevertTransactionHandler(revertTransactionStore, logger)
	payoutHandler := handlers.NewPayoutHandler(payoutTransactionStore, logger)
	mobileRechargeHandler := handlers.NewMobileRechargeHandler(mobileRechargeStore, logger)
	dthRechargeHandler := handlers.NewDTHRechargeHandler(dthRechargeStore, logger)
	electricityBillHandler := handlers.NewElectricityBillHandler(electricityBillStore, logger)

	return &Application{
		Logger:                   logger,
		DB:                       pgdb,
		AdminHandler:             adminHandler,
		MasterDistributorHandler: mdHandler,
		DistributorHandler:       distributorHandler,
		RetailerHandler:          retailerHandler,
		WalletTransactionHandler: walletTransactionHandler,
		FundTransferHandler:      fundTransferHandler,
		FundRequestHandler:       fundRequestHandler,
		BankHandler:              bankHandler,
		CommisionHandler:         commisionHandler,
		TransactionLimitHandler:  transactionLimitHandler,
		TicketHandler:            ticketHandler,
		BeneficiaryHandler:       beneficiaryHandler,
		RevertTransactionHandler: revertTransactionHandler,
		PayoutHandler:            payoutHandler,
		MobileRechargeHandler:    mobileRechargeHandler,
		DTHRechargeHandler:       dthRechargeHandler,
		ElectricityBillHandler:   electricityBillHandler,
	}, nil
}
