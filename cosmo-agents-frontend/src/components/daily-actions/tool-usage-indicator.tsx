'use client';

/**
 * ToolUsageIndicator
 *
 * Renders a subtle "Used: X, Y, Z" line below an assistant message
 * when the BD Agent invoked SDK tools to answer the query.
 *
 * FR-031: Surface which tools were invoked without disrupting chat flow.
 */

const TOOL_DISPLAY_NAMES: Record<string, string> = {
  search_contacts: 'Search Contacts',
  get_contact: 'Get Contact',
  create_contact: 'Create Contact',
  enrich_contact: 'Enrich Contact',
  suggest_outreach: 'Suggest Outreach',
  get_outreach_state: 'Outreach State',
  update_outreach: 'Update Outreach',
  generate_outreach_draft: 'Generate Draft',
  batch_generate_outreach_draft: 'Batch Drafts',
  add_interaction: 'Log Interaction',
  get_interaction_history: 'Interaction History',
  search_interactions: 'Search Interactions',
  get_meetings: 'Get Meetings',
  create_meeting: 'Create Meeting',
  update_meeting: 'Update Meeting',
  generate_meeting_prep: 'Meeting Prep',
  list_segments: 'List Segments',
  get_segment_contacts: 'Segment Contacts',
  create_segment: 'Create Segment',
  calculate_segment_scores: 'Segment Scores',
  analyze_segment_health: 'Segment Health',
  list_playbooks: 'List Playbooks',
  get_playbook: 'Get Playbook',
  enroll_contact_in_playbook: 'Enroll in Playbook',
  recommend_contacts_for_playbook: 'Playbook Recommendations',
  daily_analytics: 'Analytics',
  count_contacts_created: 'Contact Count',
  run_full_analysis: 'Full Analysis',
  calculate_relationship_score: 'Relationship Score',
  search_knowledge: 'Search Knowledge',
  apollo_people_search: 'Apollo People Search',
  apollo_organization_search: 'Apollo Org Search',
  apollo_people_enrichment: 'Apollo Enrichment',
  import_apollo_contacts_to_cosmo: 'Import from Apollo',
  vector_search_contacts: 'Semantic Search',
  hybrid_search_contacts: 'Hybrid Search',
};

export function formatToolName(toolName: string): string {
  return TOOL_DISPLAY_NAMES[toolName] ?? toolName.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

interface ToolUsageIndicatorProps {
  toolsUsed: string[];
}

export function ToolUsageIndicator({ toolsUsed }: ToolUsageIndicatorProps) {
  if (!toolsUsed || toolsUsed.length === 0) return null;

  const unique = [...new Set(toolsUsed)];

  return (
    <div className="mt-2 flex items-center gap-1.5 text-[0.65rem] text-muted-foreground/80">
      <svg
        className="h-3 w-3 shrink-0"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={2}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z"
        />
      </svg>
      <span>
        Used: {unique.map(formatToolName).join(', ')}
      </span>
    </div>
  );
}
