export const promptAIReply = () => {
  return `
      Given the "intent_type", "subject", and "content" of the original email, craft a thoughtful and appropriate reply that fits the context and tone of the conversation. 
      
      Output like system prompt output include merge tags.   
      
      Example:
      <EmailContent>
      Hi {contact_first_name},
  
      Thanks for reaching out! I’d be happy to chat about how we can support your team at {contact_company}.
  
      Could you let me know your availability for a quick conversation? I look forward to exploring potential opportunities together.
  
      Best regards,
      {agent_signature}
      </EmailContent>
      `;
};

export const promptAIReplySample = () => {
  return `
    Given the "intent_type", along with the email's "subject" and "content", generate a natural-sounding question or statement that fits the context. Your response should feel human, appropriate, and relevant to the given input.

    Output only the message, wrapped in <EmailContent> tags.

    Example:
    <EmailContent>When can we get started?</EmailContent>
      `;
};
