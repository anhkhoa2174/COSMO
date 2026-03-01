'use client';

import { ContentLayout } from '@/components/nav/content-layout';
import { useGetCampaignQuery } from '@/network/client/campaign';

export default function CampaignDetailPage({
  params,
}: {
  params: { campaign_id: string };
}) {
  const { data, isLoading, error } = useGetCampaignQuery(params.campaign_id);
  const campaign = data?.data;

  return (
    <ContentLayout title="Campaign Details" description="View campaign information">
      {isLoading && <p className="p-4">Loading...</p>}
      {!isLoading && error && (
        <p className="p-4 text-destructive">Failed to load campaign.</p>
      )}
      {!isLoading && !error && !campaign && (
        <p className="p-4">Campaign not found.</p>
      )}
      {!isLoading && !error && campaign && (
        <div className="space-y-2 rounded-md border p-4">
          <div>
            <span className="font-medium">Name:</span> {campaign.name}
          </div>
          <div>
            <span className="font-medium">Status:</span> {campaign.status}
          </div>
          <div>
            <span className="font-medium">ID:</span> {campaign.id}
          </div>
          <div>
            <span className="font-medium">Created:</span> {campaign.created_at}
          </div>
          <div>
            <span className="font-medium">Updated:</span> {campaign.updated_at}
          </div>
        </div>
      )}
    </ContentLayout>
  );
}
