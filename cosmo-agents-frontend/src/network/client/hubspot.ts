import { kyClient } from '@/lib/ky';
import { ApiResponse } from '@/models/response';
import ky from 'ky';

type HubspotProperty = {
  updatedAt: string;
  createdAt: string;
  name: string; // restricted type
  label: string;
  type:
    | 'string'
    | 'number'
    | 'datetime'
    | 'enumeration'
    | 'bool'
    | 'date'
    | 'object_coordinates'
    | 'phone_number';
  fieldType:
    | 'text'
    | 'number'
    | 'date'
    | 'booleancheckbox'
    | 'phonenumber'
    | 'select'
    | 'checkbox'
    | 'calculation_rollup'
    | 'textarea'
    | 'calculation_equation'
    | 'calculation_read_time'
    | 'radio'
    | 'calculation_score';
  description: string;
  groupName: string; // restricted type
  options: any[];
  displayOrder: 6;
  calculated: false;
  externalOptions: false;
  hasUniqueValue: false;
  hidden: false;
  hubspotDefined: true;
  modificationMetadata: {
    archivable: true;
    readOnlyDefinition: true;
    readOnlyValue: false;
  };
  formField: true;
  dataSensitivity: 'non_sensitive';
};

type HubspotList = {
  listId: string;
  listVersion: number;
  createdAt: string;
  updatedAt: string;
  filtersUpdatedAt: string;
  processingStatus: 'COMPLETE' | 'IN_PROGRESS' | 'ERROR';
  createdById: string;
  updatedById: string;
  processingType: 'MANUAL' | 'SNAPSHOT' | 'DYNAMIC';
  objectTypeId: '0-1';
  name: string;
  additionalProperties: {
    hs_last_record_removed_at?: string;
    hs_last_record_added_at?: string;
    hs_list_reference_count: string;
    hs_list_size: string;
  };
};

type HubspotOwner = {
  id: string;
  email: string;
  type: 'PERSON';
  firstName: string;
  lastName: string;
  userId: number;
  userIdIncludingInactive: number;
  createdAt: string;
  updatedAt: string;
  archived: boolean;
};

type ListsResponse = {
  offset: number;
  hasMore: boolean;
  lists: HubspotList[];
  total: number;
};

type PropertiesResponse = {
  results: HubspotProperty[];
};

type OwnersResponse = {
  results: HubspotOwner[];
};

type AuthorizeResponse = {
  url: string;
};

type TokenResponse = {
  token_type: string;
  access_token: string;
  refresh_token: string;
  expires_in: number;
};

const api = ky.create({
  prefixUrl: process.env.NEXT_PUBLIC_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
});

const HubspotApi = {
  getAuthURL: () => {
    return api.get('api/hubspot/authorize').json<AuthorizeResponse>();
  },
  getToken: (body: any) => {
    return api.post('api/hubspot/token', { json: body }).json<TokenResponse>();
  },
  processCallback: (params: any) => {
    return kyClient.get('v2/hubspot/callback', { searchParams: params }).json<ApiResponse<TokenResponse>>();
  },
  getOwners: (accessToken: string, refreshToken: string) => {
    return api.get('api/hubspot/owners', { searchParams: { accessToken, refreshToken }}).json<OwnersResponse>();
  },
  searchLists: (accessToken: string, refreshToken: string) => {
    return api.get(`api/hubspot/lists`, { searchParams: { accessToken, refreshToken }}).json<ListsResponse>();
  },
  getProperties: (accessToken: string, refreshToken: string) => {
    return api.get(`api/hubspot/properties`, { searchParams: { accessToken, refreshToken }}).json<PropertiesResponse>();
  },
  getMe: () => {
    return kyClient.get(`v2/hubspot/users/me`).json<ApiResponse<any>>();
  },
  updateMe: (body: any) => {
    return kyClient.patch(`v2/hubspot/users/me`, { json: body }).json<ApiResponse<any>>();
  }
};

export { HubspotApi };
export type { HubspotProperty, HubspotList, HubspotOwner };
