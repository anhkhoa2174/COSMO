import type { EmailIntent } from '@/models/email';

/**
 * FALLBACK/SAMPLE DATA - Email Reply Templates
 *
 * These are sample email templates used for:
 * 1. UI preview/demonstration purposes
 * 2. Fallback when API template generation fails
 * 3. Testing/development scenarios
 *
 * In production, prefer fetching templates from:
 * - GET /v1/template (list all templates)
 * - POST /v1/campaigns/:id/generate (generate templates)
 * - POST /v1/ai/emails/generate (AI-generated email templates)
 */
export const sampleTemplatesReply: Record<EmailIntent, string[]> = {
  'Do not contact': [
    "Hi {contact_first_name},\n\nGot it — we’ll make sure you're removed from our contact list.\n\nWishing you well,\n{agent_signature}",
    'Hello {contact_first_name},\n\nThanks for confirming. We won’t reach out again.\n\nTake care,\n{agent_signature}',
    'Hi {contact_first_name},\n\nUnderstood. You won’t receive further communication from us.\n\nAll the best,\n{agent_signature}',
    "Hi {contact_first_name},\n\nThanks for your message — we've noted your preference and will no longer contact you.\n\nWarm regards,\n{agent_signature}",
    'Hi {contact_first_name},\n\nYour request has been acknowledged. We’ll respect your wish and remove you from our outreach.\n\nRegards,\n{agent_signature}',
  ],
  Interested: [
    'Hi {contact_first_name},\n\nAwesome to hear from you! I’d love to dive deeper into how we can help with {mentioned_need}.\n\nLet me loop in {sales_rep_first_name} to find a time that works for a quick chat.\n\nCheers,\n{agent_signature}',
    "Hello {contact_first_name},\n\nGlad to see your interest! Let's schedule a time to walk through what we can offer {contact_company}.\n\nWould any of these work?\n- {time_1}\n- {time_2}\n- {time_3}\n\nLooking forward!\n\nBest,\n{agent_signature}",
    'Hi {contact_first_name},\n\nThanks for showing interest! Based on your needs, we can definitely help out.\n\n{sales_rep_first_name} will follow up shortly to schedule something.\n\nCheers,\n{agent_signature}',
    'Hi {contact_first_name},\n\nGreat hearing from you. Let’s set up a time to explore how we can support {contact_company}.\n\nHow about:\n- {time_1}\n- {time_2}\n- {time_3}\n\nTalk soon!\n\nRegards,\n{agent_signature}',
    "Hi {contact_first_name},\n\nI’m thrilled you're interested! I’ve asked {sales_rep_first_name} to connect with you about scheduling.\n\nLet’s chat soon!\n\nAll the best,\n{agent_signature}",
  ],
  'Not interested': [
    'Hi {contact_first_name},\n\nThanks for your response. We totally understand and appreciate your time.\n\nIf anything changes, we’re just an email away.\n\nWarm regards,\n{agent_signature}',
    'Hello {contact_first_name},\n\nNo worries at all, and thank you for letting us know.\n\nWe’ll be here if you revisit this in the future.\n\nBest,\n{agent_signature}',
    'Hi {contact_first_name},\n\nAppreciate the update. We won’t reach out again unless we hear from you.\n\nWishing you well,\n{agent_signature}',
    'Hey {contact_first_name},\n\nUnderstood — thanks again for your time. If your situation ever shifts, feel free to reconnect.\n\nRegards,\n{agent_signature}',
    'Hi {contact_first_name},\n\nGot your message and appreciate the honesty. No pressure — we’re here if things change later.\n\nTake care,\n{agent_signature}',
  ],
  'Out of office': [
    'Hi {contact_first_name},\n\nThanks for reaching out. I’m currently away and will be back on {return_date}.\n\nFor urgent matters, please contact {alternate_contact_name} at {alternate_contact_email}.\n\nBest,\n{agent_signature}',
    "Hello {contact_first_name},\n\nI'm out of the office until {return_date}. For now, {alternate_contact_name} ({alternate_contact_email}) can assist.\n\nI’ll reply when I return!\n\nCheers,\n{agent_signature}",
    "Hi {contact_first_name},\n\nI'm away right now but will get back to you as soon as I return on {return_date}.\n\nIf it’s urgent, feel free to email {alternate_contact_email}.\n\nBest,\n{agent_signature}",
    'Hi {contact_first_name},\n\nCurrently out of office until {return_date}. Please reach out to {alternate_contact_name} at {alternate_contact_email} if immediate help is needed.\n\nThanks!\n\n{agent_signature}',
    'Hi {contact_first_name},\n\nThanks for writing. I’ll be back on {return_date}. In the meantime, {alternate_contact_email} is your best contact.\n\nTalk soon,\n{agent_signature}',
  ],
  'Request for information': [
    "Hi {contact_first_name},\n\nThanks for reaching out! Could you tell me more about what you're hoping to learn?\n\nIn the meantime, here’s an overview:\n- {key_point_1}\n- {key_point_2}\n- {key_point_3}\n\nLooking forward,\n{agent_signature}",
    'Hello {contact_first_name},\n\nHappy to help! Can you clarify your main goals so I can send the most helpful info?\n\nHere’s a bit to start:\n- {key_point_1}\n- {key_point_2}\n- {key_point_3}\n\nCheers,\n{agent_signature}',
    'Hi {contact_first_name},\n\nAppreciate you reaching out! If you can share what you’re looking for, I’ll tailor my response.\n\nMeanwhile:\n- {key_point_1}\n- {key_point_2}\n- {key_point_3}\n\nBest,\n{agent_signature}',
    'Hi {contact_first_name},\n\nThanks for the note! Here’s some general info below. Let me know if you need anything more specific:\n- {key_point_1}\n- {key_point_2}\n- {key_point_3}\n\nBest,\n{agent_signature}',
    "Hi {contact_first_name},\n\nGreat to hear from you. If you share what you're focused on, I’ll send targeted info.\n\nFor now, here's an overview:\n- {key_point_1}\n- {key_point_2}\n- {key_point_3}\n\nTalk soon,\n{agent_signature}",
  ],
  'Request for pricing': [
    'Hi {contact_first_name},\n\nGlad to assist! Based on your needs, the {recommended_tier} package might be ideal.\n\nIt includes:\n- {key_feature_1}\n- {key_feature_2}\n- {key_feature_3}\n\nStarting at {price_point}/month.\n\nLet’s schedule a quick call?\n\nBest,\n{agent_signature}',
    'Hello {contact_first_name},\n\nThanks for asking! Our {recommended_tier} plan starts at {price_point}/month and includes:\n- {key_feature_1}\n- {key_feature_2}\n- {key_feature_3}\n\nWant to connect this week?\n\nCheers,\n{agent_signature}',
    'Hi {contact_first_name},\n\nAppreciate the interest in pricing. The {recommended_tier} package is often a strong fit for teams in {contact_industry}.\n\nIt offers:\n- {key_feature_1}\n- {key_feature_2}\n- {key_feature_3}\n\nLet’s talk more soon!\n\n{agent_signature}',
    'Hi {contact_first_name},\n\nHappy to provide pricing info. Based on what you need, {recommended_tier} starts at {price_point}/mo and gives you:\n- {key_feature_1}\n- {key_feature_2}\n- {key_feature_3}\n\nLet me know if you’d like to discuss further!\n\n{agent_signature}',
    'Hi {contact_first_name},\n\nThanks for your interest in our pricing. Here’s what the {recommended_tier} package includes:\n- {key_feature_1}\n- {key_feature_2}\n- {key_feature_3}\n\nIt starts at {price_point}/month. Want to hop on a quick call?\n\nBest,\n{agent_signature}',
  ],
  Referral: [
    'Hi {contact_first_name},\n\nThank you for pointing me to the right person — I appreciate it. I will reach out to them and mention you suggested it.\n\n{agent_signature}',
    "Hello {contact_first_name},\n\nThanks for the introduction. I'll follow up with them directly and keep you out of the loop unless you'd like to stay copied.\n\n{agent_signature}",
  ],
  Nurture: [
    "Hi {contact_first_name},\n\nUnderstood — the timing isn't right just now. I'll check back in a few months; in the meantime, feel free to reach out if anything changes.\n\n{agent_signature}",
    'Hello {contact_first_name},\n\nThanks for letting me know. I will keep you posted on anything relevant and reconnect later in the year.\n\n{agent_signature}',
  ],
  'Unknown intent': [
    'Hi {contact_first_name},\n\nThanks for reaching out! Could you clarify what you’re looking for so I can assist properly?\n\nBest,\n{agent_signature}',
    'Hello {contact_first_name},\n\nAppreciate your message. Can you share a bit more about your needs?\n\nHappy to help once I know more!\n\nCheers,\n{agent_signature}',
    'Hi {contact_first_name},\n\nI’m here to assist! If you can provide a bit more context, I’ll be able to point you in the right direction.\n\nThanks,\n{agent_signature}',
    "Hey {contact_first_name},\n\nGot your message — could you clarify what you'd like help with?\n\nLooking forward,\n{agent_signature}",
    'Hi {contact_first_name},\n\nThanks for the note! To help you best, could you elaborate on your request?\n\nBest,\n{agent_signature}',
  ],
};

