import type { Contact } from '@/models/contact';

/** Placeholders the importers write when a real value is unknown. */
function isPlaceholder(value: string) {
  return (
    value.includes('@linkedin.placeholder') ||
    value.includes('@na.local') ||
    value === 'N/A' ||
    value.startsWith('unknown-')
  );
}

function firstUsable(candidates: unknown[]): string | undefined {
  return candidates.find(
    (value): value is string =>
      typeof value === 'string' && value.trim() !== '' && !isPlaceholder(value)
  );
}

/**
 * `contact_information` is not part of CreateContactRequest on the backend, so
 * a value sent under that name is swallowed by ExtraFields and lands in
 * profile.custom_fields instead of its own column. Read it from there too.
 */
function fromCustomFields(contact: Contact, key: string): string | undefined {
  const field = (contact.profile?.custom_fields as any)?.[key];
  return typeof field === 'string' ? field : field?.value;
}

/**
 * The contact's primary reachable identifier — email, phone or profile URL,
 * whichever the record actually carries.
 */
export function resolveContactInfo(contact: Contact): string | undefined {
  return firstUsable([
    contact.contact_information,
    fromCustomFields(contact, 'contact_information'),
    contact.profile?.linkedin_url,
    contact.profile?.email,
    contact.email,
    contact.profile?.phone,
    contact.phone,
  ]);
}

/** Email only — never falls back to a phone number or a profile URL. */
export function resolveContactEmail(contact: Contact): string | undefined {
  const candidate = firstUsable([
    contact.email,
    contact.profile?.email,
    contact.contact_information,
    fromCustomFields(contact, 'email'),
    fromCustomFields(contact, 'contact_information'),
  ]);
  return candidate?.includes('@') ? candidate : undefined;
}

/** Phone only — the contact_information column may hold a number instead. */
export function resolveContactPhone(contact: Contact): string | undefined {
  const candidate = firstUsable([
    contact.phone,
    contact.profile?.phone,
    fromCustomFields(contact, 'phone'),
    contact.contact_information,
    fromCustomFields(contact, 'contact_information'),
  ]);
  if (!candidate || candidate.includes('@')) return undefined;
  // A profile URL is not a phone number either.
  if (candidate.startsWith('http') || candidate.includes('linkedin.com')) {
    return undefined;
  }
  return /\d/.test(candidate) ? candidate : undefined;
}

/** True when the value should render as a link rather than plain text. */
export function isProfileUrl(value: string) {
  return value.startsWith('http') || value.includes('linkedin.com');
}

/** Strips the LinkedIn prefix and trailing slash for display. */
export function shortenProfileUrl(value: string) {
  return value.replace('https://www.linkedin.com/in/', '').replace(/\/$/, '');
}
