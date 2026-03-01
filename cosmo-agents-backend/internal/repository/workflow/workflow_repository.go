package workflow

import (
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// WorkflowRepository provides access to Workflow entities.
type WorkflowRepository struct {
	*gormpkg.GormRepository[domain.Workflow]
	db *gorm.DB
}

// NewWorkflowRepository constructs a new gormpkg.
func NewWorkflowRepository(db *gorm.DB) *WorkflowRepository {
	return &WorkflowRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Workflow](db),
		db:             db,
	}
}
