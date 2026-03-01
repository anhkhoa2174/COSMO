package segmentation

import (
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	segmentationRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// Interfaces kept for parity; wiring is done in dependencies.
type Dependencies struct {
	SegmentationRepo *segmentationRepo.SegmentationRepository
	ScoreRepo        *segmentationRepo.ScoreRepository
	ContactRepo      *contactRepo.ContactRepository
	UserRepo         *userRepo.UserRepository
	RoleRepo         *roleRepo.RoleRepository
}
