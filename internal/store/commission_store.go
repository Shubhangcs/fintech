package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresCommisionStore struct {
	db *sql.DB
}

func NewPostgresCommisionStore(db *sql.DB) *PostgresCommisionStore {
	return &PostgresCommisionStore{db: db}
}

type CommisionStore interface {
	CreateCommision(c *models.CommisionModel) error
	UpdateCommision(c *models.CommisionModel) error
	DeleteCommision(commisionID int64) error
	GetAllCommisions(limit, offset int) ([]models.CommisionModel, error)
	GetCommisionByUserIDServiceAndAmount(userID, service string, amount float64) (*models.CommisionModel, error)
	GetDefaultCommision(amount float64) *models.CommisionModel
}

// Create Commision
func (cs *PostgresCommisionStore) CreateCommision(c *models.CommisionModel) error {
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
		c.TotalCommision,
		c.AdminCommision,
		c.MasterDistributorCommision,
		c.DistributorCommision,
		c.RetailerCommision,
	).Scan(&c.CommisionID, &c.CreatedAT, &c.UpdatedAT)
}

// Update Commision
func (cs *PostgresCommisionStore) UpdateCommision(c *models.CommisionModel) error {
	query := `
	UPDATE commisions
	SET total_commision              = COALESCE(NULLIF($1, 0), total_commision),
	    admin_commision              = COALESCE(NULLIF($2, 0), admin_commision),
	    master_distributor_commision = COALESCE(NULLIF($3, 0), master_distributor_commision),
	    distributor_commision        = COALESCE(NULLIF($4, 0), distributor_commision),
	    retailer_commision           = COALESCE(NULLIF($5, 0), retailer_commision),
	    updated_at                   = CURRENT_TIMESTAMP
	WHERE commision_id = $6;
	`
	res, err := cs.db.Exec(query,
		c.TotalCommision,
		c.AdminCommision,
		c.MasterDistributorCommision,
		c.DistributorCommision,
		c.RetailerCommision,
		c.CommisionID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Delete Commision
func (cs *PostgresCommisionStore) DeleteCommision(commisionID int64) error {
	res, err := cs.db.Exec(`DELETE FROM commisions WHERE commision_id = $1;`, commisionID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Get All Commisions
func (cs *PostgresCommisionStore) GetAllCommisions(limit, offset int) ([]models.CommisionModel, error) {
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

	var commisions []models.CommisionModel
	for rows.Next() {
		var c models.CommisionModel
		if err := rows.Scan(
			&c.CommisionID, &c.UserID, &c.Service,
			&c.TotalCommision, &c.AdminCommision, &c.MasterDistributorCommision,
			&c.DistributorCommision, &c.RetailerCommision,
			&c.CreatedAT, &c.UpdatedAT,
		); err != nil {
			return nil, err
		}
		commisions = append(commisions, c)
	}
	return commisions, rows.Err()
}

// Get Commision By UserID, Service And Amount
func (cs *PostgresCommisionStore) GetCommisionByUserIDServiceAndAmount(userID, service string, amount float64) (*models.CommisionModel, error) {
	query := `
		SELECT
			commision_id,
			user_id,
			service,
			total_commision,
			admin_commision,
			master_distributor_commision,
			distributor_commision,
			retailer_commision,
			created_at,
			updated_at
		FROM commisions
		WHERE user_id = $1
		AND service = $2;
	`
	var commision models.CommisionModel
	err := cs.db.QueryRow(
		query,
		userID,
		service,
	).Scan(
		&commision.CommisionID,
		&commision.UserID,
		&commision.Service,
		&commision.TotalCommision,
		&commision.AdminCommision,
		&commision.MasterDistributorCommision,
		&commision.DistributorCommision,
		&commision.RetailerCommision,
		&commision.CreatedAT,
		&commision.UpdatedAT,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var totalCommision = (commision.TotalCommision / 100) * amount

	return &models.CommisionModel{
		CommisionID:                commision.CommisionID,
		UserID:                     commision.UserID,
		Service:                    commision.Service,
		TotalCommision:             totalCommision,
		AdminCommision:             totalCommision * commision.AdminCommision,
		MasterDistributorCommision: totalCommision * commision.MasterDistributorCommision,
		DistributorCommision:       totalCommision * commision.DistributorCommision,
		RetailerCommision:          totalCommision * commision.RetailerCommision,
		CreatedAT:                  commision.CreatedAT,
		UpdatedAT:                  commision.UpdatedAT,
	}, nil
}

// Get Default Commision
func (cs *PostgresCommisionStore) GetDefaultCommision(amount float64) *models.CommisionModel {
	var totalCommision = (1.2 / 100) * amount
	return &models.CommisionModel{
		TotalCommision:             totalCommision,
		AdminCommision:             totalCommision * 0.25,
		MasterDistributorCommision: totalCommision * 0.05,
		DistributorCommision:       totalCommision * 0.20,
		RetailerCommision:          totalCommision * 0.50,
	}
}
