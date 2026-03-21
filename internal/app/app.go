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
	BeneficiaryHandler            *handlers.BeneficiaryHandler
	PayoutHandler                 *handlers.PayoutHandler
	RevertTransactionHandler      *handlers.RevertTransactionHandler
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
	payoutStore := store.NewPostgresPayoutStore(pgdb, walletTransactionStore)
	revertTransactionStore := store.NewPostgresRevertTransactionStore(pgdb, walletTransactionStore)

	// Handlers
	adminHandler := handlers.NewAdminHandler(adminStore, walletTransactionStore, logger)
	mdHandler := handlers.NewMasterDistributorHandler(mdStore, logger, awss3)
	distributorHandler := handlers.NewDistributorHandler(distributorStore, logger, awss3)
	retailerHandler := handlers.NewRetailerHandler(retailerStore, logger, awss3)
	walletTransactionHandler := handlers.NewWalletTransactionHandler(walletTransactionStore, logger)
	fundTransferHandler := handlers.NewFundTransferHandler(fundTransferStore, logger)
	fundRequestHandler := handlers.NewFundRequestHandler(fundRequestStore, logger)
	bankHandler := handlers.NewBankHandler(bankStore, logger)
	commisionHandler := handlers.NewCommisionHandler(commisionStore, logger)
	transactionLimitHandler := handlers.NewTransactionLimitHandler(transactionLimitStore, logger)
	ticketHandler := handlers.NewTicketHandler(ticketStore, logger)
	beneficiaryHandler := handlers.NewBeneficiaryHandler(beneficiaryStore, logger)
	payoutHandler := handlers.NewPayoutHandler(payoutStore, logger)
	revertTransactionHandler := handlers.NewRevertTransactionHandler(revertTransactionStore, logger)

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
		BeneficiaryHandler:            beneficiaryHandler,
		PayoutHandler:                 payoutHandler,
		RevertTransactionHandler:      revertTransactionHandler,
	}, nil
}
