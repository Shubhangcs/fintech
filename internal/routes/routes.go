package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/levionstudio/fintech/internal/app"
	"github.com/levionstudio/fintech/internal/middlewares"
)

func SetupRoutes(app *app.Application) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	router.Use(middlewares.RecoveryMiddleware(app.Logger))
	router.Use(middlewares.CORSMiddleware)

	adminRoutes(router, app)
	masterDistributorRoutes(router, app)
	distributorRoutes(router, app)
	retailerRoutes(router, app)
	walletTransactionRoutes(router, app)
	fundTransferRoutes(router, app)
	fundRequestRoutes(router, app)
	bankRoutes(router, app)
	commisionRoutes(router, app)
	transactionLimitRoutes(router, app)
	ticketRoutes(router, app)
	beneficiaryRoutes(router, app)
	revertTransactionRoutes(router, app)
	payoutRoutes(router, app)
	mobileRechargeRoutes(router, app)
	dthRechargeRoutes(router, app)
	electricityBillRoutes(router, app)
	loginActivityRoutes(router, app)

	return router
}

func adminRoutes(router *chi.Mux, app *app.Application) {
	// Public routes
	router.Post("/admin/login", app.AdminHandler.HandleAdminLogin)
	router.Post("/admin/create", app.AdminHandler.HandleCreateAdmin)

	// Protected routes
	router.Route("/admin", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)
		r.Get("/get", app.AdminHandler.HandleGetAdmins)
		r.Get("/dropdown", app.AdminHandler.HandleGetAdminsForDropdown)
		r.Get("/get/{id}", app.AdminHandler.HandleGetAdminByID)
		r.Get("/get/{id}/wallet", app.AdminHandler.HandleGetAdminWalletBalance)
		r.Get("/recharge-kit/recharge-balance", app.AdminHandler.HandleGetRechargeKitRechargeBalance)
		r.Get("/recharge-kit/primary-balance", app.AdminHandler.HandleGetRechargeKitPrimaryBalance)
		r.Put("/update/{id}", app.AdminHandler.HandleUpdateAdminDetails)
		r.Patch("/update/{id}/password", app.AdminHandler.HandleUpdateAdminPassword)
		r.Patch("/update/{id}/wallet", app.AdminHandler.HandleUpdateAdminWalletBalance)
		r.Delete("/delete/{id}", app.AdminHandler.HandleDeleteAdmin)
	})
}

func masterDistributorRoutes(router *chi.Mux, app *app.Application) {
	// Public routes
	router.Post("/master-distributor/login", app.MasterDistributorHandler.HandleMasterDistributorLogin)

	router.Route("/master-distributor", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.MasterDistributorHandler.HandleCreateMasterDistributor)
		r.Get("/get/{id}", app.MasterDistributorHandler.HandleGetMasterDistributorByID)
		r.Get("/admin/{id}", app.MasterDistributorHandler.HandleGetMasterDistributorsByAdminID)
		r.Get("/dropdown/{id}", app.MasterDistributorHandler.HandleGetMasterDistributorsByAdminIDForDropdown)
		r.Get("/get/{id}/wallet", app.MasterDistributorHandler.HandleGetMasterDistributorWalletBalance)
		r.Put("/update/{id}", app.MasterDistributorHandler.HandleUpdateMasterDistributorDetails)
		r.Patch("/update/{id}/password", app.MasterDistributorHandler.HandleUpdateMasterDistributorPassword)
		r.Patch("/update/{id}/mpin", app.MasterDistributorHandler.HandleUpdateMasterDistributorMpin)
		r.Patch("/update/{id}/kyc", app.MasterDistributorHandler.HandleUpdateMasterDistributorKYCStatus)
		r.Patch("/update/{id}/block", app.MasterDistributorHandler.HandleUpdateMasterDistributorBlockStatus)
		r.Patch("/update/{id}/aadhar", app.MasterDistributorHandler.HandleUpdateMasterDistributorAadharImage)
		r.Patch("/update/{id}/pan", app.MasterDistributorHandler.HandleUpdateMasterDistributorPanImage)
		r.Patch("/update/{id}/image", app.MasterDistributorHandler.HandleUpdateMasterDistributorImage)
		r.Delete("/delete/{id}", app.MasterDistributorHandler.HandleDeleteMasterDistributor)
	})
}

