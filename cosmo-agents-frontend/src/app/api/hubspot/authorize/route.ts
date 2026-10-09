import { NextResponse } from 'next/server';

export async function GET() {
  const clientID = process.env.HUBSPOT_CLIENT_ID!;
  const scope = [
    'oauth',
    'crm.objects.owners.read',
    'crm.objects.contacts.read',
    'crm.lists.read',
    'crm.schemas.contacts.read',
    'crm.schemas.custom.read',
    'crm.schemas.listings.read',
  ];
  const redirectURI = process.env.HUBSPOT_REDIRECT_URI!;

  return NextResponse.json({
    url: `https://app.hubspot.com/oauth/authorize?client_id=${clientID}&scope=${scope.join(' ')}&redirect_uri=${redirectURI}`,
  });
}
