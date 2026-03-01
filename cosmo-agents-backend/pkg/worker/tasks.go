package worker

// Task type constants
const (
	// Email tasks
	TypeSendEmail             = "email:send"
	TypeProcessIncoming       = "email:process_incoming"
	TypeSyncGmailHistory      = "email:sync_history"
	TypeSendInviteMemberEmail = "email:send_invite_member_email"
	TypeGmailNotification     = "gmail:notification"

	// Campaign tasks
	TypeExecuteCampaign = "campaign:execute"
	TypeGenerateEmail   = "campaign:generate_email"
	TypeScheduleTasks   = "campaign:schedule_tasks"

	// Agent tasks
	TypeSyncAgent        = "agent:sync"
	TypeRefreshToken     = "agent:refresh_token"
	TypeCheckCredentials = "agent:check_credentials"

	// Contact tasks
	TypePullHubspotContacts     = "contact:pull_hubspot"
	TypePullHubspotListContacts = "contact:pull_hubspot_lists"
	TypeEnrichContact           = "contact:enrich"
	TypeSyncContact             = "contact:sync"

	// Knowledge tasks
	TypeProcessKnowledge  = "knowledge:process"
	TypeGenerateEmbedding = "knowledge:generate_embedding"

	// Playbook automation tasks
	TypePlaybookEvaluateRules      = "playbook:evaluate_rules"
	TypePlaybookProcessEnrollments = "playbook:process_enrollments"

	// Segmentation tasks
	TypeRecalculateSegmentScores = "segmentation:recalculate_scores"

	// Relationship tasks
	TypeRecalculateRelationshipScores = "relationship:recalculate_scores"

	// Outreach tasks
	TypeRecalculateOutreachNextStep = "outreach:recalculate_next_step"

	// Orchestrator tasks
	TypeOrchestrateContact = "orchestrator:contact"
)

// Queue names
const (
	QueueCritical = "critical" // Time-sensitive tasks
	QueueDefault  = "default"  // Normal priority
	QueueLow      = "low"      // Background tasks
)