func retailerRoutes(router *chi.Mux, app *app.Application) {
	// Public routes
	router.Post("/retailer/login", app.RetailerHandler.HandleRetailerLogin)

	router.Route("/retailer", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.RetailerHandler.HandleCreateRetailer)
		r.Get("/get/{id}", app.RetailerHandler.HandleGetRetailerByID)
		r.Get("/distributor/{id}", app.RetailerHandler.HandleGetRetailersByDistributorID)
		r.Get("/md/{id}", app.RetailerHandler.HandleGetRetailersByMasterDistributorID)
		r.Get("/admin/{id}", app.RetailerHandler.HandleGetRetailersByAdminID)
		r.Get("/dropdown/distributor/{id}", app.RetailerHandler.HandleGetRetailersByDistributorIDForDropdown)
		r.Get("/dropdown/md/{id}", app.RetailerHandler.HandleGetRetailersByMasterDistributorIDForDropdown)
		r.Get("/dropdown/admin/{id}", app.RetailerHandler.HandleGetRetailersByAdminIDForDropdown)
		r.Get("/get/{id}/wallet", app.RetailerHandler.HandleGetRetailerWalletBalance)
		r.Put("/update/{id}", app.RetailerHandler.HandleUpdateRetailerDetails)
		r.Patch("/update/{id}/password", app.RetailerHandler.HandleUpdateRetailerPassword)
		r.Patch("/update/{id}/mpin", app.RetailerHandler.HandleUpdateRetailerMpin)
		r.Patch("/update/{id}/kyc", app.RetailerHandler.HandleUpdateRetailerKYCStatus)
		r.Patch("/update/{id}/block", app.RetailerHandler.HandleUpdateRetailerBlockStatus)
		r.Patch("/change/{id}/distributor", app.RetailerHandler.HandleChangeRetailersDistributor)
		r.Patch("/update/{id}/aadhar", app.RetailerHandler.HandleUpdateRetailerAadharImage)
		r.Patch("/update/{id}/pan", app.RetailerHandler.HandleUpdateRetailerPanImage)
		r.Patch("/update/{id}/image", app.RetailerHandler.HandleUpdateRetailerImage)
		r.Delete("/delete/{id}", app.RetailerHandler.HandleDeleteRetailer)
	})
}

func distributorRoutes(router *chi.Mux, app *app.Application) {
	// Public routes
	router.Post("/distributor/login", app.DistributorHandler.HandleDistributorLogin)

	router.Route("/distributor", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.DistributorHandler.HandleCreateDistributor)
		r.Get("/get/{id}", app.DistributorHandler.HandleGetDistributorByID)
		r.Get("/md/{id}", app.DistributorHandler.HandleGetDistributorsByMasterDistributorID)
		r.Get("/admin/{id}", app.DistributorHandler.HandleGetDistributorsByAdminID)
		r.Get("/dropdown/md/{id}", app.DistributorHandler.HandleGetDistributorsByMasterDistributorIDForDropdown)
		r.Get("/dropdown/admin/{id}", app.DistributorHandler.HandleGetDistributorsByAdminIDForDropdown)
		r.Get("/get/{id}/wallet", app.DistributorHandler.HandleGetDistributorWalletBalance)
		r.Put("/update/{id}", app.DistributorHandler.HandleUpdateDistributorDetails)
		r.Patch("/update/{id}/password", app.DistributorHandler.HandleUpdateDistributorPassword)
		r.Patch("/update/{id}/mpin", app.DistributorHandler.HandleUpdateDistributorMpin)
		r.Patch("/update/{id}/kyc", app.DistributorHandler.HandleUpdateDistributorKYCStatus)
		r.Patch("/update/{id}/block", app.DistributorHandler.HandleUpdateDistributorBlockStatus)
		r.Patch("/change/{id}/md", app.DistributorHandler.HandleChangeDistributorsMasterDistributor)
		r.Patch("/update/{id}/aadhar", app.DistributorHandler.HandleUpdateDistributorAadharImage)
		r.Patch("/update/{id}/pan", app.DistributorHandler.HandleUpdateDistributorPanImage)
		r.Patch("/update/{id}/image", app.DistributorHandler.HandleUpdateDistributorImage)
		r.Delete("/delete/{id}", app.DistributorHandler.HandleDeleteDistributor)
	})
}

func walletTransactionRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/wallet-transaction", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.WalletTransactionHandler.HandleCreateWalletTransaction)
		r.Get("/user/{id}", app.WalletTransactionHandler.HandleGetWalletTransactionsByUserID)
	})
}

func fundTransferRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/fund-transfer", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/admin-to-md", app.FundTransferHandler.HandleAdminToMD)
		r.Post("/admin-to-distributor", app.FundTransferHandler.HandleAdminToDistributor)
		r.Post("/admin-to-retailer", app.FundTransferHandler.HandleAdminToRetailer)
		r.Post("/md-to-distributor", app.FundTransferHandler.HandleMDToDistributor)
		r.Post("/md-to-retailer", app.FundTransferHandler.HandleMDToRetailer)
		r.Post("/distributor-to-retailer", app.FundTransferHandler.HandleDistributorToRetailer)
		r.Get("/transferer/{id}", app.FundTransferHandler.HandleGetFundTransfersByTransfererID)
		r.Get("/receiver/{id}", app.FundTransferHandler.HandleGetFundTransfersByReceiverID)
		r.Get("/all", app.FundTransferHandler.HandleGetAllFundTransfers)
	})
}

func beneficiaryRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/beneficiary", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.BeneficiaryHandler.HandleCreateBeneficiary)
		r.Put("/update/{id}", app.BeneficiaryHandler.HandleUpdateBeneficiary)
		r.Delete("/delete/{id}", app.BeneficiaryHandler.HandleDeleteBeneficiary)
		r.Get("/mobile/{mobile}", app.BeneficiaryHandler.HandleGetBeneficiaries)
		r.Post("/verify", app.BeneficiaryHandler.HandleVerifyBeneficiary)
	})
}

func ticketRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/ticket", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.TicketHandler.HandleCreateTicket)
		r.Put("/update/{id}", app.TicketHandler.HandleUpdateTicket)
		r.Delete("/delete/{id}", app.TicketHandler.HandleDeleteTicket)
		r.Patch("/update/{id}/clear", app.TicketHandler.HandleUpdateTicketClearStatus)
		r.Get("/all", app.TicketHandler.HandleGetAllTickets)
		r.Get("/user/{id}", app.TicketHandler.HandleGetTicketsByUserID)
	})
}

func transactionLimitRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/transaction-limit", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.TransactionLimitHandler.HandleCreateTransactionLimit)
		r.Put("/update/{id}", app.TransactionLimitHandler.HandleUpdateTransactionLimit)
		r.Delete("/delete/{id}", app.TransactionLimitHandler.HandleDeleteTransactionLimit)
		r.Get("/all", app.TransactionLimitHandler.HandleGetAllTransactionLimits)
		r.Post("/get/limit/service", app.TransactionLimitHandler.HandleGetTransactionLimitByRetailerIDAndService)
	})
}

func commisionRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/commision", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.CommisionHandler.HandleCreateCommision)
		r.Put("/update/{id}", app.CommisionHandler.HandleUpdateCommision)
		r.Delete("/delete/{id}", app.CommisionHandler.HandleDeleteCommision)
		r.Get("/all", app.CommisionHandler.HandleGetCommisions)
	})
}

func bankRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/bank", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.BankHandler.HandleCreateBank)
		r.Put("/update/{id}", app.BankHandler.HandleUpdateBank)
		r.Delete("/delete/{id}", app.BankHandler.HandleDeleteBank)
		r.Get("/all", app.BankHandler.HandleGetAllBanks)
	})

	router.Route("/admin-bank", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.BankHandler.HandleCreateAdminBank)
		r.Put("/update/{id}", app.BankHandler.HandleUpdateAdminBank)
		r.Delete("/delete/{id}", app.BankHandler.HandleDeleteAdminBank)
		r.Get("/all", app.BankHandler.HandleGetAllAdminBanks)
	})
}

func revertTransactionRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/revert-transaction", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/admin-to-md", app.RevertTransactionHandler.HandleAdminRevertOnMD)
		r.Post("/admin-to-distributor", app.RevertTransactionHandler.HandleAdminRevertOnDistributor)
		r.Post("/admin-to-retailer", app.RevertTransactionHandler.HandleAdminRevertOnRetailer)
		r.Post("/md-to-distributor", app.RevertTransactionHandler.HandleMDRevertOnDistributor)
		r.Post("/md-to-retailer", app.RevertTransactionHandler.HandleMDRevertOnRetailer)
		r.Post("/distributor-to-retailer", app.RevertTransactionHandler.HandleDistributorRevertOnRetailer)
		r.Get("/revert-by/{id}", app.RevertTransactionHandler.HandleGetRevertTransactionsByRevertByID)
		r.Get("/revert-on/{id}", app.RevertTransactionHandler.HandleGetRevertTransactionsByRevertOnID)
		r.Get("/all", app.RevertTransactionHandler.HandleGetAllRevertTransactions)
	})
}

func fundRequestRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/fund-request", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/upload-recipt/{id}", app.FundRequestHandler.HandleUploadFundRequestRecipt)
		r.Post("/md-to-admin", app.FundRequestHandler.HandleMDRequestToAdmin)
		r.Post("/distributor-to-admin", app.FundRequestHandler.HandleDistributorRequestToAdmin)
		r.Post("/distributor-to-md", app.FundRequestHandler.HandleDistributorRequestToMD)
		r.Post("/retailer-to-admin", app.FundRequestHandler.HandleRetailerRequestToAdmin)
		r.Post("/retailer-to-md", app.FundRequestHandler.HandleRetailerRequestToMD)
		r.Post("/retailer-to-distributor", app.FundRequestHandler.HandleRetailerRequestToDistributor)
		r.Patch("/approve/{id}", app.FundRequestHandler.HandleApproveFundRequest)
		r.Patch("/reject/{id}", app.FundRequestHandler.HandleRejectFundRequest)
		r.Get("/requester/{id}", app.FundRequestHandler.HandleGetFundRequestsByRequesterID)
		r.Get("/request-to/{id}", app.FundRequestHandler.HandleGetFundRequestsByRequestToID)
		r.Get("/all", app.FundRequestHandler.HandleGetAllFundRequests)
	})
}

func payoutRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/payout", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.PayoutHandler.HandleCreatePayoutTransaction)
		r.Post("/status-check/{id}", app.PayoutHandler.HandleCheckPayoutStatus)
		r.Post("/refund/{id}", app.PayoutHandler.HandleRefundPayout)
		r.Put("/update/{id}", app.PayoutHandler.HandleUpdatePayoutTransaction)
		r.Get("/all", app.PayoutHandler.HandleGetAllPayoutTransactions)
		r.Get("/retailer/{id}", app.PayoutHandler.HandleGetPayoutTransactionsByRetailerID)
		r.Get("/distributor/{id}", app.PayoutHandler.HandleGetPayoutTransactionsByDistributorID)
		r.Get("/md/{id}", app.PayoutHandler.HandleGetPayoutTransactionsByMasterDistributorID)
	})
}

func dthRechargeRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/dth-recharge", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.DTHRechargeHandler.HandleCreateDTHRecharge)
		r.Post("/status-check/{id}", app.DTHRechargeHandler.HandleCheckDTHRechargeStatus)
		r.Post("/refund/{id}", app.DTHRechargeHandler.HandleRefundDTHRecharge)
		r.Get("/all", app.DTHRechargeHandler.HandleGetAllDTHRecharge)
		r.Get("/retailer/{id}", app.DTHRechargeHandler.HandleGetDTHRechargeByRetailerID)
		r.Get("/distributor/{id}", app.DTHRechargeHandler.HandleGetDTHRechargeByDistributorID)
		r.Get("/md/{id}", app.DTHRechargeHandler.HandleGetDTHRechargeByMasterDistributorID)

		// Operator management
		r.Post("/operators", app.DTHRechargeHandler.HandleCreateDTHRechargeOperator)
		r.Put("/operators/{id}", app.DTHRechargeHandler.HandleUpdateDTHRechargeOperator)
		r.Delete("/operators/{id}", app.DTHRechargeHandler.HandleDeleteDTHRechargeOperator)
		r.Get("/operators", app.DTHRechargeHandler.HandleGetDTHRechargeOperators)
	})
}

func electricityBillRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/electricity-bill", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.ElectricityBillHandler.HandleCreateElectricityBill)
		r.Post("/status-check/{id}", app.ElectricityBillHandler.HandleCheckElectricityBillStatus)
		r.Post("/refund/{id}", app.ElectricityBillHandler.HandleRefundElectricityBill)
		r.Post("/bill-fetch", app.ElectricityBillHandler.HandleFetchElectricityBill)
		r.Get("/all", app.ElectricityBillHandler.HandleGetAllElectricityBills)
		r.Get("/retailer/{id}", app.ElectricityBillHandler.HandleGetElectricityBillsByRetailerID)
		r.Get("/distributor/{id}", app.ElectricityBillHandler.HandleGetElectricityBillsByDistributorID)
		r.Get("/md/{id}", app.ElectricityBillHandler.HandleGetElectricityBillsByMasterDistributorID)

		r.Post("/operators", app.ElectricityBillHandler.HandleCreateElectricityOperator)
		r.Put("/operators/{id}", app.ElectricityBillHandler.HandleUpdateElectricityOperator)
		r.Delete("/operators/{id}", app.ElectricityBillHandler.HandleDeleteElectricityOperator)
		r.Get("/operators", app.ElectricityBillHandler.HandleGetElectricityOperators)
	})
}

func loginActivityRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/login-activity", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Get("/all", app.LoginActivityHandler.HandleGetAllLoginActivities)
		r.Get("/user/{id}", app.LoginActivityHandler.HandleGetLoginActivitiesByUserID)
	})
}

func mobileRechargeRoutes(router *chi.Mux, app *app.Application) {
	router.Route("/mobile-recharge", func(r chi.Router) {
		r.Use(middlewares.AuthorizationMiddleware)

		r.Post("/create", app.MobileRechargeHandler.HandleCreateMobileRecharge)
		r.Post("/status-check/{id}", app.MobileRechargeHandler.HandleCheckMobileRechargeStatus)
		r.Post("/refund/{id}", app.MobileRechargeHandler.HandleRefundMobileRecharge)
		r.Get("/prepaid-plans", app.MobileRechargeHandler.HandleFetchPrepaidPlans)
		r.Post("/postpaid-bill", app.MobileRechargeHandler.HandleFetchPostpaidBill)
		r.Get("/all", app.MobileRechargeHandler.HandleGetAllMobileRecharge)
		r.Get("/retailer/{id}", app.MobileRechargeHandler.HandleGetMobileRechargeByRetailerID)
		r.Get("/distributor/{id}", app.MobileRechargeHandler.HandleGetMobileRechargeByDistributorID)
		r.Get("/md/{id}", app.MobileRechargeHandler.HandleGetMobileRechargeByMasterDistributorID)

		// Circle management
		r.Post("/circles", app.MobileRechargeHandler.HandleCreateMobileRechargeCircle)
		r.Put("/circles/{id}", app.MobileRechargeHandler.HandleUpdateMobileRechargeCircle)
		r.Delete("/circles/{id}", app.MobileRechargeHandler.HandleDeleteMobileRechargeCircle)
		r.Get("/circles", app.MobileRechargeHandler.HandleGetMobileRechargeCircles)

		// Operator management
		r.Post("/operators", app.MobileRechargeHandler.HandleCreateMobileRechargeOperator)
		r.Put("/operators/{id}", app.MobileRechargeHandler.HandleUpdateMobileRechargeOperator)
		r.Delete("/operators/{id}", app.MobileRechargeHandler.HandleDeleteMobileRechargeOperator)
		r.Get("/operators", app.MobileRechargeHandler.HandleGetMobileRechargeOperators)
	})
}
