import type { ApiResponse, PaginateResponse } from './response';

export type SalesRep = {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  calendar_link: string;
  picture?: string;
  created_at: string;
  updated_at: string;
};

// search
export type SalesRepSearchRequest = {
  filter: any;
};

type SalesRepSearchResponseData = {
  entity: Omit<SalesRep, 'created_at' | 'updated_at'>;
};
export type SalesRepSearchResponse =
  PaginateResponse<SalesRepSearchResponseData>;

// create
export type SalesRepCreateRequest = Omit<
  SalesRep,
  'id' | 'created_at' | 'updated_at'
> & {
  picture?: string;
  calendar_link?: string;
};

type SalesRepCreateResponseData = SalesRep;
export type SalesRepCreateResponse = ApiResponse<SalesRepCreateResponseData>;
