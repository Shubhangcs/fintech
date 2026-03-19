package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresCommissionStore struct {
	db *sql.DB
}

func NewPostgresCommissionStore(db *sql.DB) *PostgresCommissionStore {
	return &PostgresCommissionStore{db: db}
}

type CommissionStore interface {
	CreateCommission(c *models.CommissionModel) error
	UpdateCommission(commissionID int64, c *models.CommissionModel) error
	DeleteCommission(commissionID int64) error
	GetCommissions(limit, offset int) ([]models.CommissionModel, error)
}

func (cs *PostgresCommissionStore) CreateCommission(c *models.CommissionModel) error {
	query := `
	INSERT INTO commisions (
		user_id,
		service,
		total_commision,
		admin_commision,
		master_distributor_commision,
		distributor_commision,
		retailer_commision
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING commision_id, created_at, updated_at;
	`
	return cs.db.QueryRow(query,
		c.UserID,
		c.Service,
		c.TotalCommission,
		c.AdminCommission,
		c.MasterDistributorCommission,
		c.DistributorCommission,
		c.RetailerCommission,
	).Scan(&c.CommissionID, &c.CreatedAT, &c.UpdatedAT)
}

func (cs *PostgresCommissionStore) UpdateCommission(commissionID int64, c *models.CommissionModel) error {
	query := `
	UPDATE commisions
	SET total_commision              = $1,
	    admin_commision              = $2,
	    master_distributor_commision = $3,
	    distributor_commision        = $4,
	    retailer_commision           = $5,
	    updated_at                   = CURRENT_TIMESTAMP
	WHERE commision_id = $6;
	`
	res, err := cs.db.Exec(query,
		c.TotalCommission,
		c.AdminCommission,
		c.MasterDistributorCommission,
		c.DistributorCommission,
		c.RetailerCommission,
		commissionID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (cs *PostgresCommissionStore) DeleteCommission(commissionID int64) error {
	res, err := cs.db.Exec(`DELETE FROM commisions WHERE commision_id = $1;`, commissionID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (cs *PostgresCommissionStore) GetCommissions(limit, offset int) ([]models.CommissionModel, error) {
	query := `
	SELECT
		commision_id, user_id, service,
		total_commision, admin_commision, master_distributor_commision,
		distributor_commision, retailer_commision,
		created_at, updated_at
	FROM commisions
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2;
	`
	rows, err := cs.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commissions []models.CommissionModel
	for rows.Next() {
		var c models.CommissionModel
		if err := rows.Scan(
			&c.CommissionID, &c.UserID, &c.Service,
			&c.TotalCommission, &c.AdminCommission, &c.MasterDistributorCommission,
			&c.DistributorCommission, &c.RetailerCommission,
			&c.CreatedAT, &c.UpdatedAT,
		); err != nil {
			return nil, err
		}
		commissions = append(commissions, c)
	}
	return commissions, rows.Err()
}
