package domain

import "time"

type Account struct {
	ID             string    `json:"id"              db:"id"`
	DocumentNumber string    `json:"document_number" db:"document_number"`
	CreatedOn      time.Time `json:"created_on"      db:"created_on"`
}

type OperationType struct {
	ID          int       `json:"id"          db:"id"`
	Description string    `json:"description" db:"description"`
	CreatedOn   time.Time `json:"created_on"  db:"created_on"`
}

type Transaction struct {
	ID              string           `json:"id"                db:"id"`
	AccountID       string           `json:"account_id"        db:"account_id"`
	OperationTypeID int              `json:"operation_type_id" db:"operation_type_id"`
	Amount          float64          `json:"amount"            db:"amount"`
	EventDate       time.Time        `json:"event_date"        db:"event_date"`
	CreatedOn       time.Time        `json:"created_on"        db:"created_on"`
	InstallmentPlan *InstallmentPlan `json:"installment_plan,omitempty" db:"-"`
}

type InstallmentPlan struct {
	ID                string                `json:"id"                 db:"id"`
	TransactionID     string                `json:"transaction_id"     db:"transaction_id"`
	TotalInstallments int                   `json:"total_installments" db:"total_installments"`
	Schedules         []InstallmentSchedule `json:"schedules"          db:"-"`
	CreatedOn         time.Time             `json:"created_on"         db:"created_on"`
}

type InstallmentSchedule struct {
	ID                string    `json:"id"                  db:"id"`
	InstallmentPlanID string    `json:"installment_plan_id" db:"installment_plan_id"`
	InstallmentNo     int       `json:"installment_no"      db:"installment_no"`
	Amount            float64   `json:"amount"              db:"amount"`
	DueDate           time.Time `json:"due_date"            db:"due_date"`
	CreatedOn         time.Time `json:"created_on"          db:"created_on"`
}
