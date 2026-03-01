export type ApiResponse<T = any> = {
  data: T;
  status: string;
};

type PaginateData<T = any> = {
  list: T[];
  offset: number;
  limit: number;
  total: number;
};

export type PaginateResponse<T = any> = {
  data: PaginateData<T>;
  status: string;
};

export type BaseQuery = {
  offset?: number;
  limit?: number;
};
