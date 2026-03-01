import { kyClient } from '@/lib/ky';
import type { CustomField } from '@/models/custom-field';
import type { PaginateResponse } from '@/models/response';

function getCustomFields(params?: any) {
  return kyClient.get<PaginateResponse<CustomField>>('v1/custom-fields', { searchParams: params }).json();
}

function createCustomField(body: CustomField) {
  return kyClient.post('v1/custom-fields', { json: body }).json();
}

function updateCustomField(custom_field_id: string, body: CustomField) {
  return kyClient.patch(`v1/custom-fields/${custom_field_id}`, { json: body }).json();
}

function deleteCustomField(custom_field_id: string) {
  return kyClient.delete(`v1/custom-fields/${custom_field_id}`).json();
}

export { getCustomFields, createCustomField, updateCustomField, deleteCustomField };
