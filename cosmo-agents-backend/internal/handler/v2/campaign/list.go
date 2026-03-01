package campaign

import (
	"context"
	"encoding/json"

	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// buildListItems converts CampaignWithStats rows to v2 response items
func (h *Handler) buildListItems(ctx context.Context, rows []campaignRepo.CampaignWithStats) ([]v2schema.CampaignListItem, error) {
	items := make([]v2schema.CampaignListItem, len(rows))

	for i, row := range rows {
		campaign := row.Campaign

		// Get creator name
		var creator *string
		if row.Creator.Valid {
			creator = &row.Creator.String
		}

		var agent *v2schema.RelationshipAgent
		if row.AgentID != nil && row.AgentName.Valid {
			agentID := *row.AgentID

			var metadata map[string]interface{}
			if len(row.AgentCMetadata) > 0 {
				if err := json.Unmarshal(row.AgentCMetadata, &metadata); err != nil {
					logger.Logger.Warn().Err(err).Msg("failed to unmarshal agent metadata")
				}
			}

			agent = &v2schema.RelationshipAgent{
				ID:        agentID,
				Name:      row.AgentName.String,
				CMetadata: metadata,
			}
		}

		// Use mapper to convert (same as v1)
		item := v2schema.ToCampaignListItem(&campaign, creator, row.Sent, row.Reply, row.Interested, row.ReplyRate, row.InterestRate)
		if agent != nil {
			item.Agent = agent
			item.Entity.Agent = agent
		}

		items[i] = item
	}

	return items, nil
}
