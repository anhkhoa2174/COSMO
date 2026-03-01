import { useCampaign } from '@/app/(main)/campaigns/[campaign_id]/use-campaign';
import { useUser } from '@/hooks/use-user';

export default function useOrgByCampaign() {
  const [campaign] = useCampaign();
  const { user } = useUser();
  const foundOrganization = user?.organizations.find(
    (item) => item.id === campaign.organization_id
  ) as any;

  const organization = {
    company_description: foundOrganization?.company_description || '',
    company_targeting_persona:
      foundOrganization?.company_targeting_persona || '',
    company_url: foundOrganization?.company_url || '',
    name: foundOrganization?.name || '',
  };

  return [organization];
}
