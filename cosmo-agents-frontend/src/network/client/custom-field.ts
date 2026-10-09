import { kyClient } from '@/lib/ky';
import { getValuable } from '@/lib/utils';
import type { CustomField } from '@/models/custom-field';
import type { PaginateResponse } from '@/models/response';

// Callers often pass this straight as a react-query queryFn, which hands it
// the query context ({ queryKey, signal, ... }); only forward the real filters.
// Every consumer (contact form, CSV/HubSpot mapping) needs the full list, so
// ask for the backend's maximum page instead of its default of 25.
function getCustomFields(params?: any) {
  const { entity_type, offset, limit = 100 } = params ?? {};
  return kyClient
    .get<PaginateResponse<CustomField>>('v1/custom-fields', {
      searchParams: getValuable({ entity_type, offset, limit }),
    })
    .json();
}

function createCustomField(body: CustomField) {
  return kyClient.post('v1/custom-fields', { json: body }).json();
}

function updateCustomField(custom_field_id: string, body: CustomField) {
  return kyClient
    .patch(`v1/custom-fields/${custom_field_id}`, { json: body })
    .json();
}

function deleteCustomField(custom_field_id: string) {
  return kyClient.delete(`v1/custom-fields/${custom_field_id}`).json();
}

export {
  getCustomFields,
  createCustomField,
  updateCustomField,
  deleteCustomField,
};
