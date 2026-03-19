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
	CommissionHandler        *handlers.CommissionHandler
	TransactionLimitHandler  *handlers.TransactionLimitHandler
	TicketHandler            *handlers.TicketHandler
	BeneficiaryHandler       *handlers.BeneficiaryHandler
	PayoutHandler            *handlers.PayoutHandler
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
	fundTransferStore := store.NewPostgresFundTransferStore(pgdb)
	fundRequestStore := store.NewPostgresFundRequestStore(pgdb)
	bankStore := store.NewPostgresBankStore(pgdb)
	commissionStore := store.NewPostgresCommissionStore(pgdb)
	transactionLimitStore := store.NewPostgresTransactionLimitStore(pgdb)
	ticketStore := store.NewPostgresTicketStore(pgdb)
	beneficiaryStore := store.NewPostgresBeneficiaryStore(pgdb)
	payoutStore := store.NewPostgresPayoutStore(pgdb)

	// Handlers
	adminHandler := handlers.NewAdminHandler(adminStore, walletTransactionStore, logger)
	mdHandler := handlers.NewMasterDistributorHandler(mdStore, logger, awss3)
	distributorHandler := handlers.NewDistributorHandler(distributorStore, logger, awss3)
	retailerHandler := handlers.NewRetailerHandler(retailerStore, logger, awss3)
	walletTransactionHandler := handlers.NewWalletTransactionHandler(walletTransactionStore, logger)
	fundTransferHandler := handlers.NewFundTransferHandler(fundTransferStore, logger)
	fundRequestHandler := handlers.NewFundRequestHandler(fundRequestStore, logger)
	bankHandler := handlers.NewBankHandler(bankStore, logger)
	commissionHandler := handlers.NewCommissionHandler(commissionStore, logger)
	transactionLimitHandler := handlers.NewTransactionLimitHandler(transactionLimitStore, logger)
	ticketHandler := handlers.NewTicketHandler(ticketStore, logger)
	beneficiaryHandler := handlers.NewBeneficiaryHandler(beneficiaryStore, logger)
	payoutHandler := handlers.NewPayoutHandler(payoutStore, logger)

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
		CommissionHandler:        commissionHandler,
		TransactionLimitHandler:  transactionLimitHandler,
		TicketHandler:            ticketHandler,
		BeneficiaryHandler:       beneficiaryHandler,
		PayoutHandler:            payoutHandler,
	}, nil
}