/**
 * FALLBACK/SAMPLE DATA - Human Reply Samples
 *
 * These are sample human responses used for:
 * 1. Testing AI reply generation in demo/preview mode
 * 2. Simulating user responses during development
 *
 * These should NOT be used in production email flows.
 */
export const sampleTemplatesHumanReply: Record<EmailIntent, string[]> = {
  'Do not contact': [
    'Please remove me from your mailing list.',
    'I’m not interested and prefer not to receive further emails.',
    'Can you stop contacting me?',
    'Unsubscribe me, please.',
    'No more follow-ups, thank you.',
  ],
  Interested: [
    'I’d love to hear more about your solution.',
    'Can we set up a meeting to discuss this further?',
    'This sounds interesting, can you share more details?',
    "I'd like to explore how this could help our team.",
    'What are the next steps to get started?',
  ],
  'Not interested': [
    'Thanks, but we’re not interested at the moment.',
    'This isn’t something we need right now.',
    "Appreciate the message, but we'll pass.",
    'We’ve gone with another provider.',
    'It’s not a priority for us currently.',
  ],
  'Out of office': [
    "I'm currently out of the office until next week.",
    'Please contact my colleague while I’m away.',
    'I’m on vacation and will respond upon return.',
    'I’m out and will check emails when I’m back.',
    'Currently unavailable, but I’ll reply once I return.',
  ],
  'Request for information': [
    'Can you send me more details about your services?',
    'What exactly does your platform offer?',
    'Can you provide a brochure or overview?',
    'I’d like to understand how your solution works.',
    'What features are included in your product?',
  ],
  'Request for pricing': [
    'Can you share your pricing tiers?',
    'How much does this cost monthly?',
    'Do you offer volume discounts?',
    'What’s your most popular package and its cost?',
    'Can you send me a quote for our team size?',
  ],
  Referral: [
    'Hi {contact_first_name},\n\nThank you for pointing me to the right person — I appreciate it. I will reach out to them and mention you suggested it.\n\n{agent_signature}',
    "Hello {contact_first_name},\n\nThanks for the introduction. I'll follow up with them directly and keep you out of the loop unless you'd like to stay copied.\n\n{agent_signature}",
  ],
  Nurture: [
    "Hi {contact_first_name},\n\nUnderstood — the timing isn't right just now. I'll check back in a few months; in the meantime, feel free to reach out if anything changes.\n\n{agent_signature}",
    'Hello {contact_first_name},\n\nThanks for letting me know. I will keep you posted on anything relevant and reconnect later in the year.\n\n{agent_signature}',
  ],
  'Unknown intent': [
    'Just checking in, wanted to ask about something.',
    'Quick question about your services.',
    'Looking into options, not sure what fits yet.',
    'Can we talk about something related to your tool?',
    'Had a general question, wasn’t sure who to ask.',
  ],
};
